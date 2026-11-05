package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestUIHandler(t *testing.T) {
	t.Setenv("PAWS_HOME", t.TempDir())
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	h := uiHandler("127.0.0.1:4242")
	do := func(method, path, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Host = host
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := do("GET", "/", "127.0.0.1:4242")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>paws</title>") {
		t.Errorf("page: %d", rec.Code)
	}
	// The page loads its bundle from the embedded build, never from another origin.
	script := regexp.MustCompile(`src="\./(assets/[^"]+\.js)"`).FindStringSubmatch(rec.Body.String())
	if script == nil {
		t.Fatalf("page should load its bundle from ./assets: %s", rec.Body)
	}
	if rec := do("GET", "/"+script[1], "127.0.0.1:4242"); rec.Code != 200 || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Errorf("bundle %s: %d %q", script[1], rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := do("GET", "/api/catalog", "localhost:4242"); rec.Code != 200 {
		t.Errorf("catalog: %d %s", rec.Code, rec.Body)
	} else {
		var idx struct {
			Packs []struct{ Name string } `json:"packs"`
		}
		if json.Unmarshal(rec.Body.Bytes(), &idx) != nil || len(idx.Packs) == 0 {
			t.Errorf("catalog should list the embedded packs: %s", rec.Body)
		}
		if !strings.Contains(rec.Body.String(), `"repo": "github.com/blader/humanizer"`) && !strings.Contains(rec.Body.String(), `"repo":"github.com/blader/humanizer"`) {
			t.Errorf("index keys should be snake_case for the page and the website: %.300s", rec.Body)
		}
	}
	// Fresh install: no audit log yet. Lists must be [] so the page renders its empty states.
	if rec := do("GET", "/api/stats", "127.0.0.1:4242"); !strings.Contains(rec.Body.String(), `"rules":[]`) || !strings.Contains(rec.Body.String(), `"dormant":[]`) {
		t.Errorf("empty stats should serialise lists as []: %s", rec.Body)
	}
	if rec := do("GET", "/favicon.svg", "127.0.0.1:4242"); rec.Code != 200 {
		t.Errorf("favicon: %d", rec.Code)
	}
	if rec := do("GET", "/api/portal", "127.0.0.1:4242"); !strings.Contains(rec.Body.String(), `"logged_in":false`) || !strings.Contains(rec.Body.String(), `"skills":[]`) {
		t.Errorf("logged-out portal view: %s", rec.Body)
	}
	for _, p := range []string{"/api/installed", "/api/stats"} {
		if rec := do("GET", p, "127.0.0.1:4242"); rec.Code != 200 || !json.Valid(rec.Body.Bytes()) {
			t.Errorf("%s: %d %s", p, rec.Code, rec.Body)
		}
	}
	// DNS rebinding: a request for another host name is refused.
	if rec := do("GET", "/api/stats", "evil.example:4242"); rec.Code != http.StatusForbidden {
		t.Errorf("foreign Host should be refused, got %d", rec.Code)
	}
	if rec := do("POST", "/api/installed", "127.0.0.1:4242"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("writes should be refused, got %d", rec.Code)
	}
}

func TestUIPortalView(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PAWS_HOME", home)
	os.WriteFile(filepath.Join(home, "credentials.json"), []byte(`{"api":"https://paws-api.getunbound.ai","token":"t","account_email":"a@example.com"}`), 0o600)
	os.MkdirAll(filepath.Join(home, "portal"), 0o700)
	os.WriteFile(filepath.Join(home, "portal", "state.json"), []byte(`{"skills":["humanizer","humanizer--x"]}`), 0o600)
	// Skill humanizer--x's file also starts with "humanizer--"; it must not count for humanizer.
	os.WriteFile(filepath.Join(home, "portal", "humanizer--x--3.md"), []byte("---\nname: humanizer--x--3\nevent: pre-tool\nmatcher:\n  tool: Bash\naction:\n  type: require-skill\n  skill: humanizer--x\n---\n"), 0o600)
	os.WriteFile(filepath.Join(home, "portal", "humanizer--7.md"), []byte("---\nname: humanizer--7\nevent: pre-tool\nmatcher:\n  tool: Bash\naction:\n  type: require-skill\n  skill: humanizer\n---\n"), 0o600)
	v := portalView()
	if !v.LoggedIn || v.Account != "a@example.com" || v.SyncMinutes != 10 {
		t.Errorf("login state: %+v", v)
	}
	if len(v.Skills) != 2 || v.Skills[0].Name != "humanizer" || len(v.Skills[0].Bindings) != 1 || v.Skills[0].Bindings[0].Name != "humanizer--7" ||
		len(v.Skills[1].Bindings) != 1 || v.Skills[1].Bindings[0].Name != "humanizer--x--3" {
		t.Errorf("portal skills: %+v", v.Skills)
	}
}

func TestListenUIDefaultPortFallsBack(t *testing.T) {
	first, err := listenUI(-1)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if !strings.HasSuffix(first.Addr().String(), ":4747") {
		t.Skipf("default port already taken by something else: %s", first.Addr())
	}
	second, err := listenUI(-1)
	if err != nil {
		t.Fatalf("with the default port taken, ui should use another: %v", err)
	}
	defer second.Close()
	if strings.HasSuffix(second.Addr().String(), ":4747") {
		t.Errorf("second listener got the taken port: %s", second.Addr())
	}
	if _, err := listenUI(4747); err == nil {
		t.Error("an explicit --port that's taken should fail, not move")
	}
}
