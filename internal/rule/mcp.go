package rule

import "strings"

var mcpActions = map[string]bool{"send": true, "create": true, "update": true, "delete": true}

// Verb classes for MCP tool names, checked in this order. Server names vary by install
// (mcp__slack__, mcp__plugin_slack_slack__, mcp__claude_ai_Gmail__), so classification reads
// only the tool's own name: its verbs say what it does, whichever server it came from.
// A tool that reads (get_comments, list_messages) is never classified, even if it also
// names a write noun.
var mcpVerbClasses = []struct {
	class string
	verbs []string
}{
	{"", []string{"get", "list", "read", "search", "fetch", "query", "find", "view", "download", "export", "describe", "count"}},
	{"delete", []string{"delete", "remove", "trash", "archive", "destroy", "unshare", "cancel"}},
	// Drafts and comments are prose that leaves the agent: they count as sends. "message" is
	// deliberately absent: it is the object of label_message and read_message, not a verb.
	{"send", []string{"send", "reply", "forward", "post", "publish", "schedule", "draft", "comment", "respond"}},
	{"update", []string{"update", "edit", "modify", "patch", "set", "rename", "move", "label", "unlabel", "mark", "unmark", "resolve", "untrash", "restore", "apply", "merge", "submit", "share"}},
	{"create", []string{"create", "add", "insert", "new", "upload", "save", "duplicate", "spawn", "generate"}},
}

// MCPAction classifies an MCP tool as send, create, update or delete; "" for reads,
// unrecognised tools and anything that is not an MCP tool.
func MCPAction(tool string) string {
	rest, ok := strings.CutPrefix(tool, "mcp__")
	if !ok {
		return ""
	}
	i := strings.Index(rest, "__")
	if i < 0 {
		return ""
	}
	words := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(rest[i+2:]), func(r rune) bool { return r == '_' || r == '-' }) {
		words[w] = true
		words[strings.TrimSuffix(w, "s")] = true // drafts, comments, messages
	}
	for _, c := range mcpVerbClasses {
		for _, v := range c.verbs {
			if words[v] {
				return c.class
			}
		}
	}
	return ""
}

// matchMCPAction matches a pipe list of action classes against tool.
func matchMCPAction(spec, tool string) bool {
	got := MCPAction(tool)
	if got == "" {
		return false
	}
	for _, want := range strings.Split(spec, "|") {
		if strings.TrimSpace(want) == got {
			return true
		}
	}
	return false
}
