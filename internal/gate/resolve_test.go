package gate

import (
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/rule"
)

func mustRule(t *testing.T, md string) *rule.Rule {
	t.Helper()
	r, err := rule.Parse([]byte(md), "test.md")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func TestResolveSeverity(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	block := mustRule(t, "---\nname: b\nevent: pre-tool\naction:\n  type: block\n---\n")
	skill := mustRule(t, "---\nname: s\nevent: pre-tool\naction:\n  type: require-skill\n  skill: foo\n---\n")
	pause := mustRule(t, "---\nname: a\nevent: pre-tool\naction:\n  type: warn\n---\n")
	audit := mustRule(t, "---\nname: w\nevent: pre-tool\naction:\n  type: audit\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "x"}

	d := resolve(ev, []*rule.Rule{audit, pause, skill, block})
	if d.Verdict != Deny {
		t.Fatalf("expected Deny, got %+v", d)
	}
	if !strings.Contains(d.Reason, "Blocked by paws") {
		t.Errorf("block message should win, got %q", d.Reason)
	}

	d = resolve(ev, []*rule.Rule{audit, pause})
	if d.Verdict != Ask {
		t.Fatalf("expected Ask, got %+v", d)
	}

	d = resolve(ev, []*rule.Rule{audit})
	if d.Verdict != Allow || d.Context != "" {
		t.Fatalf("audit alone must allow silently, got %+v", d)
	}

	if d = resolve(ev, nil); d.Verdict != Allow || d.Context != "" {
		t.Errorf("no match should be empty allow, got %+v", d)
	}
}

func TestDecideEndToEnd(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	rules := []*rule.Rule{
		mustRule(t, "---\nname: aws\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: ^terraform apply\naction:\n  type: require-skill\n  skill: aws-tagging\n---\n"),
	}
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "terraform apply -auto-approve"}
	if d := Decide(ev, rules); d.Verdict != DenyRetry {
		t.Fatalf("expected DenyRetry, got %+v", d)
	}
	ev.Kind = ""
	if d := Decide(ev, rules); d.Verdict != Allow {
		t.Errorf("pre-tool rule fired on a kind-less event: %+v", d)
	}
}

func TestPromptInject(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	inject := mustRule(t, "---\nname: i\nevent: user-prompt\nmatcher:\n  command_regex: deploy\naction:\n  type: prompt-inject\n  message: Remember the deploy checklist.\n---\n")
	ev := Event{Kind: rule.KindPrompt, Target: "please deploy to prod"}
	if d := Decide(ev, []*rule.Rule{inject}); !strings.Contains(d.Context, "checklist") {
		t.Fatalf("expected injected context, got %+v", d)
	}
	ev.Target = "hello"
	if d := Decide(ev, []*rule.Rule{inject}); d.Context != "" {
		t.Errorf("should not inject on non-matching prompt: %+v", d)
	}
}

func TestUserPromptDeny(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	block := mustRule(t, "---\nname: b\nevent: user-prompt\naction:\n  type: block\n---\n")
	ev := Event{Kind: rule.KindPrompt, Target: "anything"}
	d := Decide(ev, []*rule.Rule{block})
	if d.Verdict != Deny || d.Reason == "" {
		t.Fatalf("expected Deny with a reason, got %+v", d)
	}
}

func TestRequireSkillUnlocksOnSkillLoad(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	rules := []*rule.Rule{mustRule(t, "---\nname: s\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: ^echo release-\naction:\n  type: require-skill\n  skill: release-checklist\n---\n")}
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "echo release-1.2.0"}

	loaded := ev
	loaded.Transcript = write(t, lnSkillOK, lnAssistant)
	if d := Decide(loaded, rules); d.Verdict != Allow {
		t.Errorf("skill loaded in window should clear the gate, got %+v", d)
	}

	notLoaded := ev
	notLoaded.Transcript = write(t, lnDenyStr, lnAssistant)
	if d := Decide(notLoaded, rules); d.Verdict != DenyRetry {
		t.Errorf("no skill load should deny, got %+v", d)
	}

	// No transcript at all: fail closed rather than opening the gate.
	if d := Decide(ev, rules); d.Verdict != DenyRetry {
		t.Errorf("missing transcript should deny, got %+v", d)
	}
}

func TestRequireSkillReasonTellsAgentToRetry(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "x"}
	rules := []*rule.Rule{mustRule(t, "---\nname: s\nevent: pre-tool\naction:\n  type: require-skill\n  skill: foo\n---\n")}

	if got := Decide(ev, rules).Reason; !strings.Contains(got, "then retry this exact command") {
		t.Errorf("require-skill should tell the agent to retry after invoking the skill: %q", got)
	}
}

// lnAdvised is the attachment Claude Code writes when a hook's additionalContext reaches the agent.
const lnAdvised = `{"type":"attachment","isSidechain":false,"attachment":{"type":"hook_additional_context","content":["paws (use-agent-browser): Prefer the agent-browser skill."],"hookName":"PreToolUse:mcp__playwright__browser_click","hookEvent":"PreToolUse"}}`

func TestAdviseOncePerWindow(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	rules := []*rule.Rule{mustRule(t, "---\nname: use-agent-browser\nevent: pre-tool\nmatcher:\n  tool: mcp__playwright__*\naction:\n  type: advise\n  message: Prefer the agent-browser skill.\n---\n")}
	ev := Event{Kind: rule.KindTool, Tool: "mcp__playwright__browser_click", Target: "{}"}

	first := ev
	first.Transcript = write(t, lnAssistant)
	d := Decide(first, rules)
	if d.Verdict != Allow || d.Context != "paws (use-agent-browser): Prefer the agent-browser skill." {
		t.Fatalf("first advise should allow with tagged context, got %+v", d)
	}

	again := ev
	again.Transcript = write(t, lnAdvised, lnAssistant)
	if d := Decide(again, rules); d.Verdict != Allow || d.Context != "" {
		t.Errorf("advice already in the window should not repeat, got %+v", d)
	}

	compacted := ev
	compacted.Transcript = write(t, lnAdvised, lnCompact, lnAssistant)
	if d := Decide(compacted, rules); d.Context == "" {
		t.Errorf("a compaction should make the advice repeat, got %+v", d)
	}
}

func TestRequireSkillOnSessionStartAddsContext(t *testing.T) {
	DryRun = true
	defer func() { DryRun = false }()

	rules := []*rule.Rule{mustRule(t, "---\nname: s\nevent: session-start\naction:\n  type: require-skill\n  skill: release-checklist\n---\n")}
	ev := Event{Kind: rule.KindSession, Target: "startup"}
	d := Decide(ev, rules)
	if d.Verdict != Allow || !strings.Contains(d.Context, `"release-checklist" skill`) {
		t.Fatalf("session-start require-skill should allow with context naming the skill, got %+v", d)
	}

	loaded := ev
	loaded.Transcript = write(t, lnSkillOK, lnAssistant)
	if d := Decide(loaded, rules); d.Context != "" {
		t.Errorf("a loaded skill needs no reminder, got %+v", d)
	}

	// Tool rules never fire on session events and vice versa.
	if d := Decide(Event{Kind: rule.KindTool, Tool: "Bash", Target: "ls"}, rules); d.Context != "" || d.Verdict != Allow {
		t.Errorf("session rule fired on a tool call: %+v", d)
	}
}
