package catalog

import (
	"strings"
	"testing"
)

// CheckTriggerCases is what `paws validate` and `paws index --portal` run on a catalog with trigger cases.
func TestCheckTriggerCases(t *testing.T) {
	root := goodRoot(t)
	c, probs := Load(root)
	if len(probs) > 0 {
		t.Fatal(probs)
	}
	ok := TriggerCases{
		"humanizer-pr":  {Fires: [][2]string{{"Bash", "gh pr create --fill"}}, Ignores: [][2]string{{"Bash", "gh pr view"}}},
		"stop-slop-pr":  {Fires: [][2]string{{"Bash", "gh pr create"}}, Ignores: [][2]string{{"Bash", "gh issue create"}}},
		"aws-s3-create": {Fires: [][2]string{{"Bash", "aws s3 mb s3://b"}}, Ignores: [][2]string{{"Bash", "aws s3 ls"}}},
	}
	if got := CheckTriggerCases(c, ok, nil); len(got) != 0 {
		t.Fatalf("good cases reported %v", got)
	}

	bad := TriggerCases{
		"humanizer-pr": {Fires: [][2]string{{"Bash", "gh pr view"}}, Ignores: [][2]string{{"Bash", "gh pr create"}}},
		"stop-slop-pr": {Fires: [][2]string{{"Bash", "gh pr create"}}},
		"gone":         {Fires: [][2]string{{"Bash", "x"}}, Ignores: [][2]string{{"Bash", "y"}}},
	}
	got := strings.Join(CheckTriggerCases(c, bad, map[string]bool{"old-rule": true}), "\n")
	for _, want := range []string{
		`humanizer-pr: should fire on Bash "gh pr view"`,
		`humanizer-pr: should ignore Bash "gh pr create"`,
		"stop-slop-pr: needs calls it fires on and near misses it ignores",
		"gone: has cases but is not in the catalog",
		"aws-s3-create: add its fire and ignore cases",
		"old-rule: exempt from cases but no longer in the catalog",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if got := CheckTriggerCases(c, ok, map[string]bool{"aws-s3-create": true}); len(got) != 1 ||
		!strings.Contains(got[0], "aws-s3-create: has cases now") {
		t.Errorf("an exempt binding with cases: %v", got)
	}
}
