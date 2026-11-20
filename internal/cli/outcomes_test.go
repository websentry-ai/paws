package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/hook"
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
	// A second rule on the same call: one deny names both skills, and each rule's block is graded.
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-audit.md"), "name: rm-audit\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: '\\brm\\s'\naction:\n  type: require-skill\n  skill: audit-trail\n")
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
	if len(outcomes) != 2 {
		t.Fatalf("%d outcome records, want one per rule: %v", len(outcomes), outcomes)
	}
	byRule := map[string]map[string]any{}
	for _, o := range outcomes {
		byRule[o["rule"].(string)] = o
	}
	if byRule["rm-audit"]["outcome"] != "same_retry" {
		t.Errorf("the second rule's block: %v; want same_retry (no checks, same call retried)", byRule["rm-audit"])
	}
	o := byRule["rm-gate"]
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
	if len(sent) != 2 {
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
	a.Expect = append(a.Expect, rule.Expect{ID: "good", When: "retry", If: "true"})
	data, err = managedRuleFile("coreutils", "coreutils--rm", a, "", t.TempDir())
	if err != nil || !strings.Contains(string(data), "id: good") || strings.Contains(string(data), "id: x") {
		t.Fatalf("rule file %q, %v; want the good check kept and the bad one dropped", data, err)
	}
	a.Expect = []rule.Expect{{ID: "listed", When: "before", If: `call.text matches "^ls"`}}
	data, err = managedRuleFile("coreutils", "coreutils--rm", a, "", t.TempDir())
	if err != nil || !strings.Contains(string(data), "listed") {
		t.Fatalf("rule file %q, %v; want the good check kept", data, err)
	}
}

// Outcome records are graded blocks, not rule firings: stats and pack fire counts leave them out.
func TestOutcomeRecordsAreNotFires(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	appendLines(t, filepath.Join(home, "audit.log"),
		`{"v":2,"ts":"2026-11-20T10:00:00Z","event":"pre-tool","rule":"rm-gate","verdict":"deny","session":"s","skill":"coreutils"}`,
		`{"v":2,"ts":"2026-11-20T10:00:09Z","event":"outcome","rule":"rm-gate","outcome":"fixed","session":"s","skill":"coreutils"}`)
	st := summarize([]string{filepath.Join(home, "audit.log")}, nil)
	for _, r := range st.Rules {
		if r.Rule == "rm-gate" && r.Fires != 1 {
			t.Errorf("rm-gate fires %d, want 1 (the outcome isn't a fire)", r.Fires)
		}
	}
	if n := fireCounts(filepath.Join(home, "audit.log"))["rm-gate"]; n != 1 {
		t.Errorf("fireCounts %d, want 1", n)
	}
}

// A catalog binding's outcome goes up with its action id: the binding's name.
func TestCatalogOutcomesCarryTheBindingID(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	src := filepath.Join(home, "packs", "writing", "writing", "humanizer-pr.md")
	up := toOutcomeUpload(auditRecord{Event: "outcome", Rule: "humanizer-pr", Source: src, Skill: "humanizer", Outcome: "fixed"})
	if up.Action != "humanizer-pr" {
		t.Errorf("action_id %q, want humanizer-pr", up.Action)
	}
}

// Two graders for one turn (overlapping async Stop hooks) write each outcome once.
func TestConcurrentGradersGradeOnce(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-c.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-c", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	transcriptLines(t, transcript, time.Now().Add(-20*time.Second), []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"rm -rf build", "", false}})
	in := hook.Input{HookEventName: "Stop", SessionID: "sess-c", TranscriptPath: transcript, CWD: cwd}
	ev := hook.ToEvent(&in)
	done := make(chan bool)
	for i := 0; i < 4; i++ {
		go func() { gradeTurn(&in, ev, time.Now()); done <- true }()
	}
	for i := 0; i < 4; i++ {
		<-done
	}
	n := 0
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d outcome records from 4 overlapping graders, want 1", n)
	}
}

// A deny that rotated into audit.log.1 before the turn ended is still graded.
func TestStopGradesADenyFromTheRotatedLog(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-r.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-r", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	log := filepath.Join(home, "audit.log")
	if err := os.Rename(log, log+".1"); err != nil { // the log rotated after the deny
		t.Fatal(err)
	}
	transcriptLines(t, transcript, time.Now().Add(-20*time.Second), []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"ls build", "", false}, {"rm -rf build", "", false}})
	runGate(t, map[string]any{"hook_event_name": "Stop", "session_id": "sess-r", "transcript_path": transcript, "cwd": cwd})
	n := 0
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" && rec["outcome"] == "fixed" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d outcomes for a deny in audit.log.1, want 1", n)
	}
}
