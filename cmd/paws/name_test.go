package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The help people read names the short command, and says the long one still exists.
func TestHelpNamesTheShortCommand(t *testing.T) {
	bin := buildPaws(t)
	var stderr bytes.Buffer
	cmd := exec.Command(bin, "--help")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("paws --help: %v", err)
	}
	help := stderr.String()
	for _, want := range []string{"\n  paws init ", "\n  paws add ", "\n  paws login ", "also installed as unbound-paws", "npx unbound-paws init"} {
		if !strings.Contains(help, want) {
			t.Errorf("help is missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "\n  unbound-paws ") {
		t.Errorf("help still lists commands as unbound-paws:\n%s", help)
	}

	// A usage error names the short command too.
	stderr.Reset()
	cmd = exec.Command(bin, "login", "--bogus")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	cmd.Stderr = &stderr
	cmd.Run()
	if !strings.Contains(stderr.String(), "usage: paws login") {
		t.Errorf("login usage error should name `paws login`, got %q", stderr.String())
	}
}

// `paws init` run through the short name registers the hook as `unbound-paws gate`, the line every
// earlier install wrote, and doesn't copy the binary: paws and unbound-paws are the same file on
// PATH, as the installer (a paws link to unbound-paws) and npm (two links to one launcher) lay
// them out.
func TestShortNameInitKeepsTheHookOnUnboundPaws(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs symlinks")
	}
	built := buildPaws(t)
	layouts := map[string]func(t *testing.T, bin string){
		"installer": func(t *testing.T, bin string) {
			copyFile(t, built, filepath.Join(bin, "unbound-paws"))
			mustSymlink(t, "unbound-paws", filepath.Join(bin, "paws"))
		},
		"npm": func(t *testing.T, bin string) {
			launcher := filepath.Join(filepath.Dir(bin), "lib", "node_modules", "unbound-paws", "bin", "paws.js")
			mustMkdirAll(t, filepath.Dir(launcher))
			copyFile(t, built, launcher)
			mustSymlink(t, launcher, filepath.Join(bin, "unbound-paws"))
			mustSymlink(t, launcher, filepath.Join(bin, "paws"))
		},
	}
	for name, layout := range layouts {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			bin := filepath.Join(root, "prefix", "bin")
			repo := filepath.Join(root, "repo")
			mustMkdirAll(t, home)
			mustMkdirAll(t, bin)
			mustMkdirAll(t, filepath.Join(repo, ".git"))
			layout(t, bin)

			cmd := exec.Command(filepath.Join(bin, "paws"), "init")
			cmd.Dir = repo
			cmd.Env = []string{"HOME=" + home, "PATH=" + bin + string(os.PathListSeparator) + "/usr/bin:/bin"}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("paws init: %v\n%s", err, out)
			}
			raw, err := os.ReadFile(filepath.Join(repo, ".claude", "settings.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Hooks map[string][]struct {
					Hooks []struct {
						Command string `json:"command"`
					} `json:"hooks"`
				} `json:"hooks"`
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			if got := doc.Hooks["PreToolUse"][0].Hooks[0].Command; got != "unbound-paws gate" {
				t.Errorf("hook command = %q, want %q", got, "unbound-paws gate")
			}
			if _, err := os.Stat(filepath.Join(home, ".paws", "bin", "paws")); !os.IsNotExist(err) {
				t.Errorf("init copied the binary to ~/.paws/bin although unbound-paws is on PATH (stat: %v)", err)
			}
		})
	}
}

// Hints name a command that runs this paws: `paws` only when it is this install, else `unbound-paws`,
// else (npx, nothing on PATH) `npx unbound-paws`. Never a bare `paws` that is another tool, never `npx paws`.
func TestHintsNameACommandThatRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs symlinks")
	}
	built := buildPaws(t)
	cases := []struct {
		name   string
		layout func(t *testing.T, bin string)
		want   string
	}{
		{"short name is ours", func(t *testing.T, bin string) {
			copyFile(t, built, filepath.Join(bin, "unbound-paws"))
			mustSymlink(t, "unbound-paws", filepath.Join(bin, "paws"))
		}, "`paws init`"},
		{"short name is another tool", func(t *testing.T, bin string) {
			copyFile(t, built, filepath.Join(bin, "unbound-paws"))
			mustWriteFile(t, filepath.Join(bin, "paws"), "#!/bin/sh\necho someone else\n")
			os.Chmod(filepath.Join(bin, "paws"), 0o755)
		}, "`unbound-paws init`"},
		{"nothing on PATH", func(t *testing.T, bin string) {}, "`npx unbound-paws init`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			home := filepath.Join(root, "home")
			bin := filepath.Join(root, "bin")
			repo := filepath.Join(root, "repo")
			mustMkdirAll(t, home)
			mustMkdirAll(t, bin)
			mustMkdirAll(t, filepath.Join(repo, ".git"))
			c.layout(t, bin)
			// `add` before init prints the "not registered yet" hint.
			cmd := exec.Command(built, "add", "writing")
			cmd.Dir = repo
			cmd.Env = []string{"HOME=" + home, "PATH=" + bin + string(os.PathListSeparator) + "/usr/bin:/bin"}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("paws add: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "run "+c.want) {
				t.Errorf("want the hint to say %s, got:\n%s", c.want, out)
			}
			// `paws index` is data for the website and portal: it always says `paws add`.
			cmd = exec.Command(built, "index")
			cmd.Dir = repo
			cmd.Env = []string{"HOME=" + home, "PATH=" + bin + string(os.PathListSeparator) + "/usr/bin:/bin"}
			idx, err := cmd.Output()
			if err != nil || !strings.Contains(string(idx), `"add": "paws add `) && !strings.Contains(string(idx), `"add":"paws add `) {
				t.Errorf("paws index should say paws add whatever is on PATH: %v %.300s", err, idx)
			}
		})
	}
}

// A hook someone wrote by hand as `paws gate` (the name the docs use) is paws's own: init replaces it
// instead of adding a second line that would run the gate twice, and uninstall removes it.
func TestHandWrittenShortHookIsRecognised(t *testing.T) {
	bin := buildPaws(t)
	home := t.TempDir()
	repo := t.TempDir()
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustMkdirAll(t, filepath.Join(repo, ".claude"))
	settings := filepath.Join(repo, ".claude", "settings.json")
	mustWriteFile(t, settings, `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"paws gate"}]}]}}`)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("paws %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	raw, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var gates []string
	for _, e := range doc.Hooks["PreToolUse"] {
		for _, h := range e.Hooks {
			if strings.HasSuffix(h.Command, " gate") {
				gates = append(gates, h.Command)
			}
		}
	}
	if len(gates) != 1 || gates[0] == "paws gate" {
		t.Errorf("want init's one gate line in PreToolUse in place of the hand-written one, got %q", gates)
	}

	mustWriteFile(t, settings, `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"paws gate"}]}]}}`)
	run("uninstall")
	if raw, err := os.ReadFile(settings); err == nil && strings.Contains(string(raw), "paws gate") {
		t.Errorf("uninstall left the hand-written paws gate:\n%s", raw)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}
