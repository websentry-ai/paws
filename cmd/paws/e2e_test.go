package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGateFailsOpen: whatever paws gate is fed, it always prints a valid decision and exits 0.
func TestGateFailsOpen(t *testing.T) {
	bin := buildPaws(t)
	home := t.TempDir()

	cases := []struct {
		name  string
		stdin string
	}{
		{"empty stdin", ""},
		{"garbage json", "not json at all"},
		{"empty object", "{}"},
		{"unknown tool", `{"hook_event_name":"PreToolUse","tool_name":"Frobnicate","tool_input":{}}`},
		{"no matching rule", `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := exec.Command(bin, "gate")
			cmd.Env = append(os.Environ(), "HOME="+home)
			cmd.Stdin = bytes.NewReader([]byte(c.stdin))
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			if err := cmd.Run(); err != nil {
				t.Fatalf("paws gate must exit 0 even on bad input, got: %v", err)
			}
			if got := stdout.String(); got != "{}" {
				t.Errorf("expected an allow decision {}, got %q", got)
			}
		})
	}
}

func TestGateDeniesAMatchingBlockRule(t *testing.T) {
	bin := buildPaws(t)
	home := t.TempDir()
	repo := t.TempDir()
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustMkdirAll(t, filepath.Join(repo, ".paws"))
	mustWriteFile(t, filepath.Join(repo, ".paws", "block.md"),
		"---\nname: block-rm\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: rm -rf\naction:\n  type: block\n---\n")

	cmd := exec.Command(bin, "gate")
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdin = bytes.NewReader([]byte(`{"cwd":"` + repo + `","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"rm -rf /tmp/x"}}`))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("gate exited non-zero: %v", err)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte(`"permissionDecision":"deny"`)) {
		t.Errorf("expected a deny decision, got %q", got)
	}
}

// The full wire path for the skill unlock: a real transcript file on disk, its path
// arriving only as transcript_path in the hook payload.
func TestGateSkillUnlockOverTheWire(t *testing.T) {
	bin := buildPaws(t)
	home := t.TempDir()
	repo := t.TempDir()
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustMkdirAll(t, filepath.Join(repo, ".paws"))
	mustWriteFile(t, filepath.Join(repo, ".paws", "release.md"),
		"---\nname: release-gate\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: ^echo release-\naction:\n  type: require-skill\n  skill: release-checklist\n---\n")

	loaded := filepath.Join(repo, "loaded.jsonl")
	mustWriteFile(t, loaded,
		`{"type":"user","isSidechain":false,"toolUseResult":{"success":true,"commandName":"release-checklist"}}`+"\n"+
			`{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[]}}`+"\n")
	never := filepath.Join(repo, "never.jsonl")
	mustWriteFile(t, never, `{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[]}}`+"\n")

	cases := []struct {
		name       string
		transcript string
		wantDeny   bool
	}{
		{"skill loaded in window", loaded, false},
		{"skill never loaded", never, true},
		{"transcript path missing from payload", "", true},
		{"transcript file does not exist", filepath.Join(repo, "gone.jsonl"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := `{"cwd":"` + repo + `","transcript_path":"` + c.transcript +
				`","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"echo release-1.2.0"}}`
			cmd := exec.Command(bin, "gate")
			cmd.Env = append(os.Environ(), "HOME="+home)
			cmd.Stdin = bytes.NewReader([]byte(in))
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			if err := cmd.Run(); err != nil {
				t.Fatalf("gate exited non-zero: %v", err)
			}
			denied := bytes.Contains(stdout.Bytes(), []byte(`"permissionDecision":"deny"`))
			if denied != c.wantDeny {
				t.Errorf("denied = %v, want %v (output %q)", denied, c.wantDeny, stdout.String())
			}
		})
	}
}

func buildPaws(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "paws")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
