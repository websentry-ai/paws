package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// askEnv is a sandboxed home with a repo, Claude Code found (~/.claude exists) and Codex not, on a terminal that
// reads input. raw says whether the terminal takes raw mode (arrow-key menus) or not (numbered questions).
func askEnv(t *testing.T, input string, raw bool) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("PATH", t.TempDir()) // neither claude nor codex on PATH
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	repo = filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)

	oldInteractive, oldRaw := interactive, makeRaw
	interactive = func() bool { return true }
	makeRaw = func() (func(), error) {
		if raw {
			return func() {}, nil
		}
		return nil, errors.New("not a raw terminal")
	}
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { interactive, makeRaw, os.Stdin = oldInteractive, oldRaw, oldStdin; r.Close() })
	return home, repo
}

// Arrow keys: Claude Code comes ticked (found), down and space tick Codex, enter confirms; enter keeps "this repo".
func TestInitAsksWithArrowKeys(t *testing.T) {
	_, repo := askEnv(t, "\x1b[B \r\r", true)
	code, out := captureStdout(t, func() int { return Init(nil) })
	if code != 0 {
		t.Fatalf("init exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) || !hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Errorf("both agents in this repo:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code (found)") || !strings.Contains(out, "paws is set up for:") || !strings.Contains(out, "Codex") {
		t.Errorf("the menu marks what it found and the end says what was set up where:\n%s", out)
	}
}

// Without raw mode the same questions are numbered: both agents, every repo.
func TestInitAsksWithNumbers(t *testing.T) {
	home, repo := askEnv(t, "1,2\n2\n", false)
	if code, out := captureStdout(t, func() int { return Init(nil) }); code != 0 {
		t.Fatalf("init exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(home, ".claude", "settings.json")) || !hasPawsHook(filepath.Join(home, ".codex", "hooks.json")) {
		t.Error("both agents, global")
	}
	if hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Error("global writes nothing in the repo")
	}
}

// A bad answer asks again; Ctrl-C in the menu sets up nothing.
func TestInitAskAgainAndCancel(t *testing.T) {
	_, repo := askEnv(t, "7\n\n\n", false)
	if code, out := captureStdout(t, func() int { return Init(nil) }); code != 0 || !strings.Contains(out, "Pick numbers from 1 to 2") {
		t.Fatalf("a bad answer asks again: exit %d\n%s", code, out)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) || hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Error("enter takes the defaults: Claude Code, this repo")
	}
	_, repo = askEnv(t, "\x03", true)
	if code, _ := captureStdout(t, func() int { return Init(nil) }); code != 1 || hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Errorf("Ctrl-C sets up nothing: exit %d", code)
	}
}

// Flags skip the questions: a comma list or repeated --agent, --project; conflicting flags fail.
func TestInitFlagsSkipTheQuestions(t *testing.T) {
	_, repo := askEnv(t, "", true) // a terminal, but nothing to read: a question would cancel
	if code, out := captureStdout(t, func() int { return Init([]string{"--agent", "claude,codex", "--project"}) }); code != 0 {
		t.Fatalf("init exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) || !hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Error("--agent claude,codex --project")
	}
	for _, args := range [][]string{{"--global", "--project"}, {"--agent", "claude", "--agent", "codex", "--settings", "x.json"}, {"--agent", "cursor"}} {
		if code, _ := captureStdout(t, func() int { return Init(args) }); code != 2 {
			t.Errorf("%v: want exit 2, got %d", args, code)
		}
	}
}

// No terminal and no flags: Claude Code in this repo, as before, without a question.
func TestInitWithoutATerminalKeepsTheDefault(t *testing.T) {
	_, repo := askEnv(t, "", true)
	interactive = func() bool { return false }
	if code, out := captureStdout(t, func() int { return Init(nil) }); code != 0 {
		t.Fatalf("init exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) || hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Error("Claude Code in this repo only")
	}
}

// Uninstall asks the same way: here Codex only, this repo only, then the usual confirm.
func TestUninstallAsksWhichAgents(t *testing.T) {
	_, repo := askEnv(t, "2\n2\ny\n", false)
	interactive = func() bool { return false }
	captureStdout(t, func() int { return Init([]string{"--agent", "claude,codex"}) })
	interactive = func() bool { return true }
	code, out := captureStdout(t, func() int { return Uninstall(nil) })
	if code != 0 {
		t.Fatalf("uninstall exit %d: %s", code, out)
	}
	if hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Errorf("the Codex hook goes:\n%s", out)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Errorf("Claude Code wasn't picked: its hook stays:\n%s", out)
	}
}
