package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/websentry-ai/paws/internal/gate"
)

// The portal is opt-in. Only `paws login`, `paws sync` and, once logged in, the async Stop
// hook talk to it. The PreToolUse gate never does.

// defaultPortalAPI is a placeholder until the hosted portal's address is final.
const defaultPortalAPI = "https://api.paws.getunbound.ai"

// Test seams: tests replace these so polling doesn't wait and no browser opens.
var (
	sleep   = time.Sleep
	openURL = openBrowser
)

// portalClient and portalCtx carry every portal request; the Stop hook swaps in a shorter
// per-request timeout and an overall deadline (portalCtx also bounds skill downloads).
var (
	portalClient = &http.Client{Timeout: 30 * time.Second}
	portalCtx    = context.Background()
)

func portalAPI() string {
	if v := os.Getenv("PAWS_PORTAL_API"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultPortalAPI
}

// checkPortalAPI allows https, and plain http only to this machine: the token and every
// upload ride on it. loopback reports the http case, which login warns about.
func checkPortalAPI(api string) (loopback bool, err error) {
	u, err := url.Parse(api)
	if err != nil || u.Host == "" {
		return false, fmt.Errorf("portal address %q is not a URL", api)
	}
	switch host := u.Hostname(); {
	case u.Scheme == "https":
		return false, nil
	case u.Scheme == "http" && (host == "localhost" || host == "127.0.0.1" || host == "::1"):
		return true, nil
	}
	return false, fmt.Errorf("portal address %q must use https (plain http is allowed for localhost only)", api)
}

// credentials is $PAWS_HOME/credentials.json, written by `paws login`.
type credentials struct {
	API       string          `json:"api"`
	Token     string          `json:"token"`
	DeviceID  json.RawMessage `json:"device_id,omitempty"` // kept as the portal sent it
	CreatedAt string          `json:"created_at"`
}

func credentialsPath() string {
	dir := gate.UserDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "credentials.json")
}

// loadCredentials returns nil, nil when the user never logged in.
func loadCredentials() (*credentials, error) {
	path := credentialsPath()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s is unreadable (%v); run `paws login` again", path, err)
	}
	if c.Token == "" || c.API == "" {
		return nil, nil
	}
	if _, err := checkPortalAPI(c.API); err != nil {
		return nil, fmt.Errorf("%s: %v; run `paws login` again", path, err)
	}
	return &c, nil
}

// writePrivate writes a 0600 file in a 0700 directory: the token is a secret.
func writePrivate(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600) // WriteFile keeps the mode of an existing, looser file
}

// postJSON posts body to url and returns the status and response body.
func postJSON(url, token string, body any) (int, []byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return portalDo(req, token)
}

// getJSON fetches url and returns the status and response body.
func getJSON(url, token string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	return portalDo(req, token)
}

func portalDo(req *http.Request, token string) (int, []byte, error) {
	req = req.WithContext(portalCtx)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := portalClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, maxPortalResponse))
	return resp.StatusCode, out, err
}

// maxPortalResponse bounds what paws reads from the portal; a config with many rule texts fits.
const maxPortalResponse = 8 << 20

// portalError turns an unexpected response into an error, using the portal's {"error": "..."}.
func portalError(status int, body []byte) error {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return fmt.Errorf("portal: %s (HTTP %d)", printable(e.Error), status)
	}
	return fmt.Errorf("portal: unexpected HTTP %d", status)
}

type deviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

type deviceToken struct {
	Status   string          `json:"status"`
	Token    string          `json:"token"`
	DeviceID json.RawMessage `json:"device_id"`
}

// printable drops control characters (escape sequences, newlines) from text the portal sent,
// so a server can't rewrite the terminal or forge extra output lines.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// Login links this machine to a portal account: `paws login` runs the device flow in the
// browser; `paws login --key <one-time-key>` connects a machine with no browser (`--key -`
// reads the key from stdin, keeping it out of shell history). Then it syncs once, so the
// portal sees this machine's skills right away.
func Login(args []string) int {
	key, ok := loginKey(args)
	if !ok {
		fmt.Fprintln(os.Stderr, "usage: paws login [--key <one-time-key> | --key -]")
		return 2
	}
	path := credentialsPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, "paws: no home directory for credentials.json; set PAWS_HOME")
		return 1
	}
	api := portalAPI()
	loopback, err := checkPortalAPI(api)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	if loopback {
		fmt.Fprintf(os.Stderr, "paws: warning: %s is plain http; use it only for a local portal\n", api)
	}
	if key == "-" {
		if key, err = readKey(os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
	}
	var tok deviceToken
	if key != "" {
		tok, err = redeemKey(api, key)
	} else {
		tok, err = deviceLogin(api)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	c := credentials{API: api, Token: tok.Token, DeviceID: tok.DeviceID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := writePrivate(path, c); err != nil {
		fmt.Fprintf(os.Stderr, "paws: saving credentials: %v\n", err)
		return 1
	}
	if st := loadPortalState(); st.Revoked {
		st.Revoked = false
		savePortalState(st)
	}
	fmt.Println("Logged in.")
	cwd, _ := os.Getwd()
	rep, err := syncNow(&c, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: first sync failed: %v (run `paws sync` to retry)\n", err)
		return 0
	}
	printSyncReport(rep)
	return 0
}

// loginKey reads `--key K` or `--key=K`; no arguments means the device flow.
func loginKey(args []string) (string, bool) {
	switch {
	case len(args) == 0:
		return "", true
	case len(args) == 2 && args[0] == "--key" && args[1] != "":
		return args[1], true
	case len(args) == 1 && strings.HasPrefix(args[0], "--key=") && len(args[0]) > len("--key="):
		return strings.TrimPrefix(args[0], "--key="), true
	}
	return "", false
}

// readKey reads a one-time key from r (`paws login --key -`): the first line, trimmed.
func readKey(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil {
		return "", fmt.Errorf("reading the key from stdin: %v", err)
	}
	line, _, _ := strings.Cut(string(data), "\n")
	if key := strings.TrimSpace(line); key != "" {
		return key, nil
	}
	return "", errors.New("no key on stdin")
}

// deviceLogin shows a code to confirm in the browser and waits for the confirmation.
func deviceLogin(api string) (deviceToken, error) {
	dc, err := requestDeviceCode(api)
	if err != nil {
		return deviceToken{}, err
	}
	fmt.Printf("Open %s and confirm code %s\n", printable(dc.VerificationURI), printable(dc.UserCode))
	if u, err := url.Parse(dc.VerificationURI); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
		openURL(dc.VerificationURI)
	}
	fmt.Println("Waiting for confirmation (Ctrl-C to cancel)...")
	return pollDeviceToken(api, dc)
}

// redeemKey trades a one-time key made in the portal for a device token.
func redeemKey(api, key string) (deviceToken, error) {
	body := deviceInfo()
	body["key"] = key
	var tok deviceToken
	status, resp, err := postJSON(api+"/paws/v1/device/key", "", body)
	if err != nil {
		return tok, err
	}
	switch status {
	case http.StatusOK:
		if json.Unmarshal(resp, &tok) == nil && tok.Status == "ok" && tok.Token != "" {
			return tok, nil
		}
		return tok, errors.New("portal: malformed token response")
	case http.StatusGone:
		return tok, errors.New("this key was already used or has expired; make a new one in the portal")
	default:
		return tok, portalError(status, resp)
	}
}

// deviceInfo names this machine to the portal.
func deviceInfo() map[string]string {
	host, _ := os.Hostname()
	return map[string]string{"device_name": host, "os": runtime.GOOS, "paws_version": Version}
}

func requestDeviceCode(api string) (deviceCode, error) {
	var dc deviceCode
	status, resp, err := postJSON(api+"/paws/v1/device/code", "", deviceInfo())
	if err != nil {
		return dc, err
	}
	if status != http.StatusOK {
		return dc, portalError(status, resp)
	}
	if err := json.Unmarshal(resp, &dc); err != nil || dc.DeviceCode == "" || dc.UserCode == "" {
		return dc, fmt.Errorf("portal: malformed device code response")
	}
	return dc, nil
}

// maxDeviceWait bounds the device flow whatever expires_in the portal sends.
const maxDeviceWait = 15 * time.Minute

// pollDeviceToken asks every interval seconds (default 5, at most 60) until the code is
// confirmed, denied or expired, giving up once expires_in (at most 15 minutes) has passed.
func pollDeviceToken(api string, dc deviceCode) (deviceToken, error) {
	interval := time.Duration(min(dc.Interval, 60)) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	limit := time.Duration(dc.ExpiresIn) * time.Second
	if limit <= 0 || limit > maxDeviceWait {
		limit = maxDeviceWait
	}
	for waited := time.Duration(0); waited < limit; waited += interval {
		sleep(interval)
		var tok deviceToken
		status, resp, err := postJSON(api+"/paws/v1/device/token", "", map[string]string{"device_code": dc.DeviceCode})
		if err != nil {
			return tok, err
		}
		json.Unmarshal(resp, &tok)
		switch status {
		case http.StatusPreconditionRequired: // 428: not confirmed yet
			continue
		case http.StatusForbidden:
			return tok, errors.New("the login was denied")
		case http.StatusGone:
			return tok, errors.New("the code expired; run `paws login` again")
		case http.StatusOK:
			if tok.Status == "ok" && tok.Token != "" {
				return tok, nil
			}
			return tok, errors.New("portal: malformed token response")
		default:
			return tok, portalError(status, resp)
		}
	}
	return deviceToken{}, errors.New("the code expired; run `paws login` again")
}

// Logout forgets the portal token and everything the portal config put in place: `paws logout`.
func Logout(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: paws logout")
		return 2
	}
	path := credentialsPath()
	if path == "" {
		fmt.Println("Not logged in.")
		return 0
	}
	if err := clearPortal(); err != nil {
		fmt.Fprintf(os.Stderr, "paws: removing portal rules: %v\n", err)
		return 1
	}
	err := os.Remove(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		fmt.Println("Not logged in.")
	case err != nil:
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	default:
		fmt.Println("Logged out.")
	}
	return 0
}
