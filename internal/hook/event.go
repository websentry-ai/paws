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
	Prompt         string         `json:"prompt,omitempty"` // UserPromptSubmit only
	AgentID        string         `json:"agent_id,omitempty"`
	AgentType      string         `json:"agent_type,omitempty"`
}

// Output is the decision a hook prints on stdout; the zero value marshals to "{}" (allow).
type Output struct {
	Decision       string          `json:"decision,omitempty"`
	Reason         string          `json:"reason,omitempty"`
	SystemMessage  string          `json:"systemMessage,omitempty"`
	SpecificOutput *SpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type SpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"` // allow|deny|ask
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// MatchTarget is the string a rule's regex runs against for this event.
func (in *Input) MatchTarget() string {
	if in.HookEventName == "UserPromptSubmit" {
		return in.Prompt
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
	case "Task":
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
