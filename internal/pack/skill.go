package pack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/websentry-ai/paws/internal/catalog"
)

// markerFile sits in every skill directory paws installed. paws only ever replaces or
// removes a skill directory that carries it, so a skill the user wrote or cloned stays theirs.
const markerFile = ".paws-skill.json"

type marker struct {
	Skill  string `json:"skill"`
	Source string `json:"source"` // repo@ref, or catalog:<path>
	Hash   string `json:"hash,omitempty"`
	// Installed is the hash of what paws wrote (after placeholders). A directory whose content
	// no longer matches was edited by the user and is left alone.
	Installed string `json:"installed"`
}

// Fetch checks out repo (a host path, validated by the catalog) at ref (a commit SHA) into
// dest, an empty directory. Tests replace it.
var Fetch = func(repo, ref, dest string) error {
	if !catalog.RepoRe.MatchString(repo) || !catalog.RefRe.MatchString(ref) {
		return fmt.Errorf("refusing to fetch %q at %q: want github.com/owner/repo and a commit SHA", repo, ref)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		// "--" ends options: nothing in the catalog can become a git flag.
		{"fetch", "-q", "--depth", "1", "--", "https://" + repo, ref},
		{"-c", "advice.detachedHead=false", "checkout", "-q", "FETCH_HEAD"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dest
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// skillFiles lists the files that make up a skill directory: regular files, skipping any
// path segment that starts with "." (.git, .github, editor and plugin metadata).
func skillFiles(fsys fs.FS) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != "." && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// HashSkill is the content hash of a skill directory: sha256 over each file's
// slash-separated path and bytes, in sorted path order. Formatted sha256:<hex>.
func HashSkill(fsys fs.FS) (string, error) {
	files, err := skillFiles(fsys)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", f, len(data))
		h.Write(data)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// FetchUpstream checks out an upstream skill and returns its directory inside a temp dir
// (caller removes tmp) and its content hash.
func FetchUpstream(repo, ref, skillPath string) (dir, tmp, hash string, err error) {
	tmp, err = os.MkdirTemp("", "paws-skill-")
	if err != nil {
		return "", "", "", err
	}
	if err := Fetch(repo, ref, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", "", "", err
	}
	dir = filepath.Join(tmp, filepath.FromSlash(path.Dir(skillPath)))
	// The skill directory must resolve inside the checkout: no symlinked parent leading out.
	realTmp, _ := filepath.EvalSymlinks(tmp)
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil || (realDir != realTmp && !strings.HasPrefix(realDir, realTmp+string(filepath.Separator))) {
		os.RemoveAll(tmp)
		return "", "", "", fmt.Errorf("skill path %s does not resolve inside the checkout", skillPath)
	}
	hash, err = HashSkill(os.DirFS(dir))
	if err != nil {
		os.RemoveAll(tmp)
		return "", "", "", err
	}
	return dir, tmp, hash, nil
}

// skillConfig is an org skill's config.yaml: placeholder defaults filled in at install.
type skillConfig struct {
	Placeholders map[string]string `yaml:"placeholders"`
}

// placeholders reads config.yaml from a skill directory; missing means none.
func placeholders(fsys fs.FS) (map[string]string, error) {
	data, err := fs.ReadFile(fsys, "config.yaml")
	if err != nil {
		return nil, nil
	}
	var c skillConfig
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config.yaml: %w", err)
	}
	return c.Placeholders, nil
}

// installSkill copies the skill in src to <skillsDir>/<name>, filling {{PLACEHOLDER}}s in .md
// files. It leaves an existing directory alone, returning installed=false, unless paws installed
// it and nobody edited it since. The swap goes through a sibling temp dir so a failure leaves
// the old copy.
func installSkill(src fs.FS, skillsDir, name string, m marker) (installed bool, err error) {
	if !catalog.ValidSkillName(name) {
		return false, fmt.Errorf("invalid skill name %q", name)
	}
	dest := filepath.Join(skillsDir, name)
	if exists(dest) && !replaceable(dest) {
		return false, nil
	}
	vals, err := placeholders(src)
	if err != nil {
		return false, err
	}
	files, err := skillFiles(src)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return false, err
	}
	stage, err := os.MkdirTemp(skillsDir, "."+name+"-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(stage)
	for _, f := range files {
		data, err := fs.ReadFile(src, f)
		if err != nil {
			return false, err
		}
		if strings.HasSuffix(f, ".md") {
			for k, v := range vals {
				data = []byte(strings.ReplaceAll(string(data), "{{"+k+"}}", v))
			}
		}
		out := filepath.Join(stage, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return false, err
		}
		mode := os.FileMode(0o644)
		if st, err := fs.Stat(src, f); err == nil && st.Mode().Perm()&0o111 != 0 {
			mode = 0o755 // upstream scripts stay runnable
		}
		if err := os.WriteFile(out, data, mode); err != nil {
			return false, err
		}
	}
	if m.Installed, err = HashSkill(os.DirFS(stage)); err != nil {
		return false, err
	}
	mb, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(stage, markerFile), mb, 0o644); err != nil {
		return false, err
	}
	if err := os.RemoveAll(dest); err != nil {
		return false, err
	}
	if err := os.Rename(stage, dest); err != nil {
		return false, err
	}
	return true, nil
}

// pawsOwned reports whether a skill directory carries paws's install marker.
func pawsOwned(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, markerFile))
	return err == nil
}

// replaceable reports whether paws may overwrite or delete a skill directory: it installed it
// and the content still hashes to what it wrote. A user's edits make it theirs.
func replaceable(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, markerFile))
	if err != nil {
		return false
	}
	var m marker
	if json.Unmarshal(data, &m) != nil || m.Installed == "" {
		return false
	}
	now, err := HashSkill(os.DirFS(dir))
	return err == nil && now == m.Installed
}

// removeSkill deletes a skill directory only if paws installed it and it is unedited; it
// reports whether it removed it.
func removeSkill(skillsDir, name string) (bool, error) {
	if !catalog.ValidSkillName(name) {
		return false, fmt.Errorf("invalid skill name %q", name)
	}
	dir := filepath.Join(skillsDir, name)
	if !replaceable(dir) {
		return false, nil
	}
	return true, os.RemoveAll(dir)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// copyFile copies one file out of fsys, creating parent directories.
func copyFile(fsys fs.FS, src, dest string) error {
	in, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// InstallBundledSkill installs a skill that ships inside the binary (fsys rooted at the skill
// directory). Same rules as pack skills: an existing directory paws doesn't own, or that the
// user edited, is kept.
func InstallBundledSkill(fsys fs.FS, skillsDir, name, source string) (bool, error) {
	return installSkill(fsys, skillsDir, name, marker{Skill: name, Source: source})
}
