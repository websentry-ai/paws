package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/websentry-ai/paws/internal/pack"
)

// storeEnv is a temp HOME with the test catalog published and a repo to work in. Claude Code's and Codex's folders
// are the HOME's own: ~/.claude, ~/.codex, ~/.agents.
func storeEnv(t *testing.T) (home, repo string) {
	t.Helper()
	b, _ := catalogEnv(t)
	b.publish(t, writeTestCatalog(t))
	t.Setenv("CODEX_HOME", "")
	home = os.Getenv("HOME")
	repo = filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	return home, repo
}

// mustRun runs one command and fails the test on a non-zero exit; it returns what the command printed.
func mustRun(t *testing.T, cmd func([]string) int, args ...string) string {
	t.Helper()
	code, out := captureStdout(t, func() int { return cmd(args) })
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	return out
}

// runAll runs one command and returns its exit code, stdout and stderr.
func runAll(t *testing.T, cmd func([]string) int, args ...string) (code int, out, errOut string) {
	t.Helper()
	code, errOut = capture(t, &os.Stderr, func() int {
		var c int
		c, out = captureStdout(t, func() int { return cmd(args) })
		return c
	})
	return code, out, errOut
}

// clashLine is what paws says when it leaves out skill because the user has their own at shown (~ paths).
func clashLine(skill, shown, what, again string) string {
	return `you already have a skill named "` + skill + `" (` + shown + `), so paws won't ` + what + `. Rename or remove yours, then ` + again + `.`
}

// sessionNotice is the systemMessage a Claude Code session start shows the user.
func sessionNotice(t *testing.T) string {
	t.Helper()
	var out struct{ SystemMessage string }
	json.Unmarshal([]byte(runGate(t, map[string]any{"hook_event_name": "SessionStart", "source": "startup", "cwd": t.TempDir()})), &out)
	return out.SystemMessage
}

func isThere(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// codexDeny is the reason Codex is denied a Bash command with, "" when it is let through.
func codexDeny(t *testing.T, id, cmd string) string {
	t.Helper()
	return reasonOf(t, runGate(t, codexCall("PreToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_use_id": id,
		"tool_input": map[string]any{"command": cmd}})))
}

// codexReads makes Codex read file in full, as the deny tells it to.
func codexReads(t *testing.T, id, file string) {
	t.Helper()
	read := "cat " + file
	data, _ := os.ReadFile(file)
	if r := codexDeny(t, id, read); r != "" {
		t.Fatalf("the read of %s was gated: %s", file, r)
	}
	runGate(t, codexCall("PostToolUse", "t1", map[string]any{"tool_name": "Bash", "tool_use_id": id,
		"tool_input": map[string]any{"command": read}, "tool_response": string(data)}))
}

// claudeLoads is a Claude Code transcript where the Skill tool loaded skill.
func claudeLoads(t *testing.T, skill string) string {
	t.Helper()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	p := filepath.Join(t.TempDir(), "s.jsonl")
	writeFile(t, p, `{"type":"user","isSidechain":false,"timestamp":"`+ts+`","message":{"role":"user","content":"open the PR"}}`+"\n"+
		`{"type":"assistant","isSidechain":false,"timestamp":"`+ts+`","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Skill","input":{"skill":"`+skill+`"}}]}}`+"\n"+
		`{"type":"user","isSidechain":false,"timestamp":"`+ts+`","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"Launching skill"}]},"toolUseResult":{"success":true,"commandName":"`+skill+`"}}`+"\n")
	return p
}

// claudeDeny is the reason Claude Code is denied a Bash command with in a session whose log is transcript.
func claudeDeny(t *testing.T, cmd, transcript string) string {
	t.Helper()
	return reasonOf(t, gateCall(t, "Bash", map[string]any{"command": cmd}, transcript))
}

// ownSkill puts a SKILL.md the user wrote (no paws marker) at dir/name, a day old.
func ownSkill(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name, "SKILL.md")
	writeFile(t, p, body)
	old := time.Now().Add(-24 * time.Hour)
	os.Chtimes(p, old, old)
	return p
}

const prCreate = "gh pr create --fill"

// With only Codex set up, paws add puts skills in Codex's folders and writes nothing for Claude Code; the deny names
// the file paws installed, and remove takes it out again.
func TestCodexOnlyAddWritesNothingForClaude(t *testing.T) {
	home, repo := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	out := mustRun(t, Add, "writing")
	if !strings.Contains(out, "humanizer    installed in ~/.agents/skills/humanizer") || strings.Contains(out, "copied to") {
		t.Errorf("add output:\n%s", out)
	}
	mustRun(t, Add, "--project", "testing/tdd")
	for _, p := range []string{filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md"), filepath.Join(repo, ".agents", "skills", "tdd", "SKILL.md")} {
		if !isThere(p) {
			t.Errorf("%s is missing", p)
		}
	}
	for _, p := range []string{filepath.Join(home, ".claude"), filepath.Join(repo, ".claude")} {
		if isThere(p) {
			t.Errorf("Claude Code isn't set up, yet paws wrote %s", p)
		}
	}
	md := filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	if r := codexDeny(t, "c1", prCreate); !strings.Contains(r, "Load or read it ("+md+"), then run the command again") {
		t.Errorf("deny %q, want it to name %s", r, md)
	}
	if out := mustRun(t, Remove, "writing"); !strings.Contains(out, "humanizer    removed") || isThere(md) {
		t.Errorf("remove takes the Codex copy:\n%s", out)
	}
}

// With only Claude Code set up, nothing is written for Codex.
func TestClaudeOnlyAddWritesNothingForCodex(t *testing.T) {
	home, repo := storeEnv(t)
	mustRun(t, Init, "--global")
	out := mustRun(t, Add, "writing")
	mustRun(t, Add, "--project", "testing/tdd")
	if !strings.Contains(out, "humanizer    installed in ~/.claude/skills/humanizer") || strings.Contains(out, "copied to") {
		t.Errorf("add output:\n%s", out)
	}
	for _, p := range []string{filepath.Join(home, ".agents"), filepath.Join(home, ".codex"), filepath.Join(repo, ".agents"), filepath.Join(repo, ".codex")} {
		if isThere(p) {
			t.Errorf("Codex isn't set up, yet paws wrote %s", p)
		}
	}
	if !isThere(filepath.Join(home, ".claude", "skills", "humanizer", "SKILL.md")) || !isThere(filepath.Join(repo, ".claude", "skills", "tdd", "SKILL.md")) {
		t.Error("the skills are in Claude Code's folders")
	}
}

// With both set up, each agent gets its own copy, and remove takes both.
func TestBothAgentsGetSkills(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	out := mustRun(t, Add, "writing")
	if !strings.Contains(out, "humanizer    installed in ~/.claude/skills/humanizer") || !strings.Contains(out, "skills       copied to ~/.agents/skills") {
		t.Errorf("add output:\n%s", out)
	}
	copies := []string{filepath.Join(home, ".claude", "skills", "humanizer"), filepath.Join(home, ".agents", "skills", "humanizer")}
	for _, d := range copies {
		if !isThere(filepath.Join(d, "SKILL.md")) {
			t.Errorf("%s is missing", d)
		}
	}
	mustRun(t, Remove, "writing")
	for _, d := range copies {
		if isThere(d) {
			t.Errorf("%s outlived its pack", d)
		}
	}
}

// Setting up Claude Code after a Codex-only setup copies what paws installed for Codex into Claude Code's folders,
// for global and project skills alike, whichever scope Claude Code's hook is in.
func TestClaudeAddedAfterCodexGetsTheSkills(t *testing.T) {
	home, repo := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mustRun(t, Add, "writing")
	mustRun(t, Add, "--project", "testing/tdd")
	out := mustRun(t, Init)
	for _, want := range []string{"copied 1 skill(s) paws installed into ~/.claude/skills: humanizer", "copied 1 skill(s) paws installed into .claude/skills: tdd"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output lacks %q:\n%s", want, out)
		}
	}
	md := filepath.Join(home, ".claude", "skills", "humanizer", "SKILL.md")
	if data, _ := os.ReadFile(md); string(data) != string(humanizerFiles["SKILL.md"].Data) {
		t.Errorf("Claude Code's copy: %q", data)
	}
	if !isThere(filepath.Join(repo, ".claude", "skills", "tdd", "SKILL.md")) {
		t.Error("the project skill is in the repo's .claude/skills")
	}
	// From now on Claude Code's folder leads, and remove still takes every copy.
	mustRun(t, Remove, "writing")
	for _, d := range []string{filepath.Dir(md), filepath.Join(home, ".agents", "skills", "humanizer")} {
		if isThere(d) {
			t.Errorf("%s outlived its pack", d)
		}
	}
}

// Taking Codex out of a two-agent setup leaves Claude Code's skills alone.
func TestUninstallingCodexKeepsClaudesSkills(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mustRun(t, Add, "writing")
	mustRun(t, Uninstall, "--agent", "codex", "--global")
	if !isThere(filepath.Join(home, ".claude", "skills", "humanizer", "SKILL.md")) {
		t.Error("Claude Code still runs paws: its copy stays")
	}
	out := mustRun(t, Add, "writing")
	if strings.Contains(out, "copied to") {
		t.Errorf("Codex no longer runs paws, nothing is copied for it:\n%s", out)
	}
}

// With only Codex set up, the portal config installs skills in Codex's folder only.
func TestCodexOnlyPortalSyncWritesNothingForClaude(t *testing.T) {
	_, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	home := os.Getenv("HOME")
	mustRun(t, Init, "--agent", "codex", "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	if out := mustRun(t, Sync); !strings.Contains(out, "Config: 1 skill (humanizer) applied.") {
		t.Fatalf("sync output:\n%s", out)
	}
	if data, _ := os.ReadFile(filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")); string(data) != "# humanizer\n" {
		t.Errorf("Codex's copy: %q", data)
	}
	if isThere(filepath.Join(home, ".claude")) {
		t.Error("Claude Code isn't set up, yet the sync wrote ~/.claude")
	}
}

// A skill of the user's own by the same name, in the folder paws installs into: paws installs neither the skill nor
// the rules that need it and says so, nothing gates, and once the user renames theirs a re-add installs paws's and
// the rule gates. Remove and uninstall leave the user's copy. Claude Code and Codex alike.
func TestAddLeavesOutASkillTheUserHas(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			home, _ := storeEnv(t)
			mustRun(t, Init, "--agent", agent, "--global")
			dir, shown := filepath.Join(home, ".claude", "skills"), "~/.claude/skills"
			if agent == "codex" {
				dir, shown = filepath.Join(home, ".agents", "skills"), "~/.agents/skills"
			}
			gated := func(id string) bool {
				if agent == "claude" {
					return claudeDeny(t, prCreate, "") != ""
				}
				runGate(t, codexCall("UserPromptSubmit", "t1", nil))
				return codexDeny(t, id, prCreate) != ""
			}
			md := ownSkill(t, dir, "humanizer", "my own humanizer\n")
			code, out, errOut := runAll(t, Add, "writing")
			want := "paws: " + clashLine("humanizer", shown+"/humanizer", "install it or turn on its 2 rules", "run `unbound-paws add writing` again")
			if code != 0 || !strings.Contains(errOut, want) || !strings.Contains(out, "Added writing: 0 rules") || strings.Contains(out, "installed in") {
				t.Errorf("add exit %d\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
			}
			if data, _ := os.ReadFile(md); string(data) != "my own humanizer\n" || isThere(filepath.Join(filepath.Dir(md), ".paws-skill.json")) {
				t.Errorf("the user's skill was changed: %q", data)
			}
			if gated("c1") {
				t.Error("no rule may gate on a skill paws didn't put there")
			}
			mustRun(t, Remove, "writing")
			mustRun(t, Uninstall, "--yes")
			if data, _ := os.ReadFile(md); string(data) != "my own humanizer\n" {
				t.Errorf("remove and uninstall must leave the user's skill: %q", data)
			}

			// The user renames theirs: the next add installs paws's, and the rule gates.
			mustRun(t, Init, "--agent", agent, "--global")
			mine := filepath.Join(dir, "my-humanizer")
			if err := os.Rename(filepath.Dir(md), mine); err != nil {
				t.Fatal(err)
			}
			if out := mustRun(t, Add, "writing"); !strings.Contains(out, "humanizer    installed in "+shown+"/humanizer") {
				t.Errorf("add output:\n%s", out)
			}
			if !gated("c2") {
				t.Error("with the user's copy renamed, the rule gates")
			}
			mustRun(t, Uninstall, "--yes")
			if data, _ := os.ReadFile(filepath.Join(mine, "SKILL.md")); string(data) != "my own humanizer\n" {
				t.Errorf("uninstall must leave the renamed copy: %q", data)
			}
		})
	}
}

// The user's own copy lives only in the other agent's folder: that agent would load it, so paws installs nothing for
// either agent, names the user's copy, and neither agent is gated.
func TestUsersSkillInTheOtherAgentsFolder(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mine := ownSkill(t, filepath.Join(home, ".agents", "skills"), "humanizer", "my own humanizer\n")
	_, out, errOut := runAll(t, Add, "writing")
	if !strings.Contains(errOut, clashLine("humanizer", "~/.agents/skills/humanizer", "install it or turn on its 2 rules", "run `unbound-paws add writing` again")) ||
		strings.Contains(out, "installed in") || isThere(filepath.Join(home, ".claude", "skills", "humanizer")) {
		t.Errorf("stdout:\n%s\nstderr:\n%s", out, errOut)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy was overwritten: %q", data)
	}
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	if r := codexDeny(t, "c1", prCreate); r != "" {
		t.Errorf("Codex: %q", r)
	}
	if r := claudeDeny(t, prCreate, ""); r != "" {
		t.Errorf("Claude Code: %q", r)
	}
}

// A pack with two skills where only one clashes: the other skill and its rule install, the clashing one's don't.
func TestAddInstallsThePartOfAPackThatDoesntClash(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--global")
	ownSkill(t, filepath.Join(home, ".claude", "skills"), "humanizer", "my own humanizer\n")
	_, out, errOut := runAll(t, Add, "craft")
	if !strings.Contains(errOut, clashLine("humanizer", "~/.claude/skills/humanizer", "install it or turn on its 1 rule", "run `unbound-paws add craft` again")) ||
		!strings.Contains(out, "Added craft: 1 rule") || !strings.Contains(out, "tdd          installed in ~/.claude/skills/tdd") {
		t.Errorf("stdout:\n%s\nstderr:\n%s", out, errOut)
	}
	if r := claudeDeny(t, prCreate, ""); r != "" {
		t.Errorf("humanizer's rule gated: %q", r)
	}
	if r := reasonOf(t, gateCall(t, "Write", map[string]any{"file_path": filepath.Join(home, "repo", "a.test.ts"), "content": "x"}, "")); !strings.Contains(r, `"tdd"`) {
		t.Errorf("tdd's rule gates: %q", r)
	}
	mustRun(t, Remove, "craft")
	if isThere(filepath.Join(home, ".claude", "skills", "tdd")) || !isThere(filepath.Join(home, ".claude", "skills", "humanizer", "SKILL.md")) {
		t.Error("remove takes paws's tdd and leaves the user's humanizer")
	}
}

// A project add with the user's own copy in the repo's skills folder, for each agent: a clash, named there.
func TestProjectAddLeavesOutTheRepoSkillTheUserHas(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			_, repo := storeEnv(t)
			mustRun(t, Init, "--agent", agent)
			dir, shown := filepath.Join(repo, ".claude", "skills"), ".claude/skills"
			if agent == "codex" {
				dir, shown = filepath.Join(repo, ".agents", "skills"), ".agents/skills"
			}
			ownSkill(t, dir, "humanizer", "my project humanizer\n")
			_, _, errOut := runAll(t, Add, "--project", "writing")
			if !strings.Contains(errOut, `you already have a skill named "humanizer" (`) || !strings.Contains(errOut, shown+"/humanizer") {
				t.Errorf("stderr:\n%s", errOut)
			}
			if isThere(filepath.Join(dir, "humanizer", ".paws-skill.json")) {
				t.Error("paws installed over the user's copy")
			}
			if agent == "claude" && claudeDeny(t, prCreate, "") != "" {
				t.Error("Claude Code is gated")
			}
			if agent == "codex" {
				runGate(t, codexCall("UserPromptSubmit", "t1", nil))
				if r := codexDeny(t, "c1", prCreate); r != "" {
					t.Errorf("Codex is gated: %q", r)
				}
			}
		})
	}
}

// A Codex-only setup with the user's own copy in $CODEX_HOME/skills, which Codex loads too: a clash, named there.
func TestUsersSkillInCodexHome(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mine := ownSkill(t, filepath.Join(home, ".codex", "skills"), "humanizer", "my own humanizer\n")
	_, _, errOut := runAll(t, Add, "writing")
	if !strings.Contains(errOut, `skill named "humanizer" (~/.codex/skills/humanizer)`) || isThere(filepath.Join(home, ".agents", "skills", "humanizer")) {
		t.Errorf("stderr:\n%s", errOut)
	}
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	if r := codexDeny(t, "c1", prCreate); r != "" {
		t.Errorf("deny %q", r)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy was changed: %q", data)
	}
}

// A copy paws installed and the user edited stays// A copy paws installed and the user edited stays as edited on a re-add, in the folder paws installs into and in a
// copy for the other agent, and the output says it was edited, not that it isn't paws's.
func TestReAddKeepsEditedCopies(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mustRun(t, Add, "writing")
	claude := filepath.Join(home, ".claude", "skills", "humanizer", "SKILL.md")
	codex := filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")
	writeFile(t, codex, "my tweaks\n")
	out := mustRun(t, Add, "writing")
	if !strings.Contains(out, "humanizer    already in ~/.agents/skills/humanizer, edited since paws installed it; left as is") {
		t.Errorf("add output:\n%s", out)
	}
	writeFile(t, claude, "my tweaks\n")
	out = mustRun(t, Add, "writing")
	if !strings.Contains(out, "humanizer    already in ~/.claude/skills/humanizer, edited since paws installed it; left as is") {
		t.Errorf("add output:\n%s", out)
	}
	// An edited paws copy is still paws's skill: its rules stay on.
	if r := claudeDeny(t, prCreate, ""); r == "" {
		t.Error("the rule gates with an edited paws copy")
	}
	mustRun(t, Remove, "writing")
	for _, p := range []string{claude, codex} {
		if data, _ := os.ReadFile(p); string(data) != "my tweaks\n" {
			t.Errorf("%s: the edit is gone: %q", p, data)
		}
	}
}

// An older copy paws installed and nobody edited is updated on add, in both agents' folders.
func TestAddUpdatesAnOlderPawsCopy(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	old := fstest.MapFS{"SKILL.md": {Data: []byte("# humanizer v0\n"), Mode: 0o644}}
	dirs := []string{filepath.Join(home, ".claude", "skills"), filepath.Join(home, ".agents", "skills")}
	for _, d := range dirs {
		if _, err := pack.ReinstallBundledSkill(old, d, "humanizer", "github.com/blader/humanizer@old"); err != nil {
			t.Fatal(err)
		}
	}
	mustRun(t, Add, "writing")
	for _, d := range dirs {
		if data, _ := os.ReadFile(filepath.Join(d, "humanizer", "SKILL.md")); string(data) != string(humanizerFiles["SKILL.md"].Data) {
			t.Errorf("%s not updated: %q", d, data)
		}
	}
}

// A project skill of the user's own doesn't stop a global install (paws's user copy is the one the agent loads). A user
// skill of theirs does stop a project one: the agent loads the user's skill over the project's. Neither is touched.
func TestUsersSkillInTheOtherScope(t *testing.T) {
	home, repo := storeEnv(t)
	mustRun(t, Init, "--global")
	project := ownSkill(t, filepath.Join(repo, ".claude", "skills"), "humanizer", "my project humanizer\n")
	mustRun(t, Add, "writing")
	if !isThere(filepath.Join(home, ".claude", "skills", "humanizer", ".paws-skill.json")) {
		t.Error("the global install goes to ~/.claude/skills")
	}
	mustRun(t, Remove, "writing")
	os.RemoveAll(filepath.Dir(project))

	user := ownSkill(t, filepath.Join(home, ".claude", "skills"), "humanizer", "my humanizer\n")
	_, _, errOut := runAll(t, Add, "--project", "writing")
	if !strings.Contains(errOut, clashLine("humanizer", "~/.claude/skills/humanizer", "install it or turn on its 2 rules", "run `unbound-paws add --project writing` again")) ||
		isThere(filepath.Join(repo, ".claude", "skills", "humanizer")) {
		t.Errorf("a project add with the user's own user skill installs nothing: %s", errOut)
	}
	if r := claudeDeny(t, prCreate, ""); r != "" {
		t.Errorf("deny %q", r)
	}
	mustRun(t, Remove, "--project", "writing")
	if data, _ := os.ReadFile(user); string(data) != "my humanizer\n" || isThere(filepath.Join(repo, ".claude", "skills", "humanizer")) {
		t.Errorf("remove takes paws's project copy and leaves the user's: %q", data)
	}
}

// Portal sync with the user's own copy in place: paws syncs neither the skill nor its rule, says why in the sync
// output and at the next session start, and keeps checking. Once the user renames theirs, the next check installs
// paws's and the rule gates. With Codex set up too, a copy of the user's in Codex's folder clashes the same way.
func TestSyncLeavesOutASkillTheUserHas(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mine := ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "my own humanizer\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	note := clashLine("humanizer", "~/.claude/skills/humanizer", "sync it", "sync again")
	out := mustRun(t, Sync)
	if !strings.Contains(out, "  skipped humanizer: "+note) || strings.Contains(out, "left as is") {
		t.Errorf("sync output:\n%s", out)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" || isThere(filepath.Join(userHome, ".agents", "skills", "humanizer")) {
		t.Errorf("the user's copy was overwritten, or paws's installed beside it: %q", data)
	}
	if len(portalFiles(t, home)) != 0 || deny(t, prCreate) != "" {
		t.Error("no rule may be written for a skill paws didn't sync")
	}
	// Applied as the user has it, so the portal sees this laptop synced; the clash itself is re-checked each round.
	if st := loadPortalState(); st.AppliedHash != "h1" || len(st.Clashed) != 1 {
		t.Errorf("state %+v", st)
	}
	if n := sessionNotice(t); n != "paws: "+note {
		t.Errorf("session start notice %q", n)
	}

	// The user renames theirs: the next check of the same config installs paws's, and the notice goes.
	os.Rename(filepath.Dir(mine), filepath.Join(userHome, ".claude", "skills", "my-humanizer"))
	out = mustRun(t, Sync)
	if !strings.Contains(out, "Config: 1 skill (humanizer) applied.") || !isThere(filepath.Join(userHome, ".claude", "skills", "humanizer", ".paws-skill.json")) {
		t.Errorf("sync output:\n%s", out)
	}
	if len(portalFiles(t, home)) != 1 || deny(t, prCreate) == "" {
		t.Error("the rule is written and gates")
	}
	if n := sessionNotice(t); n != "" {
		t.Errorf("session start notice %q after the clash is gone", n)
	}

	// A copy of the user's in Codex's folder appears later: the next check sees it (drift), and leaves the skill out.
	codexMine := ownSkill(t, filepath.Join(userHome, ".agents", "skills"), "humanizer-x", "x\n")
	os.RemoveAll(filepath.Join(userHome, ".agents", "skills", "humanizer"))
	os.Rename(filepath.Dir(codexMine), filepath.Join(userHome, ".agents", "skills", "humanizer"))
	out = mustRun(t, Sync)
	if !strings.Contains(out, "  skipped humanizer: "+clashLine("humanizer", "~/.agents/skills/humanizer", "sync it", "sync again")) {
		t.Errorf("sync output:\n%s", out)
	}
	if len(portalFiles(t, home)) != 0 || deny(t, prCreate) != "" {
		t.Error("the rule is gone again")
	}

	// The user removes that copy too: the same config's next check puts paws's copy and the rule back.
	os.RemoveAll(filepath.Join(userHome, ".agents", "skills", "humanizer"))
	out = mustRun(t, Sync)
	if !strings.Contains(out, "Config: 1 skill (humanizer) applied.") || len(portalFiles(t, home)) != 1 || deny(t, prCreate) == "" {
		t.Errorf("the rule is back once the clash is gone; sync output:\n%s", out)
	}
}

// A state file an older paws wrote, which kept the rules on for the user's own copy (no paws copy anywhere): the next
// check re-applies once, and leaves the skill out.
func TestSyncFindsAClashWithAnOlderStateFile(t *testing.T) {
	home, srv := syncEnv(t)
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	mustRun(t, Sync)
	os.RemoveAll(filepath.Join(userHome, ".claude", "skills", "humanizer"))
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	st := loadPortalState()
	st.Clashes, st.Clashed, st.Installed, st.ClashChecked = nil, nil, nil, false
	savePortalState(st)
	if out := mustRun(t, Sync); !strings.Contains(out, "skipped humanizer: you already have a skill named") || len(portalFiles(t, home)) != 0 {
		t.Errorf("sync output:\n%s", out)
	}
}

// Codex hooked in a repo where a sync finds nothing to change is still noted: a copy of the user's in Codex's folder
// that appears later is a clash for a sync run elsewhere.
func TestPortalNotesAnAgentOnAnUnchangedSync(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	mustRun(t, Sync)
	repo := filepath.Join(userHome, "repo-a")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	mustRun(t, Init, "--agent", "codex")
	mustRun(t, Sync) // same hash, nothing to apply
	t.Chdir(userHome)
	os.RemoveAll(filepath.Join(userHome, ".agents", "skills", "humanizer")) // init copied paws's there
	ownSkill(t, filepath.Join(userHome, ".agents", "skills"), "humanizer", "mine\n")
	if out := mustRun(t, Sync); !strings.Contains(out, `skipped humanizer: you already have a skill named "humanizer" (~/.agents/skills/humanizer)`) || len(portalFiles(t, home)) != 0 {
		t.Errorf("sync output:\n%s", out)
	}
}

// The user replaces paws's copy with their own in place: the same config's next check leaves the skill out.
func TestSyncFindsAClashThatReplacedPawsCopy(t *testing.T) {
	home, srv := syncEnv(t)
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	mustRun(t, Sync)
	os.RemoveAll(filepath.Join(userHome, ".claude", "skills", "humanizer"))
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	if out := mustRun(t, Sync); !strings.Contains(out, "skipped humanizer: you already have a skill named") || len(portalFiles(t, home)) != 0 {
		t.Errorf("sync output:\n%s", out)
	}
}

// Codex hooked in one repo only: a sync there counts it, and so does every later sync, wherever it runs, since the
// portal's rules are global. The user's own ~/.agents copy is a clash in both places.
func TestPortalClashesRememberAgentsHookedInARepo(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	repo := filepath.Join(userHome, "repo-a")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	mustRun(t, Init, "--agent", "codex")
	ownSkill(t, filepath.Join(userHome, ".agents", "skills"), "humanizer", "mine\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	if out := mustRun(t, Sync); !strings.Contains(out, `skipped humanizer: you already have a skill named "humanizer" (~/.agents/skills/humanizer)`) || len(portalFiles(t, home)) != 0 {
		t.Errorf("sync in the repo:\n%s", out)
	}
	t.Chdir(userHome)
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	if out := mustRun(t, Sync); !strings.Contains(out, "skipped humanizer") || len(portalFiles(t, home)) != 0 {
		t.Errorf("sync elsewhere:\n%s", out)
	}
}

// Portal sync keeps an edited paws copy, says so, and updates an unedited older one in both agents' folders.
func TestSyncKeepsEditedAndUpdatesOlderCopies(t *testing.T) {
	_, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer v1\n")))
	mustRun(t, Sync)
	claude := filepath.Join(userHome, ".claude", "skills", "humanizer", "SKILL.md")
	codex := filepath.Join(userHome, ".agents", "skills", "humanizer", "SKILL.md")

	v2 := humanizerSkill(fakeUpstream(t, "# humanizer v2\n"))
	v2["upstream"].(map[string]any)["ref"] = strings.Repeat("b", 40)
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", v2)
	mustRun(t, Sync)
	for _, p := range []string{claude, codex} {
		if data, _ := os.ReadFile(p); string(data) != "# humanizer v2\n" {
			t.Errorf("%s not updated: %q", p, data)
		}
	}

	writeFile(t, claude, "my tweaks\n")
	v3 := humanizerSkill(fakeUpstream(t, "# humanizer v3\n"))
	v3["upstream"].(map[string]any)["ref"] = strings.Repeat("c", 40)
	srv.configHash = "h3"
	srv.config = configJSON(t, "h3", v3)
	out := mustRun(t, Sync)
	if !strings.Contains(out, "humanizer    already in ~/.claude/skills/humanizer, edited since paws installed it; left as is") {
		t.Errorf("sync output:\n%s", out)
	}
	if data, _ := os.ReadFile(claude); string(data) != "my tweaks\n" {
		t.Errorf("the edit is gone: %q", data)
	}
	if deny(t, prCreate) == "" {
		t.Error("an edited paws copy still syncs: its rule gates")
	}
	// As with paws add, the edit isn't copied on into Codex's folder as paws's own.
	if data, _ := os.ReadFile(codex); string(data) != "# humanizer v2\n" {
		t.Errorf("Codex's copy: %q", data)
	}
}

// claudeRead is a Claude Code transcript where the Read tool read file in full.
func claudeRead(t *testing.T, file string) string {
	t.Helper()
	in, _ := json.Marshal(map[string]string{"file_path": file})
	p := filepath.Join(t.TempDir(), "s.jsonl")
	writeFile(t, p, `{"type":"user","isSidechain":false,"message":{"role":"user","content":"open the PR"}}`+"\n"+
		`{"type":"assistant","isSidechain":false,"message":{"content":[{"type":"tool_use","id":"toolu_r","name":"Read","input":`+string(in)+`}]}}`+"\n"+
		`{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_r","content":"x"}]},"toolUseResult":{"type":"text"}}`+"\n")
	return p
}

// Codex runs paws globally and Claude Code only in another repo: a global add from here installs for Codex only.
// Claude Code, in that repo, is told to read Codex's copy, and the read lets the call through.
func TestClaudeElsewhereReadsTheCodexCopy(t *testing.T) {
	home, repo := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	other := filepath.Join(home, "other")
	os.MkdirAll(filepath.Join(other, ".git"), 0o755)
	t.Chdir(other)
	mustRun(t, Init)
	t.Chdir(repo)
	mustRun(t, Add, "writing")
	md := filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")
	if isThere(filepath.Join(home, ".claude", "skills", "humanizer")) || !isThere(md) {
		t.Fatal("Claude Code isn't set up here: the global add installs for Codex only")
	}
	t.Chdir(other)
	if r := claudeDeny(t, prCreate, ""); r != `paws (rule humanizer-pr): this needs the "humanizer" skill. Load or read it (`+md+`), then run the command again.` {
		t.Errorf("deny %q, want it to name %s", r, md)
	}
	if r := claudeDeny(t, prCreate, claudeRead(t, md)); r != "" {
		t.Errorf("reading Codex's copy lets the call through, got %q", r)
	}
}

// Codex set up, with the user's own copy in Codex's folder: add installs nothing. Claude Code set up next is not
// gated either, and a re-add still leaves the skill out while Codex runs paws.
func TestClaudeAfterCodexWithTheUsersCopy(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mine := ownSkill(t, filepath.Join(home, ".agents", "skills"), "humanizer", "my own humanizer\n")
	runAll(t, Add, "writing")
	mustRun(t, Init, "--global")
	if r := claudeDeny(t, prCreate, ""); r != "" {
		t.Errorf("deny %q", r)
	}
	if _, _, errOut := runAll(t, Add, "writing"); !strings.Contains(errOut, "~/.agents/skills/humanizer") || isThere(filepath.Join(home, ".claude", "skills", "humanizer")) {
		t.Errorf("stderr:\n%s", errOut)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy: %q", data)
	}
}

// The folder paws installs into moves to Claude Code's, where the user has their own copy: init says it left the
// user's copy alone, the re-add leaves the skill out and takes back the copy paws made for Codex, and the user's stays.
func TestReAddAfterTheStoreMovesToTheUsersCopy(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mustRun(t, Add, "writing")
	mine := ownSkill(t, filepath.Join(home, ".claude", "skills"), "humanizer", "my own humanizer\n")
	if _, _, errOut := runAll(t, Init, "--global"); !strings.Contains(errOut, `you already have a skill named "humanizer" (~/.claude/skills/humanizer), so paws didn't copy its own there`) {
		t.Errorf("init stderr:\n%s", errOut)
	}
	if _, _, errOut := runAll(t, Add, "writing"); !strings.Contains(errOut, clashLine("humanizer", "~/.claude/skills/humanizer", "install it or turn on its 2 rules", "run `unbound-paws add writing` again")) {
		t.Errorf("add stderr:\n%s", errOut)
	}
	if isThere(filepath.Join(home, ".agents", "skills", "humanizer")) {
		t.Error("the copy paws made for Codex outlived the skill's last rule")
	}
	mustRun(t, Remove, "writing")
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy: %q", data)
	}
}

// With the user's own copy in both agents' folders, sync names both and writes no rule.
func TestClashesAreNamedInEveryFolder(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	for _, d := range []string{filepath.Join(userHome, ".claude", "skills"), filepath.Join(userHome, ".agents", "skills")} {
		ownSkill(t, d, "humanizer", "my own humanizer\n")
	}
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	out := mustRun(t, Sync)
	if want := clashLine("humanizer", "~/.claude/skills/humanizer, ~/.agents/skills/humanizer", "sync it", "sync again"); !strings.Contains(out, want) {
		t.Errorf("sync output lacks %q:\n%s", want, out)
	}
	if len(portalFiles(t, home)) != 0 {
		t.Error("no rule is written")
	}
}

// A read of a skill in Codex's folder stops counting// A read of a skill in Codex's folder stops counting for Claude Code once that SKILL.md changes, as a read of one in
// Claude Code's folders does.
func TestClaudeReadOfACodexCopyExpiresWhenItChanges(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mustRun(t, Add, "writing")
	mine := filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")
	read := claudeRead(t, mine)
	old := time.Now().Add(-time.Hour)
	ts := old.UTC().Format(time.RFC3339Nano)
	data, _ := os.ReadFile(read)
	os.WriteFile(read, []byte(strings.ReplaceAll(string(data), `"isSidechain":false,`, `"isSidechain":false,"timestamp":"`+ts+`",`)), 0o600)
	os.Chtimes(mine, old.Add(-time.Hour), old.Add(-time.Hour))
	if r := claudeDeny(t, prCreate, read); r != "" {
		t.Errorf("a read after the last change loads it, got %q", r)
	}
	os.Chtimes(mine, time.Now(), time.Now())
	if r := claudeDeny(t, prCreate, read); r == "" {
		t.Error("the SKILL.md changed after the read: the call needs a new read")
	}
}

// A hook file given with --settings gets the skills in the folder beside it, not the default ones.
func TestClaudeAddedWithSettingsGetsTheSkillsBesideIt(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mustRun(t, Add, "writing")
	settings := filepath.Join(t.TempDir(), ".claude", "settings.json")
	mustRun(t, Init, "--agent", "claude", "--settings", settings)
	if !isThere(filepath.Join(filepath.Dir(settings), "skills", "humanizer", "SKILL.md")) {
		t.Error("humanizer belongs beside the given settings file")
	}
	if isThere(filepath.Join(home, ".claude", "skills", "humanizer")) {
		t.Error("nothing is written to Claude Code's default folder")
	}
}

// The portal hears about a clash: the sync after the skill is left out reports the skill and the user's copy, and the
// sync after the user removes their copy reports none, so the Skills page clears it.
func TestSyncReportsClashesToThePortal(t *testing.T) {
	_, srv := syncEnv(t)
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	lastClashes := func() string {
		t.Helper()
		var b struct {
			Clashes *json.RawMessage `json:"clashes"`
		}
		json.Unmarshal([]byte(srv.bodies[len(srv.bodies)-1]), &b)
		if b.Clashes == nil {
			return "<absent>"
		}
		return string(*b.Clashes)
	}
	mustRun(t, Sync)
	if got := lastClashes(); got != `[{"skill":"humanizer","paths":["~/.claude/skills/humanizer"]}]` {
		t.Errorf("clashes after the clash: %s", got)
	}
	os.RemoveAll(filepath.Join(userHome, ".claude", "skills", "humanizer"))
	mustRun(t, Sync)
	if got := lastClashes(); got != `[]` {
		t.Errorf("clashes after the user's copy went: %s", got)
	}
}

// Before paws has checked a config for clashes (no state yet), the sync sends no clashes field: the portal keeps what
// it had rather than clearing it.
func TestSyncSendsNoClashesBeforeAnyCheck(t *testing.T) {
	if got := clashReports(portalState{}); got != nil {
		t.Errorf("clashReports(empty state) = %v, want nil", *got)
	}
}

// The user removes one of two copies that keep a skill out: the same config's next sync names only the one left, in
// its output and in what it tells the portal.
func TestSyncRenamesTheCopiesBehindAClash(t *testing.T) {
	_, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	mustRun(t, Init, "--global", "--agent", "codex")
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	ownSkill(t, filepath.Join(userHome, ".agents", "skills"), "humanizer", "mine\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	if out := mustRun(t, Sync); !strings.Contains(out, "~/.claude/skills/humanizer, ~/.agents/skills/humanizer") {
		t.Errorf("sync with both copies:\n%s", out)
	}
	os.RemoveAll(filepath.Join(userHome, ".agents", "skills", "humanizer"))
	out := mustRun(t, Sync)
	if !strings.Contains(out, `skipped humanizer: you already have a skill named "humanizer" (~/.claude/skills/humanizer), so`) {
		t.Errorf("sync after one copy went:\n%s", out)
	}
	var b struct {
		Clashes []clashReport `json:"clashes"`
	}
	json.Unmarshal([]byte(srv.bodies[len(srv.bodies)-1]), &b)
	if len(b.Clashes) != 1 || strings.Join(b.Clashes[0].Paths, ",") != "~/.claude/skills/humanizer" {
		t.Errorf("clashes sent: %+v", b.Clashes)
	}
}

// A same-hash sync finds a new clash but can't remove the skill's rule file: it records nothing, so the next sync,
// once it can write, retries and takes the rule away.
func TestSyncRetriesAClashItCouldNotApply(t *testing.T) {
	home, srv := syncEnv(t)
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	mustRun(t, Sync)
	if len(portalFiles(t, home)) == 0 {
		t.Fatal("no portal rule after the first sync")
	}
	os.RemoveAll(filepath.Join(userHome, ".claude", "skills", "humanizer"))
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	// A non-empty folder in place of the rule file: paws can't remove it, and can still save its state.
	rule := filepath.Join(home, "portal", portalFiles(t, home)[0])
	os.Remove(rule)
	os.MkdirAll(filepath.Join(rule, "x"), 0o755)
	captureStdout(t, func() int { return Sync(nil) })
	if st := loadPortalState(); slices.Contains(st.Clashed, "humanizer") {
		t.Errorf("a failed apply recorded the clash: %+v", st)
	}
	os.RemoveAll(rule)
	os.WriteFile(rule, []byte("left over\n"), 0o644)
	if out := mustRun(t, Sync); !strings.Contains(out, "skipped humanizer") || len(portalFiles(t, home)) != 0 {
		t.Errorf("the retry left the rule:\n%s\n%v", out, portalFiles(t, home))
	}
}

// The same failed apply on a state file an older paws wrote (no clash check yet): the retry still comes, and takes
// the rule away.
func TestSyncRetriesAFailedFirstClashCheck(t *testing.T) {
	home, srv := syncEnv(t)
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--global")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	mustRun(t, Sync)
	os.RemoveAll(filepath.Join(userHome, ".claude", "skills", "humanizer"))
	ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "mine\n")
	st := loadPortalState()
	st.Clashes, st.Clashed, st.ClashDirs, st.Installed, st.ClashChecked = nil, nil, nil, nil, false
	savePortalState(st)
	rule := filepath.Join(home, "portal", portalFiles(t, home)[0])
	os.Remove(rule)
	os.MkdirAll(filepath.Join(rule, "x"), 0o755)
	captureStdout(t, func() int { return Sync(nil) })
	os.RemoveAll(rule)
	os.WriteFile(rule, []byte("left over\n"), 0o644)
	if out := mustRun(t, Sync); !strings.Contains(out, "skipped humanizer") || len(portalFiles(t, home)) != 0 {
		t.Errorf("the retry left the rule:\n%s\n%v", out, portalFiles(t, home))
	}
}
