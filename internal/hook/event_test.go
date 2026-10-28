package hook

import "testing"

func TestCommandExtraction(t *testing.T) {
	cases := []struct {
		tool  string
		input map[string]any
		want  string
	}{
		{"Bash", map[string]any{"command": "ls -la"}, "ls -la"},
		{"Edit", map[string]any{"file_path": "a/b.env"}, "a/b.env"},
		{"WebFetch", map[string]any{"url": "http://x"}, "http://x"},
		{"Grep", map[string]any{"pattern": "TODO"}, "TODO"},
		{"mcp__gh__search", map[string]any{"q": "x"}, `{"q":"x"}`},
		{"UnknownTool", map[string]any{}, "UnknownTool"},
	}
	for _, c := range cases {
		in := &Input{ToolName: c.tool, ToolInput: c.input}
		if got := in.Command(); got != c.want {
			t.Errorf("%s: got %q want %q", c.tool, got, c.want)
		}
	}
}

func TestMatchTargetSessionSource(t *testing.T) {
	in := &Input{HookEventName: "SessionStart", Source: "resume", ToolInput: map[string]any{}}
	if got := in.MatchTarget(); got != "resume" {
		t.Errorf("session event should target the start source, got %q", got)
	}
}
