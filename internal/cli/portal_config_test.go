package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/pack"
)

const humanizerRef = "0123456789abcdef0123456789abcdef01234567"

// fakeUpstream makes pack.Fetch check out a one-file skill whose SKILL.md is body, and
// returns that skill's content hash.
func fakeUpstream(t *testing.T, body string) string {
	t.Helper()
	orig := pack.Fetch
	pack.Fetch = func(repo, ref, dest string) error {
		return os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(body), 0o644)
	}
	t.Cleanup(func() { pack.Fetch = orig })
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644)
	h, err := pack.HashSkill(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	return h
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
	want := "[backup-first--2.md digests/backup-first.md house-style--x----y.md humanizer--a1.md]"
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
	skills := filepath.Join(os.Getenv("HOME"), ".claude", "skills")
	// The user's own copy of a skill the config also names: paws must leave it alone.
	os.MkdirAll(filepath.Join(skills, "writer"), 0o755)
	os.WriteFile(filepath.Join(skills, "writer", "SKILL.md"), []byte("mine\n"), 0o644)
	writer := humanizerSkill("sha256:" + strings.Repeat("0", 64))
	writer["name"] = "writer"
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill("sha256:"+strings.Repeat("a", 64)), writer)

	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Config: 1 skill (writer) applied.\n  skipped humanizer: content hash mismatch at the pinned ref; not installed") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(skills, "humanizer")); !os.IsNotExist(err) {
		t.Error("a mismatched skill was installed")
	}
	if data, _ := os.ReadFile(filepath.Join(skills, "writer", "SKILL.md")); string(data) != "mine\n" {
		t.Errorf("the user's skill was overwritten: %q", data)
	}
	if got := fmt.Sprint(portalFiles(t, home)); got != "[writer--a1.md]" {
		t.Errorf("portal files %s", got)
	}
	if st := loadPortalState(); st.AppliedHash != "h1" {
		t.Errorf("a mismatch is final for this config; applied hash %q, want h1", st.AppliedHash)
	}
}

func TestSyncRetriesAfterAFetchFailure(t *testing.T) {
	home, srv := syncEnv(t)
	orig := pack.Fetch
	pack.Fetch = func(repo, ref, dest string) error { return errors.New("network down") }
	t.Cleanup(func() { pack.Fetch = orig })
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

// A skill the catalog ships in this binary installs from it, checked against the hash
// `paws index --portal` publishes for it.
func TestSyncInstallsCatalogSkillFromTheBinary(t *testing.T) {
	home, srv := syncEnv(t)
	c, probs := catalog.LoadFS(paws.Catalog, "embedded")
	if len(probs) > 0 {
		t.Fatal(probs)
	}
	var git *portalCatalogSkill
	for _, s := range buildPortalCatalog(c)["skills"].([]*portalCatalogSkill) {
		if s.Name == "git" {
			git = s
		}
	}
	if git == nil || git.Upstream != nil {
		t.Fatalf("want git as a catalog-shipped skill, got %+v", git)
	}
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", map[string]any{"name": "git", "kind": "skill", "load": "full", "status": "on",
		"content_hash": git.ContentHash, "version": git.Version, "actions": git.DefaultActions})

	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.Contains(out, "Config: 1 skill (git) applied.") {
		t.Fatalf("sync exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".claude", "skills", "git", "SKILL.md")); err != nil {
		t.Errorf("git skill not installed: %v", err)
	}
	if got := portalFiles(t, home); len(got) != len(git.DefaultActions) {
		t.Errorf("want one rule file per default action, got %v", got)
	}
}
