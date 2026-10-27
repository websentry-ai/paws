package gate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/websentry-ai/paws/internal/rule"
)

func repoWith(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, body := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHasDependency(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		pkg   string
		want  bool
	}{
		{"package.json dep", map[string]string{"package.json": `{"dependencies":{"stripe":"^14"}}`}, "stripe", true},
		{"package.json devDep scoped", map[string]string{"package.json": `{"devDependencies":{"@supabase/supabase-js":"2"}}`}, "@supabase/supabase-js", true},
		{"package.json absent", map[string]string{"package.json": `{"dependencies":{"stripe-mock":"1"}}`}, "stripe", false},
		{"go.mod full path", map[string]string{"go.mod": "module x\n\nrequire (\n\tgithub.com/stripe/stripe-go/v76 v76.0.0\n)\n"}, "github.com/stripe/stripe-go/v76", true},
		{"go.mod single require", map[string]string{"go.mod": "module x\nrequire gopkg.in/yaml.v3 v3.0.1\n"}, "gopkg.in/yaml.v3", true},
		{"go.mod own module ignored", map[string]string{"go.mod": "module github.com/me/stripe\n"}, "stripe", false},
		{"Cargo key", map[string]string{"Cargo.toml": "[package]\nname = \"x\"\n\n[dependencies]\nserde = \"1\"\n"}, "serde", true},
		{"Cargo dotted table", map[string]string{"Cargo.toml": "[dependencies.tokio]\nversion = \"1\"\n"}, "tokio", true},
		{"Cargo package name not a dep", map[string]string{"Cargo.toml": "[package]\nname = \"serde\"\n"}, "serde", false},
		{"pyproject list", map[string]string{"pyproject.toml": "[project]\ndependencies = [\n  \"Django>=4\",\n  \"requests\",\n]\n"}, "django", true},
		{"poetry table", map[string]string{"pyproject.toml": "[tool.poetry.dependencies]\nsupabase_py = \"^2\"\n"}, "supabase-py", true},
		{"requirements", map[string]string{"requirements.txt": "# deps\nstripe==8.0\n"}, "stripe", true},
		{"no manifests", map[string]string{}, "stripe", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasDependency(repoWith(t, c.files), c.pkg); got != c.want {
				t.Errorf("hasDependency(%q) = %v, want %v", c.pkg, got, c.want)
			}
		})
	}
}

func TestWhenRepoConditionsResolveFromCwd(t *testing.T) {
	root := repoWith(t, map[string]string{"package.json": `{"dependencies":{"next":"14"}}`, "next.config.js": "", "src/app/page.tsx": ""})
	sub := filepath.Join(root, "src", "app")
	r := mustRule(t, "---\nname: n\nevent: pre-tool\nwhen:\n  - dependency:next\n  - repo-marker:next.config.js\naction:\n  type: audit\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Bash", Target: "ls", Cwd: sub}
	if !Applies(r, ev) {
		t.Error("both repo conditions hold from a subdirectory")
	}
	ev.Cwd = t.TempDir()
	if Applies(r, ev) {
		t.Error("conditions must fail outside that repo")
	}
}

func TestWhenNewFile(t *testing.T) {
	root := repoWith(t, map[string]string{"exists.go": "package x"})
	r := mustRule(t, "---\nname: n\nevent: pre-tool\nmatcher:\n  tool: Write\nwhen: new-file\naction:\n  type: audit\n---\n")
	if !Applies(r, Event{Kind: rule.KindTool, Tool: "Write", Target: filepath.Join(root, "new.go"), Cwd: root}) {
		t.Error("a missing target is a new file")
	}
	if Applies(r, Event{Kind: rule.KindTool, Tool: "Write", Target: filepath.Join(root, "exists.go"), Cwd: root}) {
		t.Error("an existing target is not a new file")
	}
	if !Applies(r, Event{Kind: rule.KindTool, Tool: "Write", Target: "rel/new.go", Cwd: root}) {
		t.Error("a relative target resolves against cwd")
	}
}

const (
	lnEditTest  = `{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Edit","input":{"file_path":"/r/src/a_test.go","old_string":"a","new_string":"b"}}]}}`
	lnEditSide  = `{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Write","input":{"file_path":"/r/src/b_test.go"}}]}}`
	lnAgent     = `{"type":"assistant","isSidechain":false,"message":{"role":"assistant","content":[{"type":"tool_use","id":"a1","name":"Agent","input":{"prompt":"x"}},{"type":"tool_use","id":"a2","name":"Task","input":{"prompt":"y"}}]}}`
	lnUserTyped = `{"type":"user","isSidechain":false,"message":{"role":"user","content":"do the thing"}}`
	lnUserBlock = `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"do it"}]}}`
	lnToolRes   = `{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"a1","content":"done"}]}}`
)

func TestWhenNoEditMatching(t *testing.T) {
	r := mustRule(t, "---\nname: tdd\nevent: pre-tool\nmatcher:\n  file_glob: \"src/**\"\nwhen: no-edit-matching:**/*_test.go\naction:\n  type: audit\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Edit", Target: "/r/src/a.go"}
	ev.Transcript = write(t, lnAssistant)
	if !Applies(r, ev) {
		t.Error("no test edited yet: the rule applies")
	}
	ev.Transcript = write(t, lnEditTest, lnAssistant)
	if Applies(r, ev) {
		t.Error("a test file was edited: the rule must not apply")
	}
	ev.Transcript = write(t, lnEditSide, lnAssistant)
	if !Applies(r, ev) {
		t.Error("subagent edits do not count")
	}
}

func TestWhenAgentCount(t *testing.T) {
	r := mustRule(t, "---\nname: par\nevent: pre-tool\nmatcher:\n  tool: Agent\nwhen: agent-count>1\naction:\n  type: audit\n---\n")
	ev := Event{Kind: rule.KindTool, Tool: "Agent", Target: "x"}
	ev.Transcript = write(t, lnUserTyped, lnAgent, lnToolRes)
	if !Applies(r, ev) {
		t.Error("two spawns since the user turn exceed 1")
	}
	ev.Transcript = write(t, lnAgent, lnUserBlock, lnAssistant)
	if Applies(r, ev) {
		t.Error("spawns before the last user turn do not count")
	}
}

func TestWhenParseErrors(t *testing.T) {
	for name, md := range map[string]string{
		"unknown":            "---\nname: x\nevent: pre-tool\nwhen: phase-of-moon\naction:\n  type: audit\n---\n",
		"empty dependency":   "---\nname: x\nevent: pre-tool\nwhen: \"dependency:\"\naction:\n  type: audit\n---\n",
		"marker traversal":   "---\nname: x\nevent: pre-tool\nwhen: repo-marker:../x\naction:\n  type: audit\n---\n",
		"new-file on prompt": "---\nname: x\nevent: user-prompt\nwhen: new-file\naction:\n  type: audit\n---\n",
		"bad glob":           "---\nname: x\nevent: pre-tool\nwhen: \"no-edit-matching:{a\"\naction:\n  type: audit\n---\n",
		"map not list":       "---\nname: x\nevent: pre-tool\nwhen:\n  a: b\naction:\n  type: audit\n---\n",
	} {
		if _, err := rule.Parse([]byte(md), "t.md"); err == nil {
			t.Errorf("%s: should fail to parse", name)
		}
	}
}
