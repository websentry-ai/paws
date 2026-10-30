package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/pack"
)

// runGate feeds one hook event to `paws gate` and returns what it printed.
func runGate(t *testing.T, event map[string]any) string {
	t.Helper()
	data, _ := json.Marshal(event)
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	in, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	old := os.Stdin
	os.Stdin = in
	defer func() { os.Stdin = old }()
	code, out := captureStdout(t, Gate)
	if code != 0 {
		t.Fatalf("gate exit %d", code)
	}
	return out
}

// auditLines returns the audit log's records.
func auditLines(t *testing.T, home string) []map[string]any {
	t.Helper()
	f, err := os.Open(filepath.Join(home, "audit.log"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var m map[string]any
		if json.Unmarshal(sc.Bytes(), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func TestPostToolUseWritesNormalizedActions(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	userHome := os.Getenv("HOME")
	long := "echo " + strings.Repeat("x", 2000)

	cases := []struct {
		tool       string
		input      map[string]any
		action     string
		command    string // "" means no command field
		file       string // "" means no file field
		mcpServer  string
		mcpClass   string
		mustNotSee string
	}{
		{tool: "Bash", input: map[string]any{"command": "git commit -m 'fix the thing'"}, action: "git commit", command: "git commit -m 'fix the thing'"},
		{tool: "Bash", input: map[string]any{"command": "cd /x/y && git -C /x/y push origin main"}, action: "git push", command: "cd /x/y && git -C /x/y push origin main"},
		{tool: "Bash", input: map[string]any{"command": "gh pr create --title t --body b"}, action: "gh pr create", command: "gh pr create --title t --body b"},
		{tool: "Bash", input: map[string]any{"command": "gh api repos/o/r/pulls"}, action: "gh api", command: "gh api repos/o/r/pulls"},
		{tool: "Bash", input: map[string]any{"command": "aws s3 mb s3://bucket"}, action: "aws s3", command: "aws s3 mb s3://bucket"},
		{tool: "Bash", input: map[string]any{"command": "CI=1 sudo npm test -- --watch"}, action: "npm test", command: "CI=1 sudo npm test -- --watch"},
		{tool: "Bash", input: map[string]any{"command": "ls -la | grep x"}, action: "ls", command: "ls -la | grep x"},
		{tool: "Bash", input: map[string]any{"command": "cat notes.txt"}, action: "cat", command: "cat notes.txt"},
		{tool: "Bash", input: map[string]any{"command": "/usr/local/bin/python3 script.py"}, action: "python3", command: "/usr/local/bin/python3 script.py"},
		{tool: "Bash", input: map[string]any{"command": "make -C web build"}, action: "make build", command: "make -C web build"},
		{tool: "Bash", input: map[string]any{"command": `psql -c "DELETE FROM users WHERE id = 1"`}, action: "DELETE", command: `psql -c "DELETE FROM users WHERE id = 1"`},
		{tool: "Bash", input: map[string]any{"command": `sqlite3 app.db 'drop table sessions'`}, action: "DROP", command: `sqlite3 app.db 'drop table sessions'`},
		{tool: "Bash", input: map[string]any{"command": `psql -c "UPDATE users SET admin = true"`}, action: "UPDATE", command: `psql -c "UPDATE users SET admin = true"`},
		{tool: "Bash", input: map[string]any{"command": `psql -c "TRUNCATE events"`}, action: "TRUNCATE", command: `psql -c "TRUNCATE events"`},
		{tool: "Bash", input: map[string]any{"command": "truncate -s 0 app.log"}, action: "truncate", command: "truncate -s 0 app.log"},
		{tool: "Bash", input: map[string]any{"command": "apt-get update && apt-get install -y jq"}, action: "apt-get update", command: "apt-get update && apt-get install -y jq"},
		// Redaction: the secret never reaches the log, the key name stays readable.
		{tool: "Bash", input: map[string]any{"command": "curl --token=s3cr3tvalue -H x https://api.example.com"}, action: "curl", command: "curl --token=*** -H x https://api.example.com", mustNotSee: "s3cr3tvalue"},
		{tool: "Bash", input: map[string]any{"command": "GITHUB_TOKEN=ghp_abcdefghijkl1234 git push"}, action: "git push", command: "GITHUB_TOKEN=*** git push", mustNotSee: "ghp_abcdefghijkl1234"},
		{tool: "Bash", input: map[string]any{"command": long}, action: "echo", command: long[:500]},
		// Files: repo-relative, never absolute; outside the repo there is no file at all.
		{tool: "Read", input: map[string]any{"file_path": filepath.Join(repo, "src", "app", "main.go")}, action: "read src/**", file: "src/app/main.go"},
		{tool: "Edit", input: map[string]any{"file_path": filepath.Join(repo, "README.md"), "old_string": "SECRET-OLD", "new_string": "SECRET-NEW"}, action: "edit README.md", file: "README.md", mustNotSee: "SECRET-"},
		{tool: "Write", input: map[string]any{"file_path": "migrations/0002_users.sql", "content": "SECRET-CONTENT"}, action: "edit migrations/**", file: "migrations/0002_users.sql", mustNotSee: "SECRET-CONTENT"},
		{tool: "Read", input: map[string]any{"file_path": "/etc/hosts"}, action: "read", mustNotSee: "/etc/hosts"},
		{tool: "Edit", input: map[string]any{"file_path": filepath.Join(userHome, ".zshrc")}, action: "edit", mustNotSee: ".zshrc"},
		{tool: "Read", input: map[string]any{"file_path": "../outside.txt"}, action: "read", mustNotSee: "outside"},
		// MCP: server and verb, never the tool input.
		{tool: "mcp__claude_ai_Gmail__create_draft", input: map[string]any{"body": "SECRET-BODY"}, action: "gmail create-draft", mcpServer: "claude_ai_Gmail", mcpClass: "send", mustNotSee: "SECRET-BODY"},
		{tool: "mcp__claude_ai_Notion__notion-create-pages", input: map[string]any{"title": "x"}, action: "notion create-pages", mcpServer: "claude_ai_Notion", mcpClass: "create"},
		{tool: "mcp__plugin_slack_slack__slack_send_message", input: map[string]any{"text": "hi"}, action: "slack send-message", mcpServer: "plugin_slack_slack", mcpClass: "send"},
		{tool: "mcp__github__get_issue", input: map[string]any{"n": 1}, action: "github get-issue", mcpServer: "github"},
		// Other tools: the name only.
		{tool: "WebFetch", input: map[string]any{"url": "https://internal.example/SECRET-URL"}, action: "webfetch", mustNotSee: "SECRET-URL"},
		{tool: "Grep", input: map[string]any{"pattern": "SECRET-PATTERN"}, action: "grep", mustNotSee: "SECRET-PATTERN"},
	}
	for _, c := range cases {
		t.Run(c.tool+" "+c.action, func(t *testing.T) {
			out := runGate(t, map[string]any{"hook_event_name": "PostToolUse", "cwd": repo, "session_id": "s1",
				"tool_name": c.tool, "tool_input": c.input})
			if out != "{}" {
				t.Errorf("PostToolUse must answer {}, got %q", out)
			}
			recs := auditLines(t, home)
			if len(recs) < 2 || recs[len(recs)-2]["event"] != "tool_result" {
				t.Fatalf("want tool_result then action, got %v", recs)
			}
			rec := recs[len(recs)-1]
			raw, _ := json.Marshal(rec)
			if rec["event"] != "action" || rec["tool"] != c.tool || rec["action"] != c.action {
				t.Errorf("record %s: want tool %s action %q", raw, c.tool, c.action)
			}
			if got, _ := rec["command"].(string); got != c.command {
				t.Errorf("command = %q, want %q", got, c.command)
			}
			if got, _ := rec["file"].(string); got != c.file {
				t.Errorf("file = %q, want %q", got, c.file)
			}
			mcp, _ := rec["mcp"].(map[string]any)
			if c.mcpServer == "" && mcp != nil || c.mcpServer != "" && (mcp == nil || mcp["server"] != c.mcpServer || (mcp["class"] != nil && mcp["class"] != c.mcpClass) || (mcp["class"] == nil && c.mcpClass != "")) {
				t.Errorf("mcp = %v, want server %q class %q", mcp, c.mcpServer, c.mcpClass)
			}
			for _, leak := range []string{repo, userHome, c.mustNotSee} {
				if leak != "" && strings.Contains(string(raw), leak) {
					t.Errorf("record leaks %q: %s", leak, raw)
				}
			}
		})
	}
}

// failingTransport fails the test on any HTTP request made through the default transport.
type failingTransport struct{ t *testing.T }

func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL)
	return nil, errors.New("network disabled in this test")
}

// The gate decides tool calls offline. Even logged in, with uploads and config checks due,
// PreToolUse, SessionStart and PostToolUse never touch the network; only Stop does.
func TestGateNeverUsesNetworkBeforeTools(t *testing.T) {
	home, srv := syncEnv(t)
	t.Setenv("PAWS_SYNC_INTERVAL", "0s")
	t.Setenv("PAWS_CONFIG_INTERVAL", "0s")
	writeRule(t, filepath.Join(home, "r.md"), "name: r\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: deploy\naction:\n  type: require-skill\n  skill: deployer\n")
	appendLines(t, filepath.Join(home, "audit.log"), actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls","command":"ls"`))
	oldTransport := http.DefaultTransport
	http.DefaultTransport = failingTransport{t}
	cwd := t.TempDir()

	for _, ev := range []map[string]any{
		{"hook_event_name": "PreToolUse", "tool_name": "Bash", "tool_input": map[string]any{"command": "deploy prod"}},
		{"hook_event_name": "PreToolUse", "tool_name": "Bash", "tool_input": map[string]any{"command": "ls"}},
		{"hook_event_name": "PreToolUse", "tool_name": "Skill", "tool_input": map[string]any{"skill": "deployer"}},
		{"hook_event_name": "SessionStart", "source": "startup"},
		{"hook_event_name": "PostToolUse", "tool_name": "Bash", "tool_input": map[string]any{"command": "ls"}},
	} {
		ev["cwd"], ev["session_id"] = cwd, "s1"
		runGate(t, ev)
	}
	http.DefaultTransport = oldTransport
	if n := srv.count(""); n != 0 {
		t.Fatalf("the gate made %d portal request(s): %v", n, srv.requests)
	}
	// Control: the same setup does reach the portal from the Stop hook.
	if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": cwd, "session_id": "s1"}); out != "{}" {
		t.Errorf("Stop must answer {}, got %q", out)
	}
	if srv.count("POST /paws/v1/sync") == 0 {
		t.Error("Stop should have uploaded")
	}
}

func setState(t *testing.T, edit func(*portalState)) {
	t.Helper()
	st := loadPortalState()
	edit(&st)
	if err := savePortalState(st); err != nil {
		t.Fatal(err)
	}
}

func TestStopHookThrottlesUploadAndConfigCheck(t *testing.T) {
	home, srv := syncEnv(t)
	srv.configHash = "h1"
	log := filepath.Join(home, "audit.log")
	stop := func() {
		t.Helper()
		if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir(), "session_id": "s1"}); out != "{}" {
			t.Fatalf("Stop must answer {}, got %q", out)
		}
	}
	appendLines(t, log, actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls"`))

	// First turn: upload; its response names a new config, which is fetched and applied at once.
	stop()
	if got := srv.requests; strings.Join(got, ",") != "POST /paws/v1/sync,GET /paws/v1/config,POST /paws/v1/sync" {
		t.Fatalf("first Stop requests: %v", got)
	}
	if st := loadPortalState(); st.AppliedHash != "h1" {
		t.Errorf("applied hash %q, want h1", st.AppliedHash)
	}

	// Within both intervals: nothing, even with a new record.
	appendLines(t, log, actionLine("2026-10-01T10:00:01Z", `"tool":"Bash","action":"pwd"`))
	stop()
	if n := srv.count(""); n != 3 {
		t.Fatalf("a Stop inside the intervals made requests: %v", srv.requests[3:])
	}

	// Upload interval passed: the new record goes up; the hash is unchanged, so no config fetch.
	setState(t, func(st *portalState) { st.LastUpload = time.Now().Add(-11 * time.Minute) })
	stop()
	if got := srv.requests[3:]; strings.Join(got, ",") != "POST /paws/v1/sync" {
		t.Fatalf("Stop after the upload interval: %v", got)
	}
	if recs := decodeSync(t, srv.bodies[len(srv.bodies)-1]); len(recs) != 1 || recs[0]["action"] != "pwd" {
		t.Errorf("uploaded %v, want the one new record", recs)
	}

	// Config interval passed with nothing to upload: only the hash is checked.
	setState(t, func(st *portalState) { st.LastConfigCheck = time.Now().Add(-16 * time.Minute) })
	stop()
	if got := srv.requests[4:]; strings.Join(got, ",") != "GET /paws/v1/config/hash" {
		t.Fatalf("Stop after the config interval: %v", got)
	}

	// The env vars override both intervals.
	t.Setenv("PAWS_SYNC_INTERVAL", "0s")
	t.Setenv("PAWS_CONFIG_INTERVAL", "1h")
	appendLines(t, log, actionLine("2026-10-01T10:00:02Z", `"tool":"Bash","action":"date"`))
	stop()
	if got := srv.requests[5:]; strings.Join(got, ",") != "POST /paws/v1/sync" {
		t.Fatalf("Stop with PAWS_SYNC_INTERVAL=0s: %v", got)
	}
}

func TestStopHookIsBestEffort(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		home, srv := syncEnv(t)
		srv.failWith = http.StatusInternalServerError
		appendLines(t, filepath.Join(home, "audit.log"), actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls"`))
		code, errOut := capture(t, &os.Stderr, func() int {
			if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()}); out != "{}" {
				t.Errorf("Stop must answer exactly {}, got %q", out)
			}
			return 0
		})
		if code != 0 || errOut != "" {
			t.Errorf("Stop must stay silent, stderr %q", errOut)
		}
		if srv.count("") == 0 {
			t.Error("Stop should have tried the portal")
		}
		if _, err := os.Stat(filepath.Join(home, "sync-state.json")); !os.IsNotExist(err) {
			t.Error("a failed upload must not move the cursor")
		}
		if st := loadPortalState(); st.AppliedHash != "" || st.LastUpload.IsZero() {
			t.Errorf("state after a failed round: %+v", st)
		}
	})
	t.Run("portal unreachable", func(t *testing.T) {
		home := portalEnv(t, http.NotFoundHandler())
		writePrivate(filepath.Join(home, "credentials.json"), credentials{API: "http://127.0.0.1:1", Token: "t"})
		appendLines(t, filepath.Join(home, "audit.log"), actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls"`))
		if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()}); out != "{}" {
			t.Errorf("Stop must answer {}, got %q", out)
		}
	})
	t.Run("logged out", func(t *testing.T) {
		srv := &fakePortal{t: t}
		home := portalEnv(t, srv)
		appendLines(t, filepath.Join(home, "audit.log"), actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls"`))
		runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()})
		if srv.count("") != 0 {
			t.Errorf("logged out, Stop made requests: %v", srv.requests)
		}
	})
}

// A require-skill rule that matches every tool call must not deny the Skill call that would
// satisfy it, or the agent can never get past it. The gate never gates the Skill tool.
func TestGateNeverDeniesTheSkillTool(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "catch-all.md"), "name: catch-all\nevent: pre-tool\naction:\n  type: require-skill\n  skill: deployer\n")
	writeRule(t, filepath.Join(home, "skill-only.md"), "name: skill-only\nevent: pre-tool\nmatcher:\n  tool: Skill\naction:\n  type: require-skill\n  skill: other\n")
	for _, skill := range []string{"deployer", "other", "unrelated"} {
		out := runGate(t, map[string]any{"hook_event_name": "PreToolUse", "cwd": t.TempDir(), "session_id": "s",
			"tool_name": "Skill", "tool_input": map[string]any{"skill": skill}})
		if strings.Contains(out, "deny") {
			t.Errorf("Skill(%s) was denied: %s", skill, out)
		}
	}
	// Control: the catch-all still gates other tools.
	if r := deny(t, "ls"); !strings.Contains(r, `"deployer"`) {
		t.Errorf("the catch-all should still gate Bash, got %q", r)
	}
}

// A 401 from the Stop hook logs the machine out: portal rules, state and credentials go, and
// `paws sync` then says why.
func TestStopHookForgetsARevokedLogin(t *testing.T) {
	home, srv := syncEnv(t)
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", backupRule())
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatal("setup sync failed")
	}
	if len(portalFiles(t, home)) == 0 {
		t.Fatal("setup: no portal rules written")
	}
	srv.failWith = http.StatusUnauthorized
	appendLines(t, filepath.Join(home, "audit.log"), actionLine("2026-10-01T10:00:00Z", `"tool":"Bash","action":"ls"`))
	setState(t, func(st *portalState) { st.LastUpload = time.Time{} })
	code, errOut := capture(t, &os.Stderr, func() int {
		if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()}); out != "{}" {
			t.Errorf("Stop must answer exactly {}, got %q", out)
		}
		return 0
	})
	if code != 0 || errOut != "" {
		t.Errorf("Stop must stay silent, stderr %q", errOut)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("portal rules left after a 401: %v", got)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Error("credentials.json kept after a 401")
	}
	if st := loadPortalState(); !st.Revoked || st.AppliedHash != "" {
		t.Errorf("state after a 401: %+v", st)
	}
	if deny(t, `psql -c "delete from users"`) != "" {
		t.Error("a revoked login's rule still denies")
	}
	code, errOut = capture(t, &os.Stderr, func() int { return Sync(nil) })
	if code != 1 || !strings.Contains(errOut, "Your login was revoked") {
		t.Errorf("sync after revocation: exit %d, stderr %q", code, errOut)
	}
}

// A skill download that hangs can't hold the Stop hook past its deadline, and leaves no
// temp or staging directory behind.
func TestStopHookHasADeadline(t *testing.T) {
	home, srv := syncEnv(t)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	pinned := "sha256:" + strings.Repeat("a", 64)
	pinCatalog(t, pinned)
	orig, origDeadline := pack.Fetch, stopHookDeadline
	fetched := false
	pack.Fetch = func(ctx context.Context, repo, ref, dest string) error {
		fetched = true
		<-ctx.Done()
		return ctx.Err()
	}
	stopHookDeadline = 200 * time.Millisecond
	t.Cleanup(func() { pack.Fetch, stopHookDeadline = orig, origDeadline })
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", humanizerSkill(pinned))
	setState(t, func(st *portalState) {})

	start := time.Now()
	runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()})
	if d := time.Since(start); !fetched || d > 5*time.Second {
		t.Fatalf("Stop took %v with a hung download (fetch called: %v)", d, fetched)
	}
	if left, _ := filepath.Glob(filepath.Join(tmp, "*")); len(left) != 0 {
		t.Errorf("temp dirs left behind: %v", left)
	}
	skills := filepath.Join(os.Getenv("HOME"), ".claude", "skills")
	if left, _ := filepath.Glob(filepath.Join(skills, "*")); len(left) != 0 {
		t.Errorf("skills dir after a failed download: %v", left)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("rules written for a skill that never installed: %v", got)
	}
}
