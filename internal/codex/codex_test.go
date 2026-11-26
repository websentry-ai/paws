package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/rule"
)

// stdin is a hook call shaped like the ones Codex 0.155.1 sent in a live probe: Claude Code's fields plus turn_id
// and model on every event.
func stdin(t *testing.T, event, turn string, extra map[string]any) []byte {
	t.Helper()
	m := map[string]any{"session_id": "01a218bf-886e", "turn_id": turn, "cwd": "/repo", "hook_event_name": event,
		"model": "gpt-5.6-luna", "permission_mode": "bypassPermissions",
		"transcript_path": "/Users/x/.codex/sessions/2026/11/26/rollout-x.jsonl"}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return b
}

func parse(t *testing.T, data []byte) *harness.Hook {
	t.Helper()
	h, err := Adapter{}.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestParsePatch(t *testing.T) {
	patch := "*** Begin Patch\n*** Add File: notes.md\n+PROBE-TOKEN-7731\n+hi\n" +
		"*** Update File: /repo/src/a.go\n@@ func a\n-old\n+new\n context\n" +
		"*** Update File: b.go\n*** Move to: c.go\n@@\n+moved\n" +
		"*** Delete File: gone.txt\n*** End Patch"
	got := parsePatch(patch)
	want := []patchFile{
		{Path: "notes.md", New: true, Added: "PROBE-TOKEN-7731\nhi"},
		{Path: "/repo/src/a.go", Added: "new"},
		{Path: "b.go"},
		{Path: "c.go", Added: "moved"},
		{Path: "gone.txt"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("file %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if f := parsePatch("*** Begin Patch\n  *** Update File: migrations/1.sql  \n+x\n*** End Patch"); len(f) != 1 || f[0].Path != "migrations/1.sql" {
		t.Errorf("a header with spaces around it still names its file: %+v", f)
	}
	if f := parsePatch("not a patch"); len(f) != 0 {
		t.Errorf("no headers, no files: %+v", f)
	}
}

// Codex's tools map to the canonical vocabulary, so every rule written for Claude Code applies unchanged.
func TestParseMapsToolsToCanonicalNames(t *testing.T) {
	bash := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "git push"}}))
	if ev := bash.Calls[0]; len(bash.Calls) != 1 || ev.Tool != "Bash" || ev.Target != "git push" || ev.Kind != rule.KindTool || ev.Agent != "codex" {
		t.Errorf("Bash: %+v", bash.Calls)
	}
	if bash.Calls[0].Log.Model() != "gpt-5.6-luna" {
		t.Errorf("the model comes from stdin, got %q", bash.Calls[0].Log.Model())
	}
	patch := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "apply_patch", "tool_input": map[string]any{
		"command": "*** Begin Patch\n*** Add File: /repo/new.md\n+x\n*** Update File: src/a.go\n@@\n+y\n*** End Patch"}}))
	if len(patch.Calls) != 2 || patch.Calls[0].Tool != "Write" || patch.Calls[0].Target != "/repo/new.md" ||
		patch.Calls[1].Tool != "Edit" || patch.Calls[1].Target != "/repo/src/a.go" {
		t.Errorf("apply_patch: one Write or Edit per file, relative paths from cwd: %+v", patch.Calls)
	}
	agent := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "spawn_agent", "tool_input": map[string]any{"message": "research x"}}))
	if ev := agent.Calls[0]; ev.Tool != "Agent" || ev.Target != "research x" {
		t.Errorf("spawn_agent: %+v", ev)
	}
	mcp := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "mcp__linear__save_issue", "tool_input": map[string]any{"title": "x"}}))
	if ev := mcp.Calls[0]; ev.Tool != "mcp__linear__save_issue" || ev.Target != `{"title":"x"}` {
		t.Errorf("MCP: %+v", ev)
	}
	start := parse(t, stdin(t, "SessionStart", "", map[string]any{"source": "startup"}))
	if ev := start.Calls[0]; ev.Kind != rule.KindSession || ev.Target != "startup" {
		t.Errorf("SessionStart: %+v", ev)
	}
}

// The flag wins; without it, a call that names a turn and a model is Codex's, and anything else Claude Code's.
func TestPickDetectsTheHarness(t *testing.T) {
	adapters := []harness.Adapter{Adapter{}, hook.Claude{}}
	claude := []byte(`{"session_id":"s","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"},"transcript_path":"/x/.claude/projects/p/s.jsonl"}`)
	codex := stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash"})
	for _, c := range []struct {
		flag  string
		stdin []byte
		want  string
	}{
		{"", claude, "claude-code"},
		{"", codex, "codex"},
		{"codex", claude, "codex"},
		{"claude-code", codex, "claude-code"},
		{"", []byte(`not json`), "claude-code"},
	} {
		if got := harness.Pick(adapters, c.flag, c.stdin).Name(); got != c.want {
			t.Errorf("flag %q, stdin %.40s: got %s, want %s", c.flag, c.stdin, got, c.want)
		}
	}
}

// A deny goes in Claude Code's shape; Codex reads it the same way.
func TestRenderDenyForCodex(t *testing.T) {
	h := &harness.Hook{Event: "PreToolUse"}
	var out struct {
		HookSpecificOutput struct{ PermissionDecision, PermissionDecisionReason string }
	}
	json.Unmarshal(Adapter{}.Render(h, gate.Decision{Verdict: gate.Deny, Reason: "paws: read it, then retry."}, ""), &out)
	if out.HookSpecificOutput.PermissionDecision != "deny" || out.HookSpecificOutput.PermissionDecisionReason != "paws: read it, then retry." {
		t.Errorf("got %+v", out)
	}
	if got := string(Adapter{}.Render(&harness.Hook{Event: "Stop"}, gate.Decision{}, "")); got != "{}" {
		t.Errorf("Stop: got %s", got)
	}
}

// ledgerEnv is a scratch PAWS_HOME and an installed skill's SKILL.md.
func ledgerEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("PAWS_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	ledgerCache = map[string][]record{}
	old := gate.SkillRoots
	gate.SkillRoots = func(cwd string) ([]string, []string) { return Adapter{}.SkillDirs(cwd), nil }
	t.Cleanup(func() { gate.SkillRoots = old })
	md := filepath.Join(os.Getenv("HOME"), ".agents", "skills", "git", "SKILL.md")
	os.MkdirAll(filepath.Dir(md), 0o755)
	os.WriteFile(md, []byte("# git\n"), 0o644)
	return md
}

// observe runs one hook call through Parse and Observe, as the gate does, and returns its first event.
func observe(t *testing.T, data []byte, d gate.Decision) gate.Event {
	t.Helper()
	h := parse(t, data)
	Adapter{}.Observe(h, d)
	ledgerCache = map[string][]record{} // the next hook is a new process
	return h.Calls[0]
}

func bashCall(id, cmd string) map[string]any {
	return map[string]any{"tool_name": "Bash", "tool_use_id": id, "tool_input": map[string]any{"command": cmd}}
}

func result(id, output string) map[string]any {
	return map[string]any{"tool_name": "Bash", "tool_use_id": id, "tool_response": output}
}

func TestLedgerSkillWindow(t *testing.T) {
	md := ledgerEnv(t)
	read := "sed -n '1,240p' " + md
	ev := observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", read)), gate.Decision{})
	if ev.SkillLoad != "git" {
		t.Fatalf("a sed -n read of SKILL.md is a load, got %q", ev.SkillLoad)
	}
	next := func(turn string) gate.Session {
		return parse(t, stdin(t, "PreToolUse", turn, bashCall("c9", "git push"))).Calls[0].Log
	}
	if next("t1").SkillLoaded("git", 20) {
		t.Error("a read still running (no result yet) is not a load")
	}
	observe(t, stdin(t, "PostToolUse", "t1", result("c1", "---\nname: git\n---\nIf push says Permission denied, check the key.\n")), gate.Decision{})
	if !next("t1").SkillLoaded("git", 20) || next("t1").SkillLoaded("other", 20) {
		t.Error("a successful read loads that skill only")
	}
	for i := 2; i <= 20; i++ {
		observe(t, stdin(t, "UserPromptSubmit", "t"+string(rune('a'+i)), nil), gate.Decision{})
	}
	if next("t21").SkillLoaded("git", 20) {
		t.Error("21 turns later the load is out of the window")
	}
	if !next("tz").SkillLoaded("git", 21) {
		t.Error("a wider window still holds it")
	}
	observe(t, stdin(t, "PostCompact", "tz", nil), gate.Decision{})
	if next("tz").SkillLoaded("git", 100) {
		t.Error("a compaction after the load resets it")
	}
}

func TestLedgerFailedReadLoadsNothing(t *testing.T) {
	md := ledgerEnv(t)
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", "cat "+md)), gate.Decision{})
	observe(t, stdin(t, "PostToolUse", "t1", result("c1", "cat: "+md+": Permission denied\n")), gate.Decision{})
	if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c2", "git push"))).Calls[0].Log.SkillLoaded("git", 20) {
		t.Error("a read that failed is not a load")
	}
}

func TestLedgerDenyEditsAndAgents(t *testing.T) {
	ledgerEnv(t)
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", "aws s3 mb s3://x")),
		gate.Decision{Verdict: gate.Deny, Reason: "paws (rule a): x.\n\npaws digest (s3): This action requires the rules"})
	patch := map[string]any{"tool_name": "apply_patch", "tool_use_id": "p1", "tool_input": map[string]any{
		"command": "*** Begin Patch\n*** Update File: /repo/src/a_test.go\n@@\n+x\n*** End Patch"}}
	observe(t, stdin(t, "PreToolUse", "t1", patch), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "spawn_agent", "tool_use_id": "a1", "tool_input": map[string]any{"message": "x"}}), gate.Decision{})
	log := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c2", "ls"))).Calls[0].Log
	if !log.DenyDelivered("paws digest (s3):", 20) || log.DenyDelivered("paws digest (other):", 20) {
		t.Error("a digest deny in the ledger was delivered; another wasn't")
	}
	glob := regexp.MustCompile(`_test\.go$`)
	if log.EditedMatching(glob) {
		t.Error("an edit with no result yet doesn't count")
	}
	observe(t, stdin(t, "PostToolUse", "t1", map[string]any{"tool_name": "apply_patch", "tool_use_id": "p1", "tool_response": "Exit code: 0\nWall time: 0.4 seconds\nOutput:\nSuccess."}), gate.Decision{})
	log = parse(t, stdin(t, "PreToolUse", "t1", bashCall("c2", "ls"))).Calls[0].Log
	if !log.EditedMatching(glob) {
		t.Error("a successful patch to a test file counts")
	}
	if log.AgentsSinceUserTurn() != 1 {
		t.Errorf("one spawn this turn, got %d", log.AgentsSinceUserTurn())
	}
	spawn := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "spawn_agent", "tool_use_id": "a2", "tool_input": map[string]any{"message": "y"}}))
	if n := spawn.Calls[0].Log.AgentsSinceUserTurn(); n != 2 {
		t.Errorf("a spawn being gated counts itself, as on Claude Code: want 2, got %d", n)
	}
	if n := parse(t, stdin(t, "PreToolUse", "t2", bashCall("c3", "ls"))).Calls[0].Log.AgentsSinceUserTurn(); n != 0 {
		t.Errorf("a new turn has no spawns yet, got %d", n)
	}
}

// Outcomes read the turn that stopped from the ledger by its turn id, with each call's verdict.
func TestLedgerTurnCalls(t *testing.T) {
	md := ledgerEnv(t)
	observe(t, stdin(t, "UserPromptSubmit", "t1", nil), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c0", "ls")), gate.Decision{})
	time.Sleep(time.Millisecond)
	observe(t, stdin(t, "UserPromptSubmit", "t2", nil), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t2", bashCall("c1", "git push")), gate.Decision{Verdict: gate.Deny, Reason: "no"})
	observe(t, stdin(t, "PreToolUse", "t2", bashCall("c2", "cat "+md)), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t2", bashCall("c3", "git push")), gate.Decision{})
	tl := Adapter{}.Turns(parse(t, stdin(t, "Stop", "t2", nil)))
	at, ok := tl.LastRecordAt(time.Now())
	if !ok {
		t.Fatal("the stopped turn has records")
	}
	calls, start, err := tl.TurnCalls(at)
	if err != nil || len(calls) != 3 || start.After(calls[0].At) {
		t.Fatalf("want turn t2's 3 calls, got %+v %v", calls, err)
	}
	if !calls[0].Denied || calls[1].SkillLoad != "git" || calls[2].Denied || calls[2].Target != "git push" {
		t.Errorf("verdicts and loads: %+v", calls)
	}
	if strings.Contains(calls[0].ID, "c0") {
		t.Error("turn t1's call is not in t2")
	}
}

// The ledger masks secrets in every field it keeps, as the audit log does.
func TestLedgerMasksSecrets(t *testing.T) {
	ledgerEnv(t)
	secret := "sk-ant-abcdefghijklmnopqrstuvwx"
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", `curl -H "Authorization: Bearer `+secret+`" x`)), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "mcp__x__send", "tool_use_id": "m1",
		"tool_input": map[string]any{"to": "a", "body": map[string]any{"key": secret, "token=" + secret: "v"}}}), gate.Decision{})
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c2", "ls")), gate.Decision{Verdict: gate.Deny, Reason: "paws: no. Bearer " + secret})
	data, _ := os.ReadFile(ledgerPath("01a218bf-886e", ""))
	if strings.Contains(string(data), secret) || !strings.Contains(string(data), "***") {
		t.Errorf("a secret reached the ledger: %s", data)
	}
}

// Codex can run a patch through the shell: the command is still gated as Bash, and each file as the edit it is.
func TestParseShellApplyPatch(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: migrations/1.sql\n+x\n*** End Patch\nEOF"}}))
	if len(h.Calls) != 2 || h.Calls[0].Tool != "Bash" || h.Calls[1].Tool != "Write" || h.Calls[1].Target != "/repo/migrations/1.sql" {
		t.Errorf("want the Bash call and a Write of the file, got %+v", h.Calls)
	}
}

// Codex reads a long skill in chunks: it loads once the chunks that succeeded cover every line.
func TestLedgerLoadsWhenReadsCoverTheSkill(t *testing.T) {
	md := ledgerEnv(t)
	os.WriteFile(md, []byte(strings.Repeat("rule line\n", 300)), 0o644)
	loaded := func() bool {
		return parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "git push"))).Calls[0].Log.SkillLoaded("git", 20)
	}
	read := func(id, cmd string) {
		observe(t, stdin(t, "PreToolUse", "t1", bashCall(id, cmd)), gate.Decision{})
		observe(t, stdin(t, "PostToolUse", "t1", result(id, "rule line\n")), gate.Decision{})
	}
	read("c1", "sed -n '1,240p' "+md)
	if loaded() {
		t.Error("lines 1-240 of a 300-line skill don't load it")
	}
	read("c2", "head -n 5 "+md)
	if loaded() {
		t.Error("head -n 5 adds nothing past line 240")
	}
	read("c3", "sed -n '241,480p' "+md)
	if !loaded() {
		t.Error("1-240 and 241-480 cover the skill: loaded")
	}
}

// A patch run through the shell counts as an edit only when apply_patch says it succeeded.
func TestLedgerShellPatchNeedsSuccess(t *testing.T) {
	ledgerEnv(t)
	cmd := "apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: /repo/src/a_test.go\n@@\n+x\n*** End Patch\nEOF"
	glob := regexp.MustCompile(`_test\.go$`)
	edited := func(id, response string) bool {
		observe(t, stdin(t, "PreToolUse", "t1", bashCall(id, cmd)), gate.Decision{})
		post := bashCall(id, cmd) // Codex sends the call's input with its result
		post["tool_response"] = response
		observe(t, stdin(t, "PostToolUse", "t1", post), gate.Decision{})
		return parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.EditedMatching(glob)
	}
	if edited("p1", "Failed to find expected lines in /repo/src/a_test.go:\nold\n") {
		t.Error("a rejected patch is no edit")
	}
	if !edited("p2", "Success. Updated the following files:\nM /repo/src/a_test.go\n") {
		t.Error("a patch apply_patch reports done is an edit")
	}
	ledgerEnv(t)
	if edited("p3", "Success. Updated the following files:\nM /repo/src/other.go\n") {
		t.Error("a file the patch didn't change, by apply_patch's own list, is no edit")
	}
}
