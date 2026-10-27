// Package gate is the paws decision engine: discover rules, resolve them to a decision, run exec-scripts, audit.
package gate

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/websentry-ai/paws/internal/rule"
)

// Result is the rules in scope, plus per-file parse errors and the .paws dirs consulted (nearest-first).
type Result struct {
	Rules []*rule.Rule
	Errs  []FileError
	Dirs  []string
}

type FileError struct {
	Path string
	Err  error
}

// UserDir is the per-user paws directory: $PAWS_HOME when set to an absolute path, else ~/.paws.
// A relative PAWS_HOME is ignored: it would resolve against each tool call's cwd. "" if nothing resolves.
func UserDir() string {
	if d := os.Getenv("PAWS_HOME"); d != "" && filepath.IsAbs(d) {
		return filepath.Clean(d)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".paws")
}

// Collect gathers rules from every .paws dir in scope for cwd, up to the repo root (or $HOME), plus UserDir.
func Collect(cwd string) Result {
	var res Result
	for _, dir := range pawsDirs(cwd) {
		res.Dirs = append(res.Dirs, dir)
		loadDir(dir, &res)
	}
	return res
}

func pawsDirs(cwd string) []string {
	var dirs []string
	seen := map[string]bool{}
	home, _ := os.UserHomeDir()

	addDir := func(p string) {
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			dirs = append(dirs, p)
		}
	}
	user := UserDir()
	add := func(base string) {
		p := filepath.Join(base, ".paws")
		// ~/.paws is the user dir, not a project dir: when PAWS_HOME replaces it, it must not load.
		if home != "" && p == filepath.Join(home, ".paws") && p != user {
			return
		}
		addDir(p)
	}

	// Without a repo root or $HOME as an ancestor, trust only cwd itself —
	// never shared parents like /tmp or /.
	boundary := cwd
	for cur := cwd; ; {
		if hasGit(cur) || cur == home {
			boundary = cur
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	for cur := cwd; ; {
		add(cur)
		if cur == boundary {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	if user != "" {
		addDir(user)
	}
	return dirs
}

func hasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func loadDir(dir string, res *Result) {
	// WalkDir doesn't follow a symlinked root — resolve it first so a team-shared
	// .paws directory (a common pattern) doesn't silently load zero rules.
	root := dir
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		root = resolved
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, _ := filepath.Rel(root, path); !rule.IsRuleFile(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			res.Errs = append(res.Errs, FileError{path, err})
			return nil
		}
		r, err := rule.Parse(data, path)
		if err != nil {
			res.Errs = append(res.Errs, FileError{path, err})
			return nil
		}
		res.Rules = append(res.Rules, r)
		return nil
	})
}
