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
		if f.keepSkills || f.yes {
			fmt.Fprintln(os.Stderr, "paws: --hook-only takes only --global or --settings <path>")
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
	// Outside a repo, in the home folder, the project's files are the global ones.
	overlap := sameFolder(pack.ProjectTarget(projectRoot(cwd)), pack.UserTarget(gate.UserDir()), projectRoot(cwd))
	if f.project && overlap {
		if !f.global {
			fmt.Fprintln(os.Stderr, "paws: this folder is not in a repo, so its project files are your global ones. Run --project from a repo, or use --global.")
			return 2
		}
		f.project = false // --global covers them
	}
	if f.project {
		root := projectRoot(cwd)
		hooks := []string{filepath.Join(root, ".claude", "settings.json"), filepath.Join(root, ".claude", "settings.local.json")}
		plan, keep = planScope(plan, keep, "project", hooks, pack.ProjectTarget(root), f.keepSkills, nil)
	}
	if f.global {
		hooks := []string{filepath.Join(userClaudeDir(), "settings.json")}
		// The login goes first: if it stays, the portal rules stay, and so must the skills they require.
		var loginFailed *bool
		plan, loginFailed = planPortal(plan)
		plan, keep = planScope(plan, keep, "global", hooks, pack.UserTarget(gate.UserDir()), f.keepSkills, loginFailed)
		// Without --project, this project's hook stays, and it may run the shared copy of the binary.
		keepBin := !f.project && !overlap && (hasPawsHook(filepath.Join(projectRoot(cwd), ".claude", "settings.json")) ||
			hasPawsHook(filepath.Join(projectRoot(cwd), ".claude", "settings.local.json")))
		plan, keep = planUserFiles(plan, keep, keepBin)
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
	for _, r := range plan {
		if err := r.run(); err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			code = 1
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

// sameFolder reports whether a project target (at root) points at any of the files the user target does.
func sameFolder(project, user pack.Target, root string) bool {
	return filepath.Clean(project.StateFile) == filepath.Clean(user.StateFile) ||
		filepath.Clean(project.SkillsDir) == filepath.Clean(user.SkillsDir) ||
		filepath.Clean(filepath.Join(root, ".claude")) == filepath.Clean(userClaudeDir())
}

// planScope adds the removals for one scope: the hook in its settings files, the packs the state file lists, and
// the skills paws installed. Nothing is listed that paws didn't create.
func planScope(plan []removal, keep []string, scope string, settings []string, t pack.Target, keepSkills bool, loginFailed *bool) ([]removal, []string) {
	packFailed := false // a pack that stays may still need its skills, so they stay too
	// A cloned repo can commit .claude as a link to the global one; then nothing in it is the project's to remove.
	claudeLinked := scope == "project" && isLink(filepath.Dir(settings[0]))
	if claudeLinked {
		keep = append(keep, "project    "+tilde(filepath.Dir(settings[0]))+" (a symbolic link; left alone)")
	}
	for _, path := range settings {
		path := path
		if hasPawsHook(path) && !claudeLinked {
			plan = append(plan, removal{"hook      " + tilde(path), func() error { return unregisterHook(path) }})
		}
	}
	st, err := pack.LoadState(t.StateFile)
	if err != nil {
		// Without the state file the packs can't be found, and they may still need their skills.
		packFailed = true
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
			if rel, err := filepath.Rel(t.PacksDir, dir); err != nil || !filepath.IsLocal(rel) || rel == "." {
				continue // a name in the state file that points outside the packs folder is not paws's to delete
			}
			plan = append(plan, removal{fmt.Sprintf("pack      %s (%s) in %s", n, count(len(st.Packs[n].Bindings), "rule"), tilde(dir)), func() error {
				if err := removePackDir(t.PacksDir, dir, scope == "project"); err != nil {
					packFailed = true
					return err
				}
				pruneUp(filepath.Dir(dir), t.PacksDir)
				return nil
			}})
		}
		plan = append(plan, removal{"packs     " + tilde(t.StateFile), func() error {
			if packFailed {
				// The state file is how a retry finds the packs left behind.
				return fmt.Errorf("kept %s: a pack could not be removed, so run uninstall again after fixing that", tilde(t.StateFile))
			}
			err := os.Remove(t.StateFile)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
			pruneUp(t.PacksDir, filepath.Dir(t.StateFile))
			return err
		}})
	}
	skills := pack.PawsSkills(t.SkillsDir)
	if claudeLinked || isLink(t.SkillsDir) {
		skills = nil
	}
	for _, s := range skills {
		s, dir := s, filepath.Join(t.SkillsDir, s.Name)
		switch {
		case keepSkills:
			keep = append(keep, fmt.Sprintf("skill     %s (--keep-skills)", tilde(dir)))
		case s.Edited:
			keep = append(keep, fmt.Sprintf("skill     %s (paws installed it, but it was edited since)", tilde(dir)))
		default:
			plan = append(plan, removal{"skill     " + tilde(dir), func() error {
				if packFailed {
					return fmt.Errorf("kept %s: a pack that may need it could not be removed", tilde(dir))
				}
				if loginFailed != nil && *loginFailed {
					return fmt.Errorf("kept %s: the portal rules that stay may need it", tilde(dir))
				}
				_, err := pack.RemoveBundledSkill(t.SkillsDir, s.Name)
				pruneEmpty(t.SkillsDir)
				return err
			}})
		}
	}
	return plan, keep
}

// planPortal adds logging out and clearing the portal's rules and state, the way `logout` does.
func planPortal(plan []removal) ([]removal, *bool) {
	dir := portalDir()
	loginFailed := new(bool) // the portal rules go with the login: if it stays, so do they
	if path := credentialsPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			plan = append(plan, removal{"login     " + tilde(path) + " (logs this laptop out of the portal first)", func() error {
				c, err := loadCredentials()
				if err != nil {
					// Deleting a file paws can't read would leave this laptop's login live in the portal.
					*loginFailed = true
					return fmt.Errorf("can't read %s (%v); kept it. Disconnect this laptop in the portal, then delete the file", tilde(path), err)
				}
				if c != nil && logout(true) != 0 {
					*loginFailed = true
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
			plan = append(plan, removal{"portal    " + tilde(dir) + " (rules and state)", func() error {
				if *loginFailed {
					return fmt.Errorf("kept %s: the login is still there", tilde(dir))
				}
				if err := clearPortal(); err != nil {
					return fmt.Errorf("removing portal rules: %w", err)
				}
				os.Remove(filepath.Join(dir, "sync.lock"))
				pruneEmpty(filepath.Join(dir, "digests"), dir)
				return nil
			}})
		}
	}
	return plan, loginFailed
}

// planUserFiles adds the files paws writes in its own folder as it runs: the audit log, the sync cursors and the
// binary copy init makes. Rule files the user wrote there are not on the list.
func planUserFiles(plan []removal, keep []string, keepBin bool) ([]removal, []string) {
	dir := gate.UserDir()
	if dir == "" {
		return plan, keep
	}
	for _, name := range []string{"audit.log", "audit.log.1", "sync-state.json", "sync-from.json"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		plan = append(plan, removal{"file      " + tilde(path), func() error { return os.Remove(path) }})
	}
	bin := filepath.Join(dir, "bin")
	if entries, _ := os.ReadDir(bin); len(entries) > 0 {
		var paths []string
		for _, e := range entries {
			if n := e.Name(); n == "paws" || n == "paws.exe" || n == "paws.exe.old" {
				paths = append(paths, filepath.Join(bin, n))
			}
		}
		if len(paths) > 0 && keepBin {
			keep = append(keep, "binary    "+tilde(bin)+" (the hook still registered in this project runs it)")
		} else if len(paths) > 0 {
			plan = append(plan, removal{"binary    " + tilde(bin) + " (the copy `init` made for the hook)", func() error {
				var errs []error
				for _, p := range paths {
					errs = append(errs, os.Remove(p))
				}
				pruneEmpty(bin)
				return errors.Join(errs...)
			}})
		}
	}
	return plan, keep
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
