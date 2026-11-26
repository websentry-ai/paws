package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/pack"
)

// removal is one thing uninstall deletes: what it tells the user, and how it deletes it.
type removal struct {
	what string
	run  func() error
	// late steps remove what the earlier steps may still need: the login and the portal rules, the pack state, the
	// skills those rules require, the audit history and the binary a surviving hook runs. They run in plan order
	// and are skipped once any step has failed.
	late bool
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
		fmt.Fprintf(os.Stderr, "paws: %v\nusage: unbound-paws uninstall [--project] [--global] [--keep-skills] [--yes]\n       unbound-paws uninstall --hook-only [--global | --settings <path>]\n", err)
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
	if !f.project && !f.global {
		f.project, f.global = true, true
	}

	var plan []removal
	var keep []string
	cwd, _ := os.Getwd()
	root := projectRoot(cwd)
	user := pack.UserTarget(gate.UserDir())
	// Outside a repo, in the home folder, some or all of the project's files are the global ones. Each part that
	// is shared stays with the global scope.
	shared := overlaps(root, pack.ProjectTarget(root), user)
	if f.project && !f.global && shared.hooks && shared.skills && shared.packs {
		fmt.Fprintln(os.Stderr, "paws: this folder is not in a repo, so its project files are your global ones. Run --project from a repo, or use --global.")
		return 2
	}
	// Without --global the portal rules stay, and a project skill they use stays with them.
	var portalSkillsToKeep map[string]bool
	if !f.global {
		portalSkillsToKeep = map[string]bool{}
		for _, name := range portalSkillNames() {
			portalSkillsToKeep[name] = true
		}
	}
	if f.global {
		plan = planPortal(plan) // first: late steps run in plan order, and the skills below need the login gone first
	}
	if f.project {
		hooks := []string{filepath.Join(root, ".claude", "settings.json"), filepath.Join(root, ".claude", "settings.local.json")}
		plan, keep = planScope(plan, keep, "project", hooks, pack.ProjectTarget(root), f.keepSkills, shared, portalSkillsToKeep)
	}
	if f.global {
		plan, keep = planScope(plan, keep, "global", []string{filepath.Join(userClaudeDir(), "settings.json")}, user, f.keepSkills, overlap{}, nil)
		keep = planUserFiles(keep)
	}

	if len(plan) == 0 {
		fmt.Println("Nothing to remove: paws has not added anything here.")
	} else {
		fmt.Println("paws will remove:")
		for _, r := range plan {
			fmt.Println("  " + r.what)
		}
	}
	if len(keep) > 0 {
		fmt.Println("paws will keep:")
		for _, k := range keep {
			fmt.Println("  " + k)
		}
	}
	if len(plan) > 0 && !f.yes {
		fmt.Print("Remove these? [y/N] ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
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
	for _, late := range []bool{false, true} {
		for _, r := range plan {
			if r.late != late {
				continue
			}
			if late && code != 0 {
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
					fmt.Printf("Left %s: it still holds files paws didn't create.\n", tilde(dir))
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

A repo that committed the paws hook keeps it until you remove it there (unbound-paws uninstall, run in that repo).
`)
	return code
}

// overlap says which of a project's files are also the user's: all of them when the project root is the home folder.
type overlap struct{ hooks, skills, packs bool }

func overlaps(root string, project, user pack.Target) overlap {
	return overlap{
		hooks:  filepath.Clean(filepath.Join(root, ".claude")) == filepath.Clean(userClaudeDir()),
		skills: filepath.Clean(project.SkillsDir) == filepath.Clean(user.SkillsDir),
		packs:  filepath.Clean(project.StateFile) == filepath.Clean(user.StateFile),
	}
}

// planScope adds the removals for one scope: the hook in its settings files, the packs the state file lists, and
// the skills paws installed. Nothing is listed that paws didn't create.
func planScope(plan []removal, keep []string, scope string, settings []string, t pack.Target, keepSkills bool, shared overlap, portalSkills map[string]bool) ([]removal, []string) {
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
	if !claudeLinked {
		for _, path := range settings {
			path := path
			if shared.hooks && path == filepath.Join(userClaudeDir(), "settings.json") {
				continue // the global scope has this file
			}
			if project && isLink(path) {
				keep = append(keep, "project   "+tilde(path)+" (a symbolic link; left alone)")
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
				plan = append(plan, removal{what: fmt.Sprintf("pack      %s (%s) in %s", n, count(len(st.Packs[n].Bindings), "rule"), tilde(dir)), run: func() error {
					if err := removePackDir(t.PacksDir, dir, project); err != nil {
						return err
					}
					pruneUp(filepath.Dir(dir), t.PacksDir)
					return nil
				}})
			}
			// The state file is how a retry finds the packs left behind, so it goes only if every pack did.
			plan = append(plan, removal{what: "packs     " + tilde(t.StateFile), late: true, run: func() error {
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
		case portalSkills[s.Name]:
			keep = append(keep, fmt.Sprintf("skill     %s (a portal rule for this laptop uses it; uninstall --global removes both)", tilde(dir)))
		default:
			plan = append(plan, removal{what: "skill     " + tilde(dir), late: true, run: func() error {
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
			plan = append(plan, removal{what: "login     " + tilde(path) + " (logs this laptop out of the portal first)", late: true, run: func() error {
				c, err := loadCredentials()
				if err != nil {
					// Deleting a file paws can't read would leave this laptop's login live in the portal.
					return fmt.Errorf("can't read %s (%v); kept it. Disconnect this laptop in the portal, then delete the file", tilde(path), err)
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
			plan = append(plan, removal{what: "portal    " + tilde(dir) + " (rules and state)", late: true, run: func() error {
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

// isLink reports whether path is a symbolic link.
func isLink(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode()&os.ModeSymlink != 0
}

// portalSkillNames are the skills the applied portal config uses: the ones state.json lists, and those the rule files
// in the portal folder are named for (<skill>--<action>.md), so an unreadable state.json hides none of them.
func portalSkillNames() []string {
	names := loadPortalState().Skills
	if dir := portalDir(); dir != "" {
		rules, _ := filepath.Glob(filepath.Join(dir, "*--*.md"))
		for _, r := range rules {
			names = append(names, strings.SplitN(filepath.Base(r), "--", 2)[0])
		}
	}
	return names
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
