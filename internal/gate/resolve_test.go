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
	approval := mustRule(t, "---\nname: a\nevent: pre-tool\naction:\n  type: require-approval\n---\n")
	warn := mustRule(t, "---\nname: w\nevent: pre-tool\naction:\n  type: warn\n  message: careful\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "x"}

	d := resolve(ev, []*rule.Rule{warn, approval, skill, block})
	if d.Verdict != Deny {
		t.Fatalf("expected Deny, got %+v", d)
	}
	if !strings.Contains(d.Reason, "Blocked by paws") {
		t.Errorf("block message should win, got %q", d.Reason)
	}

	d = resolve(ev, []*rule.Rule{warn, approval})
	if d.Verdict != Ask {
		t.Fatalf("expected Ask, got %+v", d)
	}
	if !strings.Contains(d.Context, "careful") {
		t.Errorf("warn context should ride with ask, got %q", d.Context)
	}

	d = resolve(ev, []*rule.Rule{warn})
	if d.Verdict != Allow {
		t.Fatalf("warn must allow + context, got %+v", d)
	}
	if !strings.Contains(d.Context, "careful") {
		t.Errorf("warn context missing: %q", d.Context)
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
