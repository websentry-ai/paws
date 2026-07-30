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

// Collect gathers rules from every .paws dir in scope for cwd, up to the repo root (or $HOME), plus ~/.paws.
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

	add := func(base string) {
		p := filepath.Join(base, ".paws")
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			dirs = append(dirs, p)
		}
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
	if home != "" {
		add(home)
	}
	return dirs
}

func hasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func loadDir(dir string, res *Result) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
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
