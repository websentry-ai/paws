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
		return strings.TrimSuffix(rel, ".md"), []string{rel}, nil, nil
	}
	return "", nil, nil, fmt.Errorf("no pack or binding named %q (try `paws search`)", name)
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

	// Skills first: a binding must never be live before the skill it requires can load.
	done := map[string]bool{}
	for _, rel := range bindings {
		b := c.Bindings[rel]
		if b.Action.Type != "require-skill" || done[b.Action.Skill] {
			continue
		}
		done[b.Action.Skill] = true
		ok, err := installBindingSkill(c, b, t.SkillsDir)
		if err != nil {
			return nil, fmt.Errorf("skill %s: %w", b.Action.Skill, err)
		}
		if ok {
			res.Installed = append(res.Installed, b.Action.Skill)
		} else {
			res.Kept = append(res.Kept, b.Action.Skill)
		}
	}

	stage := filepath.Join(t.PacksDir, "."+filepath.Base(name)+".new")
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
	dest := filepath.Join(t.PacksDir, filepath.FromSlash(name))
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
	st.Packs[name] = &Installed{Bindings: bindings, Picks: picks, Skills: res.Installed, At: time.Now().UTC().Format(time.RFC3339)}
	return res, st.save(t.StateFile)
}

// installBindingSkill installs the skill a require-skill binding names, from upstream at the
// pinned ref (hash-checked) or from the catalog's own skills/ directory.
func installBindingSkill(c *catalog.Catalog, b *catalog.Binding, skillsDir string) (bool, error) {
	if dest := filepath.Join(skillsDir, b.Action.Skill); exists(dest) && !pawsOwned(dest) {
		return false, nil // the user's own copy wins; nothing to fetch
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
	if err := os.RemoveAll(filepath.Join(t.PacksDir, filepath.FromSlash(name))); err != nil {
		return nil, err
	}
	delete(st.Packs, name)
	still := map[string]bool{}
	for _, other := range st.Packs {
		for _, s := range other.Skills {
			still[s] = true
		}
	}
	for _, s := range inst.Skills {
		if still[s] {
			continue
		}
		if err := removeSkill(t.SkillsDir, s); err != nil {
			return removedSkills, err
		}
		removedSkills = append(removedSkills, s)
	}
	return removedSkills, st.save(t.StateFile)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// subFS roots the catalog FS at dir, failing if dir is not there.
func subFS(c *catalog.Catalog, dir string) (fs.FS, error) {
	if st, err := fs.Stat(c.FS, dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("no skill directory %s", c.Display(dir))
	}
	return fs.Sub(c.FS, dir)
}
