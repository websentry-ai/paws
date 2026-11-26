package cli

import (
	"encoding/json"
	"fmt"
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
	transcriptEvery(t, path, start, time.Second, calls)
}

// transcriptEvery is transcriptLines with one record every step.
func transcriptEvery(t *testing.T, path string, start time.Time, step time.Duration, calls []struct {
	cmd    string
	result string
	isErr  bool
}) {
	t.Helper()
	ts := func(i int) string { return start.Add(time.Duration(i) * step).UTC().Format(time.RFC3339Nano) }
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

// denyAt moves every deny in the audit log (and its rotated file) to at, as if paws denied the call then: live, a
// deny comes right after its call, and grading only counts what followed the deny.
func denyAt(t *testing.T, home string, at time.Time) {
	t.Helper()
	for _, path := range []string{filepath.Join(home, "audit.log"), filepath.Join(home, "audit.log.1")} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var out []string
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var rec map[string]any
			if json.Unmarshal([]byte(line), &rec) == nil && rec["verdict"] == "deny" {
				rec["ts"], rec["at"] = at.UTC().Format(time.RFC3339), at.UTC().Format(time.RFC3339Nano)
				b, _ := json.Marshal(rec)
				line = string(b)
			}
			out = append(out, line)
		}
		if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
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
	denyAt(t, home, start.Add(1100*time.Millisecond)) // just after the call at start+1s
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
	if o["retry_command"] != "rm -rf build" {
		t.Errorf("retry_command %v; want the command Claude ran after the block", o["retry_command"])
	}

	// Sync uploads it (the Stop hook may already have): the outcome, the checks and the retry command.
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
	for _, r := range sent {
		if r["retry_command"] != "rm -rf build" {
			t.Errorf("uploaded outcome %v; want retry_command %q", r, "rm -rf build")
		}
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

// A portal rule is named <skill>--<action id>: the id is what follows the skill's name, even when either holds `--`.
func TestPortalOutcomesSplitTheRuleNameAtTheSkill(t *testing.T) {
	portalEnv(t, http.NotFoundHandler())
	src := filepath.Join(portalDir(), "any.md")
	for _, c := range []struct{ skill, rule, want string }{
		{"coreutils", "coreutils--rm-1", "rm-1"},
		{"my--skill", "my--skill--rm-1", "rm-1"},
		{"coreutils", "coreutils--a--b", "a--b"},
		{"coreutils@e85a43d", "coreutils--rm-1", "rm-1"},
	} {
		if got := toOutcomeUpload(auditRecord{Event: "outcome", Rule: c.rule, Source: src, Skill: c.skill, Outcome: "fixed"}).Action; got != c.want {
			t.Errorf("skill %q rule %q: action_id %q, want %q", c.skill, c.rule, got, c.want)
		}
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
	start := time.Now().Add(-20 * time.Second)
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"rm -rf build", "", false}})
	denyAt(t, home, start.Add(1100*time.Millisecond)) // the turn's deny comes before its last record
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
	start := time.Now().Add(-20 * time.Second)
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"ls build", "", false}, {"rm -rf build", "", false}})
	denyAt(t, home, start.Add(1100*time.Millisecond))
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

// A deny at the end of the turn before, in the same second as this turn's start, isn't this turn's: it doesn't take
// this turn's blocked call, so this turn's own deny is graded.
func TestADenyFromTheTurnBeforeInTheSameSecond(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-b.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-b", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	log := filepath.Join(home, "audit.log")
	var deny map[string]any
	for _, rec := range auditLines(t, home) {
		if rec["verdict"] == "deny" {
			deny = rec
		}
	}
	if deny == nil || deny["at"] == nil {
		t.Fatalf("no deny with an exact time: %v", deny)
	}
	// The turn before: denied at base+0.1s and graded at its Stop. This turn starts at base+0.2s, Claude makes the
	// call at base+0.3s and paws denies it at base+0.35s: the same second, so only the exact times tell the two
	// denies apart.
	base := time.Now().Add(-30 * time.Second).UTC().Truncate(time.Second)
	at := func(d time.Duration) (string, string) {
		t := base.Add(d)
		return t.Format(time.RFC3339), t.Format(time.RFC3339Nano)
	}
	line := func(rec map[string]any) string { b, _ := json.Marshal(rec); return string(b) }
	before, this := map[string]any{}, map[string]any{}
	for k, v := range deny {
		before[k], this[k] = v, v
	}
	before["ts"], before["at"] = at(100 * time.Millisecond)
	this["ts"], this["at"] = at(350 * time.Millisecond)
	graded := map[string]any{"ts": before["ts"], "event": "outcome", "session": "sess-b", "rule": "rm-gate",
		"source": deny["source"], "outcome": "fixed", "blocked_at": before["ts"], "blocked_exact": before["at"]}
	if err := os.WriteFile(log, []byte(line(before)+"\n"+line(graded)+"\n"+line(this)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcriptEvery(t, transcript, base.Add(200*time.Millisecond), 100*time.Millisecond, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"ls build", "", false}, {"rm -rf build", "", false}})
	runGate(t, map[string]any{"hook_event_name": "Stop", "session_id": "sess-b", "transcript_path": transcript, "cwd": cwd})
	var got []string
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" && rec["blocked_exact"] == this["at"] {
			got = append(got, rec["outcome"].(string))
		}
	}
	if len(got) != 1 || got[0] != "fixed" {
		t.Errorf("this turn's deny graded %v, want [fixed]", got)
	}
}

// A deny after the Stop belongs to the next turn, even when it names a call this turn's log saw refused.
func TestADenyAfterTheStopIsTheNextTurns(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-n.jsonl")
	transcriptLines(t, transcript, time.Now().Add(-20*time.Second), []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: another hook refused it", true}, {"rm -rf build", "", false}})
	stopped := time.Now().Add(-5 * time.Second)
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-n", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}}) // the next turn's deny
	in := hook.Input{HookEventName: "Stop", SessionID: "sess-n", TranscriptPath: transcript, CWD: cwd}
	gradeTurn(&in, hook.ToEvent(&in), stopped)
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			t.Errorf("graded the next turn's deny against this turn: %v", rec)
		}
	}
}

// A Stop that couldn't read its turn leaves its denies eligible. If the user sends the next prompt before the next Stop,
// the turn before is read from its own start, so its deny is still graded.
func TestAnUnreadableTurnIsGradedAfterTheNextPrompt(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-p.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-p", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	start := time.Now().Add(-40 * time.Second)
	denyAt(t, home, start.Add(1100*time.Millisecond))
	stop := map[string]any{"hook_event_name": "Stop", "session_id": "sess-p", "transcript_path": transcript, "cwd": cwd}
	runGate(t, stop) // the log isn't there yet
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"rm -rf build", "", false}})
	f, err := os.OpenFile(transcript, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	next := start.Add(20 * time.Second).UTC().Format(time.RFC3339Nano)
	f.WriteString(`{"type":"user","timestamp":"` + next + `","message":{"role":"user","content":"now list the files"}}` + "\n")
	f.WriteString(`{"type":"assistant","timestamp":"` + start.Add(21*time.Second).UTC().Format(time.RFC3339Nano) +
		`","message":{"content":[{"type":"tool_use","id":"toolu_z","name":"Bash","input":{"command":"ls"}}]}}` + "\n")
	f.Close()
	runGate(t, stop)
	var got []string
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			got = append(got, fmt.Sprint(rec["outcome"], " ", rec["retry_command"]))
		}
	}
	if len(got) != 1 || got[0] != "same_retry rm -rf build" {
		t.Errorf("outcomes %v; want the first turn's deny graded once: same_retry rm -rf build", got)
	}
}

// The Stop hook starts a moment after the turn ended. A call the next turn makes in that gap, and gets denied, is after
// the turn's last record: it is not graded with the turn that stopped.
func TestADenyAfterTheLastRecordIsNotThisTurns(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-g.jsonl")
	start := time.Now().Add(-20 * time.Second)
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: another hook refused it", true}, {"rm -rf build", "", false}})
	// The next turn's call is denied now, five seconds after the log's last record (start+4s is before now-15s).
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-g", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	in := hook.Input{HookEventName: "Stop", SessionID: "sess-g", TranscriptPath: transcript, CWD: cwd}
	gradeTurn(&in, hook.ToEvent(&in), time.Now())
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			t.Errorf("graded a deny made after the turn's last record: %v", rec)
		}
	}
	if at, ok := hook.LastRecordAt(transcript, time.Now()); !ok || at.After(start.Add(5*time.Second)) {
		t.Errorf("LastRecordAt = %v, %v; want the log's last record, %v", at, ok, start.Add(4*time.Second))
	}
}

// liveTurn plays a session through the real gate as Claude Code would: each record logged now, a call logged before its
// PreToolUse hook runs, and the Stop hook run once the turn's last record is in the log.
type liveTurn struct {
	t                  *testing.T
	session, path, cwd string
	n                  int
}

// log appends one record of the given type: body is its fields after type and timestamp.
func (l *liveTurn) log(typ, body string) {
	l.t.Helper()
	time.Sleep(2 * time.Millisecond) // strictly increasing times, as in a real log
	appendLines(l.t, l.path, `{"type":"`+typ+`","timestamp":"`+time.Now().UTC().Format(time.RFC3339Nano)+`",`+body+`}`)
}

func (l *liveTurn) prompt(text string) {
	l.log("user", `"message":{"role":"user","content":`+jsonString(text)+`}`)
}

// bash logs a Bash call; gated runs it through PreToolUse first, and a deny is logged as Claude Code logs a refusal.
func (l *liveTurn) bash(cmd string, gated bool) {
	l.t.Helper()
	l.n++
	id := fmt.Sprintf("toolu_live%02d", l.n)
	l.log("assistant", `"message":{"content":[{"type":"tool_use","id":"`+id+`","name":"Bash","input":{"command":`+jsonString(cmd)+`}}]}`)
	result, isErr := "ok", "false"
	if gated {
		out := runGate(l.t, map[string]any{"hook_event_name": "PreToolUse", "session_id": l.session, "transcript_path": l.path,
			"cwd": l.cwd, "tool_name": "Bash", "tool_input": map[string]any{"command": cmd}})
		if strings.Contains(out, `"deny"`) {
			result, isErr = "PreToolUse:Bash hook error: paws: load coreutils", "true"
		}
	}
	l.log("user", `"message":{"content":[{"type":"tool_result","tool_use_id":"`+id+`","is_error":`+isErr+`,"content":`+jsonString(result)+`}]}`)
}

func (l *liveTurn) stop() {
	l.t.Helper()
	l.log("assistant", `"message":{"content":[{"type":"text","text":"done"}]}`)
	runGate(l.t, map[string]any{"hook_event_name": "Stop", "session_id": l.session, "transcript_path": l.path, "cwd": l.cwd})
}

func jsonString(s string) string { b, _ := json.Marshal(s); return string(b) }

func outcomesIn(t *testing.T, home string) []string {
	t.Helper()
	var got []string
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			got = append(got, fmt.Sprint(rec["outcome"], " ", rec["retry_command"]))
		}
	}
	return got
}

// The next prompt can come the moment a turn ends. The Stop hook reads the log well before the next turn's first call
// (the model answers first), so it grades the turn that stopped; the next turn's block is graded at that turn's own
// Stop, and the first isn't graded again.
func TestTheNextTurnRightAfterAStopIsGradedAtItsOwnStop(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	l := &liveTurn{t: t, session: "sess-next", path: filepath.Join(t.TempDir(), "sess-next.jsonl"), cwd: cwd}
	l.prompt("clean up the build folder")
	l.bash("rm -rf build", true)
	l.bash("ls build", false)
	l.bash("rm -rf build", false)
	l.stop()
	if got := outcomesIn(t, home); len(got) != 1 || got[0] != "fixed rm -rf build" {
		t.Fatalf("after the first Stop: outcomes %v; want [fixed rm -rf build]", got)
	}
	l.prompt("now the dist folder") // milliseconds after the Stop returned
	l.bash("rm -rf dist", true)
	l.bash("rm -rf dist", false)
	l.stop()
	if got := outcomesIn(t, home); len(got) != 2 || got[1] != "same_retry rm -rf dist" {
		t.Errorf("after the second Stop: outcomes %v; want [fixed rm -rf build, same_retry rm -rf dist]", got)
	}
}

// An interrupted turn fires no Stop. The short turn after it, a few seconds old at its Stop, is the turn that stopped and
// is graded in full; the interrupted turn's block, which Claude Code ends with its interrupt marker, stays ungraded.
func TestAShortTurnAfterAnInterruptIsGraded(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	l := &liveTurn{t: t, session: "sess-int", path: filepath.Join(t.TempDir(), "sess-int.jsonl"), cwd: cwd}
	l.prompt("clean up the build folder")
	l.bash("rm -rf build", true)
	l.log("user", `"message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user]"}]}`)
	l.prompt("just the dist folder")
	l.bash("rm -rf dist", true)
	l.bash("rm -rf dist", false)
	l.stop()
	l.stop() // a second Stop for the same turn grades nothing new
	if got := outcomesIn(t, home); len(got) != 1 || got[0] != "same_retry rm -rf dist" {
		t.Errorf("outcomes %v; want only the short turn's block: [same_retry rm -rf dist]", got)
	}
}

// Another hook refused the same command earlier in the turn: paws's block is its own later call, so the listing
// between the two refusals isn't "before the retry" for paws's block.
func TestAnEarlierRefusalByAnotherHookIsNotTheBlock(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	// A never check: a call between the two refusals breaks it only if the earlier one were taken as the block.
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule+`  - id: no-wipe
    label: Never wiped
    when: never
    if: call.text matches "^echo wiped"
`)
	transcript := filepath.Join(t.TempDir(), "sess-h.jsonl")
	// Calls at -19s, -17s, -15s, -13s: refused by another hook, the wipe, refused by paws, the retry.
	transcriptLines(t, transcript, time.Now().Add(-20*time.Second), []struct {
		cmd    string
		result string
		isErr  bool
	}{
		{"rm -rf build", "PreToolUse:Bash hook error: another hook refused it", true},
		{"echo wiped", "wiped", false},
		{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true},
		{"rm -rf build", "", false},
	})
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-h", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	// paws's deny happened just after its call (-15s), not at the end.
	log := filepath.Join(home, "audit.log")
	var lines []string
	for _, rec := range auditLines(t, home) {
		if rec["verdict"] == "deny" {
			at := time.Now().Add(-15*time.Second + 50*time.Millisecond).UTC()
			rec["ts"], rec["at"] = at.Format(time.RFC3339), at.Format(time.RFC3339Nano)
		}
		b, _ := json.Marshal(rec)
		lines = append(lines, string(b))
	}
	if err := os.WriteFile(log, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGate(t, map[string]any{"hook_event_name": "Stop", "session_id": "sess-h", "transcript_path": transcript, "cwd": cwd})
	var got []any
	var checks any
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			got = append(got, rec["outcome"])
			checks = rec["checks"]
		}
	}
	if c, _ := checks.(map[string]any); len(got) != 1 || c["no-wipe"] != true {
		t.Errorf("outcomes %v, checks %v; want no-wipe passed: the wipe came before paws's block", got, checks)
	}
}

// A rule that a sync bound to another skill since the deny doesn't grade it: the outcome would name the wrong skill.
func TestARuleRebindingSinceTheDenyLeavesItUngraded(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md")
	writeRule(t, path, rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-k.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-k", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	writeRule(t, path, strings.Replace(rmRule, "skill: coreutils", "skill: other", 1))
	transcriptLines(t, transcript, time.Now().Add(-20*time.Second), []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"rm -rf build", "", false}})
	runGate(t, map[string]any{"hook_event_name": "Stop", "session_id": "sess-k", "transcript_path": transcript, "cwd": cwd})
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			t.Errorf("graded with a rule now bound to another skill: %v", rec)
		}
	}
}

// A rule with a when condition: a later call it matches but didn't apply to (a Write to a file that existed, under
// new-file) isn't the retry. The retry is the call the gate logged it allowing.
func TestARuleWithWhenRetriesOnlyWhereItApplied(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "new-go.md"),
		"name: new-go\nevent: pre-tool\nmatcher:\n  tool: Write\nwhen: new-file\naction:\n  type: require-skill\n  skill: go-style\n")
	dir := t.TempDir()
	fresh, existing := filepath.Join(dir, "new.go"), filepath.Join(dir, "old.go")
	os.WriteFile(existing, []byte("package x\n"), 0o600)
	transcript := filepath.Join(t.TempDir(), "sess-w.jsonl")
	write := func(path string) map[string]any {
		return map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-w", "transcript_path": transcript, "cwd": cwd,
			"tool_name": "Write", "tool_input": map[string]any{"file_path": path, "content": "package x\n"}}
	}
	start := time.Now().Add(-20 * time.Second)
	at := func(s int) string { return start.Add(time.Duration(s) * time.Second).UTC().Format(time.RFC3339Nano) }
	call := func(s int, id, path string, refused bool) string {
		in, _ := json.Marshal(map[string]any{"file_path": path, "content": "package x\n"})
		res := map[bool]string{true: `"PreToolUse:Write hook error: paws: load go-style"`, false: `"ok"`}[refused]
		return `{"type":"assistant","timestamp":"` + at(s) + `","message":{"content":[{"type":"tool_use","id":"` + id + `","name":"Write","input":` + string(in) + `}]}}` + "\n" +
			`{"type":"user","timestamp":"` + at(s+1) + `","message":{"content":[{"type":"tool_result","tool_use_id":"` + id + `","is_error":` + map[bool]string{true: "true", false: "false"}[refused] + `,"content":` + res + `}]}}`
	}
	os.WriteFile(transcript, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"` + at(0) + `","message":{"role":"user","content":"add new.go"}}`,
		call(1, "w1", fresh, true), call(3, "w2", existing, false), call(5, "w3", fresh, false)}, "\n")+"\n"), 0o600)
	runGate(t, write(fresh)) // the deny of w1
	// The gate's records as they happened: the deny just after w1, and an allow just after w3 (the skill loaded).
	var lines []string
	for _, rec := range auditLines(t, home) {
		if rec["verdict"] == "deny" {
			deny := rec
			deny["ts"], deny["at"] = at(1)[:19]+"Z", start.Add(1100*time.Millisecond).UTC().Format(time.RFC3339Nano)
			allow := map[string]any{}
			for k, v := range rec {
				allow[k] = v
			}
			allow["verdict"], allow["satisfied_by"] = "allow", "skill-load"
			allow["ts"], allow["at"] = at(5)[:19]+"Z", start.Add(5100*time.Millisecond).UTC().Format(time.RFC3339Nano)
			for _, r := range []map[string]any{deny, allow} {
				b, _ := json.Marshal(r)
				lines = append(lines, string(b))
			}
		}
	}
	os.WriteFile(filepath.Join(home, "audit.log"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
	runGate(t, map[string]any{"hook_event_name": "Stop", "session_id": "sess-w", "transcript_path": transcript, "cwd": cwd})
	var got []any
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			got = append(got, rec["outcome"])
		}
	}
	if len(got) != 1 || got[0] != "same_retry" {
		t.Errorf("outcomes %v; want [same_retry]: the write to old.go isn't one new-file applied to", got)
	}
}

// A transcript that can't be read leaves the turn's denies eligible: the next Stop that can read it grades them.
func TestAnUnreadableTranscriptLeavesDeniesEligible(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	cwd, _ := os.Getwd()
	writeRule(t, filepath.Join(cwd, ".paws", "pre-tool", "rm-gate.md"), rmRule)
	transcript := filepath.Join(t.TempDir(), "sess-u.jsonl")
	runGate(t, map[string]any{"hook_event_name": "PreToolUse", "session_id": "sess-u", "transcript_path": transcript, "cwd": cwd,
		"tool_name": "Bash", "tool_input": map[string]any{"command": "rm -rf build"}})
	start := time.Now().Add(-20 * time.Second)
	denyAt(t, home, start.Add(1100*time.Millisecond))
	stop := map[string]any{"hook_event_name": "Stop", "session_id": "sess-u", "transcript_path": transcript, "cwd": cwd}
	runGate(t, stop) // no transcript yet
	transcriptLines(t, transcript, start, []struct {
		cmd    string
		result string
		isErr  bool
	}{{"rm -rf build", "PreToolUse:Bash hook error: paws: load coreutils", true}, {"rm -rf build", "", false}})
	runGate(t, stop)
	n := 0
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "outcome" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d outcomes; want the deny graded once the transcript could be read", n)
	}
}
