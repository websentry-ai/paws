package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const scanTriggersJSON = `{"version":"v1","skills":[
 {"name":"coreutils","actions":[{"id":"coreutils","label":"Recursive deletes","event":"pre-tool","tool":"Bash","command_regex":"rm\\s+-\\S*r"}]},
 {"name":"terraform","actions":[{"id":"tf-apply","label":"terraform apply","event":"pre-tool","tool":"Bash","command_regex":"terraform\\s+apply"},
  {"id":"tf-files","label":"Edit .tf files","event":"pre-tool","tool":"Edit|Write","file_glob":"**/*.tf"}]},
 {"name":"notion","actions":[{"id":"notion-create","label":"Notion pages created","event":"pre-tool","tool":"mcp__*notion*","mcp_action":"create"}]},
 {"name":"git","actions":[{"id":"git-commit","label":"git commit","event":"pre-commit","tool":"Bash"},
  {"id":"session","label":"Session start","event":"session-start"}]},
 {"name":"humanizer","actions":[{"id":"h","label":"PR text","event":"pre-tool","tool":"Bash","command_regex":"gh pr create"}]}
]}`

// Secrets, paths, prompt and output text the scan must never send.
var scanLeaks = []string{"ghp_SECRETTOKEN123456", "hunter2", "alice", "acme", "/Users", "Q3 revenue", "TOP-SECRET-OUTPUT", "private-skill", "deploy-acme"}

// writeScanSessions puts Claude Code logs full of things that must not leave the machine under HOME/.claude/projects.
func writeScanSessions(t *testing.T) {
	t.Helper()
	home, _ := os.UserHomeDir()
	call := func(ts, id, name, input string) string {
		return `{"type":"assistant","timestamp":"` + ts + `","message":{"content":[{"type":"text","text":"I will now hunter2"},{"type":"tool_use","id":"` + id + `","name":"` + name + `","input":` + input + `}]}}`
	}
	lines := []string{
		`{"type":"user","timestamp":"2026-11-01T09:00:00Z","message":{"content":"summarize acme Q3 revenue, my token is ghp_SECRETTOKEN123456"}}`,
		call("2026-11-01T09:00:01Z", "t1", "Bash", `{"command":"cd /Users/alice/acme && rm -rf build && PASSWORD=hunter2 ./deploy-acme.sh"}`),
		`{"type":"user","timestamp":"2026-11-01T09:00:02Z","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"TOP-SECRET-OUTPUT"}]},"toolUseResult":{"stdout":"TOP-SECRET-OUTPUT"}}`,
		call("2026-11-02T09:00:00Z", "t2", "Bash", `{"command":"terraform apply -var db_password=hunter2"}`),
		call("2026-11-02T09:00:01Z", "t3", "Edit", `{"file_path":"/Users/alice/acme/infra/main.tf","old_string":"hunter2","new_string":"x"}`),
		call("2026-11-03T09:00:00Z", "t4", "mcp__claude_ai_Notion__notion-create-pages", `{"title":"acme Q3 revenue"}`),
		call("2026-11-03T09:00:01Z", "t5", "Bash", `{"command":"git commit -m 'acme: rotate ghp_SECRETTOKEN123456'"}`),
		call("2026-11-03T09:00:02Z", "t6", "Skill", `{"skill":"private-skill"}`),
		call("2026-11-03T09:00:03Z", "t7", "Skill", `{"skill":"coreutils"}`),
		call("2026-11-04T09:00:00Z", "t8", "Bash", `{"command":"/Users/alice/bin/acme-tool --key hunter2"}`),
		call("2026-11-04T09:00:01Z", "t9", "Read", `{"file_path":"/Users/alice/acme/.env"}`),
	}
	dir := filepath.Join(home, ".claude", "projects", "-Users-alice-acme")
	writeFile(t, filepath.Join(dir, "sess-1.jsonl"), strings.Join(lines, "\n")+"\n")
	// A resumed session copies the earlier calls into its own log: they count once.
	writeFile(t, filepath.Join(dir, "sess-2.jsonl"), strings.Join(lines[1:3], "\n")+"\n")
}

func fixScanClock(t *testing.T) {
	old := scanNow
	scanNow = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { scanNow = old })
}

func TestScanUploadsCountsOnly(t *testing.T) {
	_, srv := syncEnv(t)
	srv.triggers = scanTriggersJSON
	fixScanClock(t)
	writeScanSessions(t)

	code, dry := captureStdout(t, func() int { return Scan([]string{"--dry-run"}) })
	if code != 0 {
		t.Fatalf("scan --dry-run exit %d: %s", code, dry)
	}
	if len(srv.histories) != 0 {
		t.Fatalf("--dry-run sent %d uploads", len(srv.histories))
	}
	code, out := captureStdout(t, func() int { return Scan(nil) })
	if code != 0 || len(srv.histories) != 1 {
		t.Fatalf("scan exit %d, %d uploads: %s", code, len(srv.histories), out)
	}
	if srv.histories[0] != dry {
		t.Errorf("the upload differs from what --dry-run printed:\n%s\n---\n%s", srv.histories[0], dry)
	}
	if srv.auths[len(srv.auths)-1] != "Bearer tok-abc" {
		t.Errorf("auth %q", srv.auths[len(srv.auths)-1])
	}
	for _, leak := range scanLeaks {
		if strings.Contains(dry, leak) {
			t.Errorf("the upload carries %q:\n%s", leak, dry)
		}
	}
	var got historyUpload
	if err := json.Unmarshal([]byte(dry), &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "claude-code" || got.WindowDays != 30 || got.Since != "2026-10-20" || got.Until != "2026-11-19" ||
		got.Sessions != 2 || got.ToolCalls != 7 || got.CatalogVersion != "v1" {
		t.Errorf("header %+v", got)
	}
	calls := map[string]int{}
	for _, s := range got.Skills {
		calls[s.Skill] = s.Calls
	}
	want := map[string]int{"coreutils": 1, "terraform": 2, "notion": 1, "git": 1}
	if len(calls) != len(want) {
		t.Errorf("skills %v, want %v", calls, want)
	}
	for name, n := range want {
		if calls[name] != n {
			t.Errorf("%s: %d calls, want %d (all: %v)", name, calls[name], n, calls)
		}
	}
	if len(got.Loaded) != 1 || got.Loaded[0].Name != "coreutils" {
		t.Errorf("loaded %v: only catalog skills", got.Loaded)
	}
	if !strings.Contains(out, "terraform") || !strings.Contains(out, "Sent the counts") {
		t.Errorf("output %q", out)
	}
	if st := loadScanState(); st.Days != 30 || st.ScannedAt.IsZero() {
		t.Errorf("scan.json %+v: the scan date and window are kept", st)
	}
	if strings.Contains(readFileOrEmpty(filepath.Join(os.Getenv("PAWS_HOME"), "portal", "state.json")), "scan") {
		t.Error("the scan must not write the state.json the hooks' syncs rewrite")
	}

	// A second scan sends the same summary again; the portal keeps one per laptop, so nothing doubles.
	if code, _ := captureStdout(t, func() int { return Scan(nil) }); code != 0 || len(srv.histories) != 2 || srv.histories[1] != srv.histories[0] {
		t.Errorf("a second scan should send the identical summary (exit %d, %d uploads)", code, len(srv.histories))
	}
}

func TestScanLeavesOutWhatTheHooksUploaded(t *testing.T) {
	home, srv := syncEnv(t)
	srv.triggers = scanTriggersJSON
	fixScanClock(t)
	writeScanSessions(t)
	api := strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")
	// Logged in on Nov 2 at 12:00.
	if err := writePrivate(filepath.Join(home, "credentials.json"), credentials{API: api, Token: "tok-abc", CreatedAt: "2026-11-02T12:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	scanCalls := func() historyUpload {
		t.Helper()
		code, dry := captureStdout(t, func() int { return Scan([]string{"--dry-run"}) })
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
		var got historyUpload
		json.Unmarshal([]byte(dry), &got)
		return got
	}
	// The hook logged nothing since the login (paws not registered in that session), so every call counts.
	if got := scanCalls(); got.Until != "2026-11-19" || got.ToolCalls != 7 {
		t.Errorf("no hook records: until %s, %d calls; want every call up to now", got.Until, got.ToolCalls)
	}
	// The hook logged the Notion call (t4) and the git commit (t5) after the login, and terraform apply (t2) before
	// it; not the terraform edit (t3) or the later calls (t8, t9). Only the three it logged were uploaded, so only
	// they go.
	appendLines(t, filepath.Join(home, "audit.log"),
		`{"v":2,"ts":"2026-11-03T09:00:01Z","agent":"claude-code","event":"action","session":"sess-1","tool":"mcp__claude_ai_Notion__notion-create-pages","action":"notion create-pages"}`,
		`{"v":2,"ts":"2026-11-03T09:00:02Z","agent":"claude-code","event":"action","session":"sess-1","tool":"Bash","action":"git commit","command":"git commit -m 'acme: rotate ***'"}`,
		// A Bash record whose command is another call's: it must not take t8.
		`{"v":2,"ts":"2026-11-04T09:00:01Z","agent":"claude-code","event":"action","session":"sess-1","tool":"Bash","action":"ls","command":"ls"}`,
		`{"v":2,"ts":"2026-10-30T10:00:00Z","agent":"claude-code","event":"action","session":"sess-other","tool":"Bash","action":"ls"}`,
		// The hook logged t2 before the login; the first sync uploaded it with the rest of the log.
		`{"v":2,"ts":"2026-11-02T09:00:01Z","agent":"claude-code","event":"action","session":"sess-1","tool":"Bash","action":"terraform apply","command":"terraform apply -var db_password=***"}`,
		// An Edit outside the repo, logged without a path: it can't be told apart, so it takes no call.
		`{"v":2,"ts":"2026-11-02T09:00:02Z","agent":"claude-code","event":"action","session":"sess-1","tool":"Edit","action":"edit"}`)
	if got := scanCalls(); got.ToolCalls != 4 {
		t.Errorf("hook logged 3 of sess-1's calls (one before the login): %d calls, want 7-3=4", got.ToolCalls)
	}
	// A fork copies the logged git commit (t5) into sess-3, a session the hook never saw: the copy has the same id,
	// so it stays out with the original.
	userHome, _ := os.UserHomeDir()
	writeFile(t, filepath.Join(userHome, ".claude", "projects", "-Users-alice-acme", "sess-3.jsonl"),
		`{"type":"assistant","timestamp":"2026-11-03T09:00:01Z","message":{"content":[{"type":"tool_use","id":"t5","name":"Bash","input":{"command":"git commit -m 'acme: rotate ghp_SECRETTOKEN123456'"}}]}}`+"\n")
	if got := scanCalls(); got.ToolCalls != 4 {
		t.Errorf("a fork's copy of a logged call came back: %d calls, want 4", got.ToolCalls)
	}
}

func TestScanFlags(t *testing.T) {
	syncEnv(t)
	for _, args := range [][]string{{"--days", "45"}, {"--days"}, {"--bogus"}, {"--days", "x"}} {
		if code, _ := captureStdout(t, func() int { return Scan(args) }); code != 2 {
			t.Errorf("scan %v exit %d, want 2", args, code)
		}
	}
}

func TestScanNeedsLoginToUpload(t *testing.T) {
	srv := &fakePortal{t: t, triggers: scanTriggersJSON}
	portalEnv(t, srv)
	if code, _ := captureStdout(t, func() int { return Scan(nil) }); code != 1 {
		t.Errorf("scan without login exit %d, want 1", code)
	}
	if srv.count("") != 0 {
		t.Errorf("scan without login sent %v", srv.requests)
	}
	// A dry run needs no login: the catalog is public, and nothing is sent.
	code, out := captureStdout(t, func() int { return Scan([]string{"--dry-run"}) })
	if code != 0 || !strings.Contains(out, `"source": "claude-code"`) || len(srv.histories) != 0 {
		t.Errorf("dry run without login exit %d: %s", code, out)
	}
}

func TestLoginScan(t *testing.T) {
	srv := &fakePortal{t: t, triggers: scanTriggersJSON}
	portalEnv(t, srv)
	fixScanClock(t)
	writeScanSessions(t)
	code, out := captureStdout(t, func() int { return Login([]string{"--key", "pk_abc", "--scan", "--days", "90"}) })
	if code != 0 || len(srv.histories) != 1 || !strings.Contains(out, "Sent the counts") {
		t.Fatalf("login --scan exit %d, %d uploads: %s", code, len(srv.histories), out)
	}
	if !strings.Contains(srv.histories[0], `"window_days": 90`) {
		t.Errorf("upload %s", srv.histories[0])
	}
	if code, _ := captureStdout(t, func() int { return Login([]string{"--key", "pk_abc", "--days", "90"}) }); code != 2 {
		t.Errorf("--days without --scan exit %d, want 2", code)
	}
}

func TestScanRefusesAnOversizedCatalog(t *testing.T) {
	_, srv := syncEnv(t)
	actions := make([]string, 21)
	for i := range actions {
		actions[i] = fmt.Sprintf(`{"id":"a%d","label":"x","event":"pre-tool","tool":"Bash","command_regex":"ghp_.{%d}X"}`, i, i)
	}
	srv.triggers = `{"version":"v1","skills":[{"name":"probe","actions":[` + strings.Join(actions, ",") + `]}]}`
	writeScanSessions(t)
	code, out := captureStdout(t, func() int { return Scan([]string{"--dry-run"}) })
	if code != 1 || strings.Contains(out, `"skills"`) {
		t.Fatalf("a catalog with 21 triggers on one skill must be refused, exit %d: %s", code, out)
	}
}

func readFileOrEmpty(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
