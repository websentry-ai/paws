package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

const humanizerRef = "0123456789abcdef0123456789abcdef01234567"

// serveSkills makes the paws backend answer skill-file downloads with get, and returns the hashes asked for.
func serveSkills(t *testing.T, get func(ctx context.Context, hash string) (fs.FS, error)) *[]string {
	t.Helper()
	asked := &[]string{}
	orig := skillFilesFor
	skillFilesFor = func(string) func(context.Context, string) (fs.FS, error) {
		return func(ctx context.Context, hash string) (fs.FS, error) {
			*asked = append(*asked, hash)
			return get(ctx, hash)
		}
	}
	t.Cleanup(func() { skillFilesFor = orig })
	return asked
}

// fakeUpstream makes the backend serve a one-file skill whose SKILL.md is body, whatever hash is asked for, and
// returns that skill's content hash; the CLI's hash check decides whether it installs.
func fakeUpstream(t *testing.T, body string) string {
	t.Helper()
	files := fstest.MapFS{"SKILL.md": {Data: []byte(body), Mode: 0o644}}
	serveSkills(t, func(context.Context, string) (fs.FS, error) { return files, nil })
	h, err := pack.HashSkill(files)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// userSkill puts a SKILL.md for name in the test HOME's ~/.claude/skills, as a user would.
func userSkill(t *testing.T, name, body string) string {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), ".claude", "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func configJSON(t *testing.T, hash string, skills ...map[string]any) string {
	t.Helper()
	if skills == nil {
		skills = []map[string]any{}
	}
	b, err := json.Marshal(map[string]any{"config_hash": hash, "skills": skills})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func humanizerSkill(hash string) map[string]any {
	return map[string]any{"name": "humanizer", "kind": "skill", "load": "full", "version": "8c307f3", "status": "on",
		"content_hash": hash,
		"upstream":     map[string]any{"repo": "github.com/blader/humanizer", "path": "SKILL.md", "ref": humanizerRef},
		"actions": []map[string]any{{"id": "a1", "label": "Open a pull request", "event": "pre-tool", "tool": "Bash",
			"command_regex": `(^|[;&|\n])\s*gh\s+pr\s+create\b`}}}
}

func backupRule() map[string]any {
	return map[string]any{"name": "backup-first", "kind": "rule", "load": "rule", "status": "on",
		"rule_text": "Make a backup before a big delete.",
		"actions":   []map[string]any{{"id": 2, "label": "DELETE", "event": "pre-tool", "tool": "Bash", "command_regex": `(?i)\bdelete\s+from\b`}}}
}

// portalFiles lists the rule and digest files under $PAWS_HOME/portal.
func portalFiles(t *testing.T, home string) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{"*.md", "digests/*.md"} {
		m, _ := filepath.Glob(filepath.Join(home, "portal", pattern))
		for _, f := range m {
			rel, _ := filepath.Rel(filepath.Join(home, "portal"), f)
			out = append(out, filepath.ToSlash(rel))
		}
	}
	sort.Strings(out)
	return out
}

// deny runs a Bash PreToolUse through the gate and returns its deny reason ("" if allowed).
func deny(t *testing.T, command string) string {
	t.Helper()
	out := runGate(t, map[string]any{"hook_event_name": "PreToolUse", "cwd": t.TempDir(), "session_id": "s",
		"tool_name": "Bash", "tool_input": map[string]any{"command": command}})
	var o struct {
		HookSpecificOutput struct {
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	json.Unmarshal([]byte(out), &o)
	if o.HookSpecificOutput.PermissionDecision != "deny" {
		return ""
	}
	return o.HookSpecificOutput.PermissionDecisionReason
}

func TestSyncAppliesRemovesAndTurnsOffConfig(t *testing.T) {
	home, srv := syncEnv(t)
	hash := fakeUpstream(t, "# humanizer\n")
	userSkill(t, "house-style", "# house style\n") // a custom skill is the user's to provide
	custom := map[string]any{"name": "house-style", "kind": "skill", "load": "full", "status": "on",
		"actions": []map[string]any{{"id": "x/../y", "label": "Edit docs", "event": "pre-tool", "tool": "Write|Edit", "file_glob": "docs/**"}}}
	off := map[string]any{"name": "tdd", "kind": "skill", "load": "full", "status": "off",
		"actions": []map[string]any{{"id": "t", "label": "x", "event": "pre-tool", "tool": "Bash", "command_regex": "x"}}}
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(hash), backupRule(), custom, off)

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Nothing new to upload.\nConfig: 3 skills (backup-first, house-style, humanizer) applied.\n") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	want := "[backup-first--2.md digests/backup-first.md house-style--x----y-fec01a9e.md humanizer--a1.md]"
	if got := fmt.Sprint(portalFiles(t, home)); got != want {
		t.Fatalf("portal files %s, want %s", got, want)
	}
	if st := loadPortalState(); st.AppliedHash != "h1" || fmt.Sprint(st.Skills) != "[backup-first house-style humanizer]" {
		t.Errorf("state: %+v", st)
	}
	skillDir := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "humanizer")
	if data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md")); err != nil || string(data) != "# humanizer\n" {
		t.Errorf("humanizer not installed from upstream: %q %v", data, err)
	}
	// The applied hash is reported back right away.
	var last struct {
		Applied *string `json:"applied_config_hash"`
	}
	json.Unmarshal([]byte(srv.bodies[len(srv.bodies)-1]), &last)
	if last.Applied == nil || *last.Applied != "h1" {
		t.Errorf("last /sync did not report the applied hash: %s", srv.bodies[len(srv.bodies)-1])
	}

	// The gate enforces the files at once: a full skill by name, a load=rule skill by its text.
	if r := deny(t, "gh pr create --title x"); !strings.Contains(r, `"humanizer"`) {
		t.Errorf("gh pr create should require humanizer, got %q", r)
	}
	if r := deny(t, `psql -c "delete from users"`); !strings.Contains(r, "Make a backup before a big delete.") || !strings.Contains(r, "paws digest (backup-first--2)") {
		t.Errorf("DELETE should deny with the rule text as digest, got %q", r)
	}

	// humanizer and house-style leave the config: their rule files go, backup-first stays.
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", backupRule())
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: 1 skill (backup-first) applied.") {
		t.Fatalf("second sync exit %d, output %q", code, out)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[backup-first--2.md digests/backup-first.md]" {
		t.Errorf("after removal: %s", got)
	}
	if deny(t, "gh pr create --title x") != "" {
		t.Error("a removed skill's rule still denies")
	}

	// Everything off: no portal rule is left; the unchanged hash then means nothing to fetch.
	srv.configHash = "h3"
	srv.config = configJSON(t, "h3")
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: no skills applied.") {
		t.Fatalf("third sync exit %d, output %q", code, out)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("turned off, files remain: %v", got)
	}
	before := srv.count("GET /paws/v1/config")
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: up to date, no skills.") {
		t.Fatalf("fourth sync exit %d, output %q", code, out)
	}
	if srv.count("GET /paws/v1/config") != before {
		t.Error("an unchanged hash must not fetch the config")
	}
}

func TestSyncRefusesMismatchedSkillsAndKeepsUserCopies(t *testing.T) {
	home, srv := syncEnv(t)
	fakeUpstream(t, "# humanizer, changed upstream\n")
	pinned := "sha256:" + strings.Repeat("a", 64)
	skills := filepath.Join(os.Getenv("HOME"), ".claude", "skills")
	// The user's own copy of a catalog skill the config also names: paws must leave it alone.
	userSkill(t, "git", "mine\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(pinned), map[string]any{"name": "git", "kind": "skill", "load": "full", "status": "on",
		"content_hash": "sha256:" + strings.Repeat("0", 64),
		"actions":      []map[string]any{{"id": "c", "label": "Commit", "event": "pre-commit", "tool": "Bash"}}})

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "  skipped humanizer: content hash mismatch at the pinned ref; not installed") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(skills, "humanizer")); !os.IsNotExist(err) {
		t.Error("a mismatched skill was installed")
	}
	if data, _ := os.ReadFile(filepath.Join(skills, "git", "SKILL.md")); string(data) != "mine\n" {
		t.Errorf("the user's skill was overwritten: %q", data)
	}
	// git is not in this test's catalog: a custom skill, and the user's copy carries its rules.
	if got := fmt.Sprint(portalFiles(t, home)); got != "[git--c.md]" {
		t.Errorf("portal files %s", got)
	}
	if st := loadPortalState(); st.AppliedHash != "h1" {
		t.Errorf("a mismatch is final for this config; applied hash %q, want h1", st.AppliedHash)
	}
}

// A config can only point a skill at a github.com repo at a commit, with a content hash; anything else installs
// nothing and downloads nothing. A pin the backend serves other files for is refused by the hash check.
func TestSyncRefusesABadPin(t *testing.T) {
	home, srv := syncEnv(t)
	pinned := "sha256:" + strings.Repeat("a", 64)
	asked := serveSkills(t, func(context.Context, string) (fs.FS, error) {
		return fstest.MapFS{"SKILL.md": {Data: []byte("evil\n"), Mode: 0o644}}, nil
	})
	variant := func(name string, edit func(u map[string]any)) map[string]any {
		s := humanizerSkill(pinned)
		s["name"] = name
		edit(s["upstream"].(map[string]any))
		return s
	}
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1",
		variant("humanizer", func(u map[string]any) { u["ref"] = "main" }),
		variant("gitlab", func(u map[string]any) { u["repo"] = "gitlab.com/blader/humanizer" }),
		variant("evil", func(u map[string]any) { u["repo"] = "github.com/evil/skills" }),
	)

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: no skills applied.") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if n := strings.Count(out, ": its pin is not a github.com repo at a commit with a content hash; not installed"); n != 2 {
		t.Errorf("want 2 bad-pin skips, got %d in %q", n, out)
	}
	if !strings.Contains(out, "evil: content hash mismatch at the pinned ref; not installed") {
		t.Errorf("the evil pin must fail its hash check: %q", out)
	}
	if fmt.Sprint(*asked) != "["+pinned+"]" {
		t.Errorf("downloads %v; only the well-formed pin may be fetched", *asked)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("portal files %v", got)
	}
	for _, name := range []string{"evil", "humanizer", "gitlab"} {
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".claude", "skills", name)); !os.IsNotExist(err) {
			t.Errorf("skill %s was installed", name)
		}
	}
}

// A user deletes a portal skill, or its rule files, by hand. Until the next config check the
// gate lets the action through (Claude would have no skill to load); then the check re-applies
// the config even though its hash hasn't changed.
func TestSyncRepairsSkillsAndRulesDeletedByHand(t *testing.T) {
	home, srv := syncEnv(t)
	v1 := fakeUpstream(t, "# humanizer\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(v1), backupRule())
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("first sync exit %d, output %q", code, out)
	}
	skill := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "humanizer")

	os.RemoveAll(skill)
	if r := deny(t, "gh pr create --title x"); r != "" {
		t.Errorf("denied with the skill gone, a dead end for the agent: %q", r)
	}
	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: 2 skills (backup-first, humanizer) applied.") {
		t.Fatalf("sync after deleting the skill: exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(skill, "SKILL.md")); err != nil {
		t.Errorf("skill not reinstalled: %v", err)
	}
	if r := deny(t, "gh pr create --title x"); !strings.Contains(r, `"humanizer"`) {
		t.Errorf("gate after the reinstall: %q", r)
	}

	os.Remove(filepath.Join(home, "portal", "backup-first--2.md"))
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "applied") {
		t.Fatalf("sync after deleting a rule file: exit %d, output %q", code, out)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[backup-first--2.md digests/backup-first.md humanizer--a1.md]" {
		t.Errorf("portal files %s", got)
	}
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: up to date") {
		t.Errorf("nothing drifted: want up to date, got exit %d, output %q", code, out)
	}
}

// An update approved in the portal moves the pin to a newer commit of the same repo and path;
// the laptop installs it once the download hashes to the approved content.
func TestSyncInstallsAnApprovedUpdate(t *testing.T) {
	home, srv := syncEnv(t)
	files := fstest.MapFS{"SKILL.md": {Data: []byte("# humanizer v2\n"), Mode: 0o644}}
	approved, _ := pack.HashSkill(files)
	asked := serveSkills(t, func(context.Context, string) (fs.FS, error) { return files, nil })
	newRef := strings.Repeat("b", 40)
	s := humanizerSkill(approved)
	s["upstream"].(map[string]any)["ref"] = newRef
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", s)

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: 1 skill (humanizer) applied.") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if got := fmt.Sprint(*asked); got != "["+approved+"]" {
		t.Errorf("downloaded %s", got)
	}
	if data, _ := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".claude", "skills", "humanizer", "SKILL.md")); string(data) != "# humanizer v2\n" {
		t.Errorf("installed SKILL.md %q", data)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[humanizer--a1.md]" {
		t.Errorf("portal files %s", got)
	}
	if st := loadPortalState(); st.AppliedHash != "h2" {
		t.Errorf("applied hash %q, want h2", st.AppliedHash)
	}
}

// An update whose download doesn't hash to the approved content is refused, and the version
// already installed keeps serving: its rules stay, and the config isn't reported as applied.
func TestSyncKeepsTheInstalledVersionWhenAnUpdateFails(t *testing.T) {
	home, srv := syncEnv(t)
	v1 := fakeUpstream(t, "# humanizer v1\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(v1))
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("first sync exit %d, output %q", code, out)
	}

	fakeUpstream(t, "# humanizer v2, not what was approved\n")
	s := humanizerSkill("sha256:" + strings.Repeat("d", 64))
	s["upstream"].(map[string]any)["ref"] = strings.Repeat("b", 40)
	srv.configHash = "h2"
	srv.config = configJSON(t, "h2", s)
	code, errOut := capture(t, &os.Stderr, func() int {
		code, _ := captureStdout(t, func() int { return Sync(nil) })
		return code
	})
	if code != 1 || !strings.Contains(errOut, "content hash mismatch at the pinned ref; not installed; keeping the installed version") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if data, _ := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".claude", "skills", "humanizer", "SKILL.md")); string(data) != "# humanizer v1\n" {
		t.Errorf("installed SKILL.md %q, want v1 kept", data)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[humanizer--a1.md]" {
		t.Errorf("portal files %s; the installed version's rules must stay", got)
	}
	if st := loadPortalState(); st.AppliedHash != "h1" {
		t.Errorf("applied hash %q, want h1 until the update installs", st.AppliedHash)
	}
}

func TestSyncRetriesAfterAFetchFailure(t *testing.T) {
	home, srv := syncEnv(t)
	serveSkills(t, func(context.Context, string) (fs.FS, error) { return nil, errors.New("network down") })
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill("sha256:"+strings.Repeat("a", 64)), backupRule())

	code, errOut := capture(t, &os.Stderr, func() int {
		code, _ := captureStdout(t, func() int { return Sync(nil) })
		return code
	})
	if code != 1 || !strings.Contains(errOut, "network down") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[backup-first--2.md digests/backup-first.md]" {
		t.Errorf("the skills that could apply should: %s", got)
	}
	if st := loadPortalState(); st.AppliedHash != "" {
		t.Errorf("applied hash %q recorded after a fetch failure; it must be retried", st.AppliedHash)
	}
}

// A skill Unbound wrote (no upstream) installs from the backend too, checked against its content hash.
func TestSyncInstallsABundledSkillFromTheBackend(t *testing.T) {
	home, srv := syncEnv(t)
	files := fstest.MapFS{"SKILL.md": {Data: []byte("---\nname: git\n---\nShow real output.\n"), Mode: 0o644},
		"scripts/check.sh": {Data: []byte("#!/bin/sh\necho ok\n"), Mode: 0o755}}
	hash, _ := pack.HashSkill(files)
	serveSkills(t, func(context.Context, string) (fs.FS, error) { return files, nil })
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", map[string]any{"name": "git", "kind": "skill", "load": "full", "status": "on",
		"content_hash": hash, "version": hash[7:14], "actions": []map[string]any{
			{"id": "git", "label": "A commit", "event": "pre-commit", "tool": "Bash"}}})

	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: 1 skill (git) applied.") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	dir := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "git")
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Errorf("git skill not installed: %v", err)
	}
	if st, err := os.Stat(filepath.Join(dir, "scripts", "check.sh")); err != nil || st.Mode().Perm()&0o100 == 0 {
		t.Errorf("a script stays runnable: %v %v", st, err)
	}
	if got := portalFiles(t, home); fmt.Sprint(got) != "[git--git.md]" {
		t.Errorf("want one rule file, got %v", got)
	}
}

// customSkill is a full skill with no upstream, which the user provides, and these actions.
func customSkill(name string, actions ...map[string]any) map[string]any {
	return map[string]any{"name": name, "kind": "skill", "load": "full", "status": "on", "actions": actions}
}

// An action that would gate the Skill tool (no tool and no matcher, or tool Skill) is never
// written: its rule would deny the very call that loads the skill.
func TestSyncSkipsActionsThatGateTheSkillTool(t *testing.T) {
	home, srv := syncEnv(t)
	userSkill(t, "house-style", "# house style\n")
	userSkill(t, "lockout", "# lockout\n")
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1",
		customSkill("house-style",
			map[string]any{"id": "all", "label": "Anything", "event": "pre-tool"},
			map[string]any{"id": "star", "label": "Star", "event": "pre-tool", "tool": "*"},
			map[string]any{"id": "skill", "label": "Skills", "event": "pre-tool", "tool": "Bash|Skill", "command_regex": "."},
			map[string]any{"id": "ok", "label": "Edit docs", "event": "pre-tool", "tool": "Edit", "file_glob": "docs/**"}),
		customSkill("lockout", map[string]any{"id": "x", "label": "Everything", "event": "pre-tool", "tool": ""}),
	)

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: 1 skill (house-style) applied.") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	for _, label := range []string{"Anything", "Star", "Skills", "Everything"} {
		if !strings.Contains(out, fmt.Sprintf("action %q: it would gate the Skill tool", label)) {
			t.Errorf("no skip for action %q in %q", label, out)
		}
	}
	if !strings.Contains(out, "skipped lockout: ") || !strings.Contains(out, "no usable actions") {
		t.Errorf("lockout should be skipped with no usable actions: %q", out)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[house-style--ok.md]" {
		t.Errorf("portal files %s", got)
	}
}

// A full skill's rules wait for its SKILL.md: in the user's skills or the project's.
func TestSyncWithholdsRulesUntilTheSkillIsInstalled(t *testing.T) {
	home, srv := syncEnv(t)
	action := map[string]any{"id": "d", "label": "Deploy", "event": "pre-tool", "tool": "Bash", "command_regex": "deploy"}
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", customSkill("deployer", action))
	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: no skills applied.\n  skipped deployer: skill not installed") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("rules written for a missing skill: %v", got)
	}
	if deny(t, "deploy prod") != "" {
		t.Error("a missing skill's rule denies")
	}

	// The skill in the project's .claude/skills (sync runs from the repo) counts, and the
	// unchanged config applies: a skipped missing skill doesn't mark the hash as applied.
	cwd, _ := os.Getwd()
	os.MkdirAll(filepath.Join(cwd, ".git"), 0o755)
	os.MkdirAll(filepath.Join(cwd, ".claude", "skills", "deployer"), 0o755)
	os.WriteFile(filepath.Join(cwd, ".claude", "skills", "deployer", "SKILL.md"), []byte("# deployer\n"), 0o644)
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: 1 skill (deployer) applied.") {
		t.Fatalf("second sync exit %d, output %q", code, out)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[deployer--d.md]" {
		t.Errorf("portal files %s", got)
	}
}

// Two action ids that cut to the same file name still get a file each.
func TestSyncKeepsActionIDsThatCollideApart(t *testing.T) {
	home, srv := syncEnv(t)
	userSkill(t, "house-style", "# house style\n")
	long := strings.Repeat("a", 70)
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", customSkill("house-style",
		map[string]any{"id": "a/b", "label": "One", "event": "pre-tool", "tool": "Bash", "command_regex": "one"},
		map[string]any{"id": "a?b", "label": "Two", "event": "pre-tool", "tool": "Bash", "command_regex": "two"},
		map[string]any{"id": long + "1", "label": "Three", "event": "pre-tool", "tool": "Bash", "command_regex": "three"},
		map[string]any{"id": long + "2", "label": "Four", "event": "pre-tool", "tool": "Bash", "command_regex": "four"},
		map[string]any{"id": "a-b", "label": "Five", "event": "pre-tool", "tool": "Bash", "command_regex": "five"}))
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	files := portalFiles(t, home)
	if len(files) != 5 {
		t.Fatalf("want 5 rule files, got %v", files)
	}
	for _, cmd := range []string{"one", "two", "three", "four", "five"} {
		if r := deny(t, cmd); !strings.Contains(r, `"house-style"`) {
			t.Errorf("%s: no rule denies, files %v", cmd, files)
		}
	}
}

// A load=rule skill whose text wouldn't fit in a deny is skipped, not written to be cut off.
func TestSyncSkipsOversizedRuleText(t *testing.T) {
	home, srv := syncEnv(t)
	big := backupRule()
	big["rule_text"] = strings.Repeat("x", rule.MaxDigest)
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", big)
	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, fmt.Sprintf("skipped backup-first: rule text is %d bytes, over the %d-byte limit", rule.MaxDigest+1, rule.MaxDigest)) {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("portal files %v", got)
	}
}

// Skip reasons carry server text (here a bad regex, echoed raw by the regexp error): control
// characters are dropped before printing.
func TestSyncStripsControlCharactersFromSkipReasons(t *testing.T) {
	_, srv := syncEnv(t)
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", map[string]any{"name": "deployer", "kind": "rule", "load": "rule", "status": "on", "rule_text": "r",
		"actions": []map[string]any{{"id": "d", "label": "x", "event": "pre-tool", "tool": "Bash", "command_regex": "(\x1b[2J\nforged line"}}})
	_, out := captureStdout(t, func() int { return Sync(nil) })
	if !strings.Contains(out, "skipped deployer: ") || !strings.Contains(out, "forged line") {
		t.Fatalf("want a skip for deployer echoing the regex, got %q", out)
	}
	if strings.ContainsAny(out[strings.Index(out, "skipped"):strings.LastIndex(out, "forged line")], "\x1b\n") {
		t.Errorf("control characters reached the terminal: %q", out)
	}
}
