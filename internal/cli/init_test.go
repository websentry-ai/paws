package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"/plain/path":     `'/plain/path'`,
		"/has'quote":      `'/has'\''quote'`,
		"/has $dollar":    `'/has $dollar'`,
		"/has `backtick`": "'/has `backtick`'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProjectRootWalksToGitRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := projectRoot(sub); got != root {
		t.Errorf("projectRoot(%q) = %q, want %q", sub, got, root)
	}
}

func TestProjectRootFallsBackToCwdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	if got := projectRoot(dir); got != dir {
		t.Errorf("no repo above cwd should fall back to cwd, got %q", got)
	}
}

func TestInstallSelfCopiesAndUpgrades(t *testing.T) {
	user := filepath.Join(t.TempDir(), "paws-home")
	src := filepath.Join(t.TempDir(), "paws")
	if err := os.WriteFile(src, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	dest, err := installSelf(src, user)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(user, "bin", "paws"); dest != want {
		t.Fatalf("dest = %q, want %q", dest, want)
	}
	if st, err := os.Stat(dest); err != nil || st.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed binary missing or not executable: %v %v", st, err)
	}

	// Upgrade: a new binary at the old (npx cache) path replaces the copy.
	if err := os.WriteFile(src, []byte("v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := installSelf(src, user); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "v2" {
		t.Errorf("upgrade did not replace the copy, got %q", got)
	}

	// Running init from the installed copy itself is a no-op.
	if again, err := installSelf(dest, user); err != nil || again != dest {
		t.Errorf("self-install from dest = %q, %v", again, err)
	}
	if left, _ := filepath.Glob(filepath.Join(user, "bin", ".paws-*")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestInstallSelfWithoutUserDirUsesSelf(t *testing.T) {
	if got, err := installSelf("/opt/paws", ""); err != nil || got != "/opt/paws" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestHookCommandUsesHomeVariable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := hookCommand(filepath.Join(home, ".paws", "bin", "paws")), `"$HOME/.paws/bin/paws" gate`; got != want {
		t.Errorf("binary under HOME: got %s, want %s", got, want)
	}
	if got, want := hookCommand("/opt/paws"), `'/opt/paws' gate`; got != want {
		t.Errorf("binary outside HOME: got %s, want %s", got, want)
	}
	if got, want := hookCommand(filepath.Join(home, "we$ird", "paws")), shellQuote(filepath.Join(home, "we$ird", "paws"))+" gate"; got != want {
		t.Errorf("path a double-quoted string would expand: got %s, want %s", got, want)
	}
}

func TestInitRegistersHookOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	if code := Init(nil); code != 0 {
		t.Fatalf("init exit %d", code)
	}
	if !hasPawsHook(filepath.Join(repo, ".claude", "settings.json")) {
		t.Fatal("init did not register the hook")
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude", "skills")); err == nil {
		t.Error("init must not install the paws skill; that's `paws skill`")
	}
	// Uninstall leaves no empty settings.json or .claude folder behind.
	if code := Uninstall(nil); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude")); err == nil {
		t.Error("uninstall left an empty .claude folder")
	}
	// A settings.json with other content stays.
	os.MkdirAll(filepath.Join(repo, ".claude"), 0o755)
	os.WriteFile(filepath.Join(repo, ".claude", "settings.json"), []byte(`{"env":{"A":"1"}}`), 0o644)
	Init(nil)
	Uninstall(nil)
	if data, _ := os.ReadFile(filepath.Join(repo, ".claude", "settings.json")); !strings.Contains(string(data), `"A"`) {
		t.Errorf("uninstall must keep other settings, got %q", data)
	}
}

func TestSkillInstallsAndReinstalls(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, ".claude", "settings.json")
	skillMD := filepath.Join(dir, ".claude", "skills", "paws", "SKILL.md")
	if code := Skill([]string{"--settings", settings}); code != 0 {
		t.Fatalf("paws skill exit %d", code)
	}
	if data, err := os.ReadFile(skillMD); err != nil || !strings.Contains(string(data), "name: paws") {
		t.Fatalf("paws skill not installed: %v", err)
	}
	// Reinstall replaces an edited paws copy.
	os.WriteFile(skillMD, []byte("edited"), 0o644)
	if code := Skill([]string{"--settings", settings}); code != 0 {
		t.Fatalf("reinstall exit %d", code)
	}
	if data, _ := os.ReadFile(skillMD); !strings.Contains(string(data), "name: paws") {
		t.Error("reinstall kept the edited copy")
	}
	// Uninstall removes the unedited copy, even though paws was never registered here, and the
	// folders it leaves empty.
	if code := Uninstall([]string{"--settings", settings}); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); err == nil {
		t.Error("uninstall should remove the paws skill and the empty .claude folder")
	}

	// A user's own paws skill (no paws marker) is never replaced.
	own := filepath.Join(dir, "other", ".claude")
	os.MkdirAll(filepath.Join(own, "skills", "paws"), 0o755)
	os.WriteFile(filepath.Join(own, "skills", "paws", "SKILL.md"), []byte("mine"), 0o644)
	if code := Skill([]string{"--settings", filepath.Join(own, "settings.json")}); code == 0 {
		t.Error("paws skill should refuse to replace a skill paws didn't install")
	}
	if got, _ := os.ReadFile(filepath.Join(own, "skills", "paws", "SKILL.md")); string(got) != "mine" {
		t.Error("paws skill replaced a skill it did not install")
	}

	// A settings file outside .claude has no skills dir beside it.
	if code := Skill([]string{"--settings", filepath.Join(dir, "elsewhere", "custom.json")}); code == 0 {
		t.Error("--settings outside .claude should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "elsewhere", "skills")); err == nil {
		t.Error("--settings outside .claude must not create a skills dir")
	}
}

func TestGlobalSettingsFollowClaudeConfigDir(t *testing.T) {
	home, cfg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	if got, code := settingsTarget([]string{"--global"}); code != -1 || got != filepath.Join(cfg, "settings.json") {
		t.Errorf("with CLAUDE_CONFIG_DIR: got %q (%d), want %q", got, code, filepath.Join(cfg, "settings.json"))
	}
	if got := pawsSkillDir(filepath.Join(cfg, "settings.json")); got != filepath.Join(cfg, "skills") {
		t.Errorf("paws skill beside CLAUDE_CONFIG_DIR settings: got %q", got)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if got, _ := settingsTarget([]string{"--global"}); got != filepath.Join(home, ".claude", "settings.json") {
		t.Errorf("without CLAUDE_CONFIG_DIR: got %q", got)
	}
}

// Project and global init write the same command, so a user with both runs the gate once.
func TestProjectAndGlobalHookCommandsMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	Init(nil)
	Init([]string{"--global"})
	cmd := func(path string) string {
		doc, _ := readSettings(path)
		hooks, _ := hooksBlock(doc)
		var pre []struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		}
		json.Unmarshal(hooks["PreToolUse"], &pre)
		return pre[0].Hooks[0].Command
	}
	if p, g := cmd(filepath.Join(repo, ".claude", "settings.json")), cmd(filepath.Join(home, ".claude", "settings.json")); p != g {
		t.Errorf("project %q and global %q differ; Claude Code would run the gate twice", p, g)
	}
}

// Run as the unbound-paws on PATH (installer, native npm), init names it and copies nothing.
func TestInitOnPathWritesTheCommandName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	os.MkdirAll(bin, 0o755)
	if err := os.Symlink(self, filepath.Join(bin, "unbound-paws")); err != nil {
		t.Skip(err)
	}
	t.Setenv("PATH", bin)
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	if code := Init(nil); code != 0 {
		t.Fatalf("init exit %d", code)
	}
	doc, _ := readSettings(filepath.Join(repo, ".claude", "settings.json"))
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if !containsCommand(pre, "unbound-paws gate") {
		t.Errorf("want unbound-paws gate, got %s", hooks["PreToolUse"])
	}
	if _, err := os.Stat(filepath.Join(home, ".paws", "bin")); err == nil {
		t.Error("init copied the binary although unbound-paws is on PATH")
	}
	if !isPawsCommand("unbound-paws gate") || isPawsCommand("unbound-paws explain") {
		t.Error("isPawsCommand should own exactly `unbound-paws gate`")
	}
}

// npx puts node_modules/.bin on PATH for one run only; init must not name a command from there.
func TestOnPathIgnoresNpxBin(t *testing.T) {
	self, _ := os.Executable()
	bin := filepath.Join(t.TempDir(), "_npx", "abc", "node_modules", ".bin")
	os.MkdirAll(bin, 0o755)
	if err := os.Symlink(self, filepath.Join(bin, "unbound-paws")); err != nil {
		t.Skip(err)
	}
	t.Setenv("PATH", bin)
	if onPath(self) {
		t.Error("onPath trusted an npx node_modules/.bin")
	}
}

// A project init from npx (not on PATH) reuses the global `unbound-paws gate`, so the two
// registrations stay one hook; a global init warns about a project line that differs.
func TestInitKeepsProjectAndGlobalOnOneLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", filepath.Join(home, ".paws"))
	cfg := filepath.Join(home, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	// unbound-paws is installed on PATH (some other build), but init runs off PATH (npx).
	pathBin := filepath.Join(home, "path-bin")
	os.MkdirAll(pathBin, 0o755)
	os.WriteFile(filepath.Join(pathBin, "unbound-paws"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", pathBin)
	doc := settingsDoc{}
	ensureHooks(doc, "unbound-paws gate")
	writeSettings(filepath.Join(cfg, "settings.json"), doc)
	repo := filepath.Join(home, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	if code := Init(nil); code != 0 {
		t.Fatalf("init exit %d", code)
	}
	if got := pawsCommandIn(filepath.Join(repo, ".claude", "settings.json")); got != "unbound-paws gate" {
		t.Errorf("project init should reuse the global line, got %q", got)
	}
	// A global init (off PATH here) next to a project line it can't match says so.
	t.Setenv("PATH", "")
	os.Remove(filepath.Join(cfg, "settings.json"))
	code, out := captureStdout(t, func() int { return Init([]string{"--global"}) })
	if code != 0 || !strings.Contains(out, "run `npx unbound-paws init` there too") {
		t.Errorf("global init should warn about the differing project line: %d %q", code, out)
	}
	// With unbound-paws gone from PATH, reusing the global line would register a hook that can't
	// start: keep the working fallback instead.
	doc = settingsDoc{}
	ensureHooks(doc, "unbound-paws gate")
	writeSettings(filepath.Join(cfg, "settings.json"), doc)
	Init(nil)
	if got := pawsCommandIn(filepath.Join(repo, ".claude", "settings.json")); got == "unbound-paws gate" {
		t.Errorf("project init reused unbound-paws gate with no unbound-paws on PATH")
	}
}
