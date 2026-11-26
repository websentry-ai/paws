package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSetContentHashTouchesOnlyUpstreamBlock(t *testing.T) {
	in := "---\nname: x\nupstream:\n  repo: github.com/a/b\n  path: SKILL.md\n  ref: 0123\nprovenance:\n  ref: not-this-one\ntags: [a]\n---\nBody mentions ref: and content_hash: too.\n"
	out, ok := setContentHash(in, "sha256:abc")
	if !ok || strings.Count(out, "content_hash: sha256:abc") != 1 || !strings.Contains(out, "  ref: 0123\n  content_hash: sha256:abc\nprovenance:") {
		t.Fatalf("insert went wrong:\n%s", out)
	}
	again, ok := setContentHash(out, "sha256:def")
	if !ok || strings.Contains(again, "sha256:abc") || strings.Count(again, "content_hash: sha256:def") != 1 || !strings.Contains(again, "Body mentions ref: and content_hash: too.") {
		t.Fatalf("replace went wrong:\n%s", again)
	}
	if _, ok := setContentHash("---\nname: x\n---\n", "h"); ok {
		t.Error("no upstream block: nothing to pin")
	}
}

func TestParsePackFlagsEqualsForm(t *testing.T) {
	f, err := parsePackFlags([]string{"--pick=prose=plain", "--catalog=/tmp/c", "writing"}, "--pick", "--catalog")
	if err != nil || f.picks["prose"] != "plain" || f.catalog != "/tmp/c" || len(f.args) != 1 {
		t.Errorf("got %+v, %v", f, err)
	}
}

// addNotFoundBackend answers like paws-backend: 404 for a name it doesn't know, a paged listing of its skills.
func addNotFoundBackend(t *testing.T) {
	t.Helper()
	rows := []catalogRow{
		{Name: "humanizer", Add: "unbound-paws add writing"},
		{Name: "writing-plans", Add: "unbound-paws add planning/writing-plans"},
		{Name: "tdd", Add: "unbound-paws add testing/tdd"},
	}
	portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/paws/v1/catalog/source":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "No pack, rule file or skill named %s in the catalog"}`, r.URL.Query().Get("name"))
		case "/paws/v1/catalog/skills":
			// Two pages of two and one, to prove the CLI follows next_offset.
			if r.URL.Query().Get("offset") == "2" {
				json.NewEncoder(w).Encode(map[string]any{"skills": rows[2:], "next_offset": nil})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"skills": rows[:2], "next_offset": 2})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAddUnknownNameSuggestsNearestNames(t *testing.T) {
	addNotFoundBackend(t)
	code, out := capture(t, &os.Stderr, func() int { return Add([]string{"writting"}) })
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	want := "paws: No pack, rule file or skill named writting. Did you mean writing? Run `unbound-paws search <word>` to look for it.\n"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
	if strings.Contains(out, "portal:") || strings.Contains(out, "HTTP 404") {
		t.Errorf("developer-facing text leaked: %q", out)
	}
}

func TestAddUnknownNameWithNothingCloseStillHints(t *testing.T) {
	addNotFoundBackend(t)
	code, out := capture(t, &os.Stderr, func() int { return Add([]string{"writing/nope"}) })
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	want := "paws: No pack, rule file or skill named writing/nope. Run `unbound-paws search <word>` to look for it.\n"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
}

func TestAddTargetsAreOnlyWhatAddAccepts(t *testing.T) {
	for _, c := range []struct{ add, want string }{
		{"unbound-paws add writing", "writing"},
		{"unbound-paws add ai-development/agents-sdk-docs ai-development/agents-sdk-install", "ai-development/agents-sdk-docs ai-development/agents-sdk-install"},
		{"unbound-paws add db --pick engine=postgres --project", "db"},
		{"unbound-paws add --pick engine=postgres db", "db"},
		{"", ""},
	} {
		if got := strings.Join(addTargets(c.add), " "); got != c.want {
			t.Errorf("addTargets(%q) = %q, want %q", c.add, got, c.want)
		}
	}
}

func TestAddSuggestionsDropControlCharacters(t *testing.T) {
	portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/catalog/source" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "none"}`)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"skills": []catalogRow{{Name: "x", Add: "unbound-paws add writin\u001b]g"}}})
	}))
	_, out := capture(t, &os.Stderr, func() int { return Add([]string{"writing"}) })
	if !strings.Contains(out, "Did you mean writin]g?") || strings.ContainsAny(out, "\x1b") {
		t.Errorf("a name from the server reached the terminal with control characters: %q", out)
	}
}

func TestClosestNames(t *testing.T) {
	names := []string{"writing", "writing-plans", "testing", "tdd", "planning", "waiting", "wrting"}
	got := closestNames("writting", names)
	if len(got) != 3 || got[0] != "writing" {
		t.Errorf("closest to writting: %v", got)
	}
	if got := closestNames("zzzzzz", names); len(got) != 0 {
		t.Errorf("nothing is close to zzzzzz, got %v", got)
	}
	for _, c := range []struct {
		a, b string
		want int
	}{{"", "abc", 3}, {"abc", "abc", 0}, {"writting", "writing", 1}, {"kitten", "sitting", 3}} {
		if d := editDistance(c.a, c.b); d != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, d, c.want)
		}
	}
}

func TestAddNoSuggestionWhenALaterCatalogPageFails(t *testing.T) {
	portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/paws/v1/catalog/source":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "none"}`)
		case r.URL.Query().Get("offset") == "2":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			json.NewEncoder(w).Encode(map[string]any{"skills": []catalogRow{{Name: "x", Add: "unbound-paws add waiting"}}, "next_offset": 2})
		}
	}))
	_, out := capture(t, &os.Stderr, func() int { return Add([]string{"writing"}) })
	if strings.Contains(out, "Did you mean") {
		t.Errorf("suggested from a partial catalog: %q", out)
	}
}

// A 404 with no catalog message is a missing route (a proxy, an older backend), not a mistyped name.
func TestAddRouteNotFoundIsNotReportedAsATypo(t *testing.T) {
	portalEnv(t, http.NotFoundHandler())
	code, out := capture(t, &os.Stderr, func() int { return Add([]string{"writing"}) })
	if code != 1 || strings.Contains(out, "No pack, rule file or skill named") {
		t.Errorf("exit %d, stderr %q", code, out)
	}
}

func TestAddNoSuggestionWhenTheListingNeverEnds(t *testing.T) {
	portalEnv(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/paws/v1/catalog/source" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "none"}`)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"skills": []catalogRow{{Name: "x", Add: "unbound-paws add waiting"}}, "next_offset": 1})
	}))
	_, out := capture(t, &os.Stderr, func() int { return Add([]string{"writing"}) })
	if !strings.Contains(out, "No pack, rule file or skill named writing.") || strings.Contains(out, "Did you mean") {
		t.Errorf("stderr %q", out)
	}
}
