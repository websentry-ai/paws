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

// Command is the meaningful argument per tool, mirroring unbound.py's selection.
func (in *Input) Command() string {
	ti := in.ToolInput
	switch in.ToolName {
	case "Bash":
		if c, ok := ti["command"].(string); ok {
			return c
		}
	case "Read", "Write", "Edit":
		if p, ok := ti["file_path"].(string); ok {
			return p
		}
	case "NotebookEdit":
		if p, ok := ti["notebook_path"].(string); ok {
			return p
		}
	case "Grep", "Glob":
		if p, ok := ti["pattern"].(string); ok {
			return p
		}
	case "WebFetch":
		if u, ok := ti["url"].(string); ok {
			return u
		}
	case "WebSearch":
		if q, ok := ti["query"].(string); ok {
			return q
		}
	case "Task", "Agent":
		if p, ok := ti["prompt"].(string); ok {
			return p
		}
	}
	if strings.HasPrefix(in.ToolName, "mcp__") {
		b, _ := json.Marshal(ti)
		return string(b)
	}
	return in.ToolName
}
