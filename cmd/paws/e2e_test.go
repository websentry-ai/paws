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
