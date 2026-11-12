package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/websentry-ai/paws/internal/pack"
)

func keysOfDoc(m map[string][]map[string]any) string {
	k := map[string]any{}
	for key := range m {
		k[key] = 0
	}
	return keysOf(k)
}

func keysOf(m map[string]any) string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	sort.Strings(k)
	return strings.Join(k, ",")
}

// paws index --portal writes the document the backend loads: one record per skill with exactly these fields, its
// default actions taken from the bindings that require it, beside the rule files, packs, digests and bundled skills.
func TestIndexPortalMatchesCatalogContract(t *testing.T) {
	dir := writeTestCatalog(t)
	out := filepath.Join(t.TempDir(), "catalog.json")
	if code := Index([]string{"--catalog", dir, "--portal", "-o", out}); code != 0 {
		t.Fatalf("index --portal exit %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string][]map[string]any
	if err := json.Unmarshal(data, &doc); err != nil || len(doc["skills"]) == 0 ||
		keysOf(map[string]any{"skills": 0, "bindings": 0, "packs": 0, "digests": 0, "bundled": 0}) != keysOfDoc(doc) {
		t.Fatalf("want {skills, bindings, packs, digests, bundled}, got %s", data)
	}
	bySkill := map[string]map[string]any{}
	for _, s := range doc["skills"] {
		if got := keysOf(s); got != "add,content_hash,default_actions,kind,license,name,official,source_url,tags,upstream,version" {
			t.Errorf("%v: keys %s", s["name"], got)
		}
		for _, a := range s["default_actions"].([]any) {
			am := a.(map[string]any)
			for _, k := range []string{"id", "label", "event", "tool"} {
				if _, ok := am[k]; !ok {
					t.Errorf("%v action %v lacks %s", s["name"], am["id"], k)
				}
			}
			for k := range am {
				if !strings.Contains(",id,label,event,tool,command_regex,file_glob,mcp_action,example,", ","+k+",") {
					t.Errorf("%v action has extra key %s", s["name"], k)
				}
			}
		}
		bySkill[s["name"].(string)] = s
	}

	h := bySkill["humanizer"]
	up, _ := h["upstream"].(map[string]any)
	if h["kind"] != "skill" || h["license"] != "MIT" || keysOf(up) != "path,ref,repo" || up["repo"] != "github.com/blader/humanizer" ||
		h["version"] != up["ref"].(string)[:7] || !strings.HasPrefix(h["content_hash"].(string), "sha256:") {
		t.Errorf("humanizer entry: %v", h)
	}
	// humanizer's SKILL.md is at its repo's root, so the link is the repo at the pinned ref.
	if want := "https://github.com/blader/humanizer/tree/" + up["ref"].(string); h["source_url"] != want {
		t.Errorf("humanizer source_url %v, want %s", h["source_url"], want)
	}
	tdd := bySkill["tdd"]
	tup := tdd["upstream"].(map[string]any)
	if want := "https://github.com/mattpocock/skills/tree/" + tup["ref"].(string) + "/skills/engineering/tdd"; tdd["source_url"] != want {
		t.Errorf("tdd source_url %v, want %s", tdd["source_url"], want)
	}
	var pr map[string]any
	for _, a := range h["default_actions"].([]any) {
		if a.(map[string]any)["id"] == "humanizer-pr" {
			pr = a.(map[string]any)
		}
	}
	if pr == nil || pr["label"] != "A PR title or body goes out" || pr["event"] != "pre-tool" || pr["tool"] != "Bash" || pr["command_regex"] == "" {
		t.Errorf("humanizer-pr action: %v", pr)
	}

	// A skill this repo ships: no upstream; its hash is the one sync checks before installing it.
	g := bySkill["git"]
	sub, _ := fs.Sub(os.DirFS(dir), "catalog/source-control/skills/git")
	want, _ := pack.HashSkill(sub)
	if g["upstream"] != nil || g["content_hash"] != want || g["license"] != "Apache-2.0" {
		t.Errorf("git entry: upstream %v hash %v (want %s)", g["upstream"], g["content_hash"], want)
	}
	if g["source_url"] != "" { // its files are published with the catalog, not on GitHub
		t.Errorf("git source_url %v", g["source_url"])
	}
}

// provenance.official reaches the portal document for an upstream skill; every other skill says false.
func TestIndexPortalSaysWhichSkillsAreOfficial(t *testing.T) {
	dir := writeTestCatalog(t)
	for _, f := range []string{"catalog/testing/tdd.md", "catalog/source-control/git.md"} {
		p := filepath.Join(dir, f)
		data, _ := os.ReadFile(p)
		text := strings.Replace(string(data), "  license: MIT\n", "  license: MIT\n  official: true\n", 1)
		writeFile(t, p, strings.Replace(text, "tags: [org, git]", "provenance:\n  official: true\ntags: [org, git]", 1))
	}
	out := filepath.Join(t.TempDir(), "catalog.json")
	if code := Index([]string{"--catalog", dir, "--portal", "-o", out}); code != 0 {
		t.Fatalf("index --portal exit %d", code)
	}
	data, _ := os.ReadFile(out)
	var doc struct{ Skills []map[string]any }
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	got := map[string]any{}
	for _, s := range doc.Skills {
		got[s["name"].(string)] = s["official"]
	}
	if got["tdd"] != true || got["humanizer"] != false || got["git"] != false {
		t.Errorf("official: %v, want tdd true, humanizer false, and git false since Unbound wrote it", got)
	}
}
