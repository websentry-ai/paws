// Package catalog reads the repo's catalog/ bindings and packs/ manifests and validates them.
//
// Layout under a root directory:
//
//	catalog/<category>/<binding>.md              one binding (a paws rule with catalog metadata)
//	catalog/<category>/digests/<name>.md         digest files a binding's action.digest points at
//	catalog/<category>/skills/<skill>/SKILL.md   skills we own (org tag), installed from the repo
//	packs/<pack>.yaml                            pack manifests; binding paths are relative to catalog/
package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/websentry-ai/paws/internal/rule"
)

// PickOne is a group of interchangeable bindings; installing the pack installs exactly one option.
type PickOne struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Default     string              `yaml:"default"`
	Options     map[string][]string `yaml:"options"` // option name -> binding paths
}

// Pack is one packs/<name>.yaml manifest: the unit a user installs with `paws add`.
type Pack struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags"`
	Bindings    []string  `yaml:"bindings"`
	PickOne     []PickOne `yaml:"pick_one"`

	Path string `yaml:"-"`
}

// Binding is a parsed catalog binding with its path relative to catalog/.
type Binding struct {
	*rule.Rule
	Rel string
}

// Catalog is everything under one root.
type Catalog struct {
	Root     string
	Bindings map[string]*Binding // keyed by path relative to catalog/
	Packs    map[string]*Pack    // keyed by pack name
}

// Problem is one validation finding.
type Problem struct {
	Path string
	Msg  string
}

func (p Problem) String() string { return p.Path + ": " + p.Msg }

var contentHashRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// CatalogDir and PacksDir locate the two trees under root.
func CatalogDir(root string) string { return filepath.Join(root, "catalog") }
func PacksDir(root string) string   { return filepath.Join(root, "packs") }

// isBinding reports whether a catalog-relative path is a binding file rather than a skill, digest or doc.
func isBinding(rel string) bool {
	if filepath.Ext(rel) != ".md" {
		return false
	}
	base := filepath.Base(rel)
	if base == "README.md" || base == "SKILL.md" {
		return false
	}
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/") {
		if seg == "skills" || seg == "digests" {
			return false
		}
	}
	return true
}

// Load parses every binding and pack under root. Parse failures come back as problems, not errors.
func Load(root string) (*Catalog, []Problem) {
	c := &Catalog{Root: root, Bindings: map[string]*Binding{}, Packs: map[string]*Pack{}}
	var probs []Problem

	cdir := CatalogDir(root)
	_ = filepath.WalkDir(cdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(cdir, path)
		if !isBinding(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			probs = append(probs, Problem{path, err.Error()})
			return nil
		}
		r, err := rule.Parse(data, path)
		if err != nil {
			probs = append(probs, Problem{path, err.Error()})
			return nil
		}
		c.Bindings[filepath.ToSlash(rel)] = &Binding{Rule: r, Rel: filepath.ToSlash(rel)}
		return nil
	})

	manifests, _ := filepath.Glob(filepath.Join(PacksDir(root), "*.yaml"))
	for _, path := range manifests {
		p, err := LoadPack(path)
		if err != nil {
			probs = append(probs, Problem{path, err.Error()})
			continue
		}
		if prev, dup := c.Packs[p.Name]; dup {
			probs = append(probs, Problem{path, fmt.Sprintf("pack name %q already used by %s", p.Name, prev.Path)})
			continue
		}
		c.Packs[p.Name] = p
	}
	return c, probs
}

// LoadPack parses one pack manifest.
func LoadPack(path string) (*Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Pack
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("bad pack manifest: %w", err)
	}
	p.Path = path
	return &p, nil
}

// Validate loads root and checks every binding and pack. Problems are sorted by path.
func Validate(root string) (*Catalog, []Problem) {
	c, probs := Load(root)
	add := func(path, format string, a ...any) { probs = append(probs, Problem{path, fmt.Sprintf(format, a...)}) }

	names := map[string]string{}
	for _, b := range c.Bindings {
		path := b.SourcePath
		if prev, dup := names[b.Name]; dup {
			add(path, "binding name %q already used by %s", b.Name, prev)
		}
		names[b.Name] = path
		if b.Name == "" {
			add(path, "binding needs a name")
		}
		if len(b.Tags) == 0 {
			add(path, "binding needs at least one tag")
		}
		u := b.Upstream
		if u.Repo != "" && (u.Path == "" || u.Ref == "") {
			add(path, "upstream needs repo, path and ref together")
		}
		if u.ContentHash != "" && !contentHashRe.MatchString(u.ContentHash) {
			add(path, "upstream.content_hash must be sha256:<64 hex>, got %q", u.ContentHash)
		}
		dir := filepath.Dir(path)
		if b.Action.Type == "require-skill" && u.Repo == "" {
			local := filepath.Join(filepath.Dir(dir), filepath.Base(dir), "skills", b.Action.Skill, "SKILL.md")
			if _, err := os.Stat(local); err != nil {
				add(path, "require-skill %q has no upstream and no local skill at %s", b.Action.Skill, local)
			}
		}
		if b.Action.Digest != "" {
			if _, err := os.Stat(filepath.Join(dir, b.Action.Digest)); err != nil {
				add(path, "digest file %s not found", b.Action.Digest)
			}
		}
		if b.Action.Type == "exec-script" {
			if _, err := os.Stat(filepath.Join(dir, b.Action.Script)); err != nil {
				add(path, "script %s not found", b.Action.Script)
			}
		}
	}

	for _, p := range c.Packs {
		want := strings.TrimSuffix(filepath.Base(p.Path), ".yaml")
		if p.Name != want {
			add(p.Path, "pack name %q must match its file name %q", p.Name, want)
		}
		if p.Description == "" {
			add(p.Path, "pack needs a description")
		}
		if len(p.Bindings) == 0 && len(p.PickOne) == 0 {
			add(p.Path, "pack lists no bindings")
		}
		seen := map[string]bool{}
		ref := func(rel, where string) {
			if seen[rel] {
				add(p.Path, "binding %s listed more than once (%s)", rel, where)
			}
			seen[rel] = true
			if _, ok := c.Bindings[rel]; !ok {
				add(p.Path, "binding %s (%s) is not a valid binding under catalog/", rel, where)
			}
		}
		for _, rel := range p.Bindings {
			ref(rel, "bindings")
		}
		for _, g := range p.PickOne {
			if g.Name == "" {
				add(p.Path, "pick_one group needs a name")
			}
			if len(g.Options) < 2 {
				add(p.Path, "pick_one %q needs at least two options", g.Name)
			}
			if _, ok := g.Options[g.Default]; !ok {
				add(p.Path, "pick_one %q default %q is not one of its options", g.Name, g.Default)
			}
			for opt, rels := range g.Options {
				if len(rels) == 0 {
					add(p.Path, "pick_one %q option %q lists no bindings", g.Name, opt)
				}
				for _, rel := range rels {
					ref(rel, "pick_one "+g.Name)
				}
			}
		}
	}

	sort.Slice(probs, func(i, j int) bool {
		if probs[i].Path != probs[j].Path {
			return probs[i].Path < probs[j].Path
		}
		return probs[i].Msg < probs[j].Msg
	})
	return c, probs
}
