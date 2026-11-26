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
		filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "installed.yaml"),
		filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.pawsHome, "portal"),
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
		filepath.Join(f.pawsHome, "audit.log"), // the audit history is not deleted by an unattended command
		filepath.Join(f.pawsHome, "bin"),       // committed hooks in other repos may run the copy
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

// A cloned repo controls .paws/installed.yaml and .paws/packs, so a symbolic link in there must not lead the
// deletion elsewhere, and the state file stays so a retry finds the pack.
func TestUninstallDoesNotFollowSymlinksOutOfThePacksFolder(t *testing.T) {
	f := setupUninstall(t)
	victim := filepath.Join(f.home, "victim")
	writeFile(t, filepath.Join(victim, "docs", "keep.txt"), "x")
	packs := filepath.Join(f.repo, ".paws", "packs")
	for _, name := range []string{"writing", "_bindings"} {
		os.RemoveAll(filepath.Join(packs, name))
	}
	if err := os.Symlink(victim, filepath.Join(packs, "evil")); err != nil {
		t.Skip(err)
	}
	state := filepath.Join(f.repo, ".paws", "installed.yaml")
	writeFile(t, state, "packs:\n  evil/docs:\n    bindings: [a.md]\n    at: now\n  evil:\n    bindings: [a.md]\n    at: now\n  .:\n    bindings: [a.md]\n    at: now\n")
	code, _ := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
		return c
	})
	if code != 1 {
		t.Errorf("exit %d, want 1: a pack could not be removed", code)
	}
	kept(t, filepath.Join(victim, "docs", "keep.txt"), filepath.Join(packs, "evil"), state,
		filepath.Join(f.repo, ".claude", "skills", "git")) // a pack that stays may need its skills
}

func TestUninstallKeepsSkillsWithHiddenFilesAndUnreadableLogin(t *testing.T) {
	f := setupUninstall(t)
	gitSkill := filepath.Join(f.home, ".claude", "skills", "git")
	writeFile(t, filepath.Join(gitSkill, ".notes"), "mine") // the content hash ignores hidden files
	creds := filepath.Join(f.pawsHome, "credentials.json")
	writeFile(t, creds, "not json")
	code, out := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	if code != 1 {
		t.Errorf("exit %d, want 1 for the login that could not be read: %s", code, out)
	}
	if !strings.Contains(out, "can't read") || f.logouts.Load() != 0 {
		t.Errorf("stderr %q, %d logouts", out, f.logouts.Load())
	}
	// The login stays, so the portal rules stay, so the skills they require stay too.
	kept(t, filepath.Join(gitSkill, ".notes"), creds, filepath.Join(f.home, ".claude", "skills", "humanizer"))
	gone(t, filepath.Join(f.pawsHome, "packs"))
}

func TestUninstallGlobalKeepsTheBinaryAHookHereStillRuns(t *testing.T) {
	f := setupUninstall(t)
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	kept(t, filepath.Join(f.pawsHome, "bin"), filepath.Join(f.repo, ".claude", "settings.json"))
	if hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--global left the global hook")
	}
}

func TestUninstallKeepsTheLoginWhenThePortalCannotRetireTheDevice(t *testing.T) {
	var status atomic.Int32
	pawsHome := portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/device/logout" {
			w.WriteHeader(int(status.Load()))
			w.Write([]byte("{}"))
			return
		}
		http.NotFound(w, r)
	}))
	creds := filepath.Join(pawsHome, "credentials.json")
	writeFile(t, creds, `{"api":"`+strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")+`","token":"tok"}`)
	portalState := filepath.Join(pawsHome, "portal", "state.json")
	writeFile(t, portalState, "{}")
	for _, bad := range []int{503, 429, 403, 400} {
		status.Store(int32(bad))
		code, errOut := capture(t, &os.Stderr, func() int {
			c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
			return c
		})
		if code != 1 || !strings.Contains(errOut, "couldn't disconnect this laptop") {
			t.Errorf("status %d: exit %d, stderr %q", bad, code, errOut)
		}
		kept(t, creds, portalState) // the rules stay with the login
	}

	status.Store(401) // the portal already forgot this device: nothing left to retire
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	gone(t, creds, portalState)
}

func TestUninstallHandlesANullPackRecord(t *testing.T) {
	f := setupUninstall(t)
	writeFile(t, filepath.Join(f.repo, ".paws", "installed.yaml"), "packs:\n  writing: null\n")
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	gone(t, filepath.Join(f.repo, ".paws", "packs", "writing"), filepath.Join(f.repo, ".paws", "installed.yaml"))
}

// A cloned repo can commit .paws itself as a link to a folder that has its own packs.
func TestUninstallRefusesAProjectDotPawsThatIsALink(t *testing.T) {
	f := setupUninstall(t)
	outside := filepath.Join(f.home, "outside")
	writeFile(t, filepath.Join(outside, "packs", "writing", "keep.md"), "x")
	writeFile(t, filepath.Join(outside, "installed.yaml"), "packs:\n  writing:\n    bindings: [a.md]\n    at: now\n")
	os.RemoveAll(filepath.Join(f.repo, ".paws"))
	if err := os.Symlink(outside, filepath.Join(f.repo, ".paws")); err != nil {
		t.Skip(err)
	}
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
		return c
	})
	kept(t, filepath.Join(outside, "packs", "writing", "keep.md"), filepath.Join(outside, "installed.yaml"))
}

// In the home folder with no repo, the project's files are the global ones: --project must not take them.
func TestUninstallProjectScopeInTheHomeFolder(t *testing.T) {
	f := setupUninstall(t)
	t.Chdir(f.home)
	// Only the .claude folder is shared here (PAWS_HOME is elsewhere): --project leaves it to --global.
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--project took the global hook")
	}
	kept(t, filepath.Join(f.pawsHome, "packs"), filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.home, ".claude", "skills", "git"))
	// With PAWS_HOME at ~/.paws too, every project file is a global one.
	t.Setenv("PAWS_HOME", filepath.Join(f.home, ".paws"))
	code, errOut := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
		return c
	})
	if code != 2 || !strings.Contains(errOut, "not in a repo") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	// With both scopes, the shared files are listed and removed once.
	t.Setenv("PAWS_HOME", f.pawsHome)
	code, out := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
	if code != 0 || strings.Count(out, "~/.claude/settings.json") != 1 {
		t.Errorf("exit %d:\n%s", code, out)
	}
	gone(t, filepath.Join(f.pawsHome, "packs"))
}

func TestUninstallKeepsSkillsWhenThePackStateIsUnreadable(t *testing.T) {
	f := setupUninstall(t)
	state := filepath.Join(f.repo, ".paws", "installed.yaml")
	writeFile(t, state, "packs: [not, a, map")
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) }); code != 0 || !strings.Contains(out, "can't be read") {
		t.Fatalf("exit %d: %s", code, out)
	}
	kept(t, state, filepath.Join(f.repo, ".claude", "skills", "git"), filepath.Join(f.repo, ".paws", "packs"))
}

// A cloned repo can commit .claude as a link to the global one; --project must not touch what is behind it.
func TestUninstallProjectLeavesALinkedDotClaudeAlone(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.repo, ".claude"))
	if err := os.Symlink(filepath.Join(f.home, ".claude"), filepath.Join(f.repo, ".claude")); err != nil {
		t.Skip(err)
	}
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--project removed the global hook through a linked .claude")
	}
	kept(t, filepath.Join(f.home, ".claude", "skills", "git"), filepath.Join(f.home, ".claude", "skills", "humanizer"))
}

// Portal rules stay with the login, and they require the global skills, so a failed logout keeps those too.
func TestUninstallKeepsGlobalSkillsWhenTheLoginStays(t *testing.T) {
	b := &fakeCatalogBackend{files: map[string][]pushFile{}}
	pawsHome := portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/device/logout" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b.ServeHTTP(w, r)
	}))
	b.publish(t, writeTestCatalog(t))
	captureStdout(t, func() int { return Add([]string{"writing"}) })
	writeFile(t, filepath.Join(pawsHome, "credentials.json"), `{"api":"`+strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")+`","token":"tok"}`)
	skill := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "humanizer")
	kept(t, skill)
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	kept(t, skill, filepath.Join(pawsHome, "credentials.json"))
}

// A project .claude that is a link to the global one is left alone; a project .paws that is a link is too, even
// when no pack name in its installed.yaml is usable.
func TestUninstallProjectLinkedDotPawsWithNoUsablePack(t *testing.T) {
	f := setupUninstall(t)
	outside := filepath.Join(f.home, "outside")
	state := filepath.Join(outside, "installed.yaml")
	writeFile(t, state, "packs:\n  .:\n    bindings: [a.md]\n    at: now\n")
	os.RemoveAll(filepath.Join(f.repo, ".paws"))
	if err := os.Symlink(outside, filepath.Join(f.repo, ".paws")); err != nil {
		t.Skip(err)
	}
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, state)
}

// A hook that failed to come out may still run the binary, and a pack that stayed may still need its skills.
func TestUninstallSkipsLateStepsAfterAFailure(t *testing.T) {
	f := setupUninstall(t)
	// An early step that fails: a pack folder that is a link.
	os.RemoveAll(filepath.Join(f.pawsHome, "packs", "writing"))
	victim := filepath.Join(f.home, "victim")
	writeFile(t, filepath.Join(victim, "x"), "x")
	if err := os.Symlink(victim, filepath.Join(f.pawsHome, "packs", "writing")); err != nil {
		t.Skip(err)
	}
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	kept(t, filepath.Join(victim, "x"), filepath.Join(f.pawsHome, "bin"), filepath.Join(f.pawsHome, "portal"),
		filepath.Join(f.pawsHome, "installed.yaml"), filepath.Join(f.home, ".claude", "skills", "humanizer"))
}

// The login is a late step: a failure before it keeps the login, the portal rules and the audit history, and a
// failed logout keeps the audit history and the skills.
func TestUninstallKeepsLoginAndAuditHistoryAfterAnEarlierFailure(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.pawsHome, "packs", "writing"))
	victim := filepath.Join(f.home, "victim")
	writeFile(t, filepath.Join(victim, "x"), "x")
	if err := os.Symlink(victim, filepath.Join(f.pawsHome, "packs", "writing")); err != nil {
		t.Skip(err)
	}
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	kept(t, filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.pawsHome, "audit.log"), filepath.Join(f.pawsHome, "portal", "state.json"))
	if f.logouts.Load() != 0 {
		t.Errorf("the device was retired (%d calls) although a step before it failed", f.logouts.Load())
	}
}

func TestUninstallKeepsSkillsWhenProjectDotPawsIsLinked(t *testing.T) {
	f := setupUninstall(t)
	outside := filepath.Join(f.home, "outside")
	writeFile(t, filepath.Join(outside, "installed.yaml"), "packs: {}\n")
	os.RemoveAll(filepath.Join(f.repo, ".paws"))
	if err := os.Symlink(outside, filepath.Join(f.repo, ".paws")); err != nil {
		t.Skip(err)
	}
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, filepath.Join(f.repo, ".claude", "skills", "git"))
}

// Project skills are required by the portal rules too: when the logout fails (a 503) they stay.
func TestUninstallKeepsProjectSkillsWhenLogoutFails(t *testing.T) {
	b := &fakeCatalogBackend{files: map[string][]pushFile{}}
	pawsHome := portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/device/logout" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b.ServeHTTP(w, r)
	}))
	b.publish(t, writeTestCatalog(t))
	repo := filepath.Join(os.Getenv("HOME"), "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	t.Chdir(repo)
	captureStdout(t, func() int { return Add([]string{"--project", "writing"}) })
	writeFile(t, filepath.Join(pawsHome, "credentials.json"), `{"api":"`+strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")+`","token":"tok"}`)
	skill := filepath.Join(repo, ".claude", "skills", "humanizer")
	kept(t, skill)
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
		return c
	})
	kept(t, skill, filepath.Join(pawsHome, "credentials.json"))
}

// A cloned repo can commit .claude/settings.json as a link to the global one.
func TestUninstallProjectLeavesALinkedSettingsFileAlone(t *testing.T) {
	f := setupUninstall(t)
	link := filepath.Join(f.repo, ".claude", "settings.json")
	os.Remove(link)
	if err := os.Symlink(filepath.Join(f.home, ".claude", "settings.json"), link); err != nil {
		t.Skip(err)
	}
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	if !hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("--project removed the global hook through a linked settings.json")
	}
}

// A project hook this run leaves in place still needs the binary copy.
func TestUninstallKeepsTheBinaryForALinkedProjectHook(t *testing.T) {
	f := setupUninstall(t)
	elsewhere := filepath.Join(f.home, "elsewhere")
	os.MkdirAll(elsewhere, 0o755)
	os.Rename(filepath.Join(f.repo, ".claude"), filepath.Join(elsewhere, ".claude"))
	if err := os.Symlink(filepath.Join(elsewhere, ".claude"), filepath.Join(f.repo, ".claude")); err != nil {
		t.Skip(err)
	}
	captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
	if !hasPawsHook(filepath.Join(elsewhere, ".claude", "settings.json")) {
		t.Fatal("the linked project hook should stay")
	}
	kept(t, filepath.Join(f.pawsHome, "bin"))
}

// Without --global the portal rules stay, so a project skill the applied config uses stays too.
func TestUninstallProjectKeepsSkillsThePortalConfigUses(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.home, ".claude", "skills", "humanizer")) // the project copy is the only one
	writeFile(t, filepath.Join(f.pawsHome, "portal", "humanizer--pr.md"), requireSkillRule("humanizer--pr", "humanizer"))
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, filepath.Join(f.repo, ".claude", "skills", "humanizer"))
	gone(t, filepath.Join(f.repo, ".claude", "skills", "git"), filepath.Join(f.repo, ".claude", "settings.json"))
}

func TestUninstallProjectLeavesNoEmptyDotClaude(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.repo, ".claude", "skills", "ours")) // the user's own skill, removed so only paws's files are left
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	gone(t, filepath.Join(f.repo, ".claude"))
}

func TestUninstallHookOnlyRejectsProject(t *testing.T) {
	portalEnv(t, http.NotFoundHandler())
	if code, _ := capture(t, &os.Stderr, func() int { return Uninstall([]string{"--hook-only", "--project"}) }); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

// In the home folder the project's .claude is the global one: its settings.local.json is still a separate file.
func TestUninstallHomeFolderRemovesBothSettingsHooksAndKeepsTheBinary(t *testing.T) {
	f := setupUninstall(t)
	t.Chdir(f.home)
	local := filepath.Join(f.home, ".claude", "settings.local.json")
	doc := settingsDoc{}
	ensureHooks(doc, "unbound-paws gate", false)
	writeSettings(local, doc)
	if code, out := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) }); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if hasPawsHook(local) || hasPawsHook(filepath.Join(f.home, ".claude", "settings.json")) {
		t.Error("a hook is left in ~/.claude")
	}
	kept(t, filepath.Join(f.pawsHome, "bin")) // other repos' hooks may still run the copy
}

// An unreadable state.json must not hide which skills the portal rules use.
func TestUninstallProjectKeepsPortalSkillsWhenStateIsUnreadable(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.home, ".claude", "skills", "humanizer")) // the project copy is the only one
	writeFile(t, filepath.Join(f.pawsHome, "portal", "state.json"), "not json")
	writeFile(t, filepath.Join(f.pawsHome, "portal", "my--skill--pr.md"), requireSkillRule("my--skill--pr", "humanizer"))
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, filepath.Join(f.repo, ".claude", "skills", "humanizer"))
}

func requireSkillRule(name, skill string) string {
	return "---\nname: " + name + "\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: gh pr create\naction:\n  type: require-skill\n  skill: " + skill + "\n---\n"
}

// A rule the user wrote that requires a paws skill keeps that skill, and so does a pack the run could not read.
func TestUninstallKeepsSkillsASurvivingRuleRequires(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.home, ".claude", "skills", "humanizer")) // the project copy is the only one
	writeFile(t, filepath.Join(f.repo, ".paws", "pre-tool", "mine.md"), requireSkillRule("mine", "humanizer"))
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, filepath.Join(f.repo, ".claude", "skills", "humanizer"))
	gone(t, filepath.Join(f.repo, ".claude", "skills", "git"))

	// The project's state can't be read, so its packs stay and a copy of the skill they use must stay.
	f = setupUninstall(t)
	writeFile(t, filepath.Join(f.repo, ".paws", "installed.yaml"), "packs: [")
	capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
		return c
	})
	kept(t, filepath.Join(f.repo, ".claude", "skills", "humanizer"))
}

// A settings file that can't be parsed and mentions paws may hold the hook: nothing that hook needs is removed.
func TestUninstallStopsWhenAHookFileCannotBeRead(t *testing.T) {
	f := setupUninstall(t)
	writeFile(t, filepath.Join(f.home, ".claude", "settings.json"), `{"hooks": {"PreToolUse": [ {"command": "unbound-paws gate"} `)
	code, _ := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	// The hook may still be registered, so the rules it enforces stay too.
	kept(t, filepath.Join(f.pawsHome, "credentials.json"), filepath.Join(f.home, ".claude", "skills", "humanizer"), filepath.Join(f.pawsHome, "installed.yaml"),
		filepath.Join(f.pawsHome, "packs", "writing"))
}

// A nested .paws in the same repo (a monorepo package) uses the project's skills too.
func TestUninstallKeepsSkillsANestedRuleRequires(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.home, ".claude", "skills", "humanizer"))
	writeFile(t, filepath.Join(f.repo, "packages", "web", ".paws", "pre-tool", "mine.md"), requireSkillRule("mine", "humanizer"))
	captureStdout(t, func() int { return Uninstall([]string{"--project", "--yes"}) })
	kept(t, filepath.Join(f.repo, ".claude", "skills", "humanizer"))
}

func TestUninstallStopsWhenAHookFileCannotBeOpened(t *testing.T) {
	f := setupUninstall(t)
	path := filepath.Join(f.home, ".claude", "settings.json")
	if err := os.Chmod(path, 0); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o600) })
	if f, err := os.Open(path); err == nil { // running as root: the mode doesn't stop us
		f.Close()
		t.Skip("file is readable anyway")
	}
	code, _ := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	kept(t, filepath.Join(f.pawsHome, "credentials.json"))
}

func TestUninstallKeepsACredentialsFileWithATokenAndNoAddress(t *testing.T) {
	f := setupUninstall(t)
	creds := filepath.Join(f.pawsHome, "credentials.json")
	writeFile(t, creds, `{"token":"tok"}`)
	code, _ := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Uninstall([]string{"--global", "--yes"}) })
		return c
	})
	if code != 1 || f.logouts.Load() != 0 {
		t.Errorf("exit %d, %d logouts", code, f.logouts.Load())
	}
	kept(t, creds)
}

// Outside a repo nothing is walked: the folder could be the whole home folder.
func TestRequiredSkillsDoesNotWalkOutsideARepo(t *testing.T) {
	f := setupUninstall(t)
	os.RemoveAll(filepath.Join(f.repo, ".git"))
	writeFile(t, filepath.Join(f.repo, "deep", ".paws", "pre-tool", "x.md"), requireSkillRule("x", "walked"))
	t.Chdir(f.repo)
	requiredCache = nil
	if requiredSkills()["walked"] {
		t.Error("walked a folder that is not a repo")
	}
	os.MkdirAll(filepath.Join(f.repo, ".git"), 0o755)
	requiredCache = nil
	if !requiredSkills()["walked"] {
		t.Error("did not find a nested .paws in a repo")
	}
	requiredCache = nil
}

// A user-level rule serves every repo, so a user skill stays even when this project has its own copy.
func TestUninstallKeepsAUserSkillASurvivingUserRuleRequires(t *testing.T) {
	f := setupUninstall(t)
	writeFile(t, filepath.Join(f.pawsHome, "pre-tool", "mine.md"), requireSkillRule("mine", "humanizer"))
	captureStdout(t, func() int { return Uninstall([]string{"--yes"}) })
	kept(t, filepath.Join(f.home, ".claude", "skills", "humanizer"))
	gone(t, filepath.Join(f.repo, ".claude", "skills", "humanizer")) // the user copy serves it
}
