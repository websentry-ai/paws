package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/gate"
)

// explain sets gate.DryRun for the process; put it back so later tests write audit records.
func resetDryRun(t *testing.T) { t.Cleanup(func() { gate.DryRun = false }) }

func putRule(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExplainToolPutsTextInThatToolsField(t *testing.T) {
	resetDryRun(t)
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Setenv("PAWS_HOME", filepath.Join(repo, "paws-home"))
	t.Chdir(repo)
	putRule(t, filepath.Join(repo, ".paws"), "web.md",
		"---\nname: web\nevent: pre-tool\nmatcher:\n  tool: WebFetch\n  command_regex: example\\.com\naction:\n  type: require-skill\n  skill: s\n---\n")
	code, out := captureStdout(t, func() int { return Explain([]string{"--tool", "WebFetch", "https://example.com/x"}) })
	if code != 0 || !strings.Contains(out, "input : https://example.com/x") || !strings.Contains(out, "DENY") {
		t.Errorf("explain --tool WebFetch: exit %d, output %q", code, out)
	}
}

func TestExplainWarnsAboutDuplicateRuleNames(t *testing.T) {
	resetDryRun(t)
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Setenv("PAWS_HOME", filepath.Join(repo, "paws-home"))
	t.Chdir(repo)
	r := "---\nname: same\nevent: pre-commit\naction:\n  type: require-skill\n  skill: s\n---\n"
	putRule(t, filepath.Join(repo, ".paws"), "a.md", r)
	putRule(t, filepath.Join(repo, ".paws"), "b.md", r)
	_, out := captureStdout(t, func() int { return Explain([]string{"git commit -m x"}) })
	if !strings.Contains(out, `two rules named "same"`) {
		t.Errorf("no duplicate-name warning: %q", out)
	}
}

func TestExplainWithNoInputShowsUsage(t *testing.T) {
	resetDryRun(t)
	if code := Explain(nil); code != 2 {
		t.Errorf("explain with no input: exit %d, want 2", code)
	}
}
