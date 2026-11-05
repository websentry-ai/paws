package cli

import (
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
	// Uninstall removes the unedited copy.
	removePawsSkill(settings)
	if _, err := os.Stat(filepath.Dir(skillMD)); err == nil {
		t.Error("uninstall should remove the paws skill it installed")
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
