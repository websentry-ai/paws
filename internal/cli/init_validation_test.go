package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Every way to get init or uninstall wrong ends with one clear line and the right exit, and nothing hangs.

// Ctrl-C at the scope question, or stdin closed at either question, sets up nothing.
func TestInitCancelledAtAnyStep(t *testing.T) {
	for name, c := range map[string]struct {
		input string
		raw   bool
	}{
		"Ctrl-C at the scope question": {"\r\x03", true},
		"stdin closed at the agents":   {"", true},
		"stdin closed at the scope":    {"\r", true},
		"stdin closed, numbered":       {"1\n", false},
	} {
		_, repo := askEnv(t, c.input, c.raw)
		if code, out := captureStdout(t, func() int { return Init(nil) }); code != 1 || hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
			t.Errorf("%s: want exit 1 and nothing set up, got %d\n%s", name, code, out)
		}
	}
}

// A terminal too narrow for the menu gets the numbered questions instead: a redraw would wrap and garble.
func TestInitOnANarrowTerminalAsksWithNumbers(t *testing.T) {
	_, repo := askEnv(t, "1\n1\n", true)
	old := termWidth
	termWidth = func() int { return 30 }
	t.Cleanup(func() { termWidth = old })
	code, out := captureStdout(t, func() int { return Init(nil) })
	if code != 0 || !strings.Contains(out, "1) Claude Code") || !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Errorf("init exit %d:\n%s", code, out)
	}
}

// Bad flags exit 2 with a usage line; --settings works with --agent codex, for a hooks.json of the user's choosing.
func TestInitAndUninstallFlags(t *testing.T) {
	_, repo := askEnv(t, "", true)
	interactive = func() bool { return false }
	for _, args := range [][]string{{"--global", "--project"}, {"--agent", "cursor"}, {"--agent"}, {"--settings"}, {"--bogus"}} {
		if code, _ := captureStdout(t, func() int { return Init(args) }); code != 2 {
			t.Errorf("init %v: want exit 2, got %d", args, code)
		}
	}
	if code, _ := captureStdout(t, func() int { return Uninstall([]string{"--agent", "cursor"}) }); code != 2 {
		t.Errorf("uninstall --agent cursor: want exit 2, got %d", code)
	}
	hooks := filepath.Join(repo, "my-hooks.json")
	if code, _ := captureStdout(t, func() int { return Init([]string{"--agent", "codex", "--settings", hooks}) }); code != 0 || !hasPawsHook(hooks) {
		t.Errorf("--agent codex --settings writes the Codex hook there: exit %d", code)
	}
}

// Outside a git repo, init says it sets up this folder only.
func TestInitOutsideARepoSaysSo(t *testing.T) {
	home, _ := askEnv(t, "", true)
	interactive = func() bool { return false }
	dir := filepath.Join(home, "notes")
	os.MkdirAll(dir, 0o755)
	t.Chdir(dir)
	code, out := captureStdout(t, func() int { return Init(nil) })
	if code != 0 || !strings.Contains(out, "Not in a git repo") || !hasPawsHook(filepath.Join(dir, ".claude", "settings.json")) {
		t.Errorf("init exit %d:\n%s", code, out)
	}
}

// A settings file paws can't parse, or a folder it can't write, fails with exit 1 and leaves the file as it was.
func TestInitFailsCleanly(t *testing.T) {
	home, repo := askEnv(t, "", true)
	interactive = func() bool { return false }
	settings := filepath.Join(repo, ".claude", "settings.json")
	writeFile(t, settings, "{not json")
	if code, _ := captureStdout(t, func() int { return Init(nil) }); code != 1 {
		t.Errorf("invalid JSON: want exit 1, got %d", code)
	}
	if data, _ := os.ReadFile(settings); string(data) != "{not json" {
		t.Errorf("the file stays as it was: %q", data)
	}
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // no read-only folders to test with
	}
	os.Chmod(filepath.Join(home, ".claude"), 0o500)
	t.Cleanup(func() { os.Chmod(filepath.Join(home, ".claude"), 0o755) })
	if code, _ := captureStdout(t, func() int { return Init([]string{"--global"}) }); code != 1 {
		t.Errorf("an unwritable ~/.claude: want exit 1, got %d", code)
	}
}

// init twice changes nothing and says so; uninstall with nothing set up, or of an agent that isn't, removes nothing.
func TestRunningTwiceAndRemovingNothing(t *testing.T) {
	_, repo := askEnv(t, "2\n2\n", false)
	interactive = func() bool { return false }
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) }); code != 0 || !strings.Contains(out, "Nothing to remove") {
		t.Errorf("nothing set up: exit %d\n%s", code, out)
	}
	captureStdout(t, func() int { return Init(nil) })
	before, _ := os.ReadFile(filepath.Join(repo, ".claude", "settings.json"))
	if code, out := captureStdout(t, func() int { return Init(nil) }); code != 0 || !strings.Contains(out, "already registered") {
		t.Errorf("init twice: exit %d\n%s", code, out)
	}
	if after, _ := os.ReadFile(filepath.Join(repo, ".claude", "settings.json")); string(after) != string(before) {
		t.Error("init twice leaves the file as it was")
	}
	interactive = func() bool { return true }
	// Codex isn't set up: picked anyway, this repo only, there's nothing of it to remove.
	if code, out := captureStdout(t, func() int { return Uninstall(nil) }); code != 0 || !strings.Contains(out, "Nothing to remove") ||
		!hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Errorf("uninstall of an agent that isn't set up: exit %d\n%s", code, out)
	}
}

// Uninstall cancelled at its questions, or with stdin closed at the confirm, removes nothing.
func TestUninstallCancelled(t *testing.T) {
	for name, input := range map[string]string{"Ctrl-C at the agents": "\x03", "Ctrl-C at the scope": "\r\x03", "no answer at the confirm": "\r\r"} {
		_, repo := askEnv(t, input, true)
		interactive = func() bool { return false }
		captureStdout(t, func() int { return Init(nil) })
		interactive = func() bool { return true }
		if code, out := captureStdout(t, func() int { return Uninstall(nil) }); code != 1 || !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
			t.Errorf("%s: want exit 1 and nothing removed, got %d\n%s", name, code, out)
		}
	}
}
