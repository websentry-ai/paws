package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/pack"
)

func keysOf(m map[string]any) string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	sort.Strings(k)
	return strings.Join(k, ",")
}

// paws index --portal writes the contract's /catalog: one entry per skill with exactly the
// contract's fields, its default actions taken from the bindings that require it.
func TestIndexPortalMatchesCatalogContract(t *testing.T) {
	out := filepath.Join(t.TempDir(), "catalog.json")
	if code := Index([]string{"--portal", "-o", out}); code != 0 {
		t.Fatalf("index --portal exit %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string][]map[string]any
	if err := json.Unmarshal(data, &doc); err != nil || len(doc) != 1 || len(doc["skills"]) == 0 {
		t.Fatalf("want {skills:[...]}, got %s", data)
	}
	bySkill := map[string]map[string]any{}
	for _, s := range doc["skills"] {
		if got := keysOf(s); got != "content_hash,default_actions,description,kind,license,name,tags,upstream,version" {
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
				if !strings.Contains(",id,label,event,tool,command_regex,file_glob,mcp_action,", ","+k+",") {
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
	sub, _ := fs.Sub(paws.Catalog, "catalog/source-control/skills/git")
	want, _ := pack.HashSkill(sub)
	if g["upstream"] != nil || g["content_hash"] != want || g["license"] != "Apache-2.0" {
		t.Errorf("git entry: upstream %v hash %v (want %s)", g["upstream"], g["content_hash"], want)
	}
}
