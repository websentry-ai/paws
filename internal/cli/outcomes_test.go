package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

const rmRule = `name: rm-gate
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
    if: call.tool == "Bash" && call.text matches "^(ls|du|stat)\\b"
`

// transcriptLines writes a Claude Code log: a prompt, then each call with its result.
func transcriptLines(t *testing.T, path string, start time.Time, calls []struct {
	cmd    string
	result string
	isErr  bool
}) {
	t.Helper()
	ts := func(i int) string { return start.Add(time.Duration(i) * time.Second).UTC().Format(time.RFC3339Nano) }
	lines := []string{`{"type":"user","timestamp":"` + ts(0) + `","message":{"role":"user","content":"clean up the build folder"}}`}
	for i, c := range calls {
		id := "toolu_" + string(rune('a'+i))
		in, _ := json.Marshal(map[string]any{"command": c.cmd})
		res, _ := json.Marshal(c.result)
		lines = append(lines,
			`{"type":"assistant","timestamp":"`+ts(2*i+1)+`","message":{"content":[{"type":"tool_use","id":"`+id+`","name":"Bash","input":`+string(in)+`}]}}`,
			`{"type":"user","timestamp":"`+ts(2*i+2)+`","message":{"content":[{"type":"tool_result","tool_use_id":"`+id+`","is_error":`+map[bool]string{true: "true", false: "false"}[c.isErr]+`,"content":`+string(res)+`}]}}`)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStopGradesWhatFollowedTheBlock(t *testing.T) {
	home, srv := syncEnv(t)
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-o.jsonl")
	event := func(name string, input map[string]any) map[string]any {
		e := map[string]any{"hook_event_name": name, "session_id": "sess-o", "transcript_path": transcript, "cwd": cwd}
		if input != nil {
			e["tool_name"], e["tool_input"] = "Bash", input
		}
		return e
	}
	// paws denies the delete (the skill isn't loaded), as it would live.
	out := runGate(t, event("PreToolUse", map[string]any{"command": "rm -rf build"}))
	if !strings.Contains(out, `"deny"`) {
		t.Fatalf("the rule didn't deny: %s", out)
	}
	// What Claude did in the rest of the turn, as Claude Code logs it: the denied call, a listing, the retry.
	start := time.Now().Add(-20 * time.Second)
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{
		{"rm -rf build", "PreToolUse:Bash hook error: paws (rm-gate): load coreutils first", true},
		{"ls build", "a.o b.o", false},
		{"rm -rf build", "", false},
	})
	runGate(t, event("Stop", nil))
	runGate(t, event("Stop", nil)) // a second Stop for the same turn grades nothing new
	var outcomes []map[string]any
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			outcomes = append(outcomes, rec)
		}
	}
	if len(outcomes) != 1 {
		t.Fatalf("%d outcome records, want 1: %v", len(outcomes), outcomes)
	}
	o := outcomes[0]
	checks, _ := o["checks"].(map[string]any)
	if o["outcome"] != "fixed" || o["skill"] != "coreutils" || checks["listed-first"] != true {
		t.Errorf("outcome record %v; want fixed with listed-first passing", o)
	}
	if b, _ := json.Marshal(o); strings.Contains(string(b), "build") {
		t.Errorf("the outcome record carries call text: %s", b)
	}

	// Sync uploads it (the Stop hook may already have): the outcome and the checks, no text.
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("sync exit %d: %s", code, out)
	}
	var sent []map[string]any
	for _, body := range srv.bodies {
		var b struct {
			Records []map[string]any `json:"records"`
		}
		json.Unmarshal([]byte(body), &b)
		for _, r := range b.Records {
			if r["kind"] == "outcome" {
				sent = append(sent, r)
			}
		}
	}
	if len(sent) != 1 || sent[0]["outcome"] != "fixed" || sent[0]["skill"] != "coreutils" {
		t.Fatalf("uploaded outcomes %v", sent)
	}
}

// A catalog check this paws can't compile only drops the check: the rule still enforces.
func TestManagedRuleKeepsEnforcingWithoutABadCheck(t *testing.T) {
	a := portalAction{Label: "rm", Event: "pre-tool", Tool: "Bash", CommandRegex: `\brm\b`,
		Expect: []rule.Expect{{ID: "x", When: "retry", If: "call.text matches"}}}
	data, err := managedRuleFile("coreutils", "coreutils--rm", a, "", t.TempDir())
	if err != nil || strings.Contains(string(data), "expect") || !strings.Contains(string(data), "command_regex") {
		t.Fatalf("rule file %q, %v; want the rule without the bad check", data, err)
	}
	a.Expect = []rule.Expect{{ID: "listed", When: "before", If: `call.text matches "^ls"`}}
	data, err = managedRuleFile("coreutils", "coreutils--rm", a, "", t.TempDir())
	if err != nil || !strings.Contains(string(data), "listed") {
		t.Fatalf("rule file %q, %v; want the good check kept", data, err)
	}
}
