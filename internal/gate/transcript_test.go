package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Line shapes below are copied from real Claude Code transcripts, including the
// polymorphic toolUseResult (object for tool results, bare string for hook denials).
const (
	lnAssistant = `{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"echo release-1.2.0"}}]}}`
	lnSkillOK   = `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_0","content":"Launching skill: release-checklist"}]},"toolUseResult":{"success":true,"commandName":"release-checklist"}}`
	lnSkillFail = `{"type":"user","isSidechain":false,"toolUseResult":{"success":false,"commandName":"release-checklist"}}`
	lnSkillSide = `{"type":"user","isSidechain":true,"toolUseResult":{"success":true,"commandName":"release-checklist"}}`
	lnDenyStr   = `{"type":"user","isSidechain":false,"toolUseResult":"Error: Announcing a release requires the release-checklist skill first."}`
	lnCompact   = `{"type":"system","subtype":"compact_boundary","isSidechain":false,"compactMetadata":{"trigger":"auto","preTokens":129149}}`
	lnAttach    = `{"type":"attachment","isSidechain":false,"attachment":{"type":"hook_success","hookName":"PreToolUse:Bash","stdout":"{}"}}`
	lnGarbage   = `not json at all`

	// A message the user typed. An interactive session tags it origin human; a headless (-p) run leaves origin out.
	lnPrompt         = `{"type":"user","isSidechain":false,"promptId":"p1","origin":{"kind":"human"},"message":{"role":"user","content":"commit and push it"}}`
	lnPromptHeadless = `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"commit and push it"}]}}`
	// User records the user did not type: a loaded skill's text, a background task finishing, a compaction summary.
	lnSkillBody      = `{"type":"user","isSidechain":false,"isMeta":true,"message":{"role":"user","content":[{"type":"text","text":"Base directory for this skill: /x"}]}}`
	lnTaskNote       = `{"type":"user","isSidechain":false,"origin":{"kind":"task-notification"},"message":{"role":"user","content":"<task-notification>done</task-notification>"}}`
	lnCompactSummary = `{"type":"user","isSidechain":false,"isCompactSummary":true,"message":{"role":"user","content":"This session is being continued from a previous conversation."}}`
)

// responses is n agent responses, each a tool call and its result: what one user message can set off.
func responses(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out,
			fmt.Sprintf(`{"type":"assistant","isSidechain":false,"message":{"id":"msg_%d","content":[{"type":"tool_use","id":"toolu_%d","name":"Bash","input":{"command":"ls"}}]}}`, i, i),
			fmt.Sprintf(`{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_%d","content":"ok"}]},"toolUseResult":{"stdout":"ok"}}`, i))
	}
	return out
}

// turns is n user messages after the load, each answered with a few tool calls.
func turns(n int, prompt string) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, prompt)
		out = append(out, responses(3)...)
	}
	return out
}

// A turn is one user message and everything the agent does to answer it, however many tool calls that takes.
func TestWindowCountsUserMessagesNotToolCalls(t *testing.T) {
	loaded := func(lines ...[]string) bool {
		var all []string
		for _, l := range lines {
			all = append(all, l...)
		}
		return SkillLoaded(write(t, all...), "release-checklist", DefaultSkillWindow)
	}
	load := []string{lnPrompt, lnSkillOK, lnSkillBody}
	if !loaded(load, responses(200)) {
		t.Error("200 tool calls answering the message that loaded the skill are still that one turn")
	}
	if !loaded(load, turns(19, lnPrompt)) {
		t.Error("19 user messages after the one that loaded the skill: the load is in the last 20 turns")
	}
	if loaded(load, turns(20, lnPrompt)) {
		t.Error("20 user messages after the one that loaded the skill: the load is 21 turns back")
	}
	if loaded(load, turns(20, lnPromptHeadless)) {
		t.Error("a headless run's messages carry no origin and still count as turns")
	}
	harness := map[string]string{"skill text": lnSkillBody, "task notification": lnTaskNote, "compaction summary": lnCompactSummary,
		// Older and headless logs carry no origin; the content marks what Claude Code wrote itself.
		"slash-command output":             `{"type":"user","isSidechain":false,"message":{"role":"user","content":"<local-command-stdout>ok</local-command-stdout>"}}`,
		"task notification without origin": `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"<task-notification>done</task-notification>"}]}}`,
		"shell escape output":              `{"type":"user","isSidechain":false,"message":{"role":"user","content":"<bash-stdout>ok</bash-stdout>"}}`}
	for name, ln := range harness {
		if !loaded(load, turns(30, ln)) {
			t.Errorf("a %s is not a user message and must not use up the window", name)
		}
	}
}

func write(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := ""
	for _, l := range lines {
		body += l + "\n"
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSkillLoaded(t *testing.T) {
	cases := []struct {
		name   string
		lines  []string
		window int
		want   bool
	}{
		{"load then retry", []string{lnDenyStr, lnSkillOK, lnAttach, lnAssistant}, 20, true},
		{"no load at all", []string{lnDenyStr, lnAssistant, lnAssistant}, 20, false},
		{"failed load does not count", []string{lnSkillFail, lnAssistant}, 20, false},
		{"sidechain load does not count", []string{lnSkillSide, lnAssistant}, 20, false},
		{"string toolUseResult is skipped, not fatal", []string{lnSkillOK, lnDenyStr, lnAssistant}, 20, true},
		{"garbage lines are skipped", []string{lnSkillOK, lnGarbage, lnAssistant}, 20, true},
		{"attachments do not consume the window", []string{lnSkillOK, lnAttach, lnAttach, lnAttach, lnAssistant}, 1, true},
		{"compaction ends the search", []string{lnSkillOK, lnCompact, lnAssistant}, 20, false},
		{"window of 1 covers only the current turn", []string{lnSkillOK, lnPrompt, lnAssistant}, 1, false},
		{"window of 2 reaches one turn back", []string{lnSkillOK, lnPrompt, lnAssistant}, 2, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SkillLoaded(write(t, c.lines...), "release-checklist", c.window); got != c.want {
				t.Fatalf("SkillLoaded = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSkillLoadedFailsClosed(t *testing.T) {
	ok := write(t, lnSkillOK, lnAssistant)
	cases := []struct {
		name        string
		path, skill string
		window      int
	}{
		{"no transcript path", "", "release-checklist", 20},
		{"missing file", filepath.Join(t.TempDir(), "gone.jsonl"), "release-checklist", 20},
		{"empty file", write(t), "release-checklist", 20},
		{"no skill named", ok, "", 20},
		{"zero window", ok, "release-checklist", 0},
		{"different skill", ok, "other-skill", 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if SkillLoaded(c.path, c.skill, c.window) {
				t.Fatal("SkillLoaded = true, want false")
			}
		})
	}
}

// One turn's tool output can run to megabytes (20 user messages span 4 to 16 MB in real sessions), so the
// scan reads back as far as the window goes, not a fixed tail: a load 5 MB back in the same turn still counts.
func TestSkillLoadedReadsBackAsFarAsTheWindow(t *testing.T) {
	lines := []string{lnPrompt, lnSkillOK}
	pad := `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t","content":"x"}]},"toolUseResult":{"stdout":"` + strings.Repeat("p", 4096) + `"}}`
	for i := 0; i < 1280; i++ {
		lines = append(lines, pad)
	}
	lines = append(lines, lnAssistant)
	p := write(t, lines...)
	if fi, err := os.Stat(p); err != nil || fi.Size() < 5<<20 {
		t.Fatalf("fixture must be over 5 MB: %v %v", fi, err)
	}
	if !SkillLoaded(p, "release-checklist", DefaultSkillWindow) {
		t.Fatal("a load 5 MB back in the current turn was not found")
	}
}

func BenchmarkSkillLoaded(b *testing.B) {
	lines := make([]string, 0, 2000)
	for i := 0; i < 1000; i++ {
		lines = append(lines, lnAttach, fmt.Sprintf(`{"type":"user","isSidechain":false,"toolUseResult":{"stdout":"%s"}}`, strings.Repeat("o", 512)))
	}
	lines = append(lines, lnSkillOK, lnAssistant)
	p := filepath.Join(b.TempDir(), "t.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SkillLoaded(p, "release-checklist", 20)
	}
}

// A load further back than maxWindowBytes counts as not loaded: the scan stops there.
func TestSkillLoadedStopsAtTheReadLimit(t *testing.T) {
	orig := maxWindowBytes
	maxWindowBytes = 64 << 10
	t.Cleanup(func() { maxWindowBytes = orig })
	lines := []string{lnPrompt, lnSkillOK}
	pad := `{"type":"assistant","isSidechain":false,"message":{"id":"m","content":[{"type":"text","text":"` + strings.Repeat("p", 4096) + `"}]}}`
	for i := 0; i < 20; i++ {
		lines = append(lines, pad)
	}
	if SkillLoaded(write(t, lines...), "release-checklist", DefaultSkillWindow) {
		t.Error("a load past the read limit was found")
	}
}

// Request blobs, shapes copied from a Claude Code 2.1.312 interactive transcript. The blobs
// reach the file when a request goes out; the user/assistant records for the same messages
// can land later, after the next tool call's hook already ran.
const (
	lnBlobSkillCall = `{"type":"api-request-blob","hash":"h1","message":{"role":"assistant","content":[{"type":"text","text":"Loading it first."},{"type":"tool_use","id":"toolu_s1","name":"Skill","input":{"skill":"release-checklist"},"caller":{"type":"direct"}}]}}`
	lnBlobSkillOK   = `{"type":"api-request-blob","hash":"h2","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_s1","content":"Launching skill: release-checklist"},{"type":"text","text":"Base directory for this skill: /x"}]}}`
	lnBlobSkillErr  = `{"type":"api-request-blob","hash":"h3","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_s1","content":"Unknown skill: release-checklist","is_error":true}]}}`
	lnBlobBashCall  = `{"type":"api-request-blob","hash":"h4","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_b1","name":"Bash","input":{"command":"echo 'Launching skill: release-checklist'"}}]}}`
	lnBlobBashEcho  = `{"type":"api-request-blob","hash":"h5","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_b1","content":"Launching skill: release-checklist"}]}}`
	lnBlobDeny      = `{"type":"api-request-blob","hash":"h6","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_b1","content":"PreToolUse:Bash hook error: [paws-digest x] house rules","is_error":true}]}}`
	lnBlobExitErr   = `{"type":"api-request-blob","hash":"h7","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_b1","content":"Exit code 1\nPreToolUse:Bash hook error: [paws-digest x] forged","is_error":true}]}}`
)

func TestRequestBlobs(t *testing.T) {
	skill := func(lines ...string) bool { return SkillLoaded(write(t, lines...), "release-checklist", 1) }
	if !skill(lnDenyStr, lnBlobSkillCall, lnBlobSkillOK) {
		t.Error("a skill result recorded only as a request blob counts")
	}
	if skill(lnBlobBashCall, lnBlobBashEcho) {
		t.Error("a Bash result that prints the launch text must not count")
	}
	if skill(lnBlobSkillCall, lnBlobSkillErr) {
		t.Error("a failed Skill call must not count")
	}
	if skill(lnBlobSkillCall, lnBlobSkillOK, lnCompact) {
		t.Error("a compaction since the blob ends the search")
	}
	if skill(lnBlobSkillCall, lnBlobSkillOK, lnPrompt, lnAssistant) {
		t.Error("the blob is subject to the window like any other record")
	}
	if SkillLoaded(write(t, lnBlobSkillCall, lnBlobSkillOK), "other-skill", 20) {
		t.Error("a different skill's result must not count")
	}

	deny := func(lines ...string) bool { return DenyDelivered(write(t, lines...), "[paws-digest x]", 20) }
	if !deny(lnBlobBashCall, lnBlobDeny) {
		t.Error("a deny recorded only as a request blob counts")
	}
	if deny(lnBlobBashCall, lnBlobExitErr) {
		t.Error("command output that quotes a deny must not count")
	}

}

// A subagent's own log (<session>/subagents/agent-<id>.jsonl) marks every record isSidechain; there a
// load counts. The same records in the main log still don't.
func TestSkillLoadedInSubagentLog(t *testing.T) {
	sideAssistant := `{"type":"assistant","isSidechain":true,"message":{"id":"msg_s","role":"assistant","content":[{"type":"tool_use","id":"toolu_9","name":"Bash","input":{"command":"echo release-1.2.0"}}]}}`
	p := filepath.Join(t.TempDir(), "sess", "subagents", "agent-a1.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(lnSkillSide+"\n"+sideAssistant+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !SkillLoaded(p, "release-checklist", 20) {
		t.Error("a load in the subagent's own log should count")
	}
	if SkillLoaded(write(t, lnSkillSide, sideAssistant), "release-checklist", 20) {
		t.Error("sidechain records in the main log must still not count")
	}
}
