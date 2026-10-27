// Package pack installs and removes catalog packs: bindings into a packs directory the gate
// loads, the skills they require into Claude Code's skills directory, and a state file.
package pack

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/websentry-ai/paws/internal/catalog"
)

// Target is where a pack installs.
type Target struct {
	PacksDir  string // bindings go to <PacksDir>/<pack>/<catalog-relative path>
	SkillsDir string // skills go to <SkillsDir>/<skill>
	StateFile string
}

// UserTarget installs for one user: bindings under the paws user dir (loaded in every repo),
// skills into Claude Code's user skills ($CLAUDE_CONFIG_DIR/skills or ~/.claude/skills).
func UserTarget(userDir string) Target {
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		home, _ := os.UserHomeDir()
		claude = filepath.Join(home, ".claude")
	}
	return Target{
		PacksDir:  filepath.Join(userDir, "packs"),
		SkillsDir: filepath.Join(claude, "skills"),
		StateFile: filepath.Join(userDir, "installed.yaml"),
	}
}

// ProjectTarget installs into a repo, to commit and share: .paws/packs and .claude/skills.
func ProjectTarget(root string) Target {
	return Target{
		PacksDir:  filepath.Join(root, ".paws", "packs"),
		SkillsDir: filepath.Join(root, ".claude", "skills"),
		StateFile: filepath.Join(root, ".paws", "installed.yaml"),
	}
}

// Installed is one pack's record in the state file.
type Installed struct {
	Bindings []string          `yaml:"bindings"`
	Picks    map[string]string `yaml:"picks,omitempty"`
	Skills   []string          `yaml:"skills,omitempty"` // skills paws installed for this pack
	At       string            `yaml:"at"`
}

// State is the state file: installed packs by name.
type State struct {
	Packs map[string]*Installed `yaml:"packs"`
}

// LoadState reads the state file; a missing file is an empty state.
func LoadState(file string) (*State, error) {
	s := &State{Packs: map[string]*Installed{}}
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if s.Packs == nil {
		s.Packs = map[string]*Installed{}
	}
	return s, nil
}

func (s *State) save(file string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// Resolve turns a pack name, or a single binding ("writing/humanizer-pr"), into the pack name
// to record and the bindings to install. picks chooses one option per pick_one group; groups
// left out take their default (or ask when choose is set).
func Resolve(c *catalog.Catalog, name string, picks map[string]string, choose func(catalog.PickOne) string) (string, []string, map[string]string, error) {
	if p, ok := c.Packs[name]; ok {
		bindings := append([]string{}, p.Bindings...)
		chosen := map[string]string{}
		for g := range picks {
			if !hasGroup(p, g) {
				return "", nil, nil, fmt.Errorf("pack %s has no pick_one group %q", name, g)
			}
		}
		for _, g := range p.PickOne {
			opt, ok := picks[g.Name]
			if !ok && choose != nil {
				opt = choose(g)
			}
			if opt == "" {
				opt = g.Default
			}
			rels, ok := g.Options[opt]
			if !ok {
				return "", nil, nil, fmt.Errorf("pack %s: %q is not an option of %s (have %s)", name, opt, g.Name, strings.Join(optionNames(g), ", "))
			}
			chosen[g.Name] = opt
			bindings = append(bindings, rels...)
		}
		return name, bindings, chosen, nil
	}
	rel := strings.TrimSuffix(name, ".md") + ".md"
	if _, ok := c.Bindings[rel]; ok {
		if len(picks) > 0 {
			return "", nil, nil, fmt.Errorf("--pick applies to packs, not the single binding %s", name)
		}
		return strings.TrimSuffix(rel, ".md"), []string{rel}, nil, nil
	}
	return "", nil, nil, fmt.Errorf("no pack or binding named %q (try `paws search`)", name)
}

func hasGroup(p *catalog.Pack, name string) bool {
	for _, g := range p.PickOne {
		if g.Name == name {
			return true
		}
	}
	return false
}

// InstallDir is where an install's bindings live. Packs take <packs>/<pack>; a single binding
// (category/name) goes under <packs>/_bindings so it can never share a directory with a pack
// named like its category. Pack names can't start with "_".
func InstallDir(t Target, name string) string {
	if strings.Contains(name, "/") {
		return filepath.Join(t.PacksDir, "_bindings", filepath.FromSlash(name))
	}
	return filepath.Join(t.PacksDir, name)
}

func optionNames(g catalog.PickOne) []string {
	var names []string
	for n := range g.Options {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Result describes what Add did, for the CLI to print.
type Result struct {
	Pack      string
	Bindings  []string
	Installed []string // skills paws installed or updated
	Kept      []string // skills already present that paws does not own; left as they are
}

// Add installs bindings as pack name into t. Re-adding a pack replaces it (an upgrade).
func Add(c *catalog.Catalog, name string, bindings []string, picks map[string]string, t Target) (*Result, error) {
	st, err := LoadState(t.StateFile)
	if err != nil {
		return nil, err
	}
	res := &Result{Pack: name, Bindings: bindings}
	fresh := map[string]bool{} // skills this call created where nothing was before
	ok := false
	defer func() {
		if !ok { // a later step failed: take back the skills this call created
			for s := range fresh {
				removeSkill(t.SkillsDir, s)
			}
		}
	}()

	// Skills first: a binding must never be live before the skill it requires can load.
	done := map[string]bool{}
	for _, rel := range bindings {
		b := c.Bindings[rel]
		if b.Action.Type != "require-skill" || done[b.Action.Skill] {
			continue
		}
		done[b.Action.Skill] = true
		existed := exists(filepath.Join(t.SkillsDir, b.Action.Skill))
		installed, err := installBindingSkill(c, b, t.SkillsDir)
		if err != nil {
			return nil, fmt.Errorf("skill %s: %w", b.Action.Skill, err)
		}
		if installed {
			if !existed {
				fresh[b.Action.Skill] = true
			}
			res.Installed = append(res.Installed, b.Action.Skill)
		} else {
			res.Kept = append(res.Kept, b.Action.Skill)
		}
	}

	dest := InstallDir(t, name)
	stage := filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".new")
	os.RemoveAll(stage)
	defer os.RemoveAll(stage)
	for _, rel := range bindings {
		b := c.Bindings[rel]
		files := []string{rel}
		dir := path.Dir(rel)
		if b.Action.Digest != "" {
			files = append(files, path.Join(dir, filepath.ToSlash(b.Action.Digest)))
		}
		if b.Action.Type == "exec-script" {
			files = append(files, path.Join(dir, filepath.ToSlash(b.Action.Script)))
		}
		for _, f := range files {
			dest := filepath.Join(stage, filepath.FromSlash(f))
			if err := copyFile(c.FS, catalog.BindingPath(f), dest); err != nil {
				return nil, err
			}
		}
		if b.Action.Type == "exec-script" {
			if err := os.Chmod(filepath.Join(stage, filepath.FromSlash(path.Join(dir, b.Action.Script))), 0o755); err != nil {
				return nil, err
			}
		}
	}
	if err := os.RemoveAll(dest); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(stage, dest); err != nil {
		return nil, err
	}

	// A skill another pack also installed is reinstalled (it is paws-owned), so every pack that
	// needs it lists it and Remove keeps it until the last one goes.
	var dropped []string // skills an earlier install of this pack needed and this one doesn't
	if prev := st.Packs[name]; prev != nil {
		for _, sk := range prev.Skills {
			if !done[sk] {
				dropped = append(dropped, sk)
			}
		}
	}
	st.Packs[name] = &Installed{Bindings: bindings, Picks: picks, Skills: res.Installed, At: time.Now().UTC().Format(time.RFC3339)}
	if err := st.save(t.StateFile); err != nil {
		return nil, err
	}
	ok = true
	for _, sk := range dropped {
		if !st.needs(sk) {
			removeSkill(t.SkillsDir, sk)
		}
	}
	return res, nil
}

// needs reports whether any installed pack lists skill.
func (s *State) needs(skill string) bool {
	for _, p := range s.Packs {
		for _, sk := range p.Skills {
			if sk == skill {
				return true
			}
		}
	}
	return false
}

// installBindingSkill installs the skill a require-skill binding names, from upstream at the
// pinned ref (hash-checked) or from the catalog's own skills/ directory.
func installBindingSkill(c *catalog.Catalog, b *catalog.Binding, skillsDir string) (bool, error) {
	if !catalog.ValidSkillName(b.Action.Skill) {
		return false, fmt.Errorf("invalid skill name %q", b.Action.Skill)
	}
	if dest := filepath.Join(skillsDir, b.Action.Skill); exists(dest) && !replaceable(dest) {
		return false, nil // the user's own (or edited) copy wins; nothing to fetch
	}
	u := b.Upstream
	if u.Repo == "" {
		src, err := subFS(c, catalog.LocalSkillDir(b))
		if err != nil {
			return false, err
		}
		return installSkill(src, skillsDir, b.Action.Skill, marker{Skill: b.Action.Skill, Source: "catalog:" + catalog.LocalSkillDir(b)})
	}
	dir, tmp, hash, err := FetchUpstream(u.Repo, u.Ref, u.Path)
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	if hash != u.ContentHash {
		return false, fmt.Errorf("content hash mismatch for %s@%s: catalog pins %s, upstream has %s; refusing to install", u.Repo, u.Ref, u.ContentHash, hash)
	}
	return installSkill(os.DirFS(dir), skillsDir, b.Action.Skill, marker{Skill: b.Action.Skill, Source: u.Repo + "@" + u.Ref, Hash: hash})
}

// Remove uninstalls a pack: its bindings, and skills paws installed for it that no other
// installed pack still needs.
func Remove(name string, t Target) (removedSkills []string, err error) {
	st, err := LoadState(t.StateFile)
	if err != nil {
		return nil, err
	}
	inst, ok := st.Packs[name]
	if !ok {
		return nil, fmt.Errorf("pack %q is not installed", name)
	}
	if err := os.RemoveAll(InstallDir(t, name)); err != nil {
		return nil, err
	}
	delete(st.Packs, name)
	for _, s := range inst.Skills {
		if st.needs(s) {
			continue
		}
		removed, err := removeSkill(t.SkillsDir, s)
		if err != nil {
			return removedSkills, err
		}
		if removed {
			removedSkills = append(removedSkills, s)
		}
	}
	return removedSkills, st.save(t.StateFile)
}

// subFS roots the catalog FS at dir, failing if dir is not there.
func subFS(c *catalog.Catalog, dir string) (fs.FS, error) {
	if st, err := fs.Stat(c.FS, dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("no skill directory %s", c.Display(dir))
	}
	return fs.Sub(c.FS, dir)
}
