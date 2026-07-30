package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	command := shellQuote(self) + " gate"

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
