package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/pack"
)

// `paws catalog push <dir>` and `pull <dir>`: how the published catalog is edited. The catalog lives in the paws
// backend's database; a directory laid out like a catalog (catalog/, packs/, trigger_cases.json) is only the editing
// copy. push checks it the way the CLI reads it, runs every trigger against its fire and ignore cases, uploads the
// skill files the backend lacks and replaces the catalog in one go. Both need PAWS_ADMIN_KEY.

const adminTimeout = 0 // no client timeout: a first push uploads every skill; the context bounds it instead

// CatalogAdmin is `paws catalog push|pull <dir>`.
func CatalogAdmin(args []string) int {
	usage := "usage: unbound-paws catalog push [--bundled-source URL] <dir>\n       unbound-paws catalog pull <dir>\n" +
		"PAWS_ADMIN_KEY must hold the backend's admin key; PAWS_PORTAL_API picks the backend."
	if len(args) < 2 || args[0] != "push" && args[0] != "pull" {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	key := os.Getenv("PAWS_ADMIN_KEY")
	api := portalAPI()
	if _, err := checkPortalAPI(api); err != nil || key == "" {
		fmt.Fprintf(os.Stderr, "paws: %v\n%s\n", errors.Join(err, errIf(key == "", "PAWS_ADMIN_KEY is not set")), usage)
		return 2
	}
	a := admin{api: api, key: key, client: &http.Client{Timeout: adminTimeout}}
	var err error
	if args[0] == "pull" {
		err = a.pull(args[1])
	} else {
		bundledSource, dir := "", args[1]
		if args[1] == "--bundled-source" && len(args) == 4 {
			bundledSource, dir = strings.TrimRight(args[2], "/"), args[3]
		} else if len(args) != 2 {
			fmt.Fprintln(os.Stderr, usage)
			return 2
		}
		err = a.push(dir, bundledSource)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	return 0
}

func errIf(cond bool, msg string) error {
	if cond {
		return errors.New(msg)
	}
	return nil
}

type admin struct {
	api, key string
	client   *http.Client
}

// do sends body (gzipped JSON when not nil) and decodes the answer into out.
func (a admin) do(method, p string, body, out any) error {
	var rd io.Reader
	if body != nil {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if err := json.NewEncoder(zw).Encode(body); err != nil {
			return err
		}
		zw.Close()
		rd = &buf
	}
	req, err := http.NewRequestWithContext(context.Background(), method, a.api+p, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound && strings.HasPrefix(p, "/paws/v1/admin/") {
		return fmt.Errorf("%s %s: 404; is PAWS_ADMIN_KEY the backend's admin key?", method, p)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s: %w", method, p, portalError(resp.StatusCode, data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

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
	Dir         string `json:"dir"`
	Skill       string `json:"skill"`
	ContentHash string `json:"content_hash"`
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

func (a admin) push(dir, bundledSource string) error {
	c, probs := catalog.Validate(dir)
	if len(probs) > 0 {
		for _, p := range probs {
			fmt.Fprintln(os.Stderr, "paws: catalog:", p)
		}
		return fmt.Errorf("the catalog has %d problem(s); nothing pushed", len(probs))
	}
	cases, exempt, err := readCases(dir)
	if err != nil {
		return err
	}
	if probs := catalog.CheckTriggerCases(c, cases, exempt); len(probs) > 0 {
		for _, p := range probs {
			fmt.Fprintln(os.Stderr, "paws: trigger:", p)
		}
		return fmt.Errorf("%d trigger case(s) fail; nothing pushed", len(probs))
	}
	var unlicensed []string
	for _, rel := range sortedKeys(c.Bindings) {
		if b := c.Bindings[rel]; b.Upstream.Repo != "" && noLicense(b.Provenance.License) {
			unlicensed = append(unlicensed, rel)
		}
	}
	if len(unlicensed) > 0 {
		return fmt.Errorf("%d rule file(s) pin a skill whose repo grants no licence, first %s; remove them", len(unlicensed), unlicensed[0])
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
		if s.Upstream == nil {
			s.SourceURL = ""
			if bundledSource != "" {
				s.SourceURL = bundledSource + "/" + bundledDir(c, s.Name)
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

	// Skill files: ask which the backend lacks, and fetch and upload only those.
	var hashes []string
	for _, s := range skills {
		hashes = append(hashes, s.ContentHash)
	}
	var missing struct {
		Missing []string `json:"missing"`
	}
	if err := a.do(http.MethodPost, "/paws/v1/admin/skill-files/missing", map[string]any{"hashes": hashes}, &missing); err != nil {
		return err
	}
	if err := a.uploadFiles(c, skills, missing.Missing); err != nil {
		return err
	}

	body := map[string]any{"skills": records, "bindings": []pushBinding{}, "packs": []pushPack{}, "digests": []catalogText{},
		"bundled": []pushBundled{}}
	digests := map[string]bool{}
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		text, err := fs.ReadFile(c.FS, catalog.BindingPath(rel))
		if err != nil {
			return err
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
			return err
		}
		body["digests"] = append(body["digests"].([]catalogText), catalogText{Path: d, Text: string(text)})
	}
	for _, name := range sortedKeys(c.Packs) {
		p := c.Packs[name]
		text, err := fs.ReadFile(c.FS, p.Path)
		if err != nil {
			return err
		}
		body["packs"] = append(body["packs"].([]pushPack), pushPack{Name: name, Text: string(text), Paths: packPaths(p)})
	}
	for _, s := range skills {
		if s.Upstream == nil {
			body["bundled"] = append(body["bundled"].([]pushBundled), pushBundled{Dir: bundledDir(c, s.Name), Skill: s.Name, ContentHash: s.ContentHash})
		}
	}
	var res map[string]any
	if err := a.do(http.MethodPut, "/paws/v1/admin/catalog", body, &res); err != nil {
		return err
	}
	out, _ := json.Marshal(res)
	fmt.Printf("pushed to %s: %s\n", a.api, out)
	return nil
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

// uploadFiles sends the skill directories with these hashes. Upstream ones are fetched from GitHub at their pinned
// commit, once per repo and commit, and must hash to their pin.
func (a admin) uploadFiles(c *catalog.Catalog, skills []*portalCatalogSkill, want []string) error {
	need := map[string]bool{}
	for _, h := range want {
		need[h] = true
	}
	checkouts := map[string]string{} // repo@ref -> temp checkout
	defer func() {
		for _, d := range checkouts {
			os.RemoveAll(d)
		}
	}()
	for _, s := range skills {
		if !need[s.ContentHash] {
			continue
		}
		var fsys fs.FS
		if s.Upstream == nil {
			sub, err := fs.Sub(c.FS, path.Join("catalog", bundledDir(c, s.Name)))
			if err != nil {
				return err
			}
			fsys = sub
		} else {
			key := s.Upstream.Repo + "@" + s.Upstream.Ref
			if checkouts[key] == "" {
				tmp, err := os.MkdirTemp("", "paws-push-")
				if err != nil {
					return err
				}
				checkouts[key] = tmp
				fmt.Printf("fetching %s\n", key)
				if err := pack.Fetch(context.Background(), s.Upstream.Repo, s.Upstream.Ref, tmp); err != nil {
					return fmt.Errorf("%s: %w", s.Name, err)
				}
			}
			fsys = os.DirFS(filepath.Join(checkouts[key], filepath.FromSlash(path.Dir(s.Upstream.Path))))
		}
		got, err := pack.HashSkill(fsys)
		if err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		if got != s.ContentHash {
			return fmt.Errorf("%s: its files hash to %s, not the pinned %s", s.Name, got, s.ContentHash)
		}
		files, err := dirFiles(fsys)
		if err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		if err := a.do(http.MethodPut, "/paws/v1/admin/skill-files/"+url.PathEscape(s.ContentHash), map[string]any{"files": files}, nil); err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
		delete(need, s.ContentHash)
		fmt.Printf("uploaded %s\n", s.Name)
	}
	return nil
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

// pull writes the published catalog into dir, an empty or new directory, ready to edit and push back.
func (a admin) pull(dir string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s is not empty", dir)
	}
	var cat struct {
		Bindings []struct {
			Path  string     `json:"path"`
			Text  string     `json:"text"`
			Cases *pushCases `json:"cases"`
		} `json:"bindings"`
		Digests []catalogText `json:"digests"`
		Packs   []pushPack    `json:"packs"`
		Bundled []pushBundled `json:"bundled"`
	}
	if err := a.do(http.MethodGet, "/paws/v1/admin/catalog", nil, &cat); err != nil {
		return err
	}
	write := func(rel string, data []byte, mode os.FileMode) error {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
			return fmt.Errorf("refusing path %q", rel)
		}
		p := filepath.Join(dir, clean)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, data, mode)
	}
	for _, b := range cat.Bindings {
		if err := write("catalog/"+b.Path, []byte(b.Text), 0o644); err != nil {
			return err
		}
	}
	for _, d := range cat.Digests {
		if err := write("catalog/"+d.Path, []byte(d.Text), 0o644); err != nil {
			return err
		}
	}
	for _, p := range cat.Packs {
		if err := write("packs/"+p.Name+".yaml", []byte(p.Text), 0o644); err != nil {
			return err
		}
	}
	get := skillFilesFrom(a.api)
	for _, b := range cat.Bundled {
		fsys, err := get(context.Background(), b.ContentHash)
		if err != nil {
			return fmt.Errorf("%s: %w", b.Dir, err)
		}
		err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			info, _ := d.Info()
			return write(path.Join("catalog", b.Dir, p), data, info.Mode().Perm())
		})
		if err != nil {
			return err
		}
	}
	// trigger_cases.json is keyed by binding name, which lives in each rule file.
	c, _ := catalog.Load(dir)
	cases, exempt := map[string]pushCases{}, []string{}
	for _, b := range cat.Bindings {
		rb := c.Bindings[b.Path]
		if rb == nil {
			continue
		}
		if b.Cases != nil {
			cases[rb.Name] = *b.Cases
		} else {
			exempt = append(exempt, rb.Name)
		}
	}
	data, _ := json.MarshalIndent(cases, "", "  ")
	if err := write("trigger_cases.json", append(data, '\n'), 0o644); err != nil {
		return err
	}
	sort.Strings(exempt)
	list := "# Bindings from before trigger cases; give them cases and remove them here.\n" + strings.Join(exempt, "\n") + "\n"
	if err := write("without_trigger_cases.txt", []byte(list), 0o644); err != nil {
		return err
	}
	fmt.Printf("pulled %d rule files, %d packs and %d bundled skills from %s into %s\n", len(cat.Bindings), len(cat.Packs), len(cat.Bundled), a.api, dir)
	return nil
}
