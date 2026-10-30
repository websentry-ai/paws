package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// portalEnv points PAWS_HOME and PAWS_PORTAL_API at temp dirs and a test server, makes polling
// instant and keeps the browser shut. It returns PAWS_HOME (not created yet).
func portalEnv(t *testing.T, h http.Handler) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "paws-home")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", "") // skills install under the temp HOME, never a real config dir
	t.Setenv("PAWS_HOME", home)
	for _, env := range []string{"PAWS_SYNC_INTERVAL", "PAWS_CONFIG_INTERVAL"} {
		t.Setenv(env, "")
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("PAWS_PORTAL_API", srv.URL+"/")
	// The inventory reads the working directory's repo: keep it out of this checkout.
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	oldSleep, oldOpen := sleep, openURL
	sleep, openURL = func(time.Duration) {}, func(string) {}
	t.Cleanup(func() { sleep, openURL = oldSleep, oldOpen })
	return home
}

// captureStdout runs fn and returns its exit code and what it printed to stdout.
func captureStdout(t *testing.T, fn func() int) (int, string) {
	return capture(t, &os.Stdout, fn)
}

// capture runs fn with *f redirected and returns its exit code and what went to *f.
func capture(t *testing.T, f **os.File, fn func() int) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := *f
	*f = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	code := fn()
	*f = old
	w.Close()
	return code, <-done
}

type tokenReply struct {
	status int
	body   string
}

// deviceFlow serves /device/code and answers successive /device/token polls with replies.
// Anything else (the first sync after login) goes to portal.
type deviceFlow struct {
	t        *testing.T
	code     string // the /device/code reply; "" is a well-formed one
	replies  []tokenReply
	polls    int
	codeBody map[string]string
	portal   *fakePortal
}

func (d *deviceFlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/paws/v1/device/") {
		if d.portal == nil {
			d.portal = &fakePortal{t: d.t}
		}
		d.portal.ServeHTTP(w, r)
		return
	}
	if r.Method != http.MethodPost {
		d.t.Errorf("%s %s: want POST", r.Method, r.URL.Path)
	}
	switch r.URL.Path {
	case "/paws/v1/device/code":
		json.NewDecoder(r.Body).Decode(&d.codeBody)
		if d.code != "" {
			io.WriteString(w, d.code)
			return
		}
		io.WriteString(w, `{"device_code":"dev-123","user_code":"WXYZ-4823","verification_uri":"https://portal.example/device","interval":3,"expires_in":600}`)
	case "/paws/v1/device/token":
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["device_code"] != "dev-123" {
			d.t.Errorf("token poll sent device_code %q", body["device_code"])
		}
		rep := d.replies[min(d.polls, len(d.replies)-1)]
		d.polls++
		w.WriteHeader(rep.status)
		io.WriteString(w, rep.body)
	default:
		d.t.Errorf("unexpected path %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestLoginPollsUntilConfirmed(t *testing.T) {
	flow := &deviceFlow{t: t, replies: []tokenReply{
		{428, `{"status":"pending"}`},
		{428, `{"status":"pending"}`},
		{200, `{"status":"ok","token":"tok-abc","device_id":"d-1"}`},
	}}
	home := portalEnv(t, flow)
	var slept []time.Duration
	var opened string
	sleep = func(d time.Duration) { slept = append(slept, d) }
	openURL = func(u string) { opened = u }

	code, out := captureStdout(t, func() int { return Login(nil) })
	if code != 0 {
		t.Fatalf("login exit %d, output %q", code, out)
	}
	if !strings.Contains(out, "Open https://portal.example/device and confirm code WXYZ-4823") || !strings.Contains(out, "Logged in.\nNothing new to upload.\nConfig: no skills applied.") {
		t.Errorf("login output: %q", out)
	}
	// The first sync runs at once, with the new token, so the portal sees this machine.
	if flow.portal == nil || len(flow.portal.auths) == 0 || flow.portal.auths[0] != "Bearer tok-abc" {
		t.Errorf("login must sync once with the new token")
	}
	if opened != "https://portal.example/device" {
		t.Errorf("browser opened %q", opened)
	}
	if flow.polls != 3 || len(slept) != 3 || slept[0] != 3*time.Second {
		t.Errorf("polls=%d slept=%v; want 3 polls 3s apart", flow.polls, slept)
	}
	host, _ := os.Hostname()
	if flow.codeBody["device_name"] != host || flow.codeBody["os"] != runtime.GOOS || flow.codeBody["paws_version"] != Version {
		t.Errorf("device code request body: %v", flow.codeBody)
	}

	path := filepath.Join(home, "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	if c.Token != "tok-abc" || string(c.DeviceID) != `"d-1"` || c.API != strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/") || c.CreatedAt == "" {
		t.Errorf("credentials: %+v", c)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("credentials.json mode %v, want 0600", st.Mode().Perm())
	}
	if st, _ := os.Stat(home); st.Mode().Perm() != 0o700 {
		t.Errorf("PAWS_HOME mode %v, want 0700", st.Mode().Perm())
	}
}

func TestLoginStopsOnDeniedOrExpired(t *testing.T) {
	for name, final := range map[string]tokenReply{
		"denied":  {403, `{"status":"denied"}`},
		"expired": {410, `{"status":"expired"}`},
		"error":   {500, `{"error":"boom"}`},
	} {
		t.Run(name, func(t *testing.T) {
			flow := &deviceFlow{t: t, replies: []tokenReply{{428, `{"status":"pending"}`}, final, {200, `{"status":"ok","token":"x"}`}}}
			home := portalEnv(t, flow)
			code, _ := captureStdout(t, func() int { return Login(nil) })
			if code != 1 {
				t.Errorf("exit %d, want 1", code)
			}
			if flow.polls != 2 {
				t.Errorf("polled %d times; must stop at the %s reply", flow.polls, name)
			}
			if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
				t.Errorf("no credentials may be written on %s", name)
			}
		})
	}
}

// The device flow gives up once expires_in has passed (at most 15 minutes), polling at most
// every 60 seconds, however the portal answers.
func TestLoginStopsPollingAtExpiry(t *testing.T) {
	for name, c := range map[string]struct {
		code      string
		polls     int
		firstWait time.Duration
	}{
		"expires_in":       {`{"device_code":"dev-123","user_code":"U","verification_uri":"https://p/d","interval":10,"expires_in":30}`, 3, 10 * time.Second},
		"capped at 15 min": {`{"device_code":"dev-123","user_code":"U","verification_uri":"https://p/d","interval":5,"expires_in":999999}`, 180, 5 * time.Second},
		"no expiry":        {`{"device_code":"dev-123","user_code":"U","verification_uri":"https://p/d"}`, 180, 5 * time.Second},
		"interval clamped": {`{"device_code":"dev-123","user_code":"U","verification_uri":"https://p/d","interval":3600,"expires_in":600}`, 10, 60 * time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			flow := &deviceFlow{t: t, code: c.code, replies: []tokenReply{{428, `{"status":"pending"}`}}}
			home := portalEnv(t, flow)
			var slept []time.Duration
			sleep = func(d time.Duration) { slept = append(slept, d) }
			code, errOut := capture(t, &os.Stderr, func() int {
				code, _ := captureStdout(t, func() int { return Login(nil) })
				return code
			})
			if code != 1 || !strings.Contains(errOut, "the code expired") {
				t.Errorf("exit %d, stderr %q", code, errOut)
			}
			if flow.polls != c.polls || len(slept) == 0 || slept[0] != c.firstWait {
				t.Errorf("polls=%d first wait=%v; want %d polls %v apart", flow.polls, slept[:min(1, len(slept))], c.polls, c.firstWait)
			}
			if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
				t.Error("credentials written for an expired code")
			}
		})
	}
}

// What the portal sends for display can't carry terminal escapes or extra lines.
func TestLoginStripsControlCharacters(t *testing.T) {
	flow := &deviceFlow{t: t, code: `{"device_code":"dev-123","user_code":"AB\u001b[2JCD\nforged","verification_uri":"https://p/d\u001b]8;;x","interval":1,"expires_in":5}`,
		replies: []tokenReply{{500, `{"error":"bad\u001b[31m\nforged"}`}}}
	portalEnv(t, flow)
	var out string
	_, errOut := capture(t, &os.Stderr, func() int {
		var code int
		code, out = captureStdout(t, func() int { return Login(nil) })
		return code
	})
	if !strings.Contains(out, "Open https://p/d]8;;x and confirm code AB[2JCDforged\n") {
		t.Errorf("stdout %q", out)
	}
	if !strings.Contains(errOut, "portal: bad[31mforged (HTTP 500)") {
		t.Errorf("stderr %q", errOut)
	}
	if strings.ContainsRune(out+errOut, 0x1b) {
		t.Errorf("an escape reached the terminal: %q %q", out, errOut)
	}
}

// The portal address must be https, except plain http to this machine, which warns.
func TestLoginRequiresHTTPS(t *testing.T) {
	srv := &fakePortal{t: t}
	home := portalEnv(t, srv)
	for _, api := range []string{"http://portal.example.com", "http://127.0.0.1.evil.com", "ftp://localhost", "portal.example.com"} {
		t.Setenv("PAWS_PORTAL_API", api)
		code, errOut := capture(t, &os.Stderr, func() int { return Login([]string{"--key", "pk_abc"}) })
		if code != 1 || !strings.Contains(errOut, "must use https") && !strings.Contains(errOut, "not a URL") {
			t.Errorf("%s: exit %d, stderr %q", api, code, errOut)
		}
	}
	if srv.count("") != 0 {
		t.Errorf("requests made to a refused address: %v", srv.requests)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Error("credentials written for a refused address")
	}
	// Loopback http works, with a warning.
	t.Setenv("PAWS_PORTAL_API", strings.Replace(srvURL(t, srv), "127.0.0.1", "localhost", 1))
	code, errOut := capture(t, &os.Stderr, func() int {
		code, _ := captureStdout(t, func() int { return Login([]string{"--key", "pk_abc"}) })
		return code
	})
	if code != 0 || !strings.Contains(errOut, "is plain http; use it only for a local portal") {
		t.Errorf("loopback http: exit %d, stderr %q", code, errOut)
	}
}

// srvURL starts another server for h and returns its URL.
func srvURL(t *testing.T, h http.Handler) string {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s.URL
}

// Credentials saved for a plain-http address are never sent to a host that isn't this machine.
func TestSyncRefusesHTTPCredentialsForARemoteHost(t *testing.T) {
	home, srv := syncEnv(t)
	writePrivate(filepath.Join(home, "credentials.json"), credentials{API: "http://portal.example.com", Token: "tok"})
	code, errOut := capture(t, &os.Stderr, func() int {
		code, _ := captureStdout(t, func() int { return Sync(nil) })
		return code
	})
	if code != 1 || !strings.Contains(errOut, "must use https") || !strings.Contains(errOut, "run `paws login` again") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	// The Stop hook stays silent and offline too.
	oldTransport := http.DefaultTransport
	http.DefaultTransport = failingTransport{t}
	defer func() { http.DefaultTransport = oldTransport }()
	if out := runGate(t, map[string]any{"hook_event_name": "Stop", "cwd": t.TempDir()}); out != "{}" {
		t.Errorf("Stop must answer {}, got %q", out)
	}
	if srv.count("") != 0 {
		t.Errorf("requests made: %v", srv.requests)
	}
}

// `paws login --key -` reads the key from stdin, keeping it out of shell history.
func TestLoginReadsKeyFromStdin(t *testing.T) {
	srv := &fakePortal{t: t}
	portalEnv(t, srv)
	withStdin := func(data string, fn func() int) int {
		path := filepath.Join(t.TempDir(), "stdin")
		os.WriteFile(path, []byte(data), 0o600)
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		old := os.Stdin
		os.Stdin = f
		defer func() { os.Stdin = old }()
		return fn()
	}
	code, errOut := capture(t, &os.Stderr, func() int {
		return withStdin("\n", func() int { return Login([]string{"--key", "-"}) })
	})
	if code != 1 || !strings.Contains(errOut, "no key on stdin") {
		t.Errorf("empty stdin: exit %d, stderr %q", code, errOut)
	}
	code = withStdin("  pk_abc  \n", func() int {
		code, _ := captureStdout(t, func() int { return Login([]string{"--key", "-"}) })
		return code
	})
	if code != 0 || srv.keyBody["key"] != "pk_abc" {
		t.Errorf("exit %d, key sent %q", code, srv.keyBody["key"])
	}
}

// Logging out removes every rule and digest the portal config wrote, and its state.
func TestLogoutRemovesPortalRules(t *testing.T) {
	home, srv := syncEnv(t)
	srv.configHash = "h1"
	srv.config = configJSON(t, "h1", backupRule())
	if code, _ := captureStdout(t, func() int { return Sync(nil) }); code != 0 || len(portalFiles(t, home)) == 0 {
		t.Fatal("setup: sync wrote no portal rules")
	}
	if deny(t, `psql -c "delete from users"`) == "" {
		t.Fatal("setup: the portal rule should deny")
	}
	if code, out := captureStdout(t, func() int { return Logout(nil) }); code != 0 || !strings.Contains(out, "Logged out.") {
		t.Fatalf("logout exit %d, output %q", code, out)
	}
	if got := portalFiles(t, home); len(got) != 0 {
		t.Errorf("portal rules left after logout: %v", got)
	}
	if _, err := os.Stat(filepath.Join(home, "portal", "state.json")); !os.IsNotExist(err) {
		t.Error("state.json left after logout")
	}
	if deny(t, `psql -c "delete from users"`) != "" {
		t.Error("a portal rule still denies after logout")
	}
}

func TestLogoutDeletesCredentials(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writePrivate(filepath.Join(home, "credentials.json"), credentials{API: "http://x", Token: "t"})
	if code, out := captureStdout(t, func() int { return Logout(nil) }); code != 0 || !strings.Contains(out, "Logged out.") {
		t.Errorf("logout exit %d, output %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Error("credentials.json still exists after logout")
	}
	if code, out := captureStdout(t, func() int { return Logout(nil) }); code != 0 || !strings.Contains(out, "Not logged in.") {
		t.Errorf("second logout exit %d, output %q", code, out)
	}
}

func TestLoginWithOneTimeKey(t *testing.T) {
	srv := &fakePortal{t: t}
	home := portalEnv(t, srv)
	code, out := captureStdout(t, func() int { return Login([]string{"--key", "pk_abc"}) })
	if code != 0 || !strings.Contains(out, "Logged in.\nNothing new to upload.") {
		t.Fatalf("login --key exit %d, output %q", code, out)
	}
	host, _ := os.Hostname()
	if srv.keyBody["key"] != "pk_abc" || srv.keyBody["device_name"] != host || srv.keyBody["os"] != runtime.GOOS || srv.keyBody["paws_version"] != Version {
		t.Errorf("POST /device/key body: %v", srv.keyBody)
	}
	c, err := loadCredentials()
	if err != nil || c == nil || c.Token != "tok-key" || string(c.DeviceID) != "7" {
		t.Fatalf("credentials: %+v %v", c, err)
	}
	if st, _ := os.Stat(filepath.Join(home, "credentials.json")); st.Mode().Perm() != 0o600 {
		t.Errorf("credentials.json mode %v, want 0600", st.Mode().Perm())
	}
	if len(srv.auths) == 0 || srv.auths[0] != "Bearer tok-key" {
		t.Errorf("login --key must sync once with the new token, auths %v", srv.auths)
	}
}

func TestLoginWithUsedKeyFails(t *testing.T) {
	srv := &fakePortal{t: t}
	home := portalEnv(t, srv)
	code, errOut := capture(t, &os.Stderr, func() int { return Login([]string{"--key=pk_used"}) })
	if code != 1 || !strings.Contains(errOut, "already used or has expired") {
		t.Errorf("used key: exit %d, stderr %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Error("no credentials may be written for a used key")
	}
	if code, _ := capture(t, &os.Stderr, func() int { return Login([]string{"--key"}) }); code != 2 {
		t.Errorf("--key with no value: exit %d, want 2", code)
	}
}

// A portal that names the account: login says which one (both flows), credentials.json keeps
// it, and `paws sync` leads with it. Control characters never reach the terminal or the file.
func TestLoginShowsTheAccount(t *testing.T) {
	for name, login := range map[string]func(t *testing.T) func() int{
		"device flow": func(t *testing.T) func() int {
			portalEnv(t, &deviceFlow{t: t, replies: []tokenReply{
				{200, `{"status":"ok","token":"tok-abc","device_id":"d-1","account_email":"alice@example.com\u001b[2J\n"}`},
			}})
			return func() int { return Login(nil) }
		},
		"one-time key": func(t *testing.T) func() int {
			portalEnv(t, &fakePortal{t: t, keyEmail: "alice@example.com\u001b[2J\n"})
			return func() int { return Login([]string{"--key", "pk_abc"}) }
		},
	} {
		t.Run(name, func(t *testing.T) {
			code, out := captureStdout(t, login(t))
			if code != 0 || !strings.Contains(out, "Logged in as alice@example.com[2J.\nNothing new to upload.") {
				t.Fatalf("login exit %d, output %q", code, out)
			}
			c, err := loadCredentials()
			if err != nil || c == nil || c.AccountEmail != "alice@example.com[2J" {
				t.Fatalf("credentials: %+v %v", c, err)
			}
			code, out = captureStdout(t, func() int { return Sync(nil) })
			if code != 0 || !strings.HasPrefix(out, "Account: alice@example.com[2J\nNothing new to upload.") {
				t.Errorf("sync exit %d, output %q", code, out)
			}
			code, out = captureStdout(t, func() int { return Sync([]string{"--dry-run"}) })
			if code != 0 || !strings.HasPrefix(out, "{") {
				t.Errorf("sync --dry-run must print only JSON on stdout, exit %d, output %q", code, out)
			}
			if strings.ContainsRune(out, 0x1b) {
				t.Errorf("an escape reached the terminal: %q", out)
			}
		})
	}
}

// An older portal sends no account_email: login and sync print what they always did.
func TestLoginWithoutAccountEmail(t *testing.T) {
	home := portalEnv(t, &fakePortal{t: t})
	if code, out := captureStdout(t, func() int { return Login([]string{"--key", "pk_abc"}) }); code != 0 || !strings.Contains(out, "Logged in.\nNothing new to upload.") {
		t.Fatalf("login exit %d, output %q", code, out)
	}
	data, _ := os.ReadFile(filepath.Join(home, "credentials.json"))
	if strings.Contains(string(data), "account_email") {
		t.Errorf("credentials.json: %s", data)
	}
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 || !strings.HasPrefix(out, "Nothing new to upload.") {
		t.Errorf("sync exit %d, output %q", code, out)
	}
}
