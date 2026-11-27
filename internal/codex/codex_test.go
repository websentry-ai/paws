package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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
// TestMain keeps every test off the real home: a test that skips ledgerEnv still reads and writes temp folders.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "paws-codex-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", tmp)
	os.Setenv("PAWS_HOME", filepath.Join(tmp, ".paws"))
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

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
	observe(t, stdin(t, "PostToolUse", "t1", map[string]any{"tool_name": "apply_patch", "tool_use_id": "p1", "tool_response": "Exit code: 0\nWall time: 0.4 seconds\nOutput:\nSuccess. Updated the following files:\nM /repo/src/a_test.go\n"}), gate.Decision{})
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

// A shell patch after a cd edits files in that folder.
func TestParseShellPatchAfterCd(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "ls && cd db; cd migrations && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/db/migrations/1.sql") {
		t.Errorf("want /repo/db/migrations/1.sql, got %+v", h.Calls)
	}
}

// apply_patch lists the files it changed relative to where it ran: after a cd, that folder.
func TestLedgerShellPatchResultAfterCd(t *testing.T) {
	ledgerEnv(t)
	repo := onDisk(t, "src/a_test.go")
	cmd := "cd src && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: a_test.go\n@@\n+x\n*** End Patch\nEOF"
	observe(t, stdin(t, "PreToolUse", "t1", in(repo, bashCall("p1", cmd))), gate.Decision{})
	post := in(repo, bashCall("p1", cmd))
	post["tool_response"] = "Success. Updated the following files:\nM a_test.go\n"
	observe(t, stdin(t, "PostToolUse", "t1", post), gate.Decision{})
	if !parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.EditedMatching(regexp.MustCompile(`src/a_test\.go$`)) {
		t.Error("src/a_test.go was edited")
	}
}

// Patch text that apply_patch doesn't run (written to a file, say) edits nothing; a subshell's cd still counts.
func TestShellPatchNeedsApplyPatch(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cat > fix.patch <<'EOF'\n*** Begin Patch\n*** Add File: migrations/1.sql\n+x\n*** End Patch\nEOF"}}))
	if len(h.Calls) != 1 || h.Calls[0].Tool != "Bash" {
		t.Errorf("only a Bash call, got %+v", h.Calls)
	}
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "(cd migrations && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF\n)"}}))
	if !gated(h.Calls, "/repo/migrations/1.sql") {
		t.Errorf("want /repo/migrations/1.sql, got %+v", h.Calls)
	}
}

// A cd inside a subshell that closes before apply_patch moves nothing.
func TestShellPatchAfterAClosedSubshell(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "(cd tmp && ls) && cd db && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/db/1.sql") {
		t.Errorf("want /repo/db/1.sql, got %+v", h.Calls)
	}
}

// Each apply_patch in one command resolves from its own folder.
func TestShellPatchesEachFromTheirFolder(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cd a && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF\ncd ../b && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 2.sql\n+y\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/a/1.sql") || !gated(h.Calls, "/repo/b/2.sql") {
		t.Errorf("want /repo/a/1.sql and /repo/b/2.sql, got %+v", h.Calls)
	}
}

// apply_patch's file lists pair with their own runs' folders; a cd to $VAR resolves from the environment.
func TestShellPatchResultsPairWithTheirFolders(t *testing.T) {
	ledgerEnv(t)
	repo := onDisk(t, "a/x_test.go", "db/y.sql")
	t.Setenv("MIGS", filepath.Join(repo, "db"))
	cmd := "cd a && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: x_test.go\n@@\n+x\n*** End Patch\nEOF\ncd $MIGS && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: y.sql\n@@\n+y\n*** End Patch\nEOF"
	h := parse(t, stdin(t, "PreToolUse", "t1", in(repo, bashCall("p1", cmd))))
	if !gated(h.Calls, filepath.Join(repo, "db", "y.sql")) {
		t.Fatalf("cd $MIGS: want db/y.sql, got %+v", h.Calls)
	}
	Adapter{}.Observe(h, gate.Decision{})
	post := in(repo, bashCall("p1", cmd))
	post["tool_response"] = "Success. Updated the following files:\nM x_test.go\nSuccess. Updated the following files:\nM y.sql\n"
	observe(t, stdin(t, "PostToolUse", "t1", post), gate.Decision{})
	log := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log
	if !log.EditedMatching(regexp.MustCompile(`/a/x_test\.go$`)) || !log.EditedMatching(regexp.MustCompile(`/db/y\.sql$`)) {
		t.Error("each file is the edit of its own run's folder")
	}
}

// A read Codex cut before the model saw it covers only the lines before the cut.
func TestLedgerCutReadCoversWhatTheModelSaw(t *testing.T) {
	md := ledgerEnv(t)
	os.WriteFile(md, []byte(strings.Repeat("rule line\n", 300)), 0o644)
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", "cat "+md)), gate.Decision{})
	observe(t, stdin(t, "PostToolUse", "t1", result("c1", "rule line\nrule line\n…9000 tokens truncated…\nrule line\n")), gate.Decision{})
	if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "git push"))).Calls[0].Log.SkillLoaded("git", 20) {
		t.Error("two lines of 300 reached the model: not loaded")
	}
}

// "apply_patch" inside a file name doesn't start a new patch: the file is still gated.
func TestShellPatchNamedApplyPatch(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: tools/apply_patch.py\n@@\n+x\n*** End Patch\nEOF"}}))
	if len(h.Calls) != 2 || h.Calls[1].Tool != "Edit" || h.Calls[1].Target != "/repo/tools/apply_patch.py" {
		t.Errorf("want an Edit of /repo/tools/apply_patch.py, got %+v", h.Calls)
	}
}

// A failed run lists no files; the next run's list still pairs with its own folder.
func TestShellPatchResultsAfterAFailedRun(t *testing.T) {
	ledgerEnv(t)
	repo := onDisk(t, "a/x.go", "b/y_test.go")
	cmd := "cd a && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: x.go\n@@\n+x\n*** End Patch\nEOF\ncd ../b && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: y_test.go\n@@\n+y\n*** End Patch\nEOF"
	observe(t, stdin(t, "PreToolUse", "t1", in(repo, bashCall("p1", cmd))), gate.Decision{})
	post := in(repo, bashCall("p1", cmd))
	post["tool_response"] = "Failed to find expected lines in x.go\nSuccess. Updated the following files:\nM y_test.go\n"
	observe(t, stdin(t, "PostToolUse", "t1", post), gate.Decision{})
	log := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log
	if !log.EditedMatching(regexp.MustCompile(`/b/y_test\.go$`)) || log.EditedMatching(regexp.MustCompile(`x\.go$`)) {
		t.Error("y_test.go in b was edited; x.go wasn't")
	}
}

// The Stop grades its own turn, even when an earlier turn's marker lands after its calls.
func TestTurnCallsGradesTheStopsTurn(t *testing.T) {
	ledgerEnv(t)
	observe(t, stdin(t, "PreToolUse", "t2", bashCall("c1", "git push")), gate.Decision{Verdict: gate.Deny, Reason: "no"})
	// The turn before's marker, from an async UserPromptSubmit, lands after t2's call.
	appendRecords(ledgerPath("01a218bf-886e", ""), []record{{Kind: "turn", At: time.Now().Add(-time.Minute), Turn: "t1"}})
	ledgerCache = map[string][]record{}
	tl := Adapter{}.Turns(parse(t, stdin(t, "Stop", "t2", nil)))
	at, _ := tl.LastRecordAt(time.Now())
	calls, _, err := tl.TurnCalls(at)
	if err != nil || len(calls) != 1 || calls[0].Target != "git push" {
		t.Errorf("want t2's one call, got %+v %v", calls, err)
	}
}

// A wrapper that runs the next word as the command still runs apply_patch; a path two runs both name is no one's.
func TestShellPatchWrappersAndAmbiguousResults(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "env LANG=C command apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: migrations/1.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/migrations/1.sql") {
		t.Errorf("command apply_patch: want a Write of /repo/migrations/1.sql, got %+v", h.Calls)
	}
	ledgerEnv(t)
	cmd := "cd a && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: x_test.go\n@@\n+x\n*** End Patch\nEOF\ncd ../b && apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: x_test.go\n@@\n+y\n*** End Patch\nEOF"
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("p1", cmd)), gate.Decision{})
	post := bashCall("p1", cmd)
	post["tool_response"] = "Failed to find expected lines\nSuccess. Updated the following files:\nM x_test.go\n"
	observe(t, stdin(t, "PostToolUse", "t1", post), gate.Decision{})
	if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.EditedMatching(regexp.MustCompile(`/a/x_test\.go$`)) {
		t.Error("which run changed x_test.go is unknown: a's copy isn't credited")
	}
}

// Codex's applypatch alias and a `cd -- dir` count like apply_patch and cd dir.
func TestShellPatchAliasAndCdDashDash(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cd -- migrations && applypatch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/migrations/1.sql") {
		t.Errorf("want a Write of /repo/migrations/1.sql, got %+v", h.Calls)
	}
}

// A bare cd goes home.
func TestShellPatchAfterBareCd(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cd && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: notes.md\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/home/u/notes.md") {
		t.Errorf("want /home/u/notes.md, got %+v", h.Calls)
	}
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cdx foo && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: notes.md\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/notes.md") {
		t.Errorf("cdx is not cd: want /repo/notes.md, got %+v", h.Calls)
	}
}

// Heredoc text before a patch is data, not cd commands; a patch read from a file on stdin is gated too.
func TestShellPatchHeredocDataAndStdinFile(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cat > notes.txt <<'EOF'\ncd elsewhere\nEOF\napply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/1.sql") {
		t.Errorf("heredoc data moved the folder: want /repo/1.sql, got %+v", h.Calls)
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fix.patch"), []byte("*** Begin Patch\n*** Update File: migrations/2.sql\n@@\n+y\n*** End Patch\n"), 0o644)
	for _, cmd := range []string{"apply_patch < fix.patch", "apply_patch 0< fix.patch"} {
		h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{"command": cmd}}))
		if len(h.Calls) != 2 || h.Calls[1].Tool != "Edit" || h.Calls[1].Target != filepath.Join(dir, "migrations", "2.sql") {
			t.Errorf("%s: want an Edit of migrations/2.sql, got %+v", cmd, h.Calls)
		}
	}
}

// A patch file the same call writes first is the patch apply_patch reads; a here-string is not a here-document.
func TestShellPatchFileWrittenInTheSameCall(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cat > /tmp/new.patch <<'EOF'\n*** Begin Patch\n*** Update File: migrations/3.sql\n@@\n+z\n*** End Patch\nEOF\napply_patch < /tmp/new.patch"}}))
	if !gated(h.Calls, "/repo/migrations/3.sql") {
		t.Errorf("want an Edit of /repo/migrations/3.sql, got %+v", h.Calls)
	}
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "cat <<< foo\ncd migrations && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 4.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/migrations/4.sql") {
		t.Errorf("a here-string hid the cd: want /repo/migrations/4.sql, got %+v", h.Calls)
	}
}

// A patch piped in from a file is gated like one read on stdin.
func TestShellPatchPipedFromAFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fix.patch"), []byte("*** Begin Patch\n*** Update File: migrations/5.sql\n@@\n+y\n*** End Patch\n"), 0o644)
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{"command": "cat fix.patch | apply_patch"}}))
	if !gated(h.Calls, filepath.Join(dir, "migrations", "5.sql")) {
		t.Errorf("want an Edit of migrations/5.sql, got %+v", h.Calls)
	}
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{
		"command": "cat > new.patch <<'EOF'\n*** Begin Patch\n*** Add File: migrations/6.sql\n+z\n*** End Patch\nEOF\ncat new.patch | apply_patch"}}))
	if !gated(h.Calls, filepath.Join(dir, "migrations", "6.sql")) {
		t.Errorf("a piped file written in the same call: want a Write of migrations/6.sql, got %+v", h.Calls)
	}
}

// A quoted patch file name may hold spaces.
func TestShellPatchQuotedFileName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "my fix.patch"), []byte("*** Begin Patch\n*** Update File: migrations/7.sql\n@@\n+y\n*** End Patch\n"), 0o644)
	for _, cmd := range []string{`apply_patch < "my fix.patch"`, `cat 'my fix.patch' | apply_patch`} {
		h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{"command": cmd}}))
		if !gated(h.Calls, filepath.Join(dir, "migrations", "7.sql")) {
			t.Errorf("%s: want an Edit of migrations/7.sql, got %+v", cmd, h.Calls)
		}
	}
}

// Variables the command sets itself resolve a cd and a patch file name, as in the live run (D=sub; cd $D).
func TestShellPatchWithShellLocalVariables(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "D=sub; cd $D && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: x.txt\n+hi\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/sub/x.txt") {
		t.Errorf("want /repo/sub/x.txt, got %+v", h.Calls)
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fix.patch"), []byte("*** Begin Patch\n*** Update File: migrations/8.sql\n@@\n+y\n*** End Patch\n"), 0o644)
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{"command": `P=fix.patch && apply_patch < "$P"`}}))
	if !gated(h.Calls, filepath.Join(dir, "migrations", "8.sql")) {
		t.Errorf(`apply_patch < "$P": want an Edit of migrations/8.sql, got %+v`, h.Calls)
	}
}

// gated reports whether a call targets path.
func gated(calls []gate.Event, path string) bool {
	for _, c := range calls {
		if c.Target == path && (c.Tool == "Write" || c.Tool == "Edit") {
			return true
		}
	}
	return false
}

// A cd behind && may not run: the patch's files are gated from the folder before it too.
func TestShellPatchGatesEveryFolderItMayRunIn(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "false && cd migrations; apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: 9.sql\n+x\n*** End Patch\nEOF"}}))
	if !gated(h.Calls, "/repo/9.sql") || !gated(h.Calls, "/repo/migrations/9.sql") {
		t.Errorf("want both /repo/9.sql and /repo/migrations/9.sql gated, got %+v", h.Calls)
	}
}

// A skipped cd then a relative cd lands somewhere neither path alone names; a redirected patch is read from each folder.
func TestShellPatchSkippedCdCombinations(t *testing.T) {
	h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
		"command": "false && cd a; cd b && apply_patch <<'EOF'\n*** Begin Patch\n*** Add File: x.sql\n+x\n*** End Patch\nEOF"}}))
	for _, want := range []string{"/repo/x.sql", "/repo/a/x.sql", "/repo/b/x.sql", "/repo/a/b/x.sql"} {
		if !gated(h.Calls, want) {
			t.Errorf("want %s gated, got %+v", want, h.Calls)
		}
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "fix.patch"), []byte("*** Begin Patch\n*** Update File: migrations/10.sql\n@@\n+y\n*** End Patch\n"), 0o644)
	h = parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "cwd": dir, "tool_input": map[string]any{"command": "false && cd sub; apply_patch < fix.patch"}}))
	if !gated(h.Calls, filepath.Join(dir, "migrations", "10.sql")) {
		t.Errorf("the patch file in the folder before the skipped cd is read: got %+v", h.Calls)
	}
}

// apply_patch after {, !, then, do, else or elif starts a shell patch too; a native patch with no file list edits nothing.
func TestShellPatchAfterCompoundStarts(t *testing.T) {
	for _, cmd := range []string{"{ apply_patch", "if true; then apply_patch", "! apply_patch"} {
		h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{
			"command": cmd + " <<'EOF'\n*** Begin Patch\n*** Add File: m.sql\n+x\n*** End Patch\nEOF"}}))
		if !gated(h.Calls, "/repo/m.sql") {
			t.Errorf("%s: want /repo/m.sql gated, got %+v", cmd, h.Calls)
		}
	}
	ledgerEnv(t)
	patch := map[string]any{"tool_name": "apply_patch", "tool_use_id": "n1", "tool_input": map[string]any{
		"command": "*** Begin Patch\n*** Update File: /repo/src/b_test.go\n@@\n+x\n*** End Patch"}}
	observe(t, stdin(t, "PreToolUse", "t1", patch), gate.Decision{})
	patch["tool_response"] = "Exit code: 0\nOutput:\nSuccess."
	observe(t, stdin(t, "PostToolUse", "t1", patch), gate.Decision{})
	if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.EditedMatching(regexp.MustCompile(`b_test\.go$`)) {
		t.Error("a result that lists no file credits no edit")
	}
}

// onDisk makes a folder holding files at the given relative paths, as a patch leaves them.
func onDisk(t *testing.T, rels ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, r := range rels {
		p := filepath.Join(root, r)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x\n"), 0o644)
	}
	return root
}

// in sets a hook call's cwd.
func in(cwd string, m map[string]any) map[string]any {
	m["cwd"] = cwd
	return m
}

// A read of part of SKILL.md is still a skill read, but only a read of all of it loads the skill alone.
func TestSkillReadWholeOrPart(t *testing.T) {
	md := ledgerEnv(t)
	os.WriteFile(md, []byte(strings.Repeat("rule line\n", 30)), 0o644)
	for cmd, whole := range map[string]bool{"sed -n '1,240p' " + md: true, "sed -n '1,10p' " + md: false, "cat " + md: true} {
		ev := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c1", cmd))).Calls[0]
		if ev.SkillLoad != "git" || ev.SkillWhole != whole {
			t.Errorf("%s: want skill git, whole %v; got %q, %v", cmd, whole, ev.SkillLoad, ev.SkillWhole)
		}
	}
}

// apply_patch run by another shell (bash -c, eval, xargs) is a shell patch too, and a cd with a redirect or inside
// the string still moves it; a file named apply_patch or patch text mentioning it is not a run.
func TestShellPatchInWrappersAndRedirectedCd(t *testing.T) {
	patch := "<<'EOF'\n*** Begin Patch\n*** Add File: 1.sql\n+x\n*** End Patch\nEOF"
	for cmd, want := range map[string]string{
		"bash -c \"apply_patch " + patch + "\"":                   "/repo/1.sql",
		"bash -lc \"cd migrations && apply_patch " + patch + "\"": "/repo/migrations/1.sql",
		"eval 'cd migrations'; apply_patch " + patch:              "/repo/migrations/1.sql",
		"cd migrations >/dev/null && apply_patch " + patch:        "/repo/migrations/1.sql",
		"cd migrations 2>&1 && apply_patch " + patch:              "/repo/migrations/1.sql",
		"cd >/dev/null migrations && apply_patch " + patch:        "/repo/migrations/1.sql",
		"pushd migrations && apply_patch " + patch:                "/repo/migrations/1.sql",
		"pushd migrations >/dev/null; apply_patch " + patch:       "/repo/migrations/1.sql",
		"builtin cd migrations && apply_patch " + patch:           "/repo/migrations/1.sql",
		"/usr/local/bin/apply_patch " + patch:                     "/repo/1.sql",
		"cd migrations && ./bin/applypatch " + patch:              "/repo/migrations/1.sql",
		"echo " + patch + " | xargs -0 apply_patch":               "/repo/1.sql",
	} {
		h := parse(t, stdin(t, "PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": cmd}}))
		if !gated(h.Calls, want) {
			t.Errorf("%q: want %s gated, got %+v", cmd, want, h.Calls)
		}
	}
	h := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c1", "python tools/apply_patch.py fix.diff && cat apply_patch.md")))
	if len(h.Calls) != 1 {
		t.Errorf("a file named apply_patch runs no patch, got %+v", h.Calls)
	}
	h = parse(t, stdin(t, "PreToolUse", "t1", bashCall("c2",
		"apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: docs.md\n@@\n+run apply_patch here\n*** Update File: b.md\n@@\n+y\n*** End Patch\nEOF")))
	if !gated(h.Calls, "/repo/docs.md") || !gated(h.Calls, "/repo/b.md") {
		t.Errorf("a patch that mentions apply_patch is still one patch, got %+v", h.Calls)
	}
}

// A spawn a hook denied started no agent: the retry is still the first.
func TestLedgerAgentCountSkipsDeniedSpawns(t *testing.T) {
	ledgerEnv(t)
	spawn := func(id string) map[string]any {
		return map[string]any{"tool_name": "Agent", "tool_use_id": id, "tool_input": map[string]any{"prompt": "x"}}
	}
	observe(t, stdin(t, "PreToolUse", "t1", spawn("s1")), gate.Decision{Verdict: gate.Deny, Reason: "no"})
	observe(t, stdin(t, "PreToolUse", "t1", spawn("s2")), gate.Decision{})
	if n := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.AgentsSinceUserTurn(); n != 1 {
		t.Errorf("one agent started, got %d", n)
	}
}

// A reader the shell couldn't find read nothing: the skill isn't loaded.
func TestLedgerReaderNotFoundIsNoRead(t *testing.T) {
	for _, out := range []string{"bash: sed: command not found\n", "zsh: command not found: sed\n", "sh: 1: sed: not found\n"} {
		md := ledgerEnv(t)
		read := "sed -n '1,240p' " + md
		observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", read)), gate.Decision{})
		observe(t, stdin(t, "PostToolUse", "t1", result("c1", out)), gate.Decision{})
		if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "git push"))).Calls[0].Log.SkillLoaded("git", 20) {
			t.Errorf("%q: the read never ran", out)
		}
	}
}

// Two spawns gated at the same time: both records land before either is decided, and the later one still counts
// the earlier one.
func TestLedgerConcurrentSpawnsCountEachOther(t *testing.T) {
	ledgerEnv(t)
	spawn := func(id string) map[string]any {
		return map[string]any{"tool_name": "Agent", "tool_use_id": id, "tool_input": map[string]any{"prompt": "x"}}
	}
	first := parse(t, stdin(t, "PreToolUse", "t1", spawn("s1"))).Calls[0].Log
	second := parse(t, stdin(t, "PreToolUse", "t1", spawn("s2"))).Calls[0].Log
	if a, b := first.AgentsSinceUserTurn(), second.AgentsSinceUserTurn(); a != 1 || b != 2 {
		t.Errorf("the first spawn is the first, the second the second: got %d and %d", a, b)
	}
	if n := parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "ls"))).Calls[0].Log.AgentsSinceUserTurn(); n != 2 {
		t.Errorf("a later call sees both spawns, got %d", n)
	}
}

// A context line in an Update File hunk that reads like a header is the file's text, as Codex reads it; a header
// right after Begin Patch or after an Add File's lines may be indented.
func TestPatchContextLineIsNotAHeader(t *testing.T) {
	got := parsePatch("*** Begin Patch\n*** Update File: README.md\n@@\n *** Update File: migrations/x.sql\n-old\n+new\n" +
		"*** Add File: a.txt\n+a\n  *** Update File: b.txt\n@@\n+b\n*** End Patch")
	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	if !slices.Equal(paths, []string{"README.md", "a.txt", "b.txt"}) {
		t.Errorf("want README.md, a.txt, b.txt; got %v", paths)
	}
	if got[0].Added != "new" {
		t.Errorf("README.md adds %q, want new", got[0].Added)
	}
}

// Skill text on the first line of a read is the file, even when it ends like an error.
func TestLedgerSkillTextLikeAnErrorIsARead(t *testing.T) {
	for _, first := range []string{"Returns 404 when the page is not found", "Note: Permission denied", "deploy: command not found"} {
		md := ledgerEnv(t)
		os.WriteFile(md, []byte(first+"\n"), 0o644)
		read := "sed -n '1,240p' " + md
		observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", read)), gate.Decision{})
		observe(t, stdin(t, "PostToolUse", "t1", result("c1", first+"\n")), gate.Decision{})
		if !parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "git push"))).Calls[0].Log.SkillLoaded("git", 20) {
			t.Errorf("%q: the skill was read", first)
		}
	}
	for _, out := range []string{"sed: /x/SKILL.md: No such file or directory", "cat: /x: Permission denied", "bash: line 1: sed: command not found"} {
		if resultOK(json.RawMessage(strconv.Quote(out + "\n"))) {
			t.Errorf("%q is a failed read", out)
		}
	}
}

// Codex's result header isn't part of the file: a read cut three lines short of the end is still cut.
func TestLedgerCutReadDoesNotCountTheHeader(t *testing.T) {
	md := ledgerEnv(t)
	os.WriteFile(md, []byte(strings.Repeat("rule line\n", 10)), 0o644)
	observe(t, stdin(t, "PreToolUse", "t1", bashCall("c1", "cat "+md)), gate.Decision{})
	out := "Exit code: 0\nWall time: 0.1 seconds\nOutput:\n" + strings.Repeat("rule line\n", 7) + "…40 tokens truncated…\n"
	observe(t, stdin(t, "PostToolUse", "t1", result("c1", out)), gate.Decision{})
	if parse(t, stdin(t, "PreToolUse", "t1", bashCall("c9", "git push"))).Calls[0].Log.SkillLoaded("git", 20) {
		t.Error("seven lines of ten reached the model: not loaded")
	}
}
