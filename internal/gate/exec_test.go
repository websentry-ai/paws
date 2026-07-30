package gate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

func scriptRule(t *testing.T, dir, script string) *rule.Rule {
	t.Helper()
	r := mustRule(t, "---\nname: x\nevent: pre-tool\naction:\n  type: exec-script\n  script: "+script+"\n---\n")
	r.SourcePath = filepath.Join(dir, "rule.md")
	return r
}

func TestRunScriptAllow(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "ok.sh", "#!/bin/sh\nexit 0\n")
	oc := runScript(scriptRule(t, dir, "ok.sh"), Event{})
	if oc.verdict != Allow || oc.fired {
		t.Errorf("exit 0 should allow silently, got %+v", oc)
	}
}

func TestRunScriptDeny(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "deny.sh", "#!/bin/sh\necho 'nope' >&2\nexit 2\n")
	oc := runScript(scriptRule(t, dir, "deny.sh"), Event{})
	if oc.verdict != DenyRetry || oc.reason != "nope" {
		t.Errorf("exit 2 should deny with the stderr reason, got %+v", oc)
	}
}

func TestRunScriptMissingFailsOpen(t *testing.T) {
	dir := t.TempDir()
	oc := runScript(scriptRule(t, dir, "missing.sh"), Event{})
	if oc.verdict != Allow || oc.fired {
		t.Errorf("a missing script must fail open, got %+v", oc)
	}
}

func TestRunScriptDryRunNeverExecutes(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()
	dir := t.TempDir()
	writeScript(t, dir, "deny.sh", "#!/bin/sh\nexit 2\n")
	if oc := runScript(scriptRule(t, dir, "deny.sh"), Event{}); oc.fired {
		t.Errorf("DryRun must not run the script, got %+v", oc)
	}
}

// A backgrounded grandchild inheriting the stdio pipes must not wedge the gate past the timeout.
func TestRunScriptBackgroundedGrandchildDoesNotHang(t *testing.T) {
	orig := scriptTimeout
	scriptTimeout = 300 * time.Millisecond
	defer func() { scriptTimeout = orig }()

	dir := t.TempDir()
	writeScript(t, dir, "bg.sh", "#!/bin/sh\nsleep 5 &\nexit 0\n")

	done := make(chan struct{})
	go func() {
		runScript(scriptRule(t, dir, "bg.sh"), Event{})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runScript hung past its timeout because of a backgrounded grandchild")
	}
}

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
