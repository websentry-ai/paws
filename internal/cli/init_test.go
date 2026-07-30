package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"/plain/path":     `'/plain/path'`,
		"/has'quote":      `'/has'\''quote'`,
		"/has $dollar":    `'/has $dollar'`,
		"/has `backtick`": "'/has `backtick`'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProjectRootWalksToGitRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := projectRoot(sub); got != root {
		t.Errorf("projectRoot(%q) = %q, want %q", sub, got, root)
	}
}

func TestProjectRootFallsBackToCwdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	if got := projectRoot(dir); got != dir {
		t.Errorf("no repo above cwd should fall back to cwd, got %q", got)
	}
}
