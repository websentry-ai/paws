package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// codexCall is a hook call shaped as Codex sends it: Claude Code's fields plus turn_id and model.
func codexCall(event, turn string, extra map[string]any) map[string]any {
	m := map[string]any{"hook_event_name": event, "session_id": "cx-1", "turn_id": turn, "model": "gpt-5.6-luna",
		"permission_mode": "bypassPermissions", "cwd": os.Getenv("HOME")}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func codexSkill(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(os.Getenv("HOME"), ".agents", "skills", name, "SKILL.md")
	writeFile(t, p, "---\nname: "+name+"\n---\n# "+name+"\n")
	return p
}

func reasonOf(t *testing.T, out string) string {
	t.Helper()
	var o struct {
		HookSpecificOutput struct{ PermissionDecision, PermissionDecisionReason string }
	}
	json.Unmarshal([]byte(out), &o)
	if o.HookSpecificOutput.PermissionDecision != "deny" {
		return ""
	}
	return o.HookSpecificOutput.PermissionDecisionReason
}

// The whole require-skill round on Codex: a deny that names the SKILL.md to read, the read itself let through, the
// retry allowed from the ledger, and the turn graded at Stop.
func TestCodexRequireSkillRound(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "push.md"), "name: push\nevent: pre-tool\nmatcher:\n  tool: Bash\naction:\n  type: require-skill\n  skill: git\n")
	md := codexSkill(t, "git")
	bash := func(id, cmd string) map[string]any {
		return map[string]any{"tool_name": "Bash", "tool_use_id": id, "tool_input": map[string]any{"command": cmd}}
	}
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	r := reasonOf(t, runGate(t, codexCall("PreToolUse", "t1", bash("c1", "git push"))))
	want := `paws (rule push): this is paws working as intended, nothing is broken. This command needs the "git" skill. Read ` + md + ` in full on its own, follow it, then run the command again, changed if the skill says so`
	if r != want {
		t.Fatalf("deny:\n got %q\nwant %q", r, want)
	}
	read := "sed -n '1,240p' " + md
	if out := runGate(t, codexCall("PreToolUse", "t1", bash("c2", read))); out != "{}" {
		t.Fatalf("the read of SKILL.md was gated by a catch-all Bash rule: %s", out)
	}
	runGate(t, codexCall("PostToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_use_id": "c2", "tool_input": map[string]any{"command": read}, "tool_response": "# git\n"}))
	if out := runGate(t, codexCall("PreToolUse", "t1", bash("c3", "git push"))); out != "{}" {
		t.Fatalf("the retry after the read should pass, got %s", out)
	}
	runGate(t, codexCall("Stop", "t1", map[string]any{"stop_hook_active": false, "last_assistant_message": "done"}))
	var deny, allow, graded map[string]any
	for _, rec := range auditLines(t, home) {
		switch {
		case rec["verdict"] == "deny":
			deny = rec
		case rec["verdict"] == "allow":
			allow = rec
		case rec["event"] == "outcome":
			graded = rec
		}
	}
	if deny == nil || deny["harness"] != "codex" || deny["model"] != "gpt-5.6-luna" || allow["satisfied_by"] != "skill-load" {
		t.Errorf("audit: deny %v, allow %v", deny, allow)
	}
	if graded == nil || graded["outcome"] != "same_retry" || graded["harness"] != "codex" {
		t.Errorf("want the block graded same_retry from the ledger, got %v", graded)
	}
	if st, err := os.Stat(filepath.Join(home, "sessions", "codex", "cx-1.jsonl")); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("the ledger is a 0600 file under $PAWS_HOME/sessions/codex: %v", err)
	}
}

// A patch matches a file rule through any file it touches.
func TestCodexPatchMatchesFileRules(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "mig.md"), "name: mig\nevent: pre-tool\nmatcher:\n  tool: Write|Edit\n  file_glob: \"migrations/**\"\naction:\n  type: require-skill\n  skill: db\n")
	patch := func(files ...string) map[string]any {
		body := "*** Begin Patch\n"
		for _, f := range files {
			body += "*** Update File: " + f + "\n@@\n+x\n"
		}
		return map[string]any{"tool_name": "apply_patch", "tool_use_id": "p1", "tool_input": map[string]any{"command": body + "*** End Patch"}}
	}
	if r := reasonOf(t, runGate(t, codexCall("PreToolUse", "t1", patch("src/a.go", "migrations/0001.sql")))); !strings.Contains(r, "This edit needs the \"db\" skill") || strings.HasSuffix(r, ".") {
		t.Errorf("a patch touching a migration is gated once, without a trailing period: %q", r)
	}
	if out := runGate(t, codexCall("PreToolUse", "t1", patch("src/a.go"))); out != "{}" {
		t.Errorf("a patch outside migrations passes: %s", out)
	}
	// A shell patch with a command chained after it: the command is still gated as Bash.
	writeRule(t, filepath.Join(home, "push.md"), "name: push\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: git push\naction:\n  type: require-skill\n  skill: git\n")
	chained := "apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: src/a.go\n@@\n+x\n*** End Patch\nEOF\ngit push --force"
	if r := reasonOf(t, runGate(t, codexCall("PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_use_id": "b1", "tool_input": map[string]any{"command": chained}}))); !strings.Contains(r, `"git" skill`) {
		t.Errorf("git push after a shell patch must still be gated, got %q", r)
	}
}

func TestCodexInitUninstallAndSkill(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	hooks := filepath.Join(repo, ".codex", "hooks.json")
	// Someone else's hook in the same file survives init and uninstall.
	writeFile(t, hooks, `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"notify-me"}]}]}}`)
	code, out := captureStdout(t, func() int { return Init([]string{"--agent", "codex"}) })
	if code != 0 || !strings.Contains(out, "/hooks") {
		t.Fatalf("init exit %d, want the trust note: %s", code, out)
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct {
				Command string
				Timeout int
				Async   bool
			}
		}
	}
	data, _ := os.ReadFile(hooks)
	json.Unmarshal(data, &doc)
	for _, ev := range []string{"PreToolUse", "PostToolUse", "SessionStart", "UserPromptSubmit", "Stop", "PostCompact"} {
		entries := doc.Hooks[ev]
		last := entries[len(entries)-1]
		if c := last.Hooks[0]; !strings.HasSuffix(c.Command, " gate --agent codex") || c.Timeout != 15 {
			t.Errorf("%s: %+v", ev, last)
		}
	}
	for _, ev := range []string{"PreToolUse", "PostToolUse", "PostCompact"} {
		if h := doc.Hooks[ev][0]; h.Hooks[0].Async {
			t.Errorf("%s must finish before the next call is gated: %+v", ev, h)
		}
	}
	if pre := doc.Hooks["PreToolUse"][0]; pre.Matcher != "*" {
		t.Errorf("PreToolUse runs on every tool: %+v", pre)
	}
	if _, again := captureStdout(t, func() int { return Init([]string{"--agent", "codex"}) }); !strings.Contains(again, "already registered") {
		t.Errorf("init twice changes nothing: %s", again)
	}
	if hasPawsHook(filepath.Join(home, ".claude", "settings.json")) || hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Error("init --agent codex must not touch Claude Code's settings")
	}
	if code, _ := captureStdout(t, func() int { return Skill([]string{"--agent", "codex"}) }); code != 0 {
		t.Fatal("skill --agent codex failed")
	}
	if _, err := os.Stat(filepath.Join(repo, ".agents", "skills", "paws", "SKILL.md")); err != nil {
		t.Errorf("the paws skill goes to the repo's .agents/skills: %v", err)
	}
	captureStdout(t, func() int { return Uninstall([]string{"--agent", "codex"}) })
	data, _ = os.ReadFile(hooks)
	if strings.Contains(string(data), "gate") || !strings.Contains(string(data), "notify-me") {
		t.Errorf("uninstall removes only paws's entries: %s", data)
	}
	if _, err := os.Stat(filepath.Join(repo, ".agents", "skills", "paws")); err == nil {
		t.Error("uninstall removes the paws skill too")
	}
	if code, _ := captureStdout(t, func() int { return Init([]string{"--agent", "codex", "--global"}) }); code != 0 || !hasPawsHook(filepath.Join(home, ".codex", "hooks.json")) {
		t.Error("init --agent codex --global writes ~/.codex/hooks.json")
	}
	t.Setenv("CODEX_HOME", filepath.Join(home, "cx"))
	if code, _ := captureStdout(t, func() int { return Init([]string{"--agent", "codex", "--global"}) }); code != 0 || !hasPawsHook(filepath.Join(home, "cx", "hooks.json")) {
		t.Error("--global follows $CODEX_HOME")
	}
}

// With the Codex hook installed, paws add copies each skill it installs into ~/.agents/skills too.
func TestAddCopiesSkillsForCodex(t *testing.T) {
	b, _ := catalogEnv(t)
	b.publish(t, writeTestCatalog(t))
	t.Setenv("CODEX_HOME", "")
	captureStdout(t, func() int { return Init([]string{"--agent", "codex", "--global"}) })
	if code, out := captureStdout(t, func() int { return Add([]string{"writing"}) }); code != 0 {
		t.Fatalf("add exit %d: %s", code, out)
	}
	codex := filepath.Join(os.Getenv("HOME"), ".agents", "skills", "humanizer")
	if data, _ := os.ReadFile(filepath.Join(codex, "SKILL.md")); string(data) != string(humanizerFiles["SKILL.md"].Data) {
		t.Errorf("humanizer copied for Codex: %q", data)
	}
	if st, err := os.Lstat(codex); err != nil || st.Mode()&os.ModeSymlink != 0 {
		t.Error("the copy is a real folder, not a symlink")
	}
	// The Codex hook goes before the pack does: the copy goes with the pack all the same.
	captureStdout(t, func() int { return Uninstall([]string{"--agent", "codex", "--global"}) })
	if code, _ := captureStdout(t, func() int { return Remove([]string{"writing"}) }); code != 0 {
		t.Fatal("remove failed")
	}
	if _, err := os.Stat(codex); err == nil {
		t.Error("remove takes the Codex copy too")
	}
}

func TestScanRefusesCodex(t *testing.T) {
	portalEnv(t, http.NotFoundHandler())
	code, errOut := capture(t, &os.Stderr, func() int { return Scan([]string{"--agent", "codex", "--dry-run"}) })
	if code != 2 || !strings.Contains(errOut, "can't read codex sessions") {
		t.Errorf("exit %d: %s", code, errOut)
	}
}

// A project hook from the unbound-paws on PATH is guarded like Claude Code's: a teammate without paws is told how
// to install it, and init and uninstall still know the entry as paws's.
func TestCodexProjectHookIsGuarded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	bin := filepath.Join(home, "bin")
	self, _ := os.Executable()
	os.MkdirAll(bin, 0o755)
	os.Symlink(self, filepath.Join(bin, "unbound-paws"))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	if !onPath(self) {
		t.Skip("the test binary isn't seen as the unbound-paws on PATH here")
	}
	captureStdout(t, func() int { return Init([]string{"--agent", "codex"}) })
	data, _ := os.ReadFile(filepath.Join(repo, ".codex", "hooks.json"))
	if !strings.Contains(string(data), "command -v unbound-paws") || !strings.Contains(string(data), "; unbound-paws gate --agent codex") {
		t.Errorf("want the guarded command: %s", data)
	}
	if !hasPawsHook(filepath.Join(repo, ".codex", "hooks.json")) {
		t.Error("a guarded Codex entry is still paws's")
	}
}

// A custom portal skill the user put only in Codex's skill folder is there: its rules are written, not skipped.
func TestPortalSkillInACodexFolderIsInstalled(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	codexSkill(t, "house-style")
	res, err := applyConfig(portalConfig{ConfigHash: "h1", Skills: []portalSkill{{Name: "house-style", Kind: "skill", Load: "full", Status: "on",
		Actions: []portalAction{{ID: "docs", Label: "Edit docs", Event: "pre-tool", Tool: "Write|Edit", FileGlob: "docs/**"}}}}}, t.TempDir())
	if err != nil || len(res.Applied) != 1 || res.Pending {
		t.Fatalf("want house-style applied, got %+v %v", res, err)
	}
	if rules, _ := filepath.Glob(filepath.Join(home, "portal", "house-style--*.md")); len(rules) != 1 {
		t.Errorf("want its rule file written, got %v", rules)
	}
}
