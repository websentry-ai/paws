package hook

import (
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

func TestToEventKind(t *testing.T) {
	in := &Input{HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: map[string]any{"command": "ls"}}
	if ev := ToEvent(in, nil); ev.Kind != rule.KindTool || ev.Target != "ls" {
		t.Errorf("PreToolUse should map to KindTool with the extracted command, got %+v", ev)
	}
	in = &Input{HookEventName: "UserPromptSubmit", Prompt: "hi"}
	if ev := ToEvent(in, nil); ev.Kind != rule.KindPrompt || ev.Target != "hi" {
		t.Errorf("UserPromptSubmit should map to KindPrompt with the prompt text, got %+v", ev)
	}
	in = &Input{HookEventName: "SessionStart"}
	if ev := ToEvent(in, nil); ev.Kind != "" {
		t.Errorf("unmapped events should have no Kind, got %+v", ev)
	}
}

func TestRenderDenyAsk(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Deny, Reason: "no"})
	if out.SpecificOutput == nil || out.SpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("Deny should render permissionDecision=deny, got %+v", out)
	}
	out = Render("PreToolUse", gate.Decision{Verdict: gate.DenyRetry, Reason: "retry"})
	if out.SpecificOutput.PermissionDecision != "deny" {
		t.Errorf("DenyRetry should also render as deny, got %+v", out)
	}
	out = Render("PreToolUse", gate.Decision{Verdict: gate.Ask, Reason: "confirm"})
	if out.SpecificOutput.PermissionDecision != "ask" {
		t.Errorf("Ask should render permissionDecision=ask, got %+v", out)
	}
}

func TestRenderUserPromptBlock(t *testing.T) {
	out := Render("UserPromptSubmit", gate.Decision{Verdict: gate.Deny, Reason: "no"})
	if out.Decision != "block" || out.Reason != "no" {
		t.Fatalf("UserPromptSubmit deny must use the top-level decision/reason form, got %+v", out)
	}
	if out.SpecificOutput != nil {
		t.Errorf("block form should not also set hookSpecificOutput, got %+v", out)
	}
}

func TestRenderWarnContextOnly(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Allow, Context: "careful"})
	if out.SpecificOutput == nil || out.SpecificOutput.PermissionDecision != "" || !strings.Contains(out.SpecificOutput.AdditionalContext, "careful") {
		t.Fatalf("allow+context must omit permissionDecision, got %+v", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	out := Render("PreToolUse", gate.Decision{Verdict: gate.Allow})
	if out.SpecificOutput != nil || out.Decision != "" {
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
	up := SettingsEntry("UserPromptSubmit", "'/x/paws' gate")
	if _, ok := up["matcher"]; ok {
		t.Error("UserPromptSubmit entries must not carry a matcher — Claude Code ignores it there")
	}
}
