package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/pack"
)

// removal is one thing uninstall deletes: what it tells the user, and how it deletes it.
type removal struct {
	what string
	run  func() error
	// tier orders the steps: 0 removes hooks, 1 removes pack rules, 2 removes what the earlier steps may still
	// need (the login and portal rules, the pack state, the skills those rules require). Each tier runs in plan
	// order and is skipped once any step has failed, so a hook that stays keeps its rules.
	tier int
}

type uninstallFlags struct {
	project, global, keepSkills, yes, hookOnly bool
	rest                                       []string // flags for the old hook-only uninstall
}

func parseUninstallFlags(args []string) (uninstallFlags, error) {
	var f uninstallFlags
	for _, a := range args {
		switch a {
		case "--project":
			f.project = true
		case "--global":
			f.global = true
		case "--keep-skills":
			f.keepSkills = true
		case "--yes", "-y":
			f.yes = true
		case "--hook-only":
			f.hookOnly = true
		case "--agent":
			// The full uninstall covers Claude Code; another harness's hook goes through the hook-only path.
			f.hookOnly = true
			f.rest = append(f.rest, a)
		default:
			f.rest = append(f.rest, a)
		}
	}
	if !f.hookOnly && len(f.rest) > 0 {
		return f, fmt.Errorf("unknown flag %q", f.rest[0])
	}
	return f, nil
}

// Uninstall removes everything paws added: `paws uninstall [--project] [--global] [--keep-skills] [--yes]`.
// It lists what it will remove and asks first. With --hook-only it does what uninstall used to: it removes the
// hook from one settings.json and nothing else.
func Uninstall(args []string) int {
	f, err := parseUninstallFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\nusage: unbound-paws uninstall [--project] [--global] [--keep-skills] [--yes]\n       unbound-paws uninstall --hook-only [--agent codex] [--global | --settings <path>]\n", err)
		return 2
	}
	if f.hookOnly {
		if f.keepSkills || f.yes || f.project {
			fmt.Fprintln(os.Stderr, "paws: --hook-only takes only --global or --settings <path>; the project's settings file is the default")
			return 2
		}
		if f.global {
			f.rest = append(f.rest, "--global")
		}
		return uninstallHookOnly(f.rest)
	}
	c := newChooser()
	agents := map[harness.Adapter]bool{} // the harnesses to remove paws from; empty is all of them
	if !f.project && !f.global && !f.yes && interactive() {
		var err error
		if agents, f.project, f.global, err = askUninstall(c); err != nil {
			fmt.Fprintln(os.Stderr, "paws: nothing removed")
			return 1
		}
	}
	all := len(agents) == 0
	claude := all || agents[defaultAdapter()]
	if !f.project && !f.global {
		f.project, f.global = true, true
	}
	cwd, _ := os.Getwd()
	root := projectRoot(cwd)
	// An agent left unticked that still runs paws keeps what every agent shares: then Claude Code loses only its hook.
	staying := ""
	for _, a := range adapters {
		if a != defaultAdapter() && !all && !agents[a] && (hasPawsHook(a.HookFile(false, root)) || hasPawsHook(a.HookFile(true, ""))) {
			staying = a.Title()
		}
	}

	var plan []removal
	var keep []string
	requiredCache = nil
	user := pack.UserTarget(gate.UserDir())
	// Outside a repo, in the home folder, some or all of the project's files are the global ones. Each part that
	// is shared stays with the global scope.
	shared := overlaps(root, pack.ProjectTarget(root), user)
	if f.project && !f.global && shared.hooks && shared.skills && shared.packs {
		fmt.Fprintln(os.Stderr, "paws: this folder is not in a repo, so its project files are your global ones. Run --project from a repo, or use --global.")
		return 2
	}
	// What every agent shares (the login, portal and pack rules, pack state, the skills paws installed) goes with
	// Claude Code's part, unless an agent that stays still runs paws: then Claude Code loses only its hook. A
	// Codex-only setup has no Claude Code hook to keep it, so it goes with Codex's.
	claudeHooked := hasPawsHook(filepath.Join(root, ".claude", "settings.json")) ||
		hasPawsHook(filepath.Join(root, ".claude", "settings.local.json")) || hasPawsHook(filepath.Join(hook.ConfigDir(), "settings.json"))
	if !claude && claudeHooked {
		staying = defaultAdapter().Title()
	}
	if claude && staying != "" {
		plan, keep = planClaudeHooks(plan, keep, root, f.project, f.global, staying)
	}
	claude = staying == ""
	if claude && f.global {
		plan = planPortal(plan) // first: tier 2 steps run in plan order, and the skills below need the login gone first
	}
	if claude && f.project {
		hooks := []string{filepath.Join(root, ".claude", "settings.json"), filepath.Join(root, ".claude", "settings.local.json")}
		plan, keep = planScope(plan, keep, "project", hooks, pack.ProjectTarget(root), f.keepSkills, shared)
	}
	if claude && f.global {
		plan, keep = planScope(plan, keep, "global", []string{filepath.Join(hook.ConfigDir(), "settings.json")}, user, f.keepSkills, overlap{})
		keep = planUserFiles(keep)
	}
	plan, keep = planOtherHarnesses(plan, keep, root, f.project, f.global, f.keepSkills, func(a harness.Adapter) bool { return all || agents[a] })

	if len(plan) == 0 {
		fmt.Println("Nothing to remove: paws has not added anything here.")
	} else {
		fmt.Println("paws will remove:")
		for _, r := range plan {
			fmt.Println("  " + r.what)
		}
	}
	if len(keep) > 0 {
		fmt.Println("\npaws will keep:")
		for _, k := range keep {
			fmt.Println("  " + k)
		}
	}
	if len(plan) > 0 && !f.yes {
		fmt.Print("\nRemove these? [y/N] ")
		line, err := c.in.ReadString('\n')
		if err != nil && line == "" {
			fmt.Println()
			fmt.Fprintln(os.Stderr, "paws: no answer; nothing removed. Pass --yes to skip the question.")
			return 1
		}
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			fmt.Println("Nothing removed.")
			return 0
		}
	}

	code := 0
	for tier := 0; tier <= 2; tier++ {
		for _, r := range plan {
			if r.tier != tier {
				continue
			}
			if tier > 0 && code != 0 {
				fmt.Fprintf(os.Stderr, "paws: kept %s: an earlier step failed; fix it and run uninstall again\n", strings.Join(strings.Fields(r.what), " "))
				continue
			}
			if err := r.run(); err != nil {
				fmt.Fprintf(os.Stderr, "paws: %v\n", err)
				code = 1
			}
		}
	}
	if len(plan) > 0 {
		if f.global {
			if dir := gate.UserDir(); dir != "" {
				if os.Remove(dir) == nil {
					fmt.Printf("Removed %s\n", tilde(dir))
				} else if _, err := os.Stat(dir); err == nil {
					fmt.Printf("Left %s: it still holds the audit history, the binary copy or files you wrote.\n", tilde(dir))
				}
			}
		}
		fmt.Println("Done.")
	}
	fmt.Print(`
To remove the binary itself, use the way you installed it:
  npm rm -g unbound-paws
  brew uninstall --cask unbound-paws
  rm ~/.local/bin/unbound-paws     # the install script

A repo that committed the paws hook keeps it until you remove it there:
run unbound-paws uninstall in that repo.
`)
	return code
}

// planClaudeHooks removes only Claude Code's paws hooks, for the scopes being removed: another agent still runs
// paws, so the rules, packs, skills and login it shares stay.
func planClaudeHooks(plan []removal, keep []string, root string, project, global bool, staying string) ([]removal, []string) {
	var files []string
	if project {
		files = append(files, filepath.Join(root, ".claude", "settings.json"), filepath.Join(root, ".claude", "settings.local.json"))
	}
	if global {
		files = append(files, filepath.Join(hook.ConfigDir(), "settings.json"))
	}
	for _, path := range files {
		if hasPawsHook(path) {
			plan = append(plan, removal{what: "hook      " + tilde(path), run: func() error { return unregisterHook(path) }})
		}
	}
	return plan, append(keep, fmt.Sprintf("rules, packs, skills and login (%s still runs paws)", staying))
}

// askUninstall asks which agents to remove paws from (the ones it is set up for ticked) and from where.
func askUninstall(c *chooser) (agents map[harness.Adapter]bool, project, global bool, err error) {
	cwd, _ := os.Getwd()
	root := projectRoot(cwd)
	all := installOrder()
	labels, on := make([]string, len(all)), make([]bool, len(all))
	for i, a := range all {
		labels[i] = a.Title()
		if hasPawsHook(a.HookFile(false, root)) || hasPawsHook(a.HookFile(true, "")) {
			labels[i], on[i] = labels[i]+" (paws set up)", true
		}
	}
	if !anyOn(on) {
		for i := range on {
			on[i] = true
		}
	}
	picked, err := c.many("Remove paws from which agents?", labels, on)
	if err != nil {
		return nil, false, false, err
	}
	agents = map[harness.Adapter]bool{}
	for i, p := range picked {
		agents[all[i]] = p
	}
	where, err := c.one("From where?", []string{"This repo and every repo (everything)", "This repo only", "Every repo (global) only"}, 0)
	return agents, where != 2, where != 1, err
}

// planOtherHarnesses adds each other harness's paws hook and the skill copies paws made for it (Codex's
// .codex/hooks.json and .agents/skills), for the scopes being removed. A copy goes even when its hook is gone.
func planOtherHarnesses(plan []removal, keep []string, root string, project, global, keepSkills bool, chosen func(harness.Adapter) bool) ([]removal, []string) {
	seen := map[string]bool{} // outside a repo, the project's files can be the global ones
	for _, a := range adapters {
		if a == defaultAdapter() || !chosen(a) {
			continue
		}
		for _, g := range []bool{false, true} {
			if g && !global || !g && !project {
				continue
			}
			path := a.HookFile(g, root)
			if seen[path] {
				continue
			}
			seen[path] = true
			if hasPawsHook(path) {
				plan = append(plan, removal{what: "hook      " + tilde(path), run: func() error {
					err := unregisterHook(path)
					if !g {
						pruneEmpty(filepath.Dir(path))
					}
					return err
				}})
			}
			dir := a.SkillsDirFor(path)
			if dir == "" || seen[dir] || isLink(dir) {
				continue
			}
			seen[dir] = true
			for _, s := range pack.PawsSkills(dir) {
				s, at := s, filepath.Join(dir, s.Name)
				switch {
				case keepSkills:
					keep = append(keep, fmt.Sprintf("skill     %s (--keep-skills)", tilde(at)))
				case s.Edited:
					keep = append(keep, fmt.Sprintf("skill     %s (paws installed it, but it was edited since)", tilde(at)))
				default:
					plan = append(plan, removal{what: "skill     " + tilde(at), tier: 2, run: func() error {
						if requiredSkills()[s.Name] && !hasOtherCopy(s.Name, at) {
							fmt.Printf("Kept %s: a rule that stays requires it\n", tilde(at))
							return nil
						}
						_, err := pack.RemoveBundledSkill(dir, s.Name)
						pruneEmpty(dir)
						return err
					}})
				}
			}
		}
	}
	return plan, keep
}

// overlap says which of a project's files are also the user's: all of them when the project root is the home folder.
type overlap struct{ hooks, skills, packs bool }

func overlaps(root string, project, user pack.Target) overlap {
	return overlap{
		hooks:  filepath.Clean(filepath.Join(root, ".claude")) == filepath.Clean(hook.ConfigDir()),
		skills: filepath.Clean(project.SkillsDir) == filepath.Clean(user.SkillsDir),
		packs:  filepath.Clean(project.StateFile) == filepath.Clean(user.StateFile),
	}
}

// planScope adds the removals for one scope: the hook in its settings files, the packs the state file lists, and
// the skills paws installed. Nothing is listed that paws didn't create.
func planScope(plan []removal, keep []string, scope string, settings []string, t pack.Target, keepSkills bool, shared overlap) ([]removal, []string) {
	project := scope == "project"
	claudeDir := filepath.Dir(settings[0])
	pawsDir := filepath.Dir(t.StateFile)
	// A cloned repo can commit .claude or .paws as a link to another folder; then nothing behind it is the project's.
	claudeLinked, pawsLinked := project && isLink(claudeDir), project && isLink(pawsDir)
	if claudeLinked {
		keep = append(keep, "project   "+tilde(claudeDir)+" (a symbolic link; left alone)")
	}
	if pawsLinked {
		keep = append(keep, "project   "+tilde(pawsDir)+" (a symbolic link; left alone)")
	}
	// A hook behind a link stays registered, so the rules it enforces must stay: fail, which skips the later tiers.
	failLinked := func(path string) {
		if hasPawsHook(path) || unreadableWithPaws(path) {
			plan = append(plan, removal{what: "hook      " + tilde(path) + " (behind a symbolic link)", run: func() error {
				return fmt.Errorf("kept %s: a symbolic link to a hook this run won't follow, so its rules stay too", tilde(path))
			}})
		}
	}
	if claudeLinked {
		for _, path := range settings {
			failLinked(path)
		}
	}
	if !claudeLinked {
		for _, path := range settings {
			path := path
			if shared.hooks && path == filepath.Join(hook.ConfigDir(), "settings.json") {
				continue // the global scope has this file
			}
			if project && isLink(path) {
				keep = append(keep, "project   "+tilde(path)+" (a symbolic link; left alone)")
				failLinked(path)
				continue
			}
			if unreadableWithPaws(path) {
				// Skipping it would leave the hook registered with everything it needs removed: fail instead.
				plan = append(plan, removal{what: "hook      " + tilde(path) + " (can't be read)", run: func() error {
					return fmt.Errorf("can't read %s, which mentions paws; fix or remove the file, then run uninstall again", tilde(path))
				}})
				continue
			}
			if hasPawsHook(path) {
				plan = append(plan, removal{what: "hook      " + tilde(path), run: func() error {
					err := unregisterHook(path)
					if project {
						pruneEmpty(claudeDir)
					}
					return err
				}})
			}
		}
	}
	// With a linked .paws the packs can't be found, and they may still need their skills.
	skillsOK := !claudeLinked && !pawsLinked && !shared.skills && !isLink(t.SkillsDir)
	if !pawsLinked && !shared.packs {
		st, err := pack.LoadState(t.StateFile)
		if err != nil {
			// Without the state file the packs can't be found, and they may still need their skills.
			skillsOK = false
			keep = append(keep, "packs     "+tilde(t.StateFile)+" (can't be read, so its packs and the skills they may need stay)")
		}
		if err == nil && len(st.Packs) > 0 {
			var names []string
			for n := range st.Packs {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				n, dir := n, pack.InstallDir(t, n)
				rel, relErr := filepath.Rel(t.PacksDir, dir)
				if relErr != nil || !filepath.IsLocal(rel) || rel == "." || printable(n) != n {
					continue // a name that points outside the packs folder, or hides in the terminal, is not paws's to delete
				}
				plan = append(plan, removal{what: fmt.Sprintf("pack      %s (%s) in %s", n, count(len(st.Packs[n].Bindings), "rule"), tilde(dir)), tier: 1, run: func() error {
					if err := removePackDir(t.PacksDir, dir, project); err != nil {
						return err
					}
					pruneUp(filepath.Dir(dir), t.PacksDir)
					return nil
				}})
			}
			// The state file is how a retry finds the packs left behind, so it goes only if every pack did.
			plan = append(plan, removal{what: "packs     " + tilde(t.StateFile), tier: 2, run: func() error {
				err := os.Remove(t.StateFile)
				if errors.Is(err, os.ErrNotExist) {
					err = nil
				}
				pruneUp(t.PacksDir, filepath.Dir(t.StateFile))
				return err
			}})
		}
	}
	if !skillsOK {
		return plan, keep
	}
	for _, s := range pack.PawsSkills(t.SkillsDir) {
		s, dir := s, filepath.Join(t.SkillsDir, s.Name)
		switch {
		case keepSkills:
			keep = append(keep, fmt.Sprintf("skill     %s (--keep-skills)", tilde(dir)))
		case s.Edited:
			keep = append(keep, fmt.Sprintf("skill     %s (paws installed it, but it was edited since)", tilde(dir)))
		default:
			plan = append(plan, removal{what: "skill     " + tilde(dir), tier: 2, run: func() error {
				if requiredSkills()[s.Name] && !hasOtherCopy(s.Name, dir) {
					fmt.Printf("Kept %s: a rule that stays requires it\n", tilde(dir))
					return nil
				}
				_, err := pack.RemoveBundledSkill(t.SkillsDir, s.Name)
				pruneEmpty(t.SkillsDir)
				if project {
					pruneEmpty(claudeDir) // paws may have created it; os.Remove keeps it if anything else is in it
				}
				return err
			}})
		}
	}
	return plan, keep
}

// planPortal adds logging out and clearing the portal's rules and state, the way `logout` does. The rules go
// only if the login did, since a login that stays is still served those rules.
func planPortal(plan []removal) []removal {
	dir := portalDir()
	if path := credentialsPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			plan = append(plan, removal{what: "login     " + tilde(path) + " (logs this laptop out of the portal first)", tier: 2, run: func() error {
				c, err := loadCredentials()
				if err != nil {
					// Deleting a file paws can't read would leave this laptop's login live in the portal.
					return fmt.Errorf("can't read %s (%v); kept it. Disconnect this laptop in the portal, then delete the file", tilde(path), err)
				}
				if c == nil && credentialsHaveToken(path) {
					// loadCredentials passes over a file with a token but no portal address; the device is still registered.
					return fmt.Errorf("%s has a token but no portal address; kept it. Disconnect this laptop in the portal, then delete the file", tilde(path))
				}
				if c != nil && logout(true) != 0 {
					return errors.New("logout failed; credentials.json is still there")
				}
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
				return nil
			}})
		}
	}
	if dir != "" {
		if _, err := os.Stat(dir); err == nil {
			plan = append(plan, removal{what: "portal    " + tilde(dir) + " (rules and state)", tier: 2, run: func() error {
				if err := clearPortal(); err != nil {
					return fmt.Errorf("removing portal rules: %w", err)
				}
				os.Remove(filepath.Join(dir, "sync.lock"))
				pruneEmpty(filepath.Join(dir, "digests"), dir)
				return nil
			}})
		}
	}
	return plan
}

// planUserFiles lists what stays in paws's own folder. The audit log is the record of what the gate did, and the sync
// cursors belong to it, so one unattended `uninstall --yes` must not destroy them. The binary copy `init` made is
// kept too: committed hooks in other repos may still run it, and no command here can find those repos.
func planUserFiles(keep []string) []string {
	dir := gate.UserDir()
	if dir == "" {
		return keep
	}
	for _, name := range []string{"audit.log", "audit.log.1", "sync-state.json", "sync-from.json"} {
		if path := filepath.Join(dir, name); fileExists(path) {
			keep = append(keep, "file      "+tilde(path)+" (the audit history; delete it yourself if you want it gone)")
		}
	}
	bin := filepath.Join(dir, "bin")
	for _, name := range []string{"paws", "paws.exe", "paws.exe.old"} {
		if fileExists(filepath.Join(bin, name)) {
			keep = append(keep, "binary    "+tilde(bin)+" (hooks in other repos may still run it; rm -rf it once none do)")
			break
		}
	}
	return keep
}

// unregisterHook strips the paws hook entries from a settings file, deleting the file if nothing else was in it.
func unregisterHook(path string) error {
	doc, err := readSettings(path)
	if err != nil {
		return err
	}
	changed, err := removeHooks(doc)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !changed {
		return nil
	}
	if len(doc) == 0 {
		return os.Remove(path)
	}
	return writeSettings(path, doc)
}

// pruneEmpty removes each directory in turn if it is empty; os.Remove keeps one that holds anything.
func pruneEmpty(dirs ...string) {
	for _, d := range dirs {
		os.Remove(d)
	}
}

// pruneUp removes dir if it is empty, then each parent up to and including stop, ending at the first one that holds anything.
func pruneUp(dir, stop string) {
	for {
		if os.Remove(dir) != nil || dir == stop {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// removePackDir deletes a pack's folder under packs. The state file can come from a cloned repo, so it refuses to
// walk through a symbolic link: a link inside the folder could point at anything.
func removePackDir(packs, dir string, project bool) error {
	rel, err := filepath.Rel(packs, dir)
	if err != nil || !filepath.IsLocal(rel) || rel == "." {
		return fmt.Errorf("%s is not inside %s", tilde(dir), tilde(packs))
	}
	cur := packs
	parts := append([]string{""}, strings.Split(rel, string(filepath.Separator))...)
	if project {
		// A cloned repo can commit .paws itself as a link, too.
		cur, parts = filepath.Dir(packs), append([]string{"", filepath.Base(packs)}, parts[1:]...)
	}
	for _, part := range parts {
		cur = filepath.Join(cur, part)
		st, err := os.Lstat(cur)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link; left it alone", tilde(cur))
		}
	}
	return os.RemoveAll(dir)
}

// credentialsHaveToken reports whether a credentials file holds a token, whatever else is missing from it.
func credentialsHaveToken(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var c struct {
		Token string `json:"token"`
	}
	return json.Unmarshal(data, &c) == nil && c.Token != ""
}

// unreadableWithPaws reports whether a settings file exists, can't be parsed, and mentions paws, so a hook may be
// hiding in it.
func unreadableWithPaws(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist) // permission denied and the like: can't tell, so don't skip it
	}
	doc, err := readSettings(path)
	if err == nil {
		_, err = hooksBlock(doc)
	}
	return err != nil && strings.Contains(string(data), "paws")
}

// isLink reports whether path is a symbolic link.
func isLink(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode()&os.ModeSymlink != 0
}

// requiredSkills are the skills the rules still on disk require. The skill steps run after the pack and portal
// removals, so this sees only rules that stay: user-written ones, portal rules of a scope that stays, and the packs
// of a scope whose state couldn't be read.
func requiredSkills() map[string]bool {
	if requiredCache != nil {
		return requiredCache
	}
	cwd, _ := os.Getwd()
	need := map[string]bool{}
	collect := func(dir string) {
		for _, r := range gate.Collect(dir).Rules {
			if r.Action.Skill != "" {
				need[r.Action.Skill] = true
			}
		}
	}
	collect(cwd)
	// Rules in nested folders of the repo (a monorepo package's own .paws) use the same project skills. Only a
	// real repo is walked: outside one, the root is the folder you ran this in, maybe your whole home folder.
	if root := projectRoot(cwd); fileExists(filepath.Join(root, ".git")) {
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil
			case d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules"):
				return filepath.SkipDir
			case d.Name() == ".paws": // a folder or a link to one: the gate loads both
				collect(filepath.Dir(p))
				if d.IsDir() {
					return filepath.SkipDir
				}
			}
			return nil
		})
	}
	requiredCache = need
	return need
}

// requiredCache holds requiredSkills for one run, so the walk happens once however many skills are removed.
var requiredCache map[string]bool

// hasOtherCopy reports whether a project skill is also in the user skills folder. The user copy serves every repo, so
// the project copy can go. A user skill is never covered by a project copy: other repos don't have it.
func hasOtherCopy(name, dir string) bool {
	cwd, _ := os.Getwd()
	project := pack.ProjectTarget(projectRoot(cwd)).SkillsDir
	user := pack.UserTarget(gate.UserDir()).SkillsDir
	return filepath.Dir(dir) == project && project != user && fileExists(filepath.Join(user, name, "SKILL.md"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
