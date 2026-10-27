package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

	if rec := do("GET", "/", "127.0.0.1:4242"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>paws</title>") {
		t.Errorf("page: %d", rec.Code)
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
