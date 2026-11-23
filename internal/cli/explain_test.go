package cli

import (
	"os"
	"path/filepath"
	"regexp"
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

// explain shows what each check makes of the call: a retry check passes or fails, a never check is broken or not, and
// a check no case fits doesn't apply.
func TestExplainShowsChecksOnTheCall(t *testing.T) {
	resetDryRun(t)
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Setenv("PAWS_HOME", filepath.Join(repo, "paws-home"))
	t.Chdir(repo)
	putRule(t, filepath.Join(repo, ".paws"), "git.md", `---
name: git
event: pre-tool
matcher: {tool: Bash, command_regex: '\bgit\s+(commit|push)\b'}
action: {type: require-skill, skill: git}
expect:
  - id: format
    when: retry
    cases:
      - match: 'any(call.commands, # matches "^git commit\\b")'
        if: 'call.text matches "fix: "'
  - id: no-force
    when: never
    if: 'any(call.commands, # matches "^git push\\b.* -f( |$)")'
---
`)
	_, out := captureStdout(t, func() int { return Explain([]string{`git commit -m "fix: x"`}) })
	if !regexp.MustCompile(`format\s+pass\n`).MatchString(out) || !regexp.MustCompile(`no-force\s+pass`).MatchString(out) {
		t.Errorf("commit: %q", out)
	}
	_, out = captureStdout(t, func() int { return Explain([]string{"git push -f"}) })
	if !regexp.MustCompile(`format\s+doesn't apply`).MatchString(out) || !regexp.MustCompile(`no-force\s+FAIL`).MatchString(out) {
		t.Errorf("force push: %q", out)
	}
}

// An after check never counts the retry: explain says so instead of reporting a pass.
func TestExplainAfterCheckIsNotTheRetry(t *testing.T) {
	resetDryRun(t)
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Setenv("PAWS_HOME", filepath.Join(repo, "paws-home"))
	t.Chdir(repo)
	putRule(t, filepath.Join(repo, ".paws"), "git.md", "---\nname: git\nevent: pre-tool\nmatcher: {tool: Bash, command_regex: 'git push'}\naction: {type: require-skill, skill: git}\nexpect:\n  - id: checked\n    when: after\n    if: 'call.text matches \"git push\"'\n---\n")
	_, out := captureStdout(t, func() int { return Explain([]string{"git push"}) })
	if !strings.Contains(out, "checked") || !strings.Contains(out, "not the retry") {
		t.Errorf("after check: %q", out)
	}
}
