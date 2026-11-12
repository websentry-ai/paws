package cli

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

// fakeCatalogBackend holds a published catalog (what `paws index --portal` gives the backend) and answers what `add`
// and `search` ask, the way paws-backend does.
type fakeCatalogBackend struct {
	mu      sync.Mutex
	files   map[string][]pushFile // content hash -> files
	catalog map[string]json.RawMessage
}

func (b *fakeCatalogBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply := func(v any) { json.NewEncoder(w).Encode(v) }
	p := r.URL.Path
	switch {
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
			outBundled = append(outBundled, map[string]any{"dir": bu.Dir, "content_hash": bu.ContentHash})
		}
	}
	out["bindings"], out["digests"], out["bundled"] = outB, outD, outBundled
	return out
}

// catalogEnv starts a fake backend for the CLI.
func catalogEnv(t *testing.T) (*fakeCatalogBackend, string) {
	t.Helper()
	b := &fakeCatalogBackend{files: map[string][]pushFile{}}
	return b, portalEnv(t, b)
}

// portalIndex runs `paws index --portal --catalog dir` and decodes the document.
func portalIndex(t *testing.T, dir string) (int, map[string]json.RawMessage) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "portal.json")
	code, _ := capture(t, &os.Stderr, func() int { return Index([]string{"--portal", "--catalog", dir, "-o", out}) })
	var doc map[string]json.RawMessage
	if data, err := os.ReadFile(out); err == nil {
		json.Unmarshal(data, &doc)
	}
	return code, doc
}

// publish loads the document for dir into the fake backend the way paws-backend's catalog_push does: bundled
// skills' files from the document, upstream skills' files from their repos.
func (b *fakeCatalogBackend) publish(t *testing.T, dir string) {
	t.Helper()
	code, doc := portalIndex(t, dir)
	if code != 0 {
		t.Fatalf("index --portal exit %d", code)
	}
	var bundled []pushBundled
	json.Unmarshal(doc["bundled"], &bundled)
	for i, bu := range bundled {
		b.files[bu.ContentHash] = bu.Files
		bundled[i].Files = nil
	}
	doc["bundled"] = must(json.Marshal(bundled))
	for _, f := range []fstest.MapFS{humanizerFiles, tddFiles} {
		files, err := dirFiles(f)
		if err != nil {
			t.Fatal(err)
		}
		b.files[hashOf(t, f)] = files
	}
	b.mu.Lock()
	b.catalog = doc
	b.mu.Unlock()
}

func TestPortalIndexIsTheWholeCheckedCatalog(t *testing.T) {
	code, doc := portalIndex(t, writeTestCatalog(t))
	if code != 0 {
		t.Fatalf("index --portal exit %d", code)
	}
	var skills []map[string]any
	json.Unmarshal(doc["skills"], &skills)
	by := map[string]map[string]any{}
	for _, s := range skills {
		by[s["name"].(string)] = s
		if _, ok := s["description"]; ok {
			t.Errorf("%v: the backend reads the description from SKILL.md; the document must not carry one", s["name"])
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
	if by["git"]["source_url"] != "" {
		t.Errorf("git, a skill Unbound wrote, links nowhere: %v", by["git"]["source_url"])
	}
	var bindings []pushBinding
	json.Unmarshal(doc["bindings"], &bindings)
	if len(bindings) != 4 || bindings[0].Path != "source-control/git.md" || bindings[0].Digest != "source-control/digests/git.md" ||
		bindings[0].Cases == nil || !strings.HasPrefix(bindings[0].Text, "---\nname: git\n") {
		t.Errorf("bindings %+v", bindings)
	}
	var packs []pushPack
	json.Unmarshal(doc["packs"], &packs)
	if len(packs) != 1 || strings.Join(packs[0].Paths, ",") != "writing/humanizer-pr.md,writing/humanizer-commit.md" {
		t.Errorf("packs %+v", packs)
	}
	// Only the skill Unbound wrote travels with its files; upstream ones the backend fetches at their pin.
	var bundled []pushBundled
	json.Unmarshal(doc["bundled"], &bundled)
	if len(bundled) != 1 || bundled[0].Skill != "git" || bundled[0].Dir != "source-control/skills/git" ||
		bundled[0].ContentHash != by["git"]["content_hash"] || len(bundled[0].Files) == 0 {
		t.Fatalf("bundled %+v", bundled)
	}
	fsys := fstest.MapFS{}
	for _, f := range bundled[0].Files {
		data, _ := base64.StdEncoding.DecodeString(f.Data)
		fsys[f.Path] = &fstest.MapFile{Data: data, Mode: 0o644}
	}
	if got := hashOf(t, fsys); got != bundled[0].ContentHash {
		t.Errorf("bundled files hash to %s, not %s", got, bundled[0].ContentHash)
	}
}

func TestPortalIndexRefusesWhatWeCannotStandBehind(t *testing.T) {
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
			dir := writeTestCatalog(t)
			breakIt(dir)
			if code, doc := portalIndex(t, dir); code != 1 || doc != nil {
				t.Errorf("exit %d; wrote a document: %v", code, doc != nil)
			}
		})
	}
}

func TestValidateChecksTriggerCases(t *testing.T) {
	dir := writeTestCatalog(t)
	if code, out := captureStdout(t, func() int { return Validate([]string{dir}) }); code != 0 {
		t.Fatalf("validate exit %d: %s", code, out)
	}
	writeFile(t, filepath.Join(dir, "trigger_cases.json"), `{"humanizer-pr": {"fires": [["Bash", "gh pr create"]], "ignores": [["Bash", "gh pr create --draft"]]},
  "humanizer-commit": {"fires": [["Bash", "git commit"]], "ignores": [["Bash", "git status"]]},
  "tdd": {"fires": [["Write", "a.test.ts"]], "ignores": [["Write", "a.ts"]]}, "git": {"fires": [["Bash", "git push"]], "ignores": [["Bash", "ls"]]}}`)
	code, out := 0, ""
	capture(t, &os.Stderr, func() int { code, out = captureStdout(t, func() int { return Validate([]string{dir}) }); return code })
	if code != 1 || !strings.Contains(out, "should ignore") {
		t.Errorf("validate exit %d: %s", code, out)
	}
}

func must(b []byte, _ error) []byte { return b }

func TestAddInstallsFromTheBackend(t *testing.T) {
	b, home := catalogEnv(t)
	b.publish(t, writeTestCatalog(t))
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

// Files for a skill Unbound wrote are checked against its pin too: the backend can't hand out other files.
func TestAddRefusesBundledFilesThatDontMatchTheirHash(t *testing.T) {
	b, _ := catalogEnv(t)
	b.publish(t, writeTestCatalog(t))
	var bundled []pushBundled
	json.Unmarshal(b.catalog["bundled"], &bundled)
	b.files[bundled[0].ContentHash] = []pushFile{{Path: "SKILL.md", Data: "ZXZpbAo="}} // "evil"
	code, errOut := capture(t, &os.Stderr, func() int {
		c, _ := captureStdout(t, func() int { return Add([]string{"source-control/git"}) })
		return c
	})
	if code != 1 || !strings.Contains(errOut, "refusing to install") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".claude", "skills", "git")); !os.IsNotExist(err) {
		t.Error("nothing may be installed from files that fail their hash")
	}
}

func TestSearchAsksTheBackend(t *testing.T) {
	b, _ := catalogEnv(t)
	b.publish(t, writeTestCatalog(t))
	code, out := captureStdout(t, func() int { return Search([]string{"human"}) })
	if code != 0 || !strings.Contains(out, "humanizer") || !strings.Contains(out, "unbound-paws add writing") || strings.Contains(out, "tdd") {
		t.Errorf("search exit %d: %s", code, out)
	}
	if code, out := captureStdout(t, func() int { return Search([]string{"zzz"}) }); code != 0 || !strings.Contains(out, "nothing matches") {
		t.Errorf("empty search exit %d: %s", code, out)
	}
}
