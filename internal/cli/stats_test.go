package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarize(t *testing.T) {
	dir := t.TempDir()
	rotated := filepath.Join(dir, "audit.log.1")
	current := filepath.Join(dir, "audit.log")
	lines := func(ls ...string) string { return strings.Join(ls, "\n") + "\n" }
	os.WriteFile(rotated, []byte(lines(
		`{"event":"pre-commit","rule":"humanizer-commit","verdict":"deny","session":"a"}`,
		`{"event":"skill_invoked","skill":"humanizer","session":"a"}`,
		`{"event":"pre-commit","rule":"humanizer-commit","verdict":"allow","satisfied_by":"skill-load","session":"a"}`,
	)), 0o600)
	os.WriteFile(current, []byte(lines(
		`{"event":"pre-commit","rule":"humanizer-commit","verdict":"allow","satisfied_by":"skill-load","session":"b"}`,
		`{"event":"pre-tool","rule":"s3","verdict":"allow","satisfied_by":"digest-delivered","session":"b"}`,
		`{"event":"turn_end","session":"b"}`,
		`{"event":"tool_result","tool":"Bash","session":"b"}`,
		`not json`,
	)), 0o600)

	st := summarize([]string{rotated, current}, []string{"humanizer-commit", "s3", "never-fired"})
	byName := map[string]RuleStats{}
	for _, r := range st.Rules {
		byName[r.Rule] = r
	}
	h := byName["humanizer-commit"]
	if h.Fires != 3 || h.Denied != 1 || h.LoadedAfterDeny != 1 || h.AlreadyLoaded != 1 {
		t.Errorf("humanizer-commit stats wrong: %+v", h)
	}
	if byName["s3"].Delivered != 1 {
		t.Errorf("digest delivery not counted: %+v", byName["s3"])
	}
	if len(st.Dormant) != 1 || st.Dormant[0] != "never-fired" {
		t.Errorf("dormant = %v", st.Dormant)
	}
	if st.Skills["humanizer"] != 1 || st.Turns != 1 || st.ToolResults != 1 || st.Sessions != 2 {
		t.Errorf("totals wrong: %+v", st)
	}
}
