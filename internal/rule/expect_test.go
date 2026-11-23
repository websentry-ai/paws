package rule

import (
	"slices"
	"testing"
)

// A check of the wrong YAML shape drops itself only: the rule keeps its other checks and still enforces.
func TestABadCheckKeepsTheOthers(t *testing.T) {
	src := `---
name: rm
event: pre-tool
matcher:
  tool: Bash
action:
  type: require-skill
  skill: coreutils
expect:
  - id: good
    when: before
    if: call.tool == "Bash"
  - id: bad
    when: never
    if: [not, a, string]
---
`
	r, err := Parse([]byte(src), "rm.md")
	if err != nil {
		t.Fatalf("the rule didn't load: %v", err)
	}
	if len(r.Expect) != 1 || r.Expect[0].ID != "good" || r.ExpectErr == nil {
		t.Errorf("checks %+v, err %v; want only good, with a warning", r.Expect, r.ExpectErr)
	}
}

// Commands holds what the line runs, not the text a command only reads.
func TestCommands(t *testing.T) {
	cases := map[string][]string{
		`cd web && sudo git push -f`:                              {"cd web", "git push -f"},
		"git commit -F - <<'EOF'\nfix: no git push -f\nEOF":       {"git commit -F -"},
		`git commit -m "docs: never git push --force"`:            {"git commit -m docs: never git push --force"},
		"cat > a.py <<'PY'\nimport os; os.system('rm -rf /')\nPY": {"cat"},
		`bash -c "git push -f"`:                                   {"git push -f", "bash -c git push -f"},
		"bash <<'EOF'\ngit push -f\nEOF":                          {"git push -f", "bash"},
		`echo "unterminated`:                                      {`echo "unterminated`},
		"f() { git push -f; }":                                    nil,
		"f() { git push -f; }; f":                                 {"git push -f", "f"},
		// Cut short inside a heredoc: split roughly, so the commit is still seen as one.
		"cd x && git commit -F - <<'EOF'\nfix: y": {"cd x", "git commit -F - <<'EOF'", "fix: y"},
	}
	for line, want := range cases {
		if got := Commands(line); !slices.Equal(got, want) {
			t.Errorf("Commands(%q) = %q, want %q", line, got, want)
		}
	}
}

// A check has an if or cases, not both, and each case needs a match and an if.
func TestCasesValidate(t *testing.T) {
	bad := []Expect{
		{ID: "a", When: "retry"},
		{ID: "b", When: "retry", If: "true", Cases: []ExpectCase{{Match: "true", If: "true"}}},
		{ID: "c", When: "retry", Cases: []ExpectCase{{Match: "true"}}},
		{ID: "d", When: "retry", Cases: []ExpectCase{{Match: "call.tool ==", If: "true"}}},
	}
	for _, e := range bad {
		if err := validateExpect([]Expect{e}); err == nil {
			t.Errorf("check %s validated", e.ID)
		}
	}
	e := Expect{ID: "ok", When: "retry", Cases: []ExpectCase{
		{Match: `call.tool == "Bash"`, If: `call.text matches "--body"`},
		{Match: `call.tool endsWith "create_pull_request"`, If: `call.input.body != ""`},
	}}
	if err := validateExpect([]Expect{e}); err != nil {
		t.Fatal(err)
	}
	blocked := map[string]any{"tool": "Bash"}
	mcp := map[string]any{"tool": "mcp__github__create_pull_request", "input": map[string]any{"body": "x"}}
	if c, ok := e.For(mcp, blocked); !ok || !c.Eval(mcp, blocked) {
		t.Error("the MCP case didn't apply and pass")
	}
	if _, ok := e.For(map[string]any{"tool": "Write"}, blocked); ok {
		t.Error("a Write matched a case")
	}
}
