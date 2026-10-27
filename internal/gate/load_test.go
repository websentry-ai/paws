package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPawsDirsStopsAtRepoRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, ".paws")) // above the repo — must be ignored
	repo := filepath.Join(root, "repo")
	mustMkdir(t, filepath.Join(repo, ".git"))
	mustMkdir(t, filepath.Join(repo, ".paws"))
	sub := filepath.Join(repo, "sub")
	mustMkdir(t, filepath.Join(sub, ".paws"))

	dirs := pawsDirs(sub)
	want := []string{filepath.Join(sub, ".paws"), filepath.Join(repo, ".paws")}
	if len(dirs) != len(want) {
		t.Fatalf("got %v, want %v", dirs, want)
	}
	for i := range want {
		if dirs[i] != want[i] {
			t.Errorf("dirs[%d] = %q, want %q (order matters: nearest-first)", i, dirs[i], want[i])
		}
	}
}

// Outside a repo (and outside $HOME), the walk must not ascend past cwd — no trusting shared parents like /tmp.
func TestPawsDirsOutsideRepoStopsAtCwd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	shared := t.TempDir()
	mustMkdir(t, filepath.Join(shared, ".paws"))
	work := filepath.Join(shared, "work")
	mustMkdir(t, filepath.Join(work, ".paws"))

	dirs := pawsDirs(work)
	if len(dirs) != 1 || dirs[0] != filepath.Join(work, ".paws") {
		t.Errorf("only cwd's own .paws should be trusted here, got %v", dirs)
	}
}

func TestPawsDirsIncludesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mustMkdir(t, filepath.Join(home, ".paws"))

	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, ".git"))

	dirs := pawsDirs(repo)
	if last := dirs[len(dirs)-1]; last != filepath.Join(home, ".paws") {
		t.Errorf("expected ~/.paws last in %v", dirs)
	}
}

func TestLoadDirSkipsBadRules(t *testing.T) {
	dir := t.TempDir()
	pawsDir := filepath.Join(dir, ".paws")
	mustMkdir(t, pawsDir)
	mustWrite(t, filepath.Join(pawsDir, "good.md"), "---\nname: good\nevent: pre-tool\naction:\n  type: block\n---\nok")
	mustWrite(t, filepath.Join(pawsDir, "bad.md"), "no frontmatter here")
	mustWrite(t, filepath.Join(pawsDir, "notes.txt"), "ignored, not .md")

	var res Result
	loadDir(pawsDir, &res)
	if len(res.Rules) != 1 || res.Rules[0].Name != "good" {
		t.Fatalf("expected 1 good rule, got %+v", res.Rules)
	}
	if len(res.Errs) != 1 {
		t.Errorf("expected 1 parse error for bad.md, got %d", len(res.Errs))
	}
}

func TestLoadDirFollowsSymlinkedRoot(t *testing.T) {
	real := t.TempDir()
	mustWrite(t, filepath.Join(real, "rule.md"), "---\nname: shared\nevent: pre-tool\naction:\n  type: block\n---\n")

	link := filepath.Join(t.TempDir(), ".paws")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	var res Result
	loadDir(link, &res)
	if len(res.Rules) != 1 || res.Rules[0].Name != "shared" {
		t.Fatalf("expected the symlinked dir's rule to load, got %+v (errs: %+v)", res.Rules, res.Errs)
	}
}

func TestCollectMergesAllScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mustMkdir(t, filepath.Join(home, ".paws"))
	mustWrite(t, filepath.Join(home, ".paws", "home.md"), "---\nname: home\nevent: pre-tool\naction:\n  type: audit\n---\n")

	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, ".git"))
	mustMkdir(t, filepath.Join(repo, ".paws"))
	mustWrite(t, filepath.Join(repo, ".paws", "repo.md"), "---\nname: repo\nevent: pre-tool\naction:\n  type: audit\n---\n")

	res := Collect(repo)
	if len(res.Rules) != 2 {
		t.Fatalf("expected rules from both repo and home scope, got %+v", res.Rules)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPawsDirsHonorsPawsHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	custom := filepath.Join(t.TempDir(), "paws-home")
	mustMkdir(t, custom)
	t.Setenv("PAWS_HOME", custom)

	repo := filepath.Join(t.TempDir(), "repo")
	mustMkdir(t, filepath.Join(repo, ".git"))

	dirs := pawsDirs(repo)
	if len(dirs) != 1 || dirs[0] != custom {
		t.Errorf("expected only $PAWS_HOME %s, got %v", custom, dirs)
	}
}

func TestPawsHomeReplacesHomeOutsideRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mustMkdir(t, filepath.Join(home, ".paws"))
	custom := filepath.Join(t.TempDir(), "paws-home")
	mustMkdir(t, custom)
	t.Setenv("PAWS_HOME", custom)

	cwd := filepath.Join(home, "scratch") // no .git: the walk stops at $HOME
	mustMkdir(t, cwd)

	dirs := pawsDirs(cwd)
	if len(dirs) != 1 || dirs[0] != custom {
		t.Errorf("~/.paws must not load when PAWS_HOME replaces it, got %v", dirs)
	}
}

func TestRelativePawsHomeIgnored(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PAWS_HOME", "relative/paws")
	if got, want := UserDir(), filepath.Join(home, ".paws"); got != want {
		t.Errorf("relative PAWS_HOME should fall back to %s, got %s", want, got)
	}
}

func TestLoadSkipsSkillsDigestsAndReadmes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := filepath.Join(t.TempDir(), ".paws")
	for path, body := range map[string]string{
		"aws/s3.md":                  "---\nname: s3\nevent: pre-tool\naction:\n  type: block\n---\n",
		"aws/digests/aws-s3.md":      "Block public access.\n",
		"aws/skills/aws-s3/SKILL.md": "---\nname: aws-s3\n---\n",
		"README.md":                  "# my rules\n",
	} {
		mustMkdir(t, filepath.Dir(filepath.Join(dir, path)))
		if err := os.WriteFile(filepath.Join(dir, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var res Result
	loadDir(dir, &res)
	if len(res.Errs) != 0 || len(res.Rules) != 1 {
		t.Errorf("want 1 rule and no errors, got %d rules, errs %v", len(res.Rules), res.Errs)
	}
}
