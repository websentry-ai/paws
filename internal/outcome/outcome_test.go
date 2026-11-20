package outcome

import (
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

const ruleWithChecks = `---
name: rm
event: pre-tool
matcher:
  tool: Bash
  command_regex: '\brm\s+-\S*r'
action:
  type: require-skill
  skill: coreutils
expect:
  - id: listed-first
    label: Listed what it would delete first
    when: before
    if: call.tool == "Bash" && call.text matches "(^|&&\\s*)(ls|du|stat)\\b"
  - id: not-home
    label: Never deleted the home directory
    when: never
    if: call.text matches "rm\\s+-\\S*r\\S*\\s+(~|\\$HOME)/?(\\s|$)"
---
`

func parse(t *testing.T, src string) *rule.Rule {
	t.Helper()
	r, err := rule.Parse([]byte(src), "rm.md")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var t0 = time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

func bash(sec int, cmd string, denied bool, agent string) Call {
	return Call{Tool: "Bash", Target: cmd, Text: cmd, At: t0.Add(time.Duration(sec) * time.Second), Denied: denied, Agent: agent}
}

func TestGradeOutcomes(t *testing.T) {
	withChecks := parse(t, ruleWithChecks)
	noChecks := parse(t, `---
name: rm
event: pre-tool
matcher: {tool: Bash, command_regex: '\brm\s+-\S*r'}
action: {type: require-skill, skill: coreutils}
---
`)
	blocked := bash(0, "rm -rf build", true, "")
	cases := []struct {
		name   string
		r      *rule.Rule
		calls  []Call
		want   string
		checks map[string]bool
	}{
		{"listed, then deleted", withChecks, []Call{blocked, bash(2, "ls build", false, ""), bash(3, "rm -rf build", false, "")},
			Fixed, map[string]bool{"listed-first": true, "not-home": true}},
		{"a listing chained into the retry counts", withChecks, []Call{blocked, bash(3, "ls build && rm -rf build", false, "")},
			Fixed, map[string]bool{"listed-first": true, "not-home": true}},
		{"same retry, nothing listed", withChecks, []Call{blocked, bash(3, "cd /x && rm -rf build", false, "")},
			SameRetry, map[string]bool{"listed-first": false, "not-home": true}},
		{"changed, but home deleted later", withChecks, []Call{blocked, bash(2, "ls", false, ""), bash(3, "rm -rf dist", false, ""), bash(4, "rm -rf ~", false, "")},
			ChangedFailed, map[string]bool{"listed-first": true, "not-home": false}},
		{"changed, no checks", noChecks, []Call{blocked, bash(3, "rm -rf dist", false, "")}, ChangedUnchecked, map[string]bool{}},
		{"another route", withChecks, []Call{blocked, bash(3, "trash build", false, "")},
			OtherRoute, map[string]bool{"listed-first": false, "not-home": true}},
		{"stopped", withChecks, []Call{blocked}, Stopped, map[string]bool{"listed-first": false, "not-home": true}},
		{"denied again, then retried", withChecks, []Call{blocked, bash(1, "rm -rf build", true, ""), bash(2, "du -sh build", false, ""), bash(3, "rm -rf build", false, "")},
			Fixed, map[string]bool{"listed-first": true, "not-home": true}},
	}
	for _, c := range cases {
		got := Grade(c.r, c.calls, 0)
		if got.Outcome != c.want {
			t.Errorf("%s: outcome %s, want %s", c.name, got.Outcome, c.want)
		}
		for id, want := range c.checks {
			if got.Checks[id] != want {
				t.Errorf("%s: check %s = %v, want %v", c.name, id, got.Checks[id], want)
			}
		}
	}
}

// A subagent already running is another task: its calls after the block don't count. One started after the block
// does.
func TestGradeScopesSubagents(t *testing.T) {
	r := parse(t, ruleWithChecks)
	calls := []Call{
		bash(-5, "go test ./...", false, "early"), // a subagent running before the block
		bash(0, "rm -rf build", true, ""),
		bash(1, "rm -rf build", false, "early"), // its later call: not Claude's answer to the block
	}
	if got := Grade(r, calls, 1); got.Outcome != Stopped {
		t.Errorf("a running subagent's call counted: %s", got.Outcome)
	}
	calls = append(calls, bash(2, "ls build", false, "late"), bash(3, "rm -rf build", false, "late"))
	if got := Grade(r, calls, 1); got.Outcome != Fixed {
		t.Errorf("a subagent started after the block: %s, want fixed", got.Outcome)
	}
	// Launched before the block (its log says so), but its first tool call came after: still a separate task.
	quiet := []Call{bash(0, "rm -rf build", true, ""), bash(2, "ls build", false, "quiet"), bash(3, "rm -rf build", false, "quiet")}
	quiet[1].AgentStart, quiet[2].AgentStart = t0.Add(-time.Minute), t0.Add(-time.Minute)
	if got := Grade(r, quiet, 0); got.Outcome != Stopped {
		t.Errorf("a subagent launched before the block counted: %s", got.Outcome)
	}
}

func TestExpectChecksAreValidated(t *testing.T) {
	bad := map[string]string{
		"unknown when":   `expect: [{id: x, when: later, if: "true"}]`,
		"no if":          `expect: [{id: x, when: retry}]`,
		"bad expression": `expect: [{id: x, when: retry, if: "call.text matches"}]`,
		"bad id":         `expect: [{id: "Bad Id", when: retry, if: "true"}]`,
		"repeated id":    `expect: [{id: x, when: retry, if: "true"}, {id: x, when: after, if: "true"}]`,
	}
	for name, e := range bad {
		src := "---\nname: r\nevent: pre-tool\nmatcher: {tool: Bash}\naction: {type: require-skill, skill: s}\n" + e + "\n---\n"
		if _, err := rule.Parse([]byte(src), "r.md"); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
	// Any input field is reachable, and a missing one is a plain fail, not an error.
	r := parse(t, "---\nname: r\nevent: pre-tool\nmatcher: {tool: mcp__notion__create_pages}\naction: {type: require-skill, skill: notion}\n"+
		"expect: [{id: parent, when: retry, if: \"call.input.parent != nil\"}]\n---\n")
	with := Call{Tool: "mcp__notion__create_pages", Input: map[string]any{"parent": map[string]any{"page_id": "p"}}}
	without := Call{Tool: "mcp__notion__create_pages", Input: map[string]any{"title": "x"}}
	if !r.Expect[0].Eval(with.Env(), without.Env()) || r.Expect[0].Eval(without.Env(), with.Env()) {
		t.Error("input.parent check")
	}
}
