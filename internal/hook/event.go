// Package hook is the Claude Code adapter: hook I/O contract, translated to and from the agent-neutral gate engine.
package hook

import (
	"encoding/json"
	"strings"
)

// Input is the event JSON Claude Code pipes to a hook on stdin; unknown fields are ignored.
type Input struct {
	SessionID      string         `json:"session_id"`
	TranscriptPath string         `json:"transcript_path"`
	CWD            string         `json:"cwd"`
	HookEventName  string         `json:"hook_event_name"`
	ToolName       string         `json:"tool_name"`
	ToolInput      map[string]any `json:"tool_input"`
	Source         string         `json:"source,omitempty"` // SessionStart only: startup|resume|clear|compact
	AgentID        string         `json:"agent_id,omitempty"`
	AgentType      string         `json:"agent_type,omitempty"`
}

// Output is the decision a hook prints on stdout; the zero value marshals to "{}" (allow).
type Output struct {
	SystemMessage  string          `json:"systemMessage,omitempty"`
	SpecificOutput *SpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type SpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"` // paws only sends deny
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// MatchTarget is the string a rule's regex runs against for this event.
func (in *Input) MatchTarget() string {
	if in.HookEventName == "SessionStart" {
		return in.Source
	}
	return in.Command()
}

// inputField is the tool_input key holding each tool's meaningful argument, mirroring unbound.py's selection.
var inputField = map[string]string{
	"Bash": "command", "Read": "file_path", "Write": "file_path", "Edit": "file_path",
	"NotebookEdit": "notebook_path", "Grep": "pattern", "Glob": "pattern",
	"WebFetch": "url", "WebSearch": "query", "Task": "prompt", "Agent": "prompt",
}

// InputField is the tool_input key Command reads for tool, "" for tools matched another way.
func InputField(tool string) string { return inputField[tool] }

// Command is the meaningful argument per tool: the inputField value, an MCP call's whole input
// as JSON, else the tool name.
func (in *Input) Command() string {
	ti := in.ToolInput
	if f := inputField[in.ToolName]; f != "" {
		if s, ok := ti[f].(string); ok {
			return s
		}
	}
	if strings.HasPrefix(in.ToolName, "mcp__") {
		b, _ := json.Marshal(ti)
		return string(b)
	}
	return in.ToolName
}
