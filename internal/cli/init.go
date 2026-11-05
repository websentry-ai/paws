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
	"strings"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/pack"
)

// Init registers the paws binary as an agent hook: repo root's .claude/settings.json by default, ~/.claude with --global.
func Init(args []string) int {
	settingsPath, code := settingsTarget(args)
	if code >= 0 {
		return code
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: cannot resolve own path: %v\n", err)
		return 1
	}
	self, _ = filepath.Abs(self)
	// The same command in project and user settings, so Claude Code runs the gate once. Run as the
	// unbound-paws on PATH, init names it; so does a project init (from npx, say) next to a user-level
	// `unbound-paws gate` while unbound-paws is still installed on PATH.
	user := filepath.Join(userClaudeDir(), "settings.json")
	command := cliName + " gate"
	reuse := filepath.Clean(settingsPath) != filepath.Clean(user) && pawsCommandIn(user) == command && cliPath() != ""
	if !onPath(self) && !reuse {
		// Not installed on PATH: copy the binary somewhere the npx cache can't clean up.
		bin, err := installSelf(self, gate.UserDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: install binary: %v\n", err)
			return 1
		}
		command = hookCommand(bin)
		if cliPath() == "" {
			fmt.Printf("unbound-paws isn't installed, so the hook runs this copy. Install it for the `unbound-paws` command and upgrades:\n  curl -fsSL https://paws.getunbound.ai/install.sh | bash\n")
		}
	}

	doc, err := readSettings(settingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	changed, err := ensureHooks(doc, command)
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
	other := user
	if filepath.Clean(settingsPath) == filepath.Clean(user) {
		cwd, _ := os.Getwd()
		other = filepath.Join(projectRoot(cwd), ".claude", "settings.json")
	}
	if c := pawsCommandIn(other); c != "" && c != command {
		fmt.Printf("%s registers paws as `%s`; run `%s init` there too, so Claude Code runs paws once.\n", other, c, cliName)
	}
	return 0
}

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

// Uninstall removes every paws hook entry from the target settings.json.
func Uninstall(args []string) int {
	settingsPath, code := settingsTarget(args)
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
	removePawsSkill(settingsPath)
	if changed && len(doc) == 0 {
		os.Remove(settingsPath) // nothing but paws was in it
	}
	// Don't leave empty folders behind; os.Remove keeps a folder that still has anything in it.
	if skillsDir := pawsSkillDir(settingsPath); skillsDir != "" {
		os.Remove(skillsDir)
	}
	if filepath.Base(filepath.Dir(settingsPath)) == ".claude" {
		os.Remove(filepath.Dir(settingsPath))
	}
	return 0
}

// settingsTarget resolves the settings.json path from flags; code is -1 to proceed, else the exit code.
func settingsTarget(args []string) (path string, code int) {
	global := false
	explicit := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--global":
			global = true
		case "--settings":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "paws: --settings needs a path")
				return "", 2
			}
			explicit = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "paws: unknown flag %q\n", args[i])
			return "", 2
		}
	}
	if explicit != "" {
		return explicit, -1
	}
	if global {
		return filepath.Join(userClaudeDir(), "settings.json"), -1
	}
	root, _ := os.Getwd()
	return filepath.Join(projectRoot(root), ".claude", "settings.json"), -1
}

// userClaudeDir is Claude Code's user config dir: $CLAUDE_CONFIG_DIR when set, else ~/.claude.
func userClaudeDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
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
func pawsSkillDir(settingsPath string) string {
	dir := filepath.Dir(settingsPath)
	if filepath.Base(dir) != ".claude" && dir != userClaudeDir() {
		return ""
	}
	return filepath.Join(dir, "skills")
}

// Skill installs the paws skill (how Claude Code writes and checks rules) beside the target
// settings.json: `paws skill [--global | --settings <path>]`. Re-running it replaces an older or
// edited paws copy with this version's; a skills/paws directory paws didn't install is left alone.
func Skill(args []string) int {
	settingsPath, code := settingsTarget(args)
	if code >= 0 {
		return code
	}
	skillsDir := pawsSkillDir(settingsPath)
	if skillsDir == "" {
		fmt.Fprintf(os.Stderr, "paws: %s is not in a .claude directory; no skills folder to install into\n", settingsPath)
		return 1
	}
	src, err := fs.Sub(paws.Skill, "skills/paws")
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	dest := filepath.Join(skillsDir, "paws")
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
	return 0
}

// removePawsSkill undoes `paws skill` on uninstall, keeping a skill the user edited.
func removePawsSkill(settingsPath string) {
	if skillsDir := pawsSkillDir(settingsPath); skillsDir != "" {
		if removed, _ := pack.RemoveBundledSkill(skillsDir, "paws"); removed {
			fmt.Printf("Removed the paws skill from %s\n", filepath.Join(skillsDir, "paws"))
		}
	}
}

// Version is set by main so installs can record which paws wrote them.
var Version = "dev"
