package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// syncServer records each /sync request and answers with statuses[i] (200 once they run out).
type syncServer struct {
	t        *testing.T
	statuses []int
	bodies   []string
	auths    []string
}

func (s *syncServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/paws/v1/sync" {
		s.t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}
	b, _ := io.ReadAll(r.Body)
	i := len(s.bodies)
	s.bodies = append(s.bodies, string(b))
	s.auths = append(s.auths, r.Header.Get("Authorization"))
	status := http.StatusOK
	if i < len(s.statuses) {
		status = s.statuses[i]
	}
	w.WriteHeader(status)
	if status != http.StatusOK {
		io.WriteString(w, `{"error":"unknown token"}`)
		return
	}
	fmt.Fprintf(w, `{"accepted":%d,"rejected":0}`, len(decodeSync(s.t, string(b))))
}

func decodeSync(t *testing.T, body string) []map[string]any {
	t.Helper()
	var b struct{ Records []map[string]any }
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatalf("bad sync body %q: %v", body, err)
	}
	return b.Records
}

// syncEnv is a logged-in PAWS_HOME talking to a syncServer.
func syncEnv(t *testing.T, statuses ...int) (string, *syncServer) {
	t.Helper()
	srv := &syncServer{t: t, statuses: statuses}
	home := portalEnv(t, srv)
	api := strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")
	if err := writePrivate(filepath.Join(home, "credentials.json"), credentials{API: api, Token: "tok-abc"}); err != nil {
		t.Fatal(err)
	}
	return home, srv
}

func appendLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o700)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		fmt.Fprintln(f, l)
	}
}

// ruleRecord is an audit log rule record as logAudit writes it, target and session included.
func ruleRecord(ts, name, verdict, satisfiedBy, session, source, skill string) string {
	rec := map[string]any{
		"v": 2, "ts": ts, "agent": "claude-code", "event": "pre-tool", "tool": "Bash",
		"target": "gh pr create --body 'SECRET-TEXT'", "rule": name, "action": "require-skill",
		"verdict": verdict, "source": source, "session": session,
	}
	if satisfiedBy != "" {
		rec["satisfied_by"] = satisfiedBy
	}
	if skill != "" {
		rec["skill"] = skill
	}
	b, _ := json.Marshal(rec)
	return string(b)
}

func writeRule(t *testing.T, path, frontmatter string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\n"+frontmatter+"---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncUploadsOnlyRuleCountFields(t *testing.T) {
	home, srv := syncEnv(t)
	catalogRule := filepath.Join(home, "packs", "writing", "writing", "humanizer-pr.md")
	writeRule(t, catalogRule, "name: humanizer-pr\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: \\bgh\\s+pr\\s+create\\b\naction:\n  type: require-skill\n  skill: humanizer\n")
	userRule := filepath.Join(home, "my-rule.md")
	writeRule(t, userRule, "name: my-rule\nevent: pre-tool\nmatcher:\n  command_regex: private-regex\naction:\n  type: require-skill\n  skill: tdd\n")
	gone := filepath.Join(home, "deleted.md")

	appendLines(t, filepath.Join(home, "audit.log"),
		ruleRecord("2026-10-01T10:00:00Z", "humanizer-pr", "deny", "", "sess-1", catalogRule, "humanizer"),
		`{"v":2,"ts":"2026-10-01T10:00:01Z","event":"skill_invoked","skill":"humanizer","session":"sess-1"}`,
		ruleRecord("2026-10-01T10:00:02Z", "humanizer-pr", "allow", "skill-load", "sess-1", catalogRule, "humanizer"),
		ruleRecord("2026-10-01T10:00:03Z", "humanizer-pr", "allow", "skill-load", "sess-2", catalogRule, "humanizer"),
		ruleRecord("2026-10-01T10:00:04Z", "my-rule", "allow", "skill-load", "sess-2", userRule, ""), // pre-skill-field record
		ruleRecord("2026-10-01T10:00:05Z", "gone", "deny", "", "sess-2", gone, ""),                   // skill unknowable: dropped
		ruleRecord("2026-10-01T10:00:06Z", "s3", "allow", "digest-delivered", "sess-2", gone, "aws-s3"),
		`{"v":2,"ts":"2026-10-01T10:00:07Z","event":"turn_end","session":"sess-2"}`,
		`{"v":2,"ts":"2026-10-01T10:00:08Z","event":"tool_result","tool":"Bash","session":"sess-2"}`,
		`not json`,
	)

	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("sync exit %d: %q", code, out)
	}
	if len(srv.bodies) != 1 || srv.auths[0] != "Bearer tok-abc" {
		t.Fatalf("want one request with the bearer token, got %d %v", len(srv.bodies), srv.auths)
	}
	body := srv.bodies[0]
	for _, leak := range []string{"SECRET-TEXT", "sess-", home, "private-regex", `"target"`, `"session"`, `"tool"`, `"source"`, `"agent"`} {
		if strings.Contains(body, leak) {
			t.Errorf("payload leaks %q: %s", leak, body)
		}
	}
	want := []string{
		`{"binding":"humanizer-pr","event":"pre-tool","pattern":"tool=Bash command_regex=\\bgh\\s+pr\\s+create\\b","skill":"humanizer","ts":"2026-10-01T10:00:00Z","verdict":"deny","already_loaded":false}`,
		`{"binding":"humanizer-pr","event":"pre-tool","pattern":"tool=Bash command_regex=\\bgh\\s+pr\\s+create\\b","skill":"humanizer","ts":"2026-10-01T10:00:02Z","verdict":"allow","already_loaded":false,"satisfied_by":"skill-load"}`,
		`{"binding":"humanizer-pr","event":"pre-tool","pattern":"tool=Bash command_regex=\\bgh\\s+pr\\s+create\\b","skill":"humanizer","ts":"2026-10-01T10:00:03Z","verdict":"allow","already_loaded":true,"satisfied_by":"skill-load"}`,
		`{"binding":"my-rule","event":"pre-tool","skill":"tdd","ts":"2026-10-01T10:00:04Z","verdict":"allow","already_loaded":true,"satisfied_by":"skill-load"}`,
		`{"binding":"s3","event":"pre-tool","skill":"aws-s3","ts":"2026-10-01T10:00:06Z","verdict":"allow","already_loaded":false,"satisfied_by":"digest-delivered"}`,
	}
	got := decodeSync(t, body)
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d: %s", len(got), len(want), body)
	}
	for i, w := range want {
		var wm map[string]any
		json.Unmarshal([]byte(w), &wm)
		gb, _ := json.Marshal(got[i])
		wb, _ := json.Marshal(wm)
		if string(gb) != string(wb) {
			t.Errorf("record %d:\n got %s\nwant %s", i, gb, wb)
		}
	}
}

func TestSyncUploadsEachRecordOnce(t *testing.T) {
	home, srv := syncEnv(t)
	log := filepath.Join(home, "audit.log")
	rec := func(ts, name string) string { return ruleRecord(ts, name, "deny", "", "s", "", "k") }
	appendLines(t, log, rec("2026-10-01T10:00:00Z", "a"), rec("2026-10-01T10:00:00Z", "b"))
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatal("first sync failed")
	}
	// c shares a's and b's second: the cursor must count, not just compare timestamps.
	appendLines(t, log, rec("2026-10-01T10:00:00Z", "c"), rec("2026-10-01T10:00:01Z", "d"))
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatal("second sync failed")
	}
	code, out := captureStdout(t, func() int { return Sync(nil) })
	if code != 0 || !strings.Contains(out, "Nothing new to upload.") {
		t.Errorf("third sync: exit %d %q", code, out)
	}
	// Rotation: audit.log becomes audit.log.1; only the new file's record is new.
	os.Rename(log, log+".1")
	appendLines(t, log, rec("2026-10-01T10:00:02Z", "e"))
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatal("sync after rotation failed")
	}

	var sent [][]string
	for _, b := range srv.bodies {
		var names []string
		for _, r := range decodeSync(t, b) {
			names = append(names, r["binding"].(string))
		}
		sent = append(sent, names)
	}
	if fmt.Sprint(sent) != "[[a b] [c d] [e]]" {
		t.Errorf("requests carried %v, want [[a b] [c d] [e]]", sent)
	}
}

func TestSyncBatchesAndResumesAfterRevokedLogin(t *testing.T) {
	home, srv := syncEnv(t, http.StatusOK, http.StatusUnauthorized)
	var lines []string
	for i := range 2500 {
		lines = append(lines, ruleRecord(fmt.Sprintf("2026-10-01T10:%02d:%02dZ", i/60%60, i%60), fmt.Sprintf("r%04d", i), "deny", "", "s", "", "k"))
	}
	appendLines(t, filepath.Join(home, "audit.log"), lines...)

	code, errOut := capture(t, &os.Stderr, func() int { return Sync(nil) })
	if code != 1 || !strings.Contains(errOut, "Your login was revoked. Run `paws login`.") {
		t.Fatalf("401 must stop the sync: exit %d, stderr %q", code, errOut)
	}
	if len(srv.bodies) != 2 {
		t.Fatalf("sync must stop at the 401, sent %d requests", len(srv.bodies))
	}
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatal("sync after re-login failed")
	}

	sizes := []int{}
	seen := map[string]bool{}
	for i, b := range srv.bodies {
		recs := decodeSync(t, b)
		sizes = append(sizes, len(recs))
		if i == 1 {
			continue // rejected with 401: not counted as sent
		}
		for _, r := range recs {
			name := r["binding"].(string)
			if seen[name] {
				t.Errorf("%s uploaded twice", name)
			}
			seen[name] = true
		}
	}
	if fmt.Sprint(sizes) != "[1000 1000 1000 500]" || len(seen) != 2500 {
		t.Errorf("batch sizes %v, %d distinct records; want [1000 1000 1000 500] and 2500", sizes, len(seen))
	}
}

func TestSyncDryRunSendsNothing(t *testing.T) {
	home, srv := syncEnv(t)
	var lines []string
	for i := range 25 {
		lines = append(lines, ruleRecord(fmt.Sprintf("2026-10-01T10:00:%02dZ", i), fmt.Sprintf("r%02d", i), "deny", "", "s", "", "k"))
	}
	appendLines(t, filepath.Join(home, "audit.log"), lines...)

	code, out := captureStdout(t, func() int { return Sync([]string{"--dry-run"}) })
	if code != 0 {
		t.Fatalf("dry-run exit %d", code)
	}
	if len(srv.bodies) != 0 {
		t.Errorf("dry-run sent %d requests", len(srv.bodies))
	}
	if recs := decodeSync(t, out); len(recs) != 20 || recs[0]["binding"] != "r00" {
		t.Errorf("dry-run should print the first 20 records, got %d: %s", len(recs), out)
	}
	if _, err := os.Stat(filepath.Join(home, "sync-state.json")); !os.IsNotExist(err) {
		t.Error("dry-run must not move the cursor")
	}
	// The real sync afterwards still sends all 25.
	captureStdout(t, func() int { return Sync(nil) })
	if len(srv.bodies) != 1 || len(decodeSync(t, srv.bodies[0])) != 25 {
		t.Errorf("sync after dry-run sent %v", srv.bodies)
	}
}

func TestSyncRequiresLogin(t *testing.T) {
	srv := &syncServer{t: t}
	home := portalEnv(t, srv)
	appendLines(t, filepath.Join(home, "audit.log"), ruleRecord("2026-10-01T10:00:00Z", "a", "deny", "", "s", "", "k"))

	code, errOut := capture(t, &os.Stderr, func() int { return Sync(nil) })
	if code != 1 || !strings.Contains(errOut, "Run `paws login` first.") || len(srv.bodies) != 0 {
		t.Errorf("not logged in: exit %d, stderr %q, %d requests", code, errOut, len(srv.bodies))
	}
}
