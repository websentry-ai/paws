package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

func TestToEventKind(t *testing.T) {
	in := &Input{HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: map[string]any{"command": "ls"}}
	if ev := ToEvent(in); ev.Kind != rule.KindTool || ev.Target != "ls" {
		t.Errorf("PreToolUse should map to KindTool with the extracted command, got %+v", ev)
	}
	in = &Input{HookEventName: "SessionStart", Source: "compact"}
	if ev := ToEvent(in); ev.Kind != rule.KindSession || ev.Target != "compact" {
		t.Errorf("SessionStart should map to KindSession with the source, got %+v", ev)
	}
	// Stop and UserPromptSubmit run no rules: telemetry and the portal round only.
	for _, name := range []string{"Stop", "UserPromptSubmit"} {
		if ev := ToEvent(&Input{HookEventName: name}); ev.Kind != "" {
			t.Errorf("%s should have no Kind, got %+v", name, ev)
		}
	}
}

// A subagent's call carries the main session's transcript_path and its agent_id; its own log sits at
// <session>/subagents/agent-<id>.jsonl, and that is the one paws must read for the subagent's loads.
func TestToEventFindsTheSubagentsOwnLog(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "sess.jsonl")
	sub := filepath.Join(dir, "sess", "subagents", "agent-a1.jsonl")
	if err := os.MkdirAll(filepath.Dir(sub), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{main, sub} {
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	in := &Input{HookEventName: "PreToolUse", ToolName: "Bash", TranscriptPath: main}
	if ev := ToEvent(in); ev.Transcript != main || ev.Own != "" {
		t.Errorf("main thread: transcript %q own %q, want %q and none", ev.Transcript, ev.Own, main)
	}
	in.AgentID = "a1"
	if ev := ToEvent(in); ev.Transcript != main || ev.Own != sub {
		t.Errorf("subagent: transcript %q own %q, want the session log %q and its own %q", ev.Transcript, ev.Own, main, sub)
	}
	in.AgentID = "missing"
	if ev := ToEvent(in); ev.Own != "" {
		t.Errorf("subagent without its own log: own %q, want none", ev.Own)
	}
	// An id that would leave <session>/subagents/ is never used, even when the file it names exists.
	other := filepath.Join(dir, "sess", "agent-x.jsonl")
	if err := os.WriteFile(other, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a/../../agent-x", "x/../../agent-x"} {
		in.AgentID = id
		if ev := ToEvent(in); ev.Own != "" {
			t.Errorf("agent_id %q: own %q, want none", id, ev.Own)
		}
	}
}

func TestRenderDeny(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Deny, Reason: "no"})
	if out.SpecificOutput == nil || out.SpecificOutput.PermissionDecision != "deny" || out.SpecificOutput.PermissionDecisionReason != "no" {
		t.Fatalf("Deny should render permissionDecision=deny with the reason, got %+v", out)
	}
}

func TestRenderContextOnly(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Allow, Context: "careful"})
	if out.SpecificOutput == nil || out.SpecificOutput.PermissionDecision != "" || !strings.Contains(out.SpecificOutput.AdditionalContext, "careful") {
		t.Fatalf("allow+context must omit permissionDecision, got %+v", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Allow})
	if out.SpecificOutput != nil || out.SystemMessage != "" {
		t.Errorf("no-op decision should marshal to {}, got %+v", out)
	}
}

func TestSettingsEntryTimeoutIsSeconds(t *testing.T) {
	e := SettingsEntry("PreToolUse", "'/x/paws' gate")
	inner := e["hooks"].([]any)[0].(map[string]any)
	if inner["timeout"] != 15 {
		t.Errorf("Claude Code hook timeout is in seconds; want 15, got %v", inner["timeout"])
	}
	if _, ok := e["matcher"]; !ok {
		t.Error("PreToolUse entries need a matcher")
	}
	ss := SettingsEntry("SessionStart", "'/x/paws' gate")
	if _, ok := ss["matcher"]; ok {
		t.Error("SessionStart entries must not carry a tool matcher")
	}
}

func TestRenderSessionStartContext(t *testing.T) {
	out := Render("SessionStart", gate.Decision{Verdict: gate.Allow, Context: "load caveman"})
	if out.SpecificOutput == nil || out.SpecificOutput.HookEventName != "SessionStart" ||
		out.SpecificOutput.AdditionalContext != "load caveman" || out.SpecificOutput.PermissionDecision != "" {
		t.Errorf("SessionStart context should render as additionalContext only, got %+v", out.SpecificOutput)
	}
}

func TestTelemetryHooksAreAsync(t *testing.T) {
	for _, ev := range []string{"UserPromptSubmit", "PostToolUse", "Stop"} {
		e := SettingsEntry(ev, "'/x/paws' gate")
		inner := e["hooks"].([]any)[0].(map[string]any)
		if inner["async"] != true {
			t.Errorf("%s hook should be async, got %v", ev, inner)
		}
	}
	pre := SettingsEntry("PreToolUse", "'/x/paws' gate")["hooks"].([]any)[0].(map[string]any)
	if _, ok := pre["async"]; ok {
		t.Error("PreToolUse decides; it must not be async")
	}
}
