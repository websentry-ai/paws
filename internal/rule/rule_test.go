package rule

import "testing"

func mustRule(t *testing.T, md string) *Rule {
	t.Helper()
	r, err := Parse([]byte(md), "test.md")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func TestFrontmatter(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"basic", "---\nname: x\n---\nbody", "\nname: x", false},
		{"leading blank lines", "\n\n---\nname: x\n---\n", "\nname: x", false},
		{"utf8 bom", "\xef\xbb\xbf---\nname: x\n---\n", "\nname: x", false},
		{"no fence", "name: x\nbody", "", true},
		{"unterminated", "---\nname: x\nbody", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := frontmatter([]byte(c.in))
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestParseValidation(t *testing.T) {
	cases := []struct {
		name string
		md   string
	}{
		{"missing event", "---\nname: x\naction:\n  type: block\n---\n"},
		{"unknown event", "---\nname: x\nevent: bogus\naction:\n  type: block\n---\n"},
		{"session-start rejected", "---\nname: x\nevent: session-start\naction:\n  type: block\n---\n"},
		{"missing action type", "---\nname: x\nevent: pre-tool\n---\n"},
		{"unknown action", "---\nname: x\nevent: pre-tool\naction:\n  type: nope\n---\n"},
		{"require-skill no skill", "---\nname: x\nevent: pre-tool\naction:\n  type: require-skill\n---\n"},
		{"exec-script absolute path", "---\nname: x\nevent: pre-tool\naction:\n  type: exec-script\n  script: /etc/passwd\n---\n"},
		{"exec-script traversal", "---\nname: x\nevent: pre-tool\naction:\n  type: exec-script\n  script: ../../etc/passwd\n---\n"},
		{"bad regex", "---\nname: x\nevent: pre-tool\nmatcher:\n  command_regex: \"([\"\naction:\n  type: block\n---\n"},
		{"warn on user-prompt", "---\nname: x\nevent: user-prompt\naction:\n  type: warn\n---\n"},
		{"unknown on_miss", "---\nname: x\nevent: pre-tool\naction:\n  type: require-skill\n  skill: s\n  on_miss: maybe\n---\n"},
		{"digest without file", "---\nname: x\nevent: pre-tool\naction:\n  type: require-skill\n  skill: s\n  on_miss: digest\n---\n"},
		{"on_miss on block", "---\nname: x\nevent: pre-tool\naction:\n  type: block\n  on_miss: load-required\n---\n"},
		{"digest traversal", "---\nname: x\nevent: pre-tool\naction:\n  type: require-skill\n  skill: s\n  on_miss: digest\n  digest: ../secret.md\n---\n"},
		{"audit with message", "---\nname: x\nevent: pre-tool\naction:\n  type: audit\n  message: hi\n---\n"},
		{"prompt-inject no message", "---\nname: x\nevent: user-prompt\naction:\n  type: prompt-inject\n---\n"},
		{"prompt-inject on pre-tool", "---\nname: x\nevent: pre-tool\naction:\n  type: prompt-inject\n  message: hi\n---\n"},
		{"pre-push non-bash tool", "---\nname: x\nevent: pre-push\nmatcher:\n  tool: Edit\naction:\n  type: warn\n---\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Parse([]byte(c.md), "t.md"); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestExecScriptSubdirAllowed(t *testing.T) {
	r := mustRule(t, "---\nname: x\nevent: pre-tool\naction:\n  type: exec-script\n  script: scripts/foo.sh\n---\n")
	if r.Action.Script != "scripts/foo.sh" {
		t.Errorf("expected subdir script path to be accepted, got %q", r.Action.Script)
	}
}

func TestParseWildcardToolQuoted(t *testing.T) {
	r := mustRule(t, "---\nname: x\nevent: pre-tool\nmatcher:\n  tool: \"*\"\naction:\n  type: warn\n  message: hi\n---\n")
	if !r.Matches("AnyTool", "anything") {
		t.Error("quoted \"*\" should match any tool")
	}
}

func TestMatchTool(t *testing.T) {
	cases := []struct {
		spec, tool string
		want       bool
	}{
		{"", "Bash", true},
		{"*", "Anything", true},
		{"Bash", "Bash", true},
		{"Bash", "Edit", false},
		{"Edit|Write", "Write", true},
		{"Edit|Write", "Read", false},
		{" Edit | Write ", "Edit", true},
	}
	for _, c := range cases {
		if got := matchTool(c.spec, c.tool); got != c.want {
			t.Errorf("matchTool(%q,%q)=%v want %v", c.spec, c.tool, got, c.want)
		}
	}
}

func TestMatchesGitSyntheticEvents(t *testing.T) {
	push := mustRule(t, "---\nname: p\nevent: pre-push\naction:\n  type: warn\n---\n")
	commit := mustRule(t, "---\nname: c\nevent: pre-commit\naction:\n  type: warn\n---\n")

	if !push.Matches("Bash", "git push origin main") {
		t.Error("pre-push should match git push")
	}
	if push.Matches("Bash", "git commit -m x") {
		t.Error("pre-push should NOT match git commit")
	}
	if push.Matches("Edit", "git push") {
		t.Error("pre-push should only apply to Bash")
	}
	if !commit.Matches("Bash", "git commit -am wip") {
		t.Error("pre-commit should match git commit")
	}
	if commit.Matches("Bash", `git tag -m "first commit"`) {
		t.Error("pre-commit should not fire on a commit-flavored commit message")
	}
	if push.Matches("Bash", `echo "we should git push soon"`) {
		t.Error("pre-push should not fire on echoed text, only an actual git push")
	}
	if !push.Matches("Bash", "git -C sub push") {
		t.Error("pre-push should match git with flags before the subcommand")
	}
}

func TestParseBindingSuperset(t *testing.T) {
	r := mustRule(t, `---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+create\b
action:
  type: require-skill
  skill: humanizer
  on_miss: digest
  digest: humanizer.digest.md
upstream:
  repo: github.com/blader/humanizer
  path: SKILL.md
  ref: main
  content_hash: sha256:abc
provenance:
  stars: 56905
  license: MIT
  last_push: 2026-09-28
  fetched: 2026-10-14
tags: [practice, writing]
some_future_key: ignored
---
Removes the AI tells from PR bodies.
`)
	if r.Action.OnMiss != OnMissDigest || r.Action.Digest != "humanizer.digest.md" {
		t.Errorf("on_miss/digest not parsed: %+v", r.Action)
	}
	if r.Upstream.Repo != "github.com/blader/humanizer" || r.Upstream.ContentHash != "sha256:abc" {
		t.Errorf("upstream not parsed: %+v", r.Upstream)
	}
	if r.Provenance.Stars != 56905 || r.Provenance.License != "MIT" {
		t.Errorf("provenance not parsed: %+v", r.Provenance)
	}
	if len(r.Tags) != 2 || r.Tags[1] != "writing" {
		t.Errorf("tags not parsed: %v", r.Tags)
	}
	if !r.Matches("Bash", "git add . && gh pr create --fill") {
		t.Error("binding should match like a plain rule")
	}
}
