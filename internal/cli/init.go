package cli

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
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
	bin, err := installSelf(self, gate.UserDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: install binary: %v\n", err)
		return 1
	}
	command := pawsMarker + shellQuote(bin) + " gate"

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
		installPawsSkill(settingsPath)
		return 0
	}
	if err := writeSettings(settingsPath, doc); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	fmt.Printf("Registered paws in %s\n  command: %s\n", settingsPath, command)
	installPawsSkill(settingsPath)
	fmt.Println("Open a Claude Code session in this project to activate it.")
	return 0
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
	if !changed {
		fmt.Printf("paws is not registered in %s\n", settingsPath)
		return 0
	}
	if err := writeSettings(settingsPath, doc); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	fmt.Printf("Removed paws from %s\n", settingsPath)
	removePawsSkill(settingsPath)
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
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude", "settings.json"), -1
	}
	root, _ := os.Getwd()
	return filepath.Join(projectRoot(root), ".claude", "settings.json"), -1
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

// installSelf copies the running binary to <userDir>/bin and returns the path the hook should run.
// npx and go-run binaries live in caches that get cleaned; a hook pointing there breaks silently.
// Re-running init after an upgrade replaces the copy atomically. With no user dir, self is used.
func installSelf(self, userDir string) (string, error) {
	if userDir == "" {
		return self, nil
	}
	dest := filepath.Join(userDir, "bin", filepath.Base(self))
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
// settings file isn't in a .claude directory (a --settings path elsewhere, managed settings).
func pawsSkillDir(settingsPath string) string {
	dir := filepath.Dir(settingsPath)
	if filepath.Base(dir) != ".claude" {
		return ""
	}
	return filepath.Join(dir, "skills")
}

// installPawsSkill puts the paws skill in <dir>/.claude/skills/paws beside the settings file,
// so Claude Code can write and check rules. Best-effort: a failure is reported, never fatal,
// and a skill the user owns or edited is left alone.
func installPawsSkill(settingsPath string) {
	skillsDir := pawsSkillDir(settingsPath)
	if skillsDir == "" {
		fmt.Printf("Skipped the paws skill: %s is not in a .claude directory\n", settingsPath)
		return
	}
	src, err := fs.Sub(paws.Skill, "skills/paws")
	if err != nil {
		return
	}
	status, err := pack.InstallBundledSkill(src, skillsDir, "paws", "paws "+Version)
	dest := filepath.Join(skillsDir, "paws")
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "paws: could not install the paws skill: %v\n", err)
	case status == pack.BundledInstalled:
		fmt.Printf("Installed the paws skill in %s\n", dest)
	case status == pack.BundledKept:
		fmt.Printf("Left %s as is (not installed by paws, or edited); it may describe an older rule format\n", dest)
	}
}

// removePawsSkill undoes installPawsSkill on uninstall, keeping a skill the user edited.
func removePawsSkill(settingsPath string) {
	if skillsDir := pawsSkillDir(settingsPath); skillsDir != "" {
		if removed, _ := pack.RemoveBundledSkill(skillsDir, "paws"); removed {
			fmt.Printf("Removed the paws skill from %s\n", filepath.Join(skillsDir, "paws"))
		}
	}
}

// Version is set by main so installs can record which paws wrote them.
var Version = "dev"
