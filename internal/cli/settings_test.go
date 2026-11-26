package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/hook"
)

const pawsCmd = `"$HOME/.paws/bin/paws" gate`

func TestEnsureHooksMergesAndIsIdempotent(t *testing.T) {
	doc := settingsDoc{
		"env":   json.RawMessage(`{"FOO":"bar"}`),
		"hooks": json.RawMessage(`{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"\"/x/unbound.py\"","timeout":15000}]}]}`),
	}

	changed, err := ensureHooks(doc, pawsCmd, false)
	if err != nil || !changed {
		t.Fatalf("first ensureHooks should report a change: %v %v", changed, err)
	}
	changed, err = ensureHooks(doc, pawsCmd, false)
	if err != nil || changed {
		t.Fatalf("second ensureHooks should be a no-op: %v %v", changed, err)
	}

	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	if err := json.Unmarshal(hooks["PreToolUse"], &pre); err != nil {
		t.Fatal(err)
	}
	if len(pre) != 2 {
		t.Fatalf("expected 2 PreToolUse entries, got %d", len(pre))
	}
	if !containsCommand(pre, `"/x/unbound.py"`) {
		t.Error("existing unbound.py entry was dropped")
	}
	if !containsCommand(pre, pawsCmd) {
		t.Error("paws entry was not added")
	}
	if string(doc["env"]) != `{"FOO":"bar"}` {
		t.Error("env block was mutated")
	}
}

func TestEnsureHooksFromEmpty(t *testing.T) {
	doc := settingsDoc{}
	if changed, err := ensureHooks(doc, pawsCmd, false); err != nil || !changed {
		t.Fatalf("should add hooks to empty settings: %v %v", changed, err)
	}
	hooks, err := hooksBlock(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Fatal("PreToolUse not created")
	}
}

func TestEnsureHooksRepointsMovedBinary(t *testing.T) {
	doc := settingsDoc{}
	if _, err := ensureHooks(doc, `'/old/path/paws' gate`, false); err != nil {
		t.Fatal(err)
	}
	changed, err := ensureHooks(doc, `'/new/path/paws' gate`, false)
	if err != nil || !changed {
		t.Fatalf("repointing to a new path should report a change: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 {
		t.Fatalf("moved binary should repoint the existing entry, not duplicate it; got %d entries", len(pre))
	}
	if !containsCommand(pre, `'/new/path/paws' gate`) {
		t.Error("entry was not repointed to the new path")
	}
}

// init registers UserPromptSubmit as an async hook, and repoints a blocking paws entry an earlier build
// left there, keeping other hooks on the event.
func TestEnsureHooksRegistersAsyncPromptHook(t *testing.T) {
	old, _ := json.Marshal(map[string]any{"hooks": []any{map[string]any{"type": "command", "command": pawsCmd, "timeout": 15}}})
	other := `{"hooks":[{"type":"command","command":"/x/other.sh"}]}`
	doc := settingsDoc{"hooks": json.RawMessage(`{"UserPromptSubmit":[` + string(old) + `,` + other + `]}`)}
	if changed, err := ensureHooks(doc, pawsCmd, false); err != nil || !changed {
		t.Fatalf("ensureHooks: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var up []json.RawMessage
	json.Unmarshal(hooks["UserPromptSubmit"], &up)
	if len(up) != 2 || !containsCommand(up, pawsCmd) || !containsCommand(up, "/x/other.sh") {
		t.Fatalf("want the paws entry and the other hook, got %s", hooks["UserPromptSubmit"])
	}
	if !jsonEqual(up[pawsEntryIndex(up)], hook.RawEntry("UserPromptSubmit", pawsCmd)) ||
		!strings.Contains(string(up[pawsEntryIndex(up)]), `"async":true`) {
		t.Errorf("the paws UserPromptSubmit entry should be the async one, got %s", up[pawsEntryIndex(up)])
	}
}

func TestIsPawsCommand(t *testing.T) {
	for _, c := range []string{
		pawsCmd,
		`'/opt/paws' gate`,
		`'C:\Users\me\.paws\bin\paws.exe' gate`,
		`PAWS_MANAGED=1 '/Users/me/.paws/bin/paws' gate`, // written by 0.3.0 and earlier
		`'/opt/O'\''Brien/paws/bin/paws' gate`,           // shellQuote of a path with an apostrophe
		`PAWS_MANAGED=1 '/home/o'\''neil/.paws/bin/paws' gate`,
		`PAWS_MANAGED=1 '/Users/me/.paws/bin/paws-dev' gate`, // 0.3.0 kept a source build's name
	} {
		if !isPawsCommand(c) {
			t.Errorf("isPawsCommand(%q) = false, want true", c)
		}
	}
	for _, c := range []string{
		`'/opt/other/paws-utils/check' gate`, // "paws" only as a path component
		`echo paws && something gate`,
		`'/opt/paws' explain`,
		`my-scanner --strict; /x/paws gate`, // a chained command is someone else's hook
		`curl -s x | sh; paws gate`,
		`'/a/scan' && '/x/paws' gate`,
		`"$(evil)" "/x/paws" gate`,
		`"$(some-cmd)/x/paws" gate`, // expands inside double quotes
		`"$HOME/$(x)/paws" gate`,
		`/usr/local/bin/paws gate`, // never written by init
		`'/a'b'/paws' gate`,
		`'/Users/me/.paws/bin/paws-dev' gate`,          // unmarked: only the name paws counts
		`PAWS_MANAGED=1 '/x/scan'; '/x/paws-dev' gate`, // marked, but not one quoted path
	} {
		if isPawsCommand(c) {
			t.Errorf("isPawsCommand(%q) = true, want false", c)
		}
	}
}

// A hook a user put in the same entry as ours survives init (repoint) and uninstall.
func TestSiblingHookInPawsEntrySurvives(t *testing.T) {
	entry := `{"matcher":"*","hooks":[{"type":"command","command":"PAWS_MANAGED=1 '/old/paws' gate"},{"type":"command","command":"/x/secret-scan"}]}`
	doc := settingsDoc{"hooks": json.RawMessage(`{"PreToolUse":[` + entry + `]}`)}
	if _, err := ensureHooks(doc, pawsCmd, false); err != nil {
		t.Fatal(err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if !containsCommand(pre, "/x/secret-scan") || !containsCommand(pre, pawsCmd) || containsCommand(pre, "PAWS_MANAGED=1 '/old/paws' gate") {
		t.Fatalf("init should repoint paws and keep the sibling hook, got %s", hooks["PreToolUse"])
	}
	if _, err := removeHooks(doc); err != nil {
		t.Fatal(err)
	}
	hooks, _ = hooksBlock(doc)
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 || !containsCommand(pre, "/x/secret-scan") || containsCommand(pre, pawsCmd) {
		t.Fatalf("uninstall should leave only the sibling hook, got %s", hooks["PreToolUse"])
	}
}

// Re-running init over a 0.3.0 entry replaces it instead of adding a second paws hook.
func TestEnsureHooksUpgradesLegacyEntry(t *testing.T) {
	doc := settingsDoc{}
	if _, err := ensureHooks(doc, `PAWS_MANAGED=1 '/Users/me/.paws/bin/paws' gate`, false); err != nil {
		t.Fatal(err)
	}
	if changed, err := ensureHooks(doc, pawsCmd, false); err != nil || !changed {
		t.Fatalf("ensureHooks: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 || !containsCommand(pre, pawsCmd) {
		t.Fatalf("want the legacy entry replaced by %s, got %s", pawsCmd, hooks["PreToolUse"])
	}
}

func TestRemoveHooksStripsOnlyPaws(t *testing.T) {
	doc := settingsDoc{
		"hooks": json.RawMessage(`{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"\"/x/unbound.py\"","timeout":15000}]}]}`),
	}
	if _, err := ensureHooks(doc, pawsCmd, false); err != nil {
		t.Fatal(err)
	}
	changed, err := removeHooks(doc)
	if err != nil || !changed {
		t.Fatalf("removeHooks should report a change: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 || containsCommand(pre, pawsCmd) {
		t.Fatalf("expected only the unbound.py entry left, got %s", hooks["PreToolUse"])
	}
}

func TestReadSettingsMalformedDoesNotDestroy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{invalid`), 0o644)

	if _, err := readSettings(path); err == nil {
		t.Fatal("malformed settings.json should error, not silently reset")
	}
	data, _ := os.ReadFile(path)
	if string(data) != `{invalid` {
		t.Error("readSettings must not modify the file on parse failure")
	}
}

func TestReadSettingsMissingIsEmpty(t *testing.T) {
	doc, err := readSettings(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || len(doc) != 0 {
		t.Fatalf("missing file should yield an empty doc, got %v %v", doc, err)
	}
}

func TestReadSettingsEmptyFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(" \n"), 0o644)
	if doc, err := readSettings(path); err != nil || len(doc) != 0 {
		t.Fatalf("an empty file should yield an empty doc, got %v %v", doc, err)
	}
}

func TestWriteSettingsRoundTripsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	doc := settingsDoc{"someOtherPlugin": json.RawMessage(`{"n":123456789012345}`)}
	if err := writeSettings(path, doc); err != nil {
		t.Fatal(err)
	}
	got, err := readSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	var v struct{ N int64 }
	if err := json.Unmarshal(got["someOtherPlugin"], &v); err != nil || v.N != 123456789012345 {
		t.Errorf("large integer lost precision, got %s", got["someOtherPlugin"])
	}
}

func containsCommand(list []json.RawMessage, command string) bool {
	for _, entry := range list {
		var e struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		}
		if json.Unmarshal(entry, &e) != nil {
			continue
		}
		for _, h := range e.Hooks {
			if h.Command == command {
				return true
			}
		}
	}
	return false
}

func TestGuardedCommandIsStillPawsAndUnguards(t *testing.T) {
	for _, plain := range []string{"unbound-paws gate", `"$HOME/.paws/bin/paws" gate`, `'/opt/it'\''s/paws' gate`} {
		for _, event := range []string{"PreToolUse", "Stop"} {
			g := guardedCommand(plain, event)
			if !isPawsCommand(g) || unguard(g) != plain {
				t.Errorf("%s: isPawsCommand=%v unguard=%q", g, isPawsCommand(g), unguard(g))
			}
		}
	}
	for _, bad := range []string{
		"command -v unbound-paws gate",
		"command -v x" + guardMid("2") + "rm -rf ~; unbound-paws gate",
		"command -v unbound-paws" + guardMid("2") + "unbound-paws gate; rm -rf ~",
		"command -v unbound-paws" + guardMid("3") + "unbound-paws gate",
		"command -v unbound-paws" + guardMid("2") + "other gate",
	} {
		if isPawsCommand(bad) {
			t.Errorf("isPawsCommand accepted %q", bad)
		}
	}
	if guardedCommand("unbound-paws gate", "PreToolUse") == guardedCommand("unbound-paws gate", "Stop") {
		t.Error("PreToolUse blocks with exit 2; the other events must not")
	}
}

// The guard runs in sh. Without the binary it prints the install hint to stderr and exits 2 for PreToolUse (1 for
// the other events); with it, the event on stdin reaches the binary.
func TestGuardedCommandInSh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the guard needs sh")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip(err)
	}
	run := func(path, event, stdin string) (int, string, string) {
		cmd := exec.Command(sh, "-c", guardedCommand("unbound-paws gate", event))
		cmd.Env = []string{"PATH=" + path}
		cmd.Stdin = strings.NewReader(stdin)
		var out, errb strings.Builder
		cmd.Stdout, cmd.Stderr = &out, &errb
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return code, out.String(), errb.String()
	}
	empty := t.TempDir()
	for event, want := range map[string]int{"PreToolUse": 2, "Stop": 1} {
		code, out, errOut := run(empty, event, "{}")
		if code != want || out != "" || strings.TrimSpace(errOut) != missingMsg {
			t.Errorf("%s without paws: exit %d (want %d), stdout %q, stderr %q", event, code, want, out, errOut)
		}
	}
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "unbound-paws"), []byte("#!/bin/sh\n/bin/cat\n"), 0o755)
	if code, out, errOut := run(bin+":"+empty, "PreToolUse", `{"tool":"x"}`); code != 0 || out != `{"tool":"x"}` || errOut != "" {
		t.Errorf("with paws: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestWriteSettingsKeepsTheGuardReadable(t *testing.T) {
	doc := settingsDoc{}
	ensureHooks(doc, "unbound-paws gate", true)
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := writeSettings(path, doc); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "\\u003e") || !strings.Contains(string(data), ">/dev/null 2>&1") {
		t.Errorf("settings.json escapes the shell redirects:\n%s", data)
	}
	if !hasPawsHook(path) {
		t.Error("the file with the guard no longer reads as a paws hook")
	}
}

// A sibling hook with a literal backslash-u0026 in its command must come back byte for byte, not as an invalid escape.
func TestWriteSettingsKeepsEscapedBackslashesInOtherHooks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	u := "\\" + "u0026" // a backslash, then u0026, as text
	sibling := "printf '" + u + "' > /dev/null 2>&1"
	src := `{"hooks":{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"printf '\\` + "u0026" + `' > /dev/null 2>&1"}]}]}}`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := readSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ensureHooks(doc, "unbound-paws gate", true); err != nil {
		t.Fatal(err)
	}
	if err := writeSettings(path, doc); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var back struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("settings.json is no longer valid JSON: %v\n%s", err, data)
	}
	found := false
	for _, e := range back.Hooks["PreToolUse"] {
		for _, h := range e.Hooks {
			found = found || h.Command == sibling
		}
	}
	if !found {
		t.Errorf("the sibling hook changed:\n%s", data)
	}
	if strings.Contains(string(data), "\\"+"u003e") {
		t.Errorf("a > was escaped:\n%s", data)
	}
}

func TestInitGuardsOnlyTheUnboundPawsOnPathCommand(t *testing.T) {
	if !hookGuard() {
		t.Skip("no guard on this OS")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("PATH", "") // off PATH: init copies the binary and registers "$HOME/.paws/bin/paws" gate
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	if code, _ := captureStdout(t, func() int { return Init(nil) }); code != 0 {
		t.Fatalf("init exit %d", code)
	}
	got := pawsCommandIn(filepath.Join(repo, ".claude", "settings.json"))
	if strings.Contains(got, "command -v") || !strings.HasSuffix(got, ` gate`) {
		t.Errorf("a copied binary can't be installed by npm; the hint would not help, so no guard: %q", got)
	}
}
