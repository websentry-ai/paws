package cli

import (
	"os"
	"path/filepath"
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
	if r := codexDeny(t, "c1", prCreate); !strings.Contains(r, "Read "+md+", then run the command again.") {
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

// A skill of the user's own by the same name, in the folder paws installs into, is kept and named; the rule still
// gates, and loading the user's copy lets the call through. Claude Code and Codex alike.
func TestAddKeepsTheUsersOwnSkill(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			home, _ := storeEnv(t)
			mustRun(t, Init, "--agent", agent, "--global")
			dir, shown := filepath.Join(home, ".claude", "skills"), "~/.claude/skills"
			if agent == "codex" {
				dir, shown = filepath.Join(home, ".agents", "skills"), "~/.agents/skills"
			}
			md := ownSkill(t, dir, "humanizer", "my own humanizer\n")
			out := mustRun(t, Add, "writing")
			if !strings.Contains(out, "humanizer    already in "+shown+"/humanizer, not from paws; left as is") || strings.Contains(out, "installed in") {
				t.Errorf("add output:\n%s", out)
			}
			if data, _ := os.ReadFile(md); string(data) != "my own humanizer\n" || isThere(filepath.Join(filepath.Dir(md), ".paws-skill.json")) {
				t.Errorf("the user's skill was changed: %q", data)
			}
			if agent == "claude" {
				if r := claudeDeny(t, prCreate, ""); r == "" {
					t.Error("the rule gates with the user's copy in place")
				}
				if r := claudeDeny(t, prCreate, claudeLoads(t, "humanizer")); r != "" {
					t.Errorf("loading the user's copy lets the call through, got %q", r)
				}
			} else {
				runGate(t, codexCall("UserPromptSubmit", "t1", nil))
				if r := codexDeny(t, "c1", prCreate); !strings.Contains(r, "Read "+md+",") {
					t.Errorf("deny %q, want the user's %s", r, md)
				}
				codexReads(t, "c2", md)
				if r := codexDeny(t, "c3", prCreate); r != "" {
					t.Errorf("reading the user's copy lets the call through, got %q", r)
				}
			}
			mustRun(t, Remove, "writing")
			mustRun(t, Uninstall, "--yes")
			if data, _ := os.ReadFile(md); string(data) != "my own humanizer\n" {
				t.Errorf("remove and uninstall must leave the user's skill: %q", data)
			}
		})
	}
}

// The user's own copy lives only in the other agent's folder: paws installs its own copy where it installs, names
// the user's copy as kept, and Codex is told to read the user's file, which is the one it resolves first.
func TestUsersSkillInTheOtherAgentsFolder(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mine := ownSkill(t, filepath.Join(home, ".agents", "skills"), "humanizer", "my own humanizer\n")
	out := mustRun(t, Add, "writing")
	if !strings.Contains(out, "humanizer    installed in ~/.claude/skills/humanizer") ||
		!strings.Contains(out, "humanizer    already in ~/.agents/skills/humanizer, not from paws; left as is") || strings.Contains(out, "copied to") {
		t.Errorf("add output:\n%s", out)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy was overwritten: %q", data)
	}
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	if r := codexDeny(t, "c1", prCreate); !strings.Contains(r, "Read "+mine+",") {
		t.Errorf("deny %q, want the user's %s (a file that exists)", r, mine)
	}
	mustRun(t, Remove, "writing")
	if isThere(filepath.Join(home, ".claude", "skills", "humanizer")) || !isThere(mine) {
		t.Error("remove takes paws's copy and leaves the user's")
	}
}

// A Codex-only setup with the user's own copy in $CODEX_HOME/skills: paws still installs into ~/.agents/skills, and
// the deny names paws's copy there, the folder Codex's gate resolves first.
func TestUsersSkillInCodexHome(t *testing.T) {
	home, _ := storeEnv(t)
	mustRun(t, Init, "--agent", "codex", "--global")
	mine := ownSkill(t, filepath.Join(home, ".codex", "skills"), "humanizer", "my own humanizer\n")
	mustRun(t, Add, "writing")
	installed := filepath.Join(home, ".agents", "skills", "humanizer", "SKILL.md")
	runGate(t, codexCall("UserPromptSubmit", "t1", nil))
	if r := codexDeny(t, "c1", prCreate); !strings.Contains(r, "Read "+installed+",") {
		t.Errorf("deny %q, want %s", r, installed)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy was changed: %q", data)
	}
}

// A copy paws installed and the user edited stays as edited on a re-add, in the folder paws installs into and in a
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

// A project skill of the user's own doesn't stop a global install, and a user skill of theirs doesn't stop a project
// one; neither is touched, and neither goes with the pack.
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
	mustRun(t, Add, "--project", "writing")
	if !isThere(filepath.Join(repo, ".claude", "skills", "humanizer", ".paws-skill.json")) {
		t.Error("the project install goes to the repo's .claude/skills")
	}
	// Claude Code ranks the user's skill above the project's: loading it by name lets the call through.
	if r := claudeDeny(t, prCreate, claudeLoads(t, "humanizer")); r != "" {
		t.Errorf("deny %q", r)
	}
	mustRun(t, Remove, "--project", "writing")
	if data, _ := os.ReadFile(user); string(data) != "my humanizer\n" || isThere(filepath.Join(repo, ".claude", "skills", "humanizer")) {
		t.Errorf("remove takes paws's project copy and leaves the user's: %q", data)
	}
}

// Portal sync with the user's own copy in place: kept, said in one line as paws add says it, the rule still written,
// and the call gated. With Codex set up too, its copy of the user's skill is named the same way.
func TestSyncKeepsTheUsersOwnSkillAndSaysSo(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("CODEX_HOME", "")
	userHome := os.Getenv("HOME")
	mustRun(t, Init, "--agent", "claude,codex", "--global")
	mine := ownSkill(t, filepath.Join(userHome, ".claude", "skills"), "humanizer", "my own humanizer\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	out := mustRun(t, Sync)
	if !strings.Contains(out, "humanizer    already in ~/.claude/skills/humanizer, not from paws; left as is") {
		t.Errorf("sync output:\n%s", out)
	}
	if data, _ := os.ReadFile(mine); string(data) != "my own humanizer\n" {
		t.Errorf("the user's copy was overwritten: %q", data)
	}
	if len(portalFiles(t, home)) != 1 || deny(t, prCreate) == "" {
		t.Error("the rule is written and gates")
	}
	if r := claudeDeny(t, prCreate, claudeLoads(t, "humanizer")); r != "" {
		t.Errorf("loading the user's copy lets the call through, got %q", r)
	}

	// The user's copy is Codex's only; a new config installs paws's for Claude Code and names the Codex one.
	os.RemoveAll(filepath.Dir(mine))
	codexMine := ownSkill(t, filepath.Join(userHome, ".agents", "skills"), "humanizer", "my own humanizer\n")
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", humanizerSkill(fakeUpstream(t, "# humanizer\n")))
	out = mustRun(t, Sync)
	if !strings.Contains(out, "humanizer    already in ~/.agents/skills/humanizer, not from paws; left as is") {
		t.Errorf("sync output:\n%s", out)
	}
	if data, _ := os.ReadFile(codexMine); string(data) != "my own humanizer\n" || !isThere(filepath.Join(userHome, ".claude", "skills", "humanizer", ".paws-skill.json")) {
		t.Errorf("Codex keeps the user's copy, Claude Code gets paws's: %q", data)
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
	// As with paws add, the edit isn't copied on into Codex's folder as paws's own.
	if data, _ := os.ReadFile(codex); string(data) != "# humanizer v2\n" {
		t.Errorf("Codex's copy: %q", data)
	}
}
