package rule

import "strings"

var mcpActions = map[string]bool{"send": true, "create": true, "update": true, "delete": true}

// mcpVerbs maps a verb word in an MCP tool name to its action class ("" = read). Server
// names vary by install (mcp__slack__, mcp__plugin_slack_slack__, mcp__claude_ai_Gmail__),
// so classification reads only the tool's own name, and the leftmost verb wins: in
// slack_create_list "create" is the verb and "list" its object; in list_messages "list"
// is the verb. Words left out on purpose: "message" and "comment"-as-noun (objects, see
// sendObjects), "respond" (RSVPs), "resolve" (resolve-library-id reads).
var mcpVerbs = map[string]string{}

func init() {
	for class, verbs := range map[string][]string{
		"":       {"get", "list", "read", "search", "fetch", "query", "find", "view", "download", "export", "describe", "count"},
		"delete": {"delete", "remove", "trash", "archive", "destroy", "unshare", "cancel"},
		"send":   {"send", "reply", "forward", "publish", "draft", "comment"},
		"update": {"update", "edit", "modify", "patch", "set", "rename", "move", "label", "unlabel", "mark", "unmark", "untrash", "restore", "apply", "merge", "submit", "share", "push"},
		"create": {"create", "add", "insert", "new", "upload", "save", "duplicate", "spawn", "generate", "schedule", "post", "write"},
	} {
		for _, v := range verbs {
			mcpVerbs[v] = class
		}
	}
}

// sendObjects turn a create into a send when they are what gets created: add_issue_comment,
// create_draft, schedule_message, post_message. Prose that leaves the agent is a send.
var sendObjects = map[string]bool{"comment": true, "draft": true, "message": true, "reply": true}

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
	words := strings.FieldsFunc(strings.ToLower(rest[i+2:]), func(r rune) bool { return r == '_' || r == '-' })
	for j, w := range words {
		class, ok := mcpVerbs[w]
		if !ok {
			class, ok = mcpVerbs[strings.TrimSuffix(w, "s")]
		}
		if !ok {
			continue
		}
		// Promote on the object, the last word: add_issue_comment, create_draft, update_draft.
		last := strings.TrimSuffix(words[len(words)-1], "s")
		if j < len(words)-1 && (class == "create" && sendObjects[last] || class == "update" && last == "draft") {
			return "send"
		}
		return class
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
