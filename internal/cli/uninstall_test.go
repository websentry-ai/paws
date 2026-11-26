package cli

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// uninstallFixture sets up a machine with paws fully installed under a temp HOME: a global and a project hook, packs
// and their skills in both scopes, a login, and an audit log, next to files the user made. It never touches the
// real ~/.claude or ~/.paws.
type uninstallFixture struct {
	home, pawsHome, repo string
	logouts              *atomic.Int32
}

func (f uninstallFixture) path(parts ...string) string { return filepath.Join(parts...) }

func setupUninstall(t *testing.T) uninstallFixture {
	t.Helper()
	b := &fakeCatalogBackend{files: map[string][]pushFile{}}
	logouts := &atomic.Int32{}
	pawsHome := portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/device/logout" {
			logouts.Add(1)
			w.Write([]byte("{}"))
			return
		}
		b.ServeHTTP(w, r)
	}))
	b.publish(t, writeTestCatalog(t))
	home := os.Getenv("HOME")
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	captureStdout(t, func() int {
		for _, args := range [][]string{{"writing", "source-control/git"}, {"--project", "writing", "source-control/git"}} {
			if code := Add(args); code != 0 {
				t.Fatalf("add %v exit %d", args, code)
			}
		}
		for _, args := range [][]string{{"--global"}, {}} {
			if code := Init(args); code != 0 {
				t.Fatalf("init %v exit %d", args, code)
			}
		}
		return 0
	})
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"env":{"A":"1"}}`)
	captureStdout(t, func() int { return Init([]string{"--global"}) })
	writeFile(t, filepath.Join(pawsHome, "credentials.json"), `{"api":"`+strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")+`","token":"tok"}`)
	writeFile(t, filepath.Join(pawsHome, "audit.log"), "{}\n")
	writeFile(t, filepath.Join(pawsHome, "portal", "state.json"), "{}")

	// What the user made.
	writeFile(t, filepath.Join(home, ".claude", "skills", "mine", "SKILL.md"), "mine")
	writeFile(t, filepath.Join(repo, ".claude", "skills", "ours", "SKILL.md"), "ours")
	writeFile(t, filepath.Join(pawsHome, "pre-tool", "mine.md"), "---\nname: mine\n---\n")
	writeFile(t, filepath.Join(repo, ".paws", "pre-tool", "ours.md"), "---\nname: ours\n---\n")
	return uninstallFixture{home: home, pawsHome: pawsHome, repo: repo, logouts: logouts}
}

func gone(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be gone (%v)", p, err)
		}
	}
}

func kept(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s should still be there: %v", p, err)
		}
	}
}

func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.WriteString(input)
	w.Close()
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; r.Close() })
}

func TestUninstallRemovesEverythingPawsAddedAndNothingElse(t *testing.T) {
	f := setupUninstall(t)
	editedSkill := filepath.Join(f.home, ".claude", "skills", "humanizer")
	writeFile(t, filepath.Join(editedSkill, "NOTES.md"), "my notes") // the user changed a paws skill

	code, out := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}

	gone(t,
		filepath.Join(f.repo, ".claude", "settings.json"),
		filepath.Join(f.repo, ".paws", "packs"), filepath.Join(f.repo, ".paws", "installed.yaml"),
		filepath.Join(f.repo, ".claude", "skills", "git"), filepath.Join(f.repo, ".claude", "skills", "humanizer"),
		filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "installed.yaml"), filepath.Join(f.pawsHome, "bin"),
		filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.pawsHome, "audit.log"), filepath.Join(f.pawsHome, "portal"),
		filepath.Join(f.home, ".claude", "skills", "git"))
	if hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("the global hook is still registered")
	}
	if f.logouts.Load() != 1 {
		t.Errorf("uninstall should log out of the portal once, got %d calls", f.logouts.Load())
	}
	kept(t,
		filepath.Join(f.home, ".claude", "settings.json"), // still holds the user's env block
		filepath.Join(f.home, ".claude", "skills", "mine", "SKILL.md"),
		filepath.Join(f.repo, ".claude", "skills", "ours", "SKILL.md"),
		filepath.Join(f.pawsHome, "pre-tool", "mine.md"),
		filepath.Join(f.repo, ".paws", "pre-tool", "ours.md"),
		editedSkill)
	if data, _ := os.ReadFile(filepath.Join(f.home, ".claude", "settings.json")); !strings.Contains(string(data), `"A"`) {
		t.Errorf("the user's settings were lost: %s", data)
	}
	for _, want := range []string{"paws will remove:", "paws will keep:", "was edited since", "npm rm -g unbound-paws", "brew uninstall --cask unbound-paws", "~/.local/bin/unbound-paws"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestUninstallScopesAndKeepSkills(t *testing.T) {
	f := setupUninstall(t)
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--project", "--keep-skills", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	gone(t, filepath.Join(f.repo, ".claude", "settings.json"), filepath.Join(f.repo, ".paws", "packs"), filepath.Join(f.repo, ".paws", "installed.yaml"))
	kept(t, filepath.Join(f.repo, ".claude", "skills", "git"), filepath.Join(f.repo, ".claude", "skills", "humanizer"),
		filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.pawsHome, "bin"))
	if !hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--project removed the global hook")
	}
	if f.logouts.Load() != 0 {
		t.Error("--project must not log out")
	}

	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--global", "--keep-skills", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	gone(t, filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"))
	kept(t, filepath.Join(f.home, ".claude", "skills", "git"), filepath.Join(f.home, ".claude", "skills", "humanizer"))
	if f.logouts.Load() != 1 {
		t.Errorf("--global logs out once, got %d", f.logouts.Load())
	}
}

func TestUninstallAsksFirstAndListsWhatItWillRemove(t *testing.T) {
	f := setupUninstall(t)
	withStdin(t, "n\n")
	code, out := captureStdout(t, func() int { return Uninstall(nil) })
	if code != 0 || !strings.Contains(out, "paws will remove:") || !strings.Contains(out, "Remove these? [y/N]") || !strings.Contains(out, "Nothing removed.") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	kept(t, filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.repo, ".claude", "settings.json"))

	withStdin(t, "") // a script with no answer and no --yes removes nothing
	if code, _ := capture(t, &os.Stderr, func() int { c, _ := captureStdout(t, func() int { return Uninstall(nil) }); return c }); code != 1 {
		t.Errorf("no answer: exit %d, want 1", code)
	}
	kept(t, filepath.Join(f.pawsHome, "packs"))

	withStdin(t, "y\n")
	if code, out := captureStdout(t, func() int { return Uninstall(nil) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	gone(t, filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"))
}

func TestUninstallHookOnlyKeepsEverythingElse(t *testing.T) {
	f := setupUninstall(t)
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--hook-only", "--global"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--hook-only left the global hook")
	}
	kept(t, filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.home, ".claude", "skills", "git"),
		filepath.Join(f.repo, ".claude", "settings.json"))
	if f.logouts.Load() != 0 {
		t.Error("--hook-only must not log out")
	}
}

func TestUninstallNeverDeletesOutsideThePacksFolder(t *testing.T) {
	f := setupUninstall(t)
	victim := filepath.Join(f.home, "victim")
	writeFile(t, filepath.Join(victim, "keep.txt"), "x")
	state := filepath.Join(f.pawsHome, "installed.yaml")
	writeFile(t, state, "packs:\n  ../../victim:\n    bindings: [a.md]\n    at: now\n")
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	kept(t, filepath.Join(victim, "keep.txt"))
}

func TestUninstallWithNothingInstalled(t *testing.T) {
	portalEnv(t, http.NotFoundHandler())
	code, out := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
	if code != 0 || !strings.Contains(out, "Nothing to remove") || !strings.Contains(out, "npm rm -g unbound-paws") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	if code, _ := capture(t, &os.Stderr, func() int { return Uninstall([]string{"--bogus"}) }); code != 2 {
		t.Errorf("unknown flag: exit %d, want 2", code)
	}
}
