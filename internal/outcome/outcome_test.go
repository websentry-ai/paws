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
		// A bad check only measures: the rule still loads and enforces, without it, and says why.
		r, err := rule.Parse([]byte(src), "r.md")
		if err != nil || r.ExpectErr == nil {
			t.Errorf("%s: err %v, ExpectErr %v; want the rule loaded with ExpectErr set", name, err, r.ExpectErr)
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

// A Write retried to the same file with different content is a changed call, not the same one.
func TestGradeComparesTheEdit(t *testing.T) {
	r := parse(t, "---\nname: w\nevent: pre-tool\nmatcher: {tool: Write}\naction: {type: require-skill, skill: s}\n---\n")
	w := func(sec int, content string, denied bool) Call {
		return Call{Tool: "Write", Target: "/x/a.go", Text: content, At: t0.Add(time.Duration(sec) * time.Second), Denied: denied}
	}
	if got := Grade(r, []Call{w(0, "v1", true), w(1, "v2", false)}, 0); got.Outcome != ChangedUnchecked {
		t.Errorf("changed content: %s, want changed_unchecked", got.Outcome)
	}
	if got := Grade(r, []Call{w(0, "v1", true), w(1, "v1", false)}, 0); got.Outcome != SameRetry {
		t.Errorf("same content: %s, want same_retry", got.Outcome)
	}
}

// A malformed expect block, or one on a session-start rule, only measures: the rule loads without it.
func TestBadExpectNeverDropsTheRule(t *testing.T) {
	for name, src := range map[string]string{
		"if of the wrong type": "---\nname: r\nevent: pre-tool\nmatcher: {tool: Bash}\naction: {type: require-skill, skill: s}\nexpect: [{id: x, when: retry, if: [bad]}]\n---\n",
		"on session start":     "---\nname: r\nevent: session-start\naction: {type: require-skill, skill: s}\nexpect: [{id: x, when: retry, if: \"true\"}]\n---\n",
	} {
		r, err := rule.Parse([]byte(src), "r.md")
		if err != nil || r.ExpectErr == nil || len(r.Expect) != 0 || r.Action.Skill != "s" {
			t.Errorf("%s: rule %v, err %v; want it loaded, without checks, ExpectErr set", name, r, err)
		}
	}
}

// Edits compare exactly: a whitespace change is a change.
func TestGradeComparesEditsExactly(t *testing.T) {
	r := parse(t, "---\nname: w\nevent: pre-tool\nmatcher: {tool: Write}\naction: {type: require-skill, skill: s}\n---\n")
	w := func(sec int, content string, denied bool) Call {
		return Call{Tool: "Write", Target: "/x/a.go", Text: content, At: t0.Add(time.Duration(sec) * time.Second), Denied: denied}
	}
	if got := Grade(r, []Call{w(0, "a  b", true), w(1, "a b", false)}, 0); got.Outcome != ChangedUnchecked {
		t.Errorf("whitespace change: %s, want changed_unchecked", got.Outcome)
	}
}

// A never check whose if fails to run on a call can't show the call didn't happen: it fails.
func TestNeverFailsWhenItCantRun(t *testing.T) {
	r := parse(t, `---
name: rm
event: pre-tool
matcher:
  tool: Bash
  command_regex: '\brm\s+-\S*r'
action:
  type: require-skill
  skill: coreutils
expect:
  - id: small
    when: never
    if: call.text > 3
---
`)
	calls := []Call{bash(1, "rm -rf build", true, ""), bash(2, "rm -rf build", false, "")}
	if res := Grade(r, calls, 0); res.Checks["small"] || res.Outcome == Fixed {
		t.Errorf("a never check that errors passed: %+v", res)
	}
}

// Spacing between words doesn't make a different command; spacing inside quotes does.
func TestSameKeepsQuotedSpacing(t *testing.T) {
	cases := []struct {
		a, b string
		same bool
	}{
		{"rm  -rf   build", "rm -rf build", true},
		{"cd x && rm -rf build", "rm -rf build", true},
		{"rm -rf 'a  b'", "rm -rf 'a b'", false},
		{`rm -rf "a  b"`, `rm  -rf "a  b"`, true},
		{`rm -rf "x\" a  b"`, `rm -rf "x\" a b"`, false},       // an escaped quote doesn't end the string
		{`rm -rf a\  b`, `rm -rf a\ b`, false},                 // an escaped space is part of the name
		{"echo a\nrm -rf build", "echo a rm -rf build", false}, // a newline ends a command
		{"echo a\n  rm -rf build", "echo a; rm -rf build", true},
		{"cat <<EOF > f\na  b\nEOF", "cat <<EOF > f\na b\nEOF", false}, // a heredoc body is input
		{"cat  <<EOF > f\na  b\nEOF", "cat <<EOF > f\na  b\nEOF", true},
	}
	for _, c := range cases {
		if got := Same(c.a, c.b); got != c.same {
			t.Errorf("Same(%q, %q) = %v, want %v", c.a, c.b, got, c.same)
		}
	}
}

// Another tool with the same input is a changed call, not the same one.
func TestSameCallNeedsTheSameTool(t *testing.T) {
	a := Call{Tool: "mcp__fs__move_file", Target: `{"path":"x"}`, Text: "x"}
	b := Call{Tool: "mcp__fs__delete_file", Target: `{"path":"x"}`, Text: "x"}
	if sameCall(a, b) {
		t.Error("two MCP tools with one input read as the same call")
	}
}

// A rule over every tool matches the Skill call that answers its deny: that load isn't the retry.
func TestASkillLoadIsNotTheRetry(t *testing.T) {
	r := parse(t, `---
name: any
event: pre-tool
matcher: {}
action:
  type: require-skill
  skill: coreutils
---
`)
	calls := []Call{bash(1, "rm -rf build", true, ""), {Tool: "Skill", Target: "coreutils", Text: "coreutils", At: t0.Add(2 * time.Second)},
		bash(3, "rm -rf build", false, "")}
	res := Grade(r, calls, 0)
	if res.Retry == nil || res.Retry.Tool != "Bash" || res.Outcome != SameRetry {
		t.Errorf("graded %+v; want the Bash retry, same_retry", res)
	}
}

// An Edit retried with the same new text but another old_string changes a different part of the file.
func TestSameCallComparesTheWholeEdit(t *testing.T) {
	edit := func(old string) Call {
		return Call{Tool: "Edit", Target: "/x/a.go", Text: "b", Input: map[string]any{"file_path": "/x/a.go", "old_string": old, "new_string": "b"}}
	}
	if sameCall(edit("a"), edit("c")) {
		t.Error("two edits with different old_string read as the same call")
	}
	if !sameCall(edit("a"), edit("a")) {
		t.Error("the same edit read as a change")
	}
}
