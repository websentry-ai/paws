package cli

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const deletedReply = `{"error":"account_deleted","message":"This account was deleted. Contact support to restore it."}`

// deletedPortal answers every request the way the portal answers a deleted account's laptop or login.
type deletedPortal struct{ requests []string }

func (d *deletedPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.requests = append(d.requests, r.Method+" "+r.URL.Path)
	if r.URL.Path == "/paws/v1/device/code" {
		io.WriteString(w, `{"device_code":"dev-123","user_code":"WXYZ-4823","verification_uri":"https://portal.example/device","interval":3,"expires_in":600}`)
		return
	}
	w.WriteHeader(http.StatusForbidden)
	io.WriteString(w, deletedReply)
}

// loggedInWithRule logs this machine in and gives it one portal rule, as a sync would have.
func loggedInWithRule(t *testing.T, home string) string {
	t.Helper()
	api := strings.TrimSuffix(os.Getenv("PAWS_PORTAL_API"), "/")
	if err := writePrivate(filepath.Join(home, "credentials.json"), credentials{API: api, Token: "tok-abc"}); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(home, "portal", "git--git.md")
	os.MkdirAll(filepath.Dir(rule), 0o700)
	if err := os.WriteFile(rule, []byte("---\nname: git--git\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return rule
}

func assertLoggedOutAsDeleted(t *testing.T, home, rule string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
		t.Error("the credentials must be removed")
	}
	if _, err := os.Stat(rule); !os.IsNotExist(err) {
		t.Error("the portal rules must be removed")
	}
	if st := loadPortalState(); !st.Revoked || !st.AccountDeleted {
		t.Errorf("state must remember the account was deleted: %+v", st)
	}
}

func TestSyncForADeletedAccountLogsOutAndSaysWhy(t *testing.T) {
	home := portalEnv(t, &deletedPortal{})
	rule := loggedInWithRule(t, home)

	code, errOut := capture(t, &os.Stderr, func() int { return Sync(nil) })
	if code != 1 || !strings.Contains(errOut, "This account was deleted. Contact support to restore it.") ||
		!strings.Contains(errOut, "paws logged this laptop out and removed its portal rules.") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if strings.Contains(errOut, "revoked") {
		t.Errorf("a deleted account must not read as a revoked login: %q", errOut)
	}
	assertLoggedOutAsDeleted(t, home, rule)

	// The next sync, now logged out, still says why.
	code, errOut = capture(t, &os.Stderr, func() int { return Sync(nil) })
	if code != 1 || !strings.Contains(errOut, "This account was deleted. Contact support to restore it.") {
		t.Errorf("second sync: exit %d, stderr %q", code, errOut)
	}
}

func TestBackgroundRoundForADeletedAccountLogsOut(t *testing.T) {
	home := portalEnv(t, &deletedPortal{})
	rule := loggedInWithRule(t, home)
	syncRound(t.TempDir(), 0)
	assertLoggedOutAsDeleted(t, home, rule)
}

func TestLoginToADeletedAccountFailsWithoutSavingCredentials(t *testing.T) {
	for name, args := range map[string][]string{"browser": nil, "key": {"--key", "pk_one-time"}} {
		t.Run(name, func(t *testing.T) {
			portal := &deletedPortal{}
			home := portalEnv(t, portal)
			var code int
			errOut := ""
			captureStdout(t, func() int {
				code, errOut = capture(t, &os.Stderr, func() int { return Login(args) })
				return code
			})
			last := strings.TrimSpace(errOut)
			last = last[strings.LastIndex(last, "\n")+1:] // after the plain-http warning for the test server
			if code != 1 || last != "This account was deleted. Contact support to restore it." {
				t.Errorf("exit %d, stderr %q", code, errOut)
			}
			if _, err := os.Stat(filepath.Join(home, "credentials.json")); !os.IsNotExist(err) {
				t.Error("no credentials may be written")
			}
			for _, r := range portal.requests {
				if strings.HasSuffix(r, "/sync") {
					t.Errorf("no sync may run after a refused login: %v", portal.requests)
				}
			}
		})
	}
}
