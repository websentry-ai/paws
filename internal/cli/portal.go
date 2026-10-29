package cli

import (
	"bytes"
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

	"github.com/websentry-ai/paws/internal/gate"
)

// The portal is opt-in: only `paws login` and `paws sync` talk to it. The hook never does.

// defaultPortalAPI is a placeholder until the hosted portal's address is final.
const defaultPortalAPI = "http://localhost:8000"

// Test seams: tests replace these so polling doesn't wait and no browser opens.
var (
	sleep   = time.Sleep
	openURL = openBrowser
)

var portalClient = &http.Client{Timeout: 30 * time.Second}

func portalAPI() string {
	if v := os.Getenv("PAWS_PORTAL_API"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultPortalAPI
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := portalClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, out, err
}

// portalError turns an unexpected response into an error, using the portal's {"error": "..."}.
func portalError(status int, body []byte) error {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return fmt.Errorf("portal: %s (HTTP %d)", e.Error, status)
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

// Login links this machine to a portal account with the device flow: `paws login`.
func Login(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: paws login")
		return 2
	}
	path := credentialsPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, "paws: no home directory for credentials.json; set PAWS_HOME")
		return 1
	}
	api := portalAPI()
	dc, err := requestDeviceCode(api)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	fmt.Printf("Open %s and confirm code %s\n", dc.VerificationURI, dc.UserCode)
	if u, err := url.Parse(dc.VerificationURI); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
		openURL(dc.VerificationURI)
	}
	fmt.Println("Waiting for confirmation (Ctrl-C to cancel)...")
	tok, err := pollDeviceToken(api, dc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	c := credentials{API: api, Token: tok.Token, DeviceID: tok.DeviceID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := writePrivate(path, c); err != nil {
		fmt.Fprintf(os.Stderr, "paws: saving credentials: %v\n", err)
		return 1
	}
	fmt.Println("Logged in. Run `paws sync` to upload counts.")
	return 0
}

func requestDeviceCode(api string) (deviceCode, error) {
	host, _ := os.Hostname()
	body := map[string]string{"device_name": host, "os": runtime.GOOS, "paws_version": Version}
	var dc deviceCode
	status, resp, err := postJSON(api+"/paws/v1/device/code", "", body)
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

// pollDeviceToken asks every interval seconds until the code is confirmed, denied or expired.
func pollDeviceToken(api string, dc deviceCode) (deviceToken, error) {
	interval := time.Duration(dc.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for {
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
}

// Logout forgets the portal token: `paws logout`.
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
