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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/websentry-ai/paws/internal/rule"
)

// PickOne is a group of interchangeable bindings; installing the pack installs exactly one option.
type PickOne struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description" json:"description,omitempty"`
	Default     string              `yaml:"default" json:"default"`
	Options     map[string][]string `yaml:"options" json:"options"` // option name -> binding paths
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

// Catalog is everything under one root: a directory on disk or the catalog embedded in the binary.
type Catalog struct {
	FS       fs.FS               // holds catalog/ and packs/
	Root     string              // how problems name FS paths (a directory, or "embedded")
	Bindings map[string]*Binding // keyed by path relative to catalog/
	Packs    map[string]*Pack    // keyed by pack name
}

// Display names an FS path for messages.
func (c *Catalog) Display(p string) string { return filepath.Join(c.Root, filepath.FromSlash(p)) }

// BindingPath is a binding's FS path, e.g. catalog/writing/humanizer-pr.md.
func BindingPath(rel string) string { return path.Join("catalog", rel) }

// LocalSkillDir is the FS directory of a skill we ship beside a binding: catalog/<category>/skills/<skill>.
func LocalSkillDir(b *Binding) string {
	return path.Join(path.Dir(b.SourcePath), "skills", b.Action.Skill)
}

// Problem is one validation finding.
type Problem struct {
	Path string
	Msg  string
}

func (p Problem) String() string { return p.Path + ": " + p.Msg }

var (
	contentHashRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	skillNameRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`) // also a directory name under skills/
	packNameRe    = skillNameRe                                // also a directory name under packs/
	// RepoRe is a bare host path (github.com/owner/repo): fetched over https only, never a local path or URL.
	RepoRe = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*\.[a-z]{2,}(/[A-Za-z0-9][A-Za-z0-9._-]*)+$`)
	// RefRe is a full commit SHA: the pin is a commit, never a branch or tag that can move.
	RefRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// ValidSkillName reports whether s can name a skill directory.
func ValidSkillName(s string) bool { return skillNameRe.MatchString(s) }

// isRegular reports whether name exists in fsys and is a regular file.
func isRegular(fsys fs.FS, name string) bool {
	st, err := fs.Stat(fsys, name)
	return err == nil && st.Mode().IsRegular()
}

// CatalogDir and PacksDir locate the two trees under a directory root.
func CatalogDir(root string) string { return filepath.Join(root, "catalog") }
func PacksDir(root string) string   { return filepath.Join(root, "packs") }

// Load parses every binding and pack under a directory root.
func Load(root string) (*Catalog, []Problem) { return LoadFS(os.DirFS(root), root) }

// LoadFS parses every binding and pack in fsys. Parse failures come back as problems, not errors.
// A binding's SourcePath is its FS path (catalog/<category>/<name>.md).
func LoadFS(fsys fs.FS, display string) (*Catalog, []Problem) {
	c := &Catalog{FS: fsys, Root: display, Bindings: map[string]*Binding{}, Packs: map[string]*Pack{}}
	var probs []Problem
	bad := func(p string, err error) { probs = append(probs, Problem{c.Display(p), err.Error()}) }

	_ = fs.WalkDir(fsys, "catalog", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, "catalog/")
		if !rule.IsRuleFile(rel) {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			bad(p, err)
			return nil
		}
		if err := rule.CheckKnownFields(data); err != nil {
			bad(p, err)
			return nil
		}
		r, err := rule.Parse(data, p)
		if err != nil {
			bad(p, err)
			return nil
		}
		c.Bindings[rel] = &Binding{Rule: r, Rel: rel}
		return nil
	})

	yml, _ := fs.Glob(fsys, "packs/*.yml")
	for _, p := range yml {
		probs = append(probs, Problem{c.Display(p), "pack manifests use the .yaml extension"})
	}
	manifests, _ := fs.Glob(fsys, "packs/*.yaml")
	for _, p := range manifests {
		pk, err := loadPack(fsys, p)
		if err != nil {
			bad(p, err)
			continue
		}
		if prev, dup := c.Packs[pk.Name]; dup {
			probs = append(probs, Problem{c.Display(p), fmt.Sprintf("pack name %q already used by %s", pk.Name, c.Display(prev.Path))})
			continue
		}
		c.Packs[pk.Name] = pk
	}
	return c, probs
}

// loadPack parses one pack manifest; Path is its FS path.
func loadPack(fsys fs.FS, p string) (*Pack, error) {
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		return nil, err
	}
	var pk Pack
	if err := yaml.Unmarshal(data, &pk); err != nil {
		return nil, fmt.Errorf("bad pack manifest: %w", err)
	}
	pk.Path = p
	return &pk, nil
}

// Validate checks every binding and pack under a directory root.
func Validate(root string) (*Catalog, []Problem) { return ValidateFS(os.DirFS(root), root) }

// ValidateFS checks every binding and pack in fsys. Problems are sorted by path.
func ValidateFS(fsys fs.FS, display string) (*Catalog, []Problem) {
	c, probs := LoadFS(fsys, display)
	add := func(p, format string, a ...any) {
		probs = append(probs, Problem{c.Display(p), fmt.Sprintf(format, a...)})
	}

	names := map[string]string{}
	for _, b := range c.Bindings {
		p := b.SourcePath
		if prev, dup := names[b.Name]; dup {
			add(p, "binding name %q already used by %s", b.Name, c.Display(prev))
		}
		names[b.Name] = p
		if b.Name == "" {
			add(p, "binding needs a name")
		}
		if len(b.Tags) == 0 {
			add(p, "binding needs at least one tag")
		}
		u := b.Upstream
		if u.Repo != "" && (u.Path == "" || u.Ref == "") {
			add(p, "upstream needs repo, path and ref together")
		}
		if u.Repo != "" && !RepoRe.MatchString(u.Repo) {
			add(p, "upstream.repo must be a host path like github.com/owner/repo, got %q", u.Repo)
		}
		if u.Ref != "" && !RefRe.MatchString(u.Ref) {
			add(p, "upstream.ref must be a full 40-character commit SHA, got %q", u.Ref)
		}
		if u.Repo != "" && !contentHashRe.MatchString(u.ContentHash) {
			add(p, "upstream.content_hash must be sha256:<64 hex> (pin it with `paws pin`), got %q", u.ContentHash)
		}
		if u.Path != "" {
			if err := rule.RelativeInside(u.Path); err != nil {
				add(p, "upstream.path %v", err)
			}
		}
		dir := path.Dir(p)
		if b.Action.Type == "require-skill" {
			if !skillNameRe.MatchString(b.Action.Skill) {
				add(p, "skill %q must be lowercase letters, digits and dashes", b.Action.Skill)
			} else if u.Repo == "" {
				local := LocalSkillDir(b) + "/SKILL.md"
				if !isRegular(fsys, local) {
					add(p, "require-skill %q has no upstream and no local skill at %s", b.Action.Skill, c.Display(local))
				}
			}
		}
		if b.Action.Digest != "" {
			d := path.Join(dir, filepath.ToSlash(b.Action.Digest))
			if st, err := fs.Stat(fsys, d); err != nil || !st.Mode().IsRegular() {
				add(p, "digest file %s not found", b.Action.Digest)
			} else if st.Size() > rule.MaxDigest {
				add(p, "digest file %s is %d bytes, over the %d-byte limit", b.Action.Digest, st.Size(), rule.MaxDigest)
			}
		}
		if b.Action.Type == "exec-script" && !isRegular(fsys, path.Join(dir, filepath.ToSlash(b.Action.Script))) {
			add(p, "script %s not found", b.Action.Script)
		}
	}

	for _, p := range c.Packs {
		want := strings.TrimSuffix(path.Base(p.Path), ".yaml")
		if p.Name != want {
			add(p.Path, "pack name %q must match its file name %q", p.Name, want)
		}
		if !packNameRe.MatchString(p.Name) {
			add(p.Path, "pack name %q must be lowercase letters, digits and dashes", p.Name)
		}
		if p.Description == "" {
			add(p.Path, "pack needs a description")
		}
		if len(p.Bindings) == 0 && len(p.PickOne) == 0 {
			add(p.Path, "pack lists no bindings")
		}
		// Installing a pack takes the top-level bindings plus one option per group, so a binding
		// may repeat across options of one group but never within a set that installs together.
		top := map[string]bool{}
		ref := func(rel, where string, seen map[string]bool) {
			if seen[rel] || (seen != nil && top[rel] && where != "bindings") {
				add(p.Path, "binding %s listed more than once (%s)", rel, where)
			}
			seen[rel] = true
			if _, ok := c.Bindings[rel]; !ok {
				add(p.Path, "binding %s (%s) is not a valid binding under catalog/", rel, where)
			}
		}
		for _, rel := range p.Bindings {
			ref(rel, "bindings", top)
		}
		groups := map[string]bool{}
		for _, g := range p.PickOne {
			if g.Name == "" {
				add(p.Path, "pick_one group needs a name")
			} else if groups[g.Name] {
				add(p.Path, "pick_one group %q defined more than once", g.Name)
			}
			groups[g.Name] = true
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
				seen := map[string]bool{}
				for _, rel := range rels {
					ref(rel, "pick_one "+g.Name, seen)
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
