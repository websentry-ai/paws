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

func TestInstallPawsSkillBesideSettings(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, ".claude", "settings.json")
	installPawsSkill(settings)
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "paws", "SKILL.md"))
	if err != nil || !strings.Contains(string(data), "name: paws") {
		t.Fatalf("paws skill not installed: %v", err)
	}
	// A user's own paws skill is never replaced.
	own := filepath.Join(dir, "other", ".claude")
	if err := os.MkdirAll(filepath.Join(own, "skills", "paws"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(own, "skills", "paws", "SKILL.md"), []byte("mine"), 0o644)
	installPawsSkill(filepath.Join(own, "settings.json"))
	if got, _ := os.ReadFile(filepath.Join(own, "skills", "paws", "SKILL.md")); string(got) != "mine" {
		t.Error("init replaced a paws skill it did not install")
	}
}
