package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

const upstreamBinding = `---
name: %[1]s
event: pre-tool
matcher:
  tool: Bash
  command_regex: gh\s+pr\s+create
action:
  type: require-skill
  skill: %[2]s
upstream:
  repo: github.com/example/%[1]s
  path: SKILL.md
  ref: main
tags: [writing]
---
`

func binding(name, skill string) string {
	return fmt.Sprintf(upstreamBinding, name, skill)
}

// goodRoot builds a catalog with one upstream binding, one org binding with a local skill and digest, and a pack.
func goodRoot(t *testing.T) string {
	root := t.TempDir()
	write(t, filepath.Join(root, "catalog/writing/humanizer-pr.md"), binding("humanizer-pr", "humanizer"))
	write(t, filepath.Join(root, "catalog/writing/stop-slop-pr.md"), binding("stop-slop-pr", "stop-slop"))
	write(t, filepath.Join(root, "catalog/writing/README.md"), "# not a binding\n")
	write(t, filepath.Join(root, "catalog/aws/s3-create.md"), `---
name: aws-s3-create
event: pre-tool
matcher:
  tool: Bash
  command_regex: aws\s+s3\s+mb
action:
  type: require-skill
  skill: aws-s3
  on_miss: digest
  digest: digests/aws-s3.md
tags: [org, aws]
---
`)
	write(t, filepath.Join(root, "catalog/aws/digests/aws-s3.md"), "Block public access.\n")
	write(t, filepath.Join(root, "catalog/aws/skills/aws-s3/SKILL.md"), "---\nname: aws-s3\n---\n")
	write(t, filepath.Join(root, "packs/writing.yaml"), `name: writing
description: Humanize prose before it ships.
tags: [writing]
pick_one:
  - name: prose
    default: humanizer
    options:
      humanizer: [writing/humanizer-pr.md]
      stop-slop: [writing/stop-slop-pr.md]
`)
	write(t, filepath.Join(root, "packs/aws.yaml"), "name: aws\ndescription: AWS guardrails.\nbindings: [aws/s3-create.md]\n")
	return root
}

func TestValidateGoodCatalog(t *testing.T) {
	c, probs := Validate(goodRoot(t))
	if len(probs) != 0 {
		t.Fatalf("unexpected problems: %v", probs)
	}
	if len(c.Bindings) != 3 {
		t.Errorf("want 3 bindings (README, SKILL.md and digests skipped), got %d: %v", len(c.Bindings), c.Bindings)
	}
	if len(c.Packs) != 2 {
		t.Errorf("want 2 packs, got %d", len(c.Packs))
	}
}

func TestValidateProblems(t *testing.T) {
	cases := []struct {
		name    string
		breakIt func(t *testing.T, root string)
		want    string
	}{
		{"unparseable binding", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/bad.md"), "---\nname: bad\nevent: nope\n---\n")
		}, "unknown event"},
		{"duplicate binding name", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/other/dup.md"), binding("humanizer-pr", "humanizer"))
		}, `binding name "humanizer-pr" already used`},
		{"missing tags", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/untagged.md"), "---\nname: untagged\nevent: pre-tool\naction:\n  type: block\n---\n")
		}, "at least one tag"},
		{"partial upstream", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/partial.md"), strings.Replace(binding("partial", "x"), "  ref: main\n", "", 1))
		}, "repo, path and ref together"},
		{"bad content hash", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/hash.md"), strings.Replace(binding("hash", "x"), "  ref: main\n", "  ref: main\n  content_hash: md5:1\n", 1))
		}, "sha256:<64 hex>"},
		{"require-skill with no source", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/aws/orphan.md"), "---\nname: orphan\nevent: pre-tool\naction:\n  type: require-skill\n  skill: ghost\ntags: [org]\n---\n")
		}, `require-skill "ghost" has no upstream and no local skill`},
		{"missing digest file", func(t *testing.T, root string) {
			os.Remove(filepath.Join(root, "catalog/aws/digests/aws-s3.md"))
		}, "digest file digests/aws-s3.md not found"},
		{"oversize digest", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/aws/digests/aws-s3.md"), strings.Repeat("x", 7001))
		}, "over the 7000-byte limit"},
		{"pack name mismatch", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/aws.yaml"), "name: amazon\ndescription: x\nbindings: [aws/s3-create.md]\n")
		}, `must match its file name "aws"`},
		{"pack references missing binding", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/aws.yaml"), "name: aws\ndescription: x\nbindings: [aws/nope.md]\n")
		}, "aws/nope.md (bindings) is not a valid binding"},
		{"pick_one default not an option", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: x\npick_one:\n  - name: prose\n    default: nope\n    options:\n      humanizer: [writing/humanizer-pr.md]\n      stop-slop: [writing/stop-slop-pr.md]\n")
		}, `default "nope" is not one of its options`},
		{"pick_one with one option", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: x\npick_one:\n  - name: prose\n    default: humanizer\n    options:\n      humanizer: [writing/humanizer-pr.md]\n")
		}, "at least two options"},
		{"binding listed twice", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/aws.yaml"), "name: aws\ndescription: x\nbindings: [aws/s3-create.md, aws/s3-create.md]\n")
		}, "listed more than once"},
		{"misspelled frontmatter key", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/typo.md"), strings.Replace(binding("typo", "x"), "command_regex", "comand_regex", 1))
		}, "field comand_regex not found"},
		{"skill name traversal", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/evil.md"), binding("evil", "../../evil"))
		}, "lowercase letters, digits and dashes"},
		{"upstream path traversal", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "catalog/writing/up.md"), strings.Replace(binding("up", "x"), "path: SKILL.md", "path: ../../SKILL.md", 1))
		}, "upstream.path must be a relative path"},
		{"yml manifest", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/extra.yml"), "name: extra\n")
		}, "use the .yaml extension"},
		{"digest is a directory", func(t *testing.T, root string) {
			os.Remove(filepath.Join(root, "catalog/aws/digests/aws-s3.md"))
			if err := os.MkdirAll(filepath.Join(root, "catalog/aws/digests/aws-s3.md"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "digest file digests/aws-s3.md not found"},
		{"duplicate pick_one group", func(t *testing.T, root string) {
			g := "  - name: prose\n    default: humanizer\n    options:\n      humanizer: [writing/humanizer-pr.md]\n      stop-slop: [writing/stop-slop-pr.md]\n"
			write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: x\npick_one:\n"+g+g)
		}, `pick_one group "prose" defined more than once`},
		{"option repeats a top-level binding", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: x\nbindings: [writing/humanizer-pr.md]\npick_one:\n  - name: prose\n    default: humanizer\n    options:\n      humanizer: [writing/humanizer-pr.md]\n      stop-slop: [writing/stop-slop-pr.md]\n")
		}, "listed more than once (pick_one prose)"},
		{"empty pack", func(t *testing.T, root string) {
			write(t, filepath.Join(root, "packs/aws.yaml"), "name: aws\ndescription: x\n")
		}, "lists no bindings"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := goodRoot(t)
			c.breakIt(t, root)
			_, probs := Validate(root)
			for _, p := range probs {
				if strings.Contains(p.Msg, c.want) {
					return
				}
			}
			t.Errorf("want a problem containing %q, got %v", c.want, probs)
		})
	}
}

func TestPickOneOptionsMayShareABinding(t *testing.T) {
	root := goodRoot(t)
	write(t, filepath.Join(root, "catalog/writing/shared.md"), binding("shared", "shared"))
	write(t, filepath.Join(root, "packs/writing.yaml"), "name: writing\ndescription: x\npick_one:\n  - name: prose\n    default: humanizer\n    options:\n      humanizer: [writing/humanizer-pr.md, writing/shared.md]\n      stop-slop: [writing/stop-slop-pr.md, writing/shared.md]\n")
	if _, probs := Validate(root); len(probs) != 0 {
		t.Errorf("options installing separately may share a binding, got %v", probs)
	}
}
