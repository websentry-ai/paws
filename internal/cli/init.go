package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/pack"
)

// Init registers the paws binary as an agent hook: `paws init [--agent claude|codex ...] [--global | --project |
// --settings <path>]`. On a terminal with no flags it asks which agents and where; without a terminal and flags it
// does what it always did: Claude Code, the repo root's .claude/settings.json.
func Init(args []string) int {
	targets, code := initTargets(args)
	if code >= 0 {
		return code
	}
	var set []string
	for _, t := range targets {
		if c := initOne(t.a, t.path); c != 0 {
			code = c
			continue
		}
		set = append(set, fmt.Sprintf("  %-12s %s", t.a.Title(), tilde(t.path)))
	}
	if len(set) > 0 {
		fmt.Println("paws is set up for:")
		fmt.Println(strings.Join(set, "\n"))
	}
	return max(code, 0)
}

// initOne registers the hook for one harness in one hook file.
func initOne(a harness.Adapter, settingsPath string) int {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: cannot resolve own path: %v\n", err)
		return 1
	}
	self, _ = filepath.Abs(self)
	// The same command in project and user settings, so the harness runs the gate once. Run as the
	// unbound-paws on PATH, init names it; so does a project init (from npx, say) next to a user-level
	// `unbound-paws gate` while unbound-paws is still installed on PATH.
	user := a.HookFile(true, "")
	command := a.HookCommand(cliName + " gate")
	reuse := filepath.Clean(settingsPath) != filepath.Clean(user) && unguard(pawsCommandIn(user)) == command && cliPath() != ""
	if !onPath(self) && !reuse {
		// Not installed on PATH: copy the binary somewhere the npx cache can't clean up.
		bin, err := installSelf(self, gate.UserDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: install binary: %v\n", err)
			return 1
		}
		command = a.HookCommand(hookCommand(bin))
		if cliPath() == "" {
			fmt.Printf("unbound-paws isn't installed, so the hook runs this copy. Install it for the `unbound-paws` command and upgrades:\n  curl -fsSL https://paws.getunbound.ai/install.sh | bash\n")
		}
	}

	doc, err := readSettings(settingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	// A committed `unbound-paws gate` runs on teammates' machines too: guard it so a missing binary says how to
	// install paws. The copied binary under ~/.paws/bin is not guarded, because `npm i -g` would not create it.
	guard := hookGuard() && command == a.HookCommand(cliName+" gate")
	written := command
	if guard {
		written = guardedCommand(command, "PreToolUse")
	}
	changed, err := ensureHooksFor(a, doc, command, guard)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %s: %v\n", settingsPath, err)
		return 1
	}
	if !changed {
		fmt.Printf("paws already registered in %s\n", settingsPath)
		return 0
	}
	if err := writeSettings(settingsPath, doc); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	fmt.Printf("Registered paws in %s\n  command: %s\n", settingsPath, command)
	if guard {
		fmt.Println("  Where paws isn't installed, the hook says how to install it and blocks the tool call.")
	}
	if note := a.InstallNote(); note != "" {
		fmt.Println(note)
	}
	// A different paws hook in the other scope makes the harness run paws twice: say which init fixes it.
	other, fix := user, "`"+cliName+" init --global"+agentFlag(a)+"`"
	if filepath.Clean(settingsPath) == filepath.Clean(user) {
		cwd, _ := os.Getwd()
		root := projectRoot(cwd)
		other, fix = a.HookFile(false, root), "`"+cliName+" init"+agentFlag(a)+"` in "+root
	}
	if c := pawsCommandIn(other); c != "" && c != written {
		fmt.Printf("%s has a different paws hook. Run %s too, so %s runs paws once.\n", other, fix, a.Title())
	}
	return 0
}

// initTarget is one hook file init writes: a harness and its hook file.
type initTarget struct {
	a    harness.Adapter
	path string
}

// initTargets reads init's flags, or asks on a terminal when there are none. code is -1 to go on, else the exit.
func initTargets(args []string) (targets []initTarget, code int) {
	var agents []harness.Adapter
	global, project, explicit, flagged := false, false, "", len(args) > 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			if i+1 >= len(args) {
				return nil, usageAgent()
			}
			for _, name := range strings.Split(args[i+1], ",") {
				a := findAgent(strings.TrimSpace(name))
				if a == nil {
					return nil, usageAgent()
				}
				if !slices.Contains(agents, a) {
					agents = append(agents, a)
				}
			}
			i++
		case "--global":
			global = true
		case "--project":
			project = true
		case "--settings":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "paws: --settings needs a path")
				return nil, 2
			}
			explicit = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "paws: unknown flag %q\n", args[i])
			return nil, 2
		}
	}
	switch {
	case global && project:
		fmt.Fprintln(os.Stderr, "paws: --global and --project: pick one")
		return nil, 2
	case explicit != "" && (global || project || len(agents) > 1):
		fmt.Fprintln(os.Stderr, "paws: --settings names one hook file: use it with at most one --agent, without --global or --project")
		return nil, 2
	}
	if !flagged && interactive() {
		var err error
		if agents, global, err = askInit(); err != nil {
			fmt.Fprintln(os.Stderr, "paws: nothing set up")
			return nil, 1
		}
	}
	if len(agents) == 0 {
		agents = []harness.Adapter{defaultAdapter()}
	}
	cwd, _ := os.Getwd()
	for _, a := range agents {
		path := explicit
		if path == "" {
			path = a.HookFile(global, projectRoot(cwd))
		}
		targets = append(targets, initTarget{a, path})
	}
	return targets, -1
}

// askInit asks which agents paws gates (the ones found on this machine ticked) and where.
func askInit() (agents []harness.Adapter, global bool, err error) {
	c := newChooser()
	all := installOrder()
	labels, on := make([]string, len(all)), make([]bool, len(all))
	for i, a := range all {
		labels[i], on[i] = a.Title(), agentFound(a)
		if agentFound(a) {
			labels[i] += " (found)"
		}
	}
	if !anyOn(on) {
		on[0] = true // nothing found: Claude Code, as without a terminal
	}
	picked, err := c.many("Which agents should paws gate?", labels, on)
	if err != nil {
		return nil, false, err
	}
	for i, p := range picked {
		if p {
			agents = append(agents, all[i])
		}
	}
	where, err := c.one("Where?", []string{"This repo only", "Every repo (global)"}, 0)
	return agents, where == 1, err
}

// installOrder lists the harnesses as init offers them: Claude Code first.
func installOrder() []harness.Adapter {
	all := []harness.Adapter{defaultAdapter()}
	for _, a := range adapters {
		if a != defaultAdapter() {
			all = append(all, a)
		}
	}
	return all
}

// agentFound reports whether a harness is on this machine: its command on PATH, or its config folder.
func agentFound(a harness.Adapter) bool {
	bin, dir := "claude", hook.ConfigDir()
	if a != defaultAdapter() {
		bin, dir = a.Name(), filepath.Dir(a.HookFile(true, ""))
	}
	if _, err := exec.LookPath(bin); err == nil {
		return true
	}
	st, err := os.Stat(dir)
	return err == nil && st.IsDir()
}

// findAgent is the harness --agent names: its name, or "claude" for Claude Code.
func findAgent(name string) harness.Adapter {
	if name == "claude" {
		return defaultAdapter()
	}
	return harness.Find(adapters, name)
}

func usageAgent() int {
	fmt.Fprintf(os.Stderr, "paws: --agent needs one or more of: claude, %s\n", strings.Join(adapterNames(), ", "))
	return 2
}

// hookGuard reports whether init may wrap the hook command in the missing-binary guard, which needs sh.
func hookGuard() bool { return runtime.GOOS != "windows" }

// agentFlag is the --agent flag that names a, "" for the default harness.
func agentFlag(a harness.Adapter) string {
	if a == defaultAdapter() {
		return ""
	}
	return " --agent " + a.Name()
}

// defaultAdapter is the harness init and friends use without --agent: Claude Code, the fallback.
func defaultAdapter() harness.Adapter { return adapters[len(adapters)-1] }

// pawsCommandIn is the paws hook command a settings file registers for PreToolUse, or "".
func pawsCommandIn(path string) string {
	doc, err := readSettings(path)
	if err != nil {
		return ""
	}
	hooks, err := hooksBlock(doc)
	if err != nil {
		return ""
	}
	var list []struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	json.Unmarshal(hooks["PreToolUse"], &list)
	for _, e := range list {
		for _, h := range e.Hooks {
			if isPawsCommand(h.Command) {
				return h.Command
			}
		}
	}
	return ""
}

// cliName is the command people type and the hook runs: the installer (and npm) put it on PATH.
const cliName = "unbound-paws"

// cliPath is the first `unbound-paws` on PATH, or "". npx (and npm run) put a node_modules/.bin
// in front of PATH only for that run; Claude Code won't have it, so those folders are skipped, and
// so are relative entries like `.` (they follow the cwd, and a bare name would search PATH again).
func cliPath() string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if !filepath.IsAbs(dir) || filepath.Base(dir) == ".bin" && filepath.Base(filepath.Dir(dir)) == "node_modules" {
			continue
		}
		if p, err := exec.LookPath(filepath.Join(dir, cliName)); err == nil {
			return p
		}
	}
	return ""
}

// onPath reports whether init is running as the unbound-paws on PATH, so the hook can name it
// instead of copying the binary: true for the installer's copy and a native npm install, false
// for npx, a source build, or an older unbound-paws earlier on PATH.
func onPath(self string) bool {
	p := cliPath()
	if p == "" {
		return false
	}
	a, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	b, err := filepath.EvalSymlinks(self)
	return err == nil && a == b
}

// hookCommand is the settings.json command that runs bin as the gate. A binary under $HOME is
// written as "$HOME/..." so a committed project settings.json works on every teammate's machine.
// Project and user settings get the same string, so Claude Code runs the gate once, not twice.
func hookCommand(bin string) string {
	home, _ := os.UserHomeDir()
	if rel, err := filepath.Rel(home, bin); runtime.GOOS != "windows" && home != "" && err == nil &&
		filepath.IsLocal(rel) && !strings.ContainsAny(rel, "\"$`\\") {
		return `"$HOME/` + filepath.ToSlash(rel) + `" gate`
	}
	return shellQuote(bin) + " gate"
}

// uninstallHookOnly removes every paws hook entry from the target settings.json, and the paws skill beside it.
func uninstallHookOnly(args []string) int {
	a, settingsPath, code := installTarget(args)
	if code >= 0 {
		return code
	}
	doc, err := readSettings(settingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	changed, err := removeHooks(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %s: %v\n", settingsPath, err)
		return 1
	}
	if changed {
		if err := writeSettings(settingsPath, doc); err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
		fmt.Printf("Removed paws from %s\n", settingsPath)
	} else {
		fmt.Printf("paws is not registered in %s\n", settingsPath)
	}
	// The skill goes too, even when `paws skill` ran without init.
	removePawsSkill(a, settingsPath)
	if changed && len(doc) == 0 {
		os.Remove(settingsPath) // nothing but paws was in it
	}
	// Don't leave empty folders behind; os.Remove keeps a folder that still has anything in it.
	if skillsDir := a.SkillsDirFor(settingsPath); skillsDir != "" {
		os.Remove(skillsDir)
	}
	if dir := filepath.Dir(settingsPath); filepath.Dir(a.HookFile(false, filepath.Dir(dir))) == dir {
		os.Remove(dir) // the project's config folder (.claude), as init made it
	}
	return 0
}

// settingsTarget resolves the settings.json path from flags; code is -1 to proceed, else the exit code.
func settingsTarget(args []string) (path string, code int) {
	_, path, code = installTarget(args)
	return path, code
}

// installTarget resolves the harness (--agent, default Claude Code) and its hook file from flags: the project's,
// the user's with --global, or --settings <path>. code is -1 to proceed, else the exit code.
func installTarget(args []string) (a harness.Adapter, path string, code int) {
	a = defaultAdapter()
	global := false
	explicit := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			if i+1 >= len(args) || findAgent(args[i+1]) == nil {
				fmt.Fprintf(os.Stderr, "paws: --agent needs one of: claude, %s\n", strings.Join(adapterNames(), ", "))
				return nil, "", 2
			}
			a = findAgent(args[i+1])
			i++
		case "--global":
			global = true
		case "--settings":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "paws: --settings needs a path")
				return nil, "", 2
			}
			explicit = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "paws: unknown flag %q\n", args[i])
			return nil, "", 2
		}
	}
	if explicit != "" {
		return a, explicit, -1
	}
	root, _ := os.Getwd()
	return a, a.HookFile(global, projectRoot(root)), -1
}

func adapterNames() []string {
	var names []string
	for _, a := range adapters {
		names = append(names, a.Name())
	}
	return names
}

// projectRoot walks up to the git repo root so init from a subdirectory writes the file Claude Code actually reads.
func projectRoot(cwd string) string {
	home, _ := os.UserHomeDir()
	for cur := cwd; ; {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if cur == home || parent == cur {
			return cwd
		}
		cur = parent
	}
}

// shellQuote wraps p in POSIX single quotes so $, backticks and spaces survive.
func shellQuote(p string) string {
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

// installSelf copies the running binary to <userDir>/bin/paws and returns the path the hook should run.
// npx and go-run binaries live in caches that get cleaned; a hook pointing there breaks silently.
// Re-running init after an upgrade replaces the copy atomically. With no user dir, self is used.
func installSelf(self, userDir string) (string, error) {
	if userDir == "" {
		return self, nil
	}
	name := "paws"
	if runtime.GOOS == "windows" {
		name = "paws.exe"
	}
	dest := filepath.Join(userDir, "bin", name)
	if same, _ := sameFile(self, dest); same {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", err
	}
	src, err := os.Open(self)
	if err != nil {
		return "", err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".paws-*")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(tmp, src)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o755)
	}
	if err == nil && runtime.GOOS == "windows" {
		// Windows can't replace a running .exe but can rename it: move the old one aside
		// (a hook mid-run keeps using it) and clear the previous leftover first.
		os.Remove(dest + ".old")
		if rerr := os.Rename(dest, dest+".old"); rerr != nil && !os.IsNotExist(rerr) {
			err = rerr
		}
	}
	if err == nil {
		// On unix, rename is atomic: a hook mid-run keeps the old inode, the next call gets the new binary.
		err = os.Rename(tmp.Name(), dest)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if self != dest {
		fmt.Printf("Installed paws binary to %s\n", dest)
	}
	return dest, nil
}

// sameFile reports whether a and b are the same file, or byte-identical copies.
func sameFile(a, b string) (bool, error) {
	sa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	sb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if os.SameFile(sa, sb) {
		return true, nil
	}
	if sa.Size() != sb.Size() {
		return false, nil
	}
	da, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	db, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(da, db), nil
}

// pawsSkillDir is the skills directory Claude Code reads beside settingsPath, or "" when the
// settings file isn't in a .claude directory or $CLAUDE_CONFIG_DIR (a --settings path elsewhere,
// managed settings).
func pawsSkillDir(settingsPath string) string { return hook.Claude{}.SkillsDirFor(settingsPath) }

// Skill installs the paws skill (how the agent writes and checks rules) beside the target
// hook file: `paws skill [--agent A] [--global | --settings <path>]`. Re-running it replaces an older or
// edited paws copy with this version's; a skills/paws directory paws didn't install is left alone.
func Skill(args []string) int {
	a, settingsPath, code := installTarget(args)
	if code >= 0 {
		return code
	}
	skillsDir := a.SkillsDirFor(settingsPath)
	if skillsDir == "" {
		fmt.Fprintf(os.Stderr, "paws: %s has no %s skills folder beside it to install into\n", settingsPath, a.Title())
		return 1
	}
	src, err := fs.Sub(paws.Skill, "skills/paws")
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	dest := filepath.Join(skillsDir, "paws")
	mirrorFailed := false
	if !slices.Contains(args, "--agent") && !slices.Contains(args, "--settings") {
		// No harness named: every other harness that runs paws here gets the skill too, in its own folder.
		cwd, _ := os.Getwd()
		for _, m := range skillMirrors(slices.Contains(args, "--global"), projectRoot(cwd)) {
			st, err := pack.ReinstallBundledSkill(src, m, "paws", "paws "+Version)
			switch {
			case err != nil:
				fmt.Fprintf(os.Stderr, "paws: install the paws skill in %s: %v\n", m, err)
				mirrorFailed = true
			case st != pack.BundledKept:
				fmt.Printf("Installed the paws skill (paws %s) in %s\n", Version, filepath.Join(m, "paws"))
			}
		}
	}
	status, err := pack.ReinstallBundledSkill(src, skillsDir, "paws", "paws "+Version)
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "paws: install the paws skill: %v\n", err)
		return 1
	case status == pack.BundledKept:
		fmt.Fprintf(os.Stderr, "paws: %s wasn't installed by paws; left it alone (move it away and re-run to replace it)\n", dest)
		return 1
	}
	fmt.Printf("Installed the paws skill (paws %s) in %s\n", Version, dest)
	if mirrorFailed {
		return 1 // another harness's copy is missing or stale
	}
	return 0
}

// removePawsSkill undoes `paws skill` on uninstall, keeping a skill the user edited.
func removePawsSkill(a harness.Adapter, settingsPath string) {
	if skillsDir := a.SkillsDirFor(settingsPath); skillsDir != "" {
		if removed, _ := pack.RemoveBundledSkill(skillsDir, "paws"); removed {
			fmt.Printf("Removed the paws skill from %s\n", filepath.Join(skillsDir, "paws"))
		}
	}
}

// Version is set by main so installs can record which paws wrote them.
var Version = "dev"
