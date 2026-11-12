package cli

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/pack"
)

const adminKey = "test-admin-key"

// fakeCatalogBackend keeps what `paws catalog push` sends and answers what `add`, `search` and `pull` ask, the way
// paws-backend does.
type fakeCatalogBackend struct {
	mu      sync.Mutex
	files   map[string][]pushFile // content hash -> files
	catalog map[string]json.RawMessage
}

func (b *fakeCatalogBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	body := func(v any) {
		var rd io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "bad gzip", 400)
				return
			}
			rd = zr
		}
		json.NewDecoder(rd).Decode(v)
	}
	reply := func(v any) { json.NewEncoder(w).Encode(v) }
	p := r.URL.Path
	if strings.HasPrefix(p, "/paws/v1/admin/") && r.Header.Get("Authorization") != "Bearer "+adminKey {
		http.NotFound(w, r)
		return
	}
	switch {
	case p == "/paws/v1/admin/skill-files/missing":
		var req struct{ Hashes []string }
		body(&req)
		missing := []string{}
		for _, h := range req.Hashes {
			if b.files[h] == nil {
				missing = append(missing, h)
			}
		}
		reply(map[string]any{"missing": missing})
	case strings.HasPrefix(p, "/paws/v1/admin/skill-files/"):
		var req struct{ Files []pushFile }
		body(&req)
		b.files[strings.TrimPrefix(p, "/paws/v1/admin/skill-files/")] = req.Files
		w.WriteHeader(201)
		reply(map[string]any{})
	case p == "/paws/v1/admin/catalog" && r.Method == http.MethodPut:
		b.catalog = map[string]json.RawMessage{}
		body(&b.catalog)
		reply(map[string]any{"skills": 3})
	case p == "/paws/v1/admin/catalog":
		reply(b.catalog)
	case strings.HasPrefix(p, "/paws/v1/skill-files/"):
		files := b.files[strings.TrimPrefix(p, "/paws/v1/skill-files/")]
		if files == nil {
			http.NotFound(w, r)
			return
		}
		reply(map[string]any{"files": files})
	case p == "/paws/v1/catalog/source":
		reply(b.source(r.URL.Query().Get("name")))
	case p == "/paws/v1/catalog/skills":
		var skills []catalogRow
		json.Unmarshal(b.catalog["skills"], &skills)
		hits := []catalogRow{}
		for _, s := range skills {
			if strings.Contains(s.Name, r.URL.Query().Get("q")) {
				hits = append(hits, s)
			}
		}
		reply(map[string]any{"skills": hits, "total": len(hits)})
	default:
		http.NotFound(w, r)
	}
}

func (b *fakeCatalogBackend) source(name string) map[string]any {
	var bindings []pushBinding
	var packs []pushPack
	var digests []catalogText
	var bundled []pushBundled
	json.Unmarshal(b.catalog["bindings"], &bindings)
	json.Unmarshal(b.catalog["packs"], &packs)
	json.Unmarshal(b.catalog["digests"], &digests)
	json.Unmarshal(b.catalog["bundled"], &bundled)
	skills, out := map[string]bool{}, map[string]any{"packs": []any{}}
	for _, p := range packs {
		if p.Name == name {
			out["packs"] = []any{map[string]string{"name": p.Name, "text": p.Text}}
			for _, path := range p.Paths {
				for _, bd := range bindings {
					if bd.Path == path {
						skills[bd.Skill] = true
					}
				}
			}
		}
	}
	for _, bd := range bindings {
		if bd.Path == name+".md" {
			skills[bd.Skill] = true
		}
	}
	var outB []catalogText
	var outD []catalogText
	for _, bd := range bindings {
		if skills[bd.Skill] {
			outB = append(outB, catalogText{Path: bd.Path, Text: bd.Text})
			for _, d := range digests {
				if d.Path == bd.Digest {
					outD = append(outD, d)
				}
			}
		}
	}
	var outBundled []any
	for _, bu := range bundled {
		if skills[bu.Skill] {
			outBundled = append(outBundled, map[string]any{"dir": bu.Dir, "files": b.files[bu.ContentHash]})
		}
	}
	out["bindings"], out["digests"], out["bundled"] = outB, outD, outBundled
	return out
}

// catalogEnv starts a fake backend for the CLI, and makes git fetches of the test catalog's upstream repos check
// out their skill files.
func catalogEnv(t *testing.T) (*fakeCatalogBackend, string) {
	t.Helper()
	b := &fakeCatalogBackend{files: map[string][]pushFile{}}
	home := portalEnv(t, b)
	t.Setenv("PAWS_ADMIN_KEY", adminKey)
	orig := pack.Fetch
	pack.Fetch = func(_ context.Context, repo, ref, dest string) error {
		files := map[string]fstest.MapFS{"github.com/blader/humanizer": humanizerFiles, "github.com/mattpocock/skills": tddFiles}[repo]
		sub := map[string]string{"github.com/blader/humanizer": "", "github.com/mattpocock/skills": "skills/engineering/tdd"}[repo]
		for name, f := range files {
			writeFile(t, filepath.Join(dest, sub, name), string(f.Data))
		}
		return nil
	}
	t.Cleanup(func() { pack.Fetch = orig })
	return b, home
}

func TestCatalogPushSendsTheWholeCheckedCatalog(t *testing.T) {
	b, _ := catalogEnv(t)
	dir := writeTestCatalog(t)
	if code, out := captureStdout(t, func() int {
		return CatalogAdmin([]string{"push", "--bundled-source", "https://example.com/tree/abc", dir})
	}); code != 0 {
		t.Fatalf("push exit %d: %s", code, out)
	}
	var skills []map[string]any
	json.Unmarshal(b.catalog["skills"], &skills)
	by := map[string]map[string]any{}
	for _, s := range skills {
		by[s["name"].(string)] = s
		if _, ok := s["description"]; ok {
			t.Errorf("%v: the backend reads the description from SKILL.md; push must not send one", s["name"])
		}
		if _, ok := s["summary"]; ok {
			t.Errorf("%v: summary is not part of the record", s["name"])
		}
	}
	for name, want := range map[string]string{"humanizer": "unbound-paws add writing", "tdd": "unbound-paws add testing/tdd",
		"git": "unbound-paws add source-control/git"} {
		if by[name]["add"] != want {
			t.Errorf("%s add %v, want %s", name, by[name]["add"], want)
		}
	}
	examples := map[string]any{}
	for _, a := range by["humanizer"]["default_actions"].([]any) {
		examples[a.(map[string]any)["id"].(string)] = a.(map[string]any)["example"]
	}
	if examples["humanizer-pr"] != "gh pr create --fill" || examples["humanizer-commit"] != "git commit -m x" {
		t.Errorf("each action carries its first reviewed fire case as an example: %v", examples)
	}
	if by["git"]["source_url"] != "https://example.com/tree/abc/source-control/skills/git" {
		t.Errorf("git source_url %v", by["git"]["source_url"])
	}
	var bindings []pushBinding
	json.Unmarshal(b.catalog["bindings"], &bindings)
	if len(bindings) != 4 || bindings[0].Path != "source-control/git.md" || bindings[0].Digest != "source-control/digests/git.md" ||
		bindings[0].Cases == nil || !strings.HasPrefix(bindings[0].Text, "---\nname: git\n") {
		t.Errorf("bindings %+v", bindings)
	}
	var packs []pushPack
	json.Unmarshal(b.catalog["packs"], &packs)
	if len(packs) != 1 || strings.Join(packs[0].Paths, ",") != "writing/humanizer-pr.md,writing/humanizer-commit.md" {
		t.Errorf("packs %+v", packs)
	}
	var hashes []string
	for h := range b.files {
		hashes = append(hashes, h)
	}
	sort.Strings(hashes)
	want := []string{hashOf(t, humanizerFiles), hashOf(t, tddFiles), by["git"]["content_hash"].(string)}
	sort.Strings(want)
	if strings.Join(hashes, ",") != strings.Join(want, ",") {
		t.Errorf("uploaded %v, want every skill's files %v", hashes, want)
	}

	// A second push uploads nothing new: the backend already has every hash.
	fetched := 0
	orig := pack.Fetch
	pack.Fetch = func(context.Context, string, string, string) error { fetched++; return nil }
	defer func() { pack.Fetch = orig }()
	if code, _ := captureStdout(t, func() int { return CatalogAdmin([]string{"push", dir}) }); code != 0 || fetched != 0 {
		t.Errorf("second push exit %d, fetched %d repos", code, fetched)
	}
}

func TestCatalogPushRefusesWhatItCannotStandBehind(t *testing.T) {
	cases := map[string]func(dir string){
		"a trigger that fires on a near miss": func(dir string) {
			writeFile(t, filepath.Join(dir, "trigger_cases.json"), `{"humanizer-pr": {"fires": [["Bash", "gh pr create"]], "ignores": [["Bash", "gh pr create --draft"]]},
  "humanizer-commit": {"fires": [["Bash", "git commit"]], "ignores": [["Bash", "git status"]]},
  "tdd": {"fires": [["Write", "a.test.ts"]], "ignores": [["Write", "a.ts"]]}, "git": {"fires": [["Bash", "git push"]], "ignores": [["Bash", "ls"]]}}`)
		},
		"a skill whose repo grants no licence": func(dir string) {
			p := filepath.Join(dir, "catalog/testing/tdd.md")
			data, _ := os.ReadFile(p)
			writeFile(t, p, strings.Replace(string(data), "license: MIT", "license: NOASSERTION", 1))
		},
		"a rule file that doesn't validate": func(dir string) {
			writeFile(t, filepath.Join(dir, "catalog/testing/broken.md"), "---\nname: broken\n---\n")
		},
	}
	for label, breakIt := range cases {
		t.Run(label, func(t *testing.T) {
			b, _ := catalogEnv(t)
			dir := writeTestCatalog(t)
			breakIt(dir)
			code, _ := capture(t, &os.Stderr, func() int {
				c, _ := captureStdout(t, func() int { return CatalogAdmin([]string{"push", dir}) })
				return c
			})
			if code != 1 || b.catalog != nil || len(b.files) != 0 {
				t.Errorf("exit %d; pushed %v, uploaded %d", code, b.catalog != nil, len(b.files))
			}
		})
	}
	t.Run("no admin key", func(t *testing.T) {
		catalogEnv(t)
		t.Setenv("PAWS_ADMIN_KEY", "")
		if code, _ := capture(t, &os.Stderr, func() int { return CatalogAdmin([]string{"push", writeTestCatalog(t)}) }); code != 2 {
			t.Errorf("exit %d", code)
		}
	})
}

func TestCatalogPullGivesBackAnEditableCatalog(t *testing.T) {
	b, _ := catalogEnv(t)
	dir := writeTestCatalog(t)
	captureStdout(t, func() int { return CatalogAdmin([]string{"push", dir}) })
	first := b.catalog
	out := filepath.Join(t.TempDir(), "pulled")
	if code, msg := captureStdout(t, func() int { return CatalogAdmin([]string{"pull", out}) }); code != 0 {
		t.Fatalf("pull exit %d: %s", code, msg)
	}
	c, probs := catalog.Validate(out)
	if len(probs) > 0 {
		t.Fatalf("pulled catalog: %v", probs)
	}
	tc, exempt, err := readCases(out)
	if err != nil || len(catalog.CheckTriggerCases(c, tc, exempt)) != 0 {
		t.Fatalf("pulled cases: %v %v", err, catalog.CheckTriggerCases(c, tc, exempt))
	}
	if code, _ := captureStdout(t, func() int { return CatalogAdmin([]string{"push", out}) }); code != 0 {
		t.Fatal("pushing the pulled copy back failed")
	}
	for _, k := range []string{"bindings", "packs", "digests", "bundled"} {
		var a, b2 any
		json.Unmarshal(first[k], &a)
		json.Unmarshal(b.catalog[k], &b2)
		if ja, _ := json.Marshal(a); string(ja) != string(must(json.Marshal(b2))) {
			t.Errorf("%s changed in a pull and push round trip", k)
		}
	}
}

func must(b []byte, _ error) []byte { return b }

func TestAddInstallsFromTheBackend(t *testing.T) {
	_, home := catalogEnv(t)
	captureStdout(t, func() int { return CatalogAdmin([]string{"push", writeTestCatalog(t)}) })
	if code, out := captureStdout(t, func() int { return Add([]string{"writing", "source-control/git"}) }); code != 0 {
		t.Fatalf("add exit %d: %s", code, out)
	}
	skills := filepath.Join(os.Getenv("HOME"), ".claude", "skills")
	if data, _ := os.ReadFile(filepath.Join(skills, "humanizer", "SKILL.md")); string(data) != string(humanizerFiles["SKILL.md"].Data) {
		t.Errorf("humanizer installed from the backend: %q", data)
	}
	if _, err := os.Stat(filepath.Join(skills, "git", "SKILL.md")); err != nil {
		t.Errorf("git, a skill Unbound wrote, installed: %v", err)
	}
	installed := map[string]bool{}
	filepath.Walk(filepath.Join(home, "packs"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			installed[filepath.Base(p)] = true
		}
		return nil
	})
	for _, f := range []string{"humanizer-pr.md", "humanizer-commit.md", "git.md"} {
		if !installed[f] {
			t.Errorf("rule file %s not installed: %v", f, installed)
		}
	}
	if code, _ := capture(t, &os.Stderr, func() int { return Add([]string{"nope"}) }); code != 1 {
		t.Errorf("add of an unknown name exit %d", code)
	}
}

func TestSearchAsksTheBackend(t *testing.T) {
	catalogEnv(t)
	captureStdout(t, func() int { return CatalogAdmin([]string{"push", writeTestCatalog(t)}) })
	code, out := captureStdout(t, func() int { return Search([]string{"human"}) })
	if code != 0 || !strings.Contains(out, "humanizer") || !strings.Contains(out, "unbound-paws add writing") || strings.Contains(out, "tdd") {
		t.Errorf("search exit %d: %s", code, out)
	}
	if code, out := captureStdout(t, func() int { return Search([]string{"zzz"}) }); code != 0 || !strings.Contains(out, "nothing matches") {
		t.Errorf("empty search exit %d: %s", code, out)
	}
}
