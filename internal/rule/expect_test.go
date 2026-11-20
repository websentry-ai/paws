package rule

import "testing"

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
