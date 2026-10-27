package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/rule"
)

func TestLogAuditWritesJSONL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	r := mustRule(t, "---\nname: audited\nevent: pre-tool\naction:\n  type: audit\n---\n")
	logAudit(Event{Agent: "claude-code", Tool: "Bash", Target: "ls"}, r, outcome{verdict: Allow, fired: true})

	data, err := os.ReadFile(filepath.Join(home, ".paws", "audit.log"))
	if err != nil {
		t.Fatalf("expected an audit log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, `"rule":"audited"`) || !strings.Contains(line, `"verdict":"allow"`) {
		t.Errorf("audit record missing expected fields: %s", line)
	}
}

func TestLogAuditDryRunWritesNothing(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()
	home := t.TempDir()
	t.Setenv("HOME", home)

	r := mustRule(t, "---\nname: audited\nevent: pre-tool\naction:\n  type: audit\n---\n")
	logAudit(Event{}, r, outcome{verdict: Allow, fired: true})

	if _, err := os.Stat(filepath.Join(home, ".paws", "audit.log")); !os.IsNotExist(err) {
		t.Error("DryRun must not write the audit log")
	}
}

func TestDecideAuditFiresAndStaysInvisible(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	r := mustRule(t, "---\nname: a\nevent: pre-tool\naction:\n  type: audit\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "ls", Agent: "claude-code"}
	d := Decide(ev, []*rule.Rule{r})
	if d.Verdict != Allow || d.Context != "" {
		t.Fatalf("audit must resolve to a silent allow, got %+v", d)
	}
	if _, err := os.Stat(filepath.Join(home, ".paws", "audit.log")); err != nil {
		t.Error("audit rule should still have logged despite the silent decision")
	}
}

func TestLogAuditHonorsPawsHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	custom := filepath.Join(t.TempDir(), "paws-home")
	t.Setenv("PAWS_HOME", custom)

	r := mustRule(t, "---\nname: audited\nevent: pre-tool\naction:\n  type: audit\n---\n")
	logAudit(Event{Agent: "claude-code", Tool: "Bash", Target: "ls"}, r, outcome{verdict: Allow, fired: true})

	if _, err := os.Stat(filepath.Join(custom, "audit.log")); err != nil {
		t.Errorf("expected audit log under $PAWS_HOME: %v", err)
	}
}

func TestAuditMarksContextAdded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := mustRule(t, "---\nname: s\nevent: session-start\naction:\n  type: require-skill\n  skill: k\n---\n")
	Decide(Event{Kind: rule.KindSession, Target: "startup"}, []*rule.Rule{r})
	data, _ := os.ReadFile(filepath.Join(home, ".paws", "audit.log"))
	if !strings.Contains(string(data), `"context_added":true`) {
		t.Errorf("session-start require-skill should audit context_added, got %s", data)
	}
}
