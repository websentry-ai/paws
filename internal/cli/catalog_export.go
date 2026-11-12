package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/websentry-ai/paws/internal/catalog"
)

// The portal document: `paws index --portal --catalog <dir>` turns a catalog folder into the one JSON document the
// paws backend loads into its database. It publishes nothing itself. It refuses a folder that doesn't validate,
// whose triggers fail their fire and ignore cases, or that pins a skill whose repo grants no licence.

// pushFile is one file of a skill directory as the backend keeps it.
type pushFile struct {
	Path string `json:"path"`
	Data string `json:"data"`
	Exec bool   `json:"exec,omitempty"`
}

func dirFiles(fsys fs.FS) ([]pushFile, error) {
	var out []pushFile
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != "." {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, pushFile{Path: p, Data: base64.StdEncoding.EncodeToString(data), Exec: info.Mode().Perm()&0o111 != 0})
		return nil
	})
	return out, err
}

type pushBinding struct {
	Path   string     `json:"path"`
	Skill  string     `json:"skill"`
	Text   string     `json:"text"`
	Cases  *pushCases `json:"cases"`
	Digest string     `json:"digest,omitempty"`
}

type pushCases struct {
	Fires   [][2]string `json:"fires"`
	Ignores [][2]string `json:"ignores"`
}

type pushPack struct {
	Name  string   `json:"name"`
	Text  string   `json:"text"`
	Paths []string `json:"paths"`
}

type pushBundled struct {
	Dir         string     `json:"dir"`
	Skill       string     `json:"skill"`
	ContentHash string     `json:"content_hash"`
	Files       []pushFile `json:"files,omitempty"` // the skill's own files, in the document only
}

// readCases reads trigger_cases.json and without_trigger_cases.txt from dir.
func readCases(dir string) (catalog.TriggerCases, map[string]bool, error) {
	var cases catalog.TriggerCases
	data, err := os.ReadFile(filepath.Join(dir, "trigger_cases.json"))
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, nil, fmt.Errorf("trigger_cases.json: %w", err)
	}
	exempt := map[string]bool{}
	if list, err := os.ReadFile(filepath.Join(dir, "without_trigger_cases.txt")); err == nil {
		for _, line := range strings.Split(string(list), "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
				exempt[line] = true
			}
		}
	}
	return cases, exempt, nil
}

// noLicense: a source repo that grants no licence; its skills are not redistributed.
func noLicense(l string) bool { return l == "" || l == "NOASSERTION" }

func portalDocument(dir string) (map[string]any, error) {
	c, probs := catalog.Validate(dir)
	if len(probs) > 0 {
		for _, p := range probs {
			fmt.Fprintln(os.Stderr, "paws: catalog:", p)
		}
		return nil, fmt.Errorf("the catalog has %d problem(s)", len(probs))
	}
	cases, exempt, err := readCases(dir)
	if err != nil {
		return nil, err
	}
	if probs := catalog.CheckTriggerCases(c, cases, exempt); len(probs) > 0 {
		for _, p := range probs {
			fmt.Fprintln(os.Stderr, "paws: trigger:", p)
		}
		return nil, fmt.Errorf("%d trigger case(s) fail", len(probs))
	}
	var unlicensed []string
	for _, rel := range sortedKeys(c.Bindings) {
		if b := c.Bindings[rel]; b.Upstream.Repo != "" && noLicense(b.Provenance.License) {
			unlicensed = append(unlicensed, rel)
		}
	}
	if len(unlicensed) > 0 {
		return nil, fmt.Errorf("%d rule file(s) pin a skill whose repo grants no licence, first %s; remove them", len(unlicensed), unlicensed[0])
	}

	skills := buildPortalCatalog(c)["skills"].([]*portalCatalogSkill)
	ruleOf := map[string]string{} // binding name -> path without .md
	for rel, b := range c.Bindings {
		ruleOf[b.Name] = strings.TrimSuffix(rel, ".md")
	}
	records := make([]map[string]any, 0, len(skills))
	for _, s := range skills {
		var rules []string
		for i, act := range s.DefaultActions {
			rules = append(rules, ruleOf[act.ID.(string)])
			if tc, ok := cases[act.ID.(string)]; ok && len(tc.Fires) > 0 {
				s.DefaultActions[i].Example = tc.Fires[0][1]
			}
		}
		raw, _ := json.Marshal(s)
		var rec map[string]any
		json.Unmarshal(raw, &rec)
		delete(rec, "summary")
		delete(rec, "description") // the backend reads it from the skill's SKILL.md
		rec["add"] = addCommand(c, rules)
		records = append(records, rec)
	}

	body := map[string]any{"skills": records, "bindings": []pushBinding{}, "packs": []pushPack{}, "digests": []catalogText{},
		"bundled": []pushBundled{}}
	digests := map[string]bool{}
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		text, err := fs.ReadFile(c.FS, catalog.BindingPath(rel))
		if err != nil {
			return nil, err
		}
		pb := pushBinding{Path: rel, Skill: b.Action.Skill, Text: string(text)}
		if tc, ok := cases[b.Name]; ok {
			pb.Cases = &pushCases{Fires: tc.Fires, Ignores: tc.Ignores}
		}
		if b.Action.Digest != "" {
			pb.Digest = path.Join(path.Dir(rel), b.Action.Digest)
			digests[pb.Digest] = true
		}
		body["bindings"] = append(body["bindings"].([]pushBinding), pb)
	}
	for _, d := range sortedKeys(digests) {
		text, err := fs.ReadFile(c.FS, catalog.BindingPath(d))
		if err != nil {
			return nil, err
		}
		body["digests"] = append(body["digests"].([]catalogText), catalogText{Path: d, Text: string(text)})
	}
	for _, name := range sortedKeys(c.Packs) {
		p := c.Packs[name]
		text, err := fs.ReadFile(c.FS, p.Path)
		if err != nil {
			return nil, err
		}
		body["packs"] = append(body["packs"].([]pushPack), pushPack{Name: name, Text: string(text), Paths: packPaths(p)})
	}
	for _, s := range skills {
		if s.Upstream != nil {
			continue // the backend fetches upstream skills from GitHub at their pin and checks the hash
		}
		d := bundledDir(c, s.Name)
		sub, err := fs.Sub(c.FS, path.Join("catalog", d))
		if err != nil {
			return nil, err
		}
		files, err := dirFiles(sub)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Name, err)
		}
		body["bundled"] = append(body["bundled"].([]pushBundled), pushBundled{Dir: d, Skill: s.Name, ContentHash: s.ContentHash, Files: files})
	}
	return body, nil
}

// bundledDir is where a catalog skill's own files sit, relative to catalog/: <category>/skills/<name>.
func bundledDir(c *catalog.Catalog, skill string) string {
	for _, rel := range sortedKeys(c.Bindings) {
		if b := c.Bindings[rel]; b.Action.Skill == skill && b.Upstream.Repo == "" {
			return strings.TrimPrefix(catalog.LocalSkillDir(b), "catalog/")
		}
	}
	return ""
}

func packPaths(p *catalog.Pack) []string {
	paths := append([]string{}, p.Bindings...)
	for _, g := range p.PickOne {
		for _, opt := range sortedKeys(g.Options) {
			paths = append(paths, g.Options[opt]...)
		}
	}
	return paths
}

// addCommand is the `paws add` command that installs exactly these rule files: a pack when one installs them (with
// --pick for a non-default option), else the rule files by path.
func addCommand(c *catalog.Catalog, rules []string) string {
	same := func(files []string) bool {
		if len(files) != len(rules) {
			return false
		}
		have := map[string]bool{}
		for _, r := range rules {
			have[r] = true
		}
		for _, f := range files {
			if !have[strings.TrimSuffix(f, ".md")] {
				return false
			}
		}
		return true
	}
	for _, name := range sortedKeys(c.Packs) {
		p := c.Packs[name]
		install := func(g *catalog.PickOne, option string) []string {
			files := append([]string{}, p.Bindings...)
			for i := range p.PickOne {
				h := &p.PickOne[i]
				pick := h.Default
				if g != nil && h.Name == g.Name {
					pick = option
				}
				files = append(files, h.Options[pick]...)
			}
			return files
		}
		if same(install(nil, "")) {
			return cliName + " add " + name
		}
		for i := range p.PickOne {
			g := &p.PickOne[i]
			for _, option := range sortedKeys(g.Options) {
				if option != g.Default && same(install(g, option)) {
					return fmt.Sprintf("%s add --pick %s=%s %s", cliName, g.Name, option, name)
				}
			}
		}
	}
	sorted := append([]string{}, rules...)
	sort.Strings(sorted)
	return cliName + " add " + strings.Join(sorted, " ")
}
