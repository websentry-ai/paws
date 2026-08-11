package hook

import (
	"encoding/json"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

// Agent identifies this adapter in audit records and exec-script payloads.
const Agent = "claude-code"

// RegisteredEvents are the Claude Code hook events `paws init` subscribes to.
var RegisteredEvents = []string{"PreToolUse", "UserPromptSubmit"}

// ToEvent normalizes a Claude Code hook event into the engine's terms; raw passes through untouched for exec-scripts.
func ToEvent(in *Input, raw []byte) gate.Event {
	ev := gate.Event{
		Agent:      Agent,
		Tool:       in.ToolName,
		Target:     in.MatchTarget(),
		Session:    in.SessionID,
		Transcript: in.TranscriptPath,
		AgentID:    in.AgentID,
		AgentType:  in.AgentType,
		Raw:        raw,
	}
	switch in.HookEventName {
	case "PreToolUse":
		ev.Kind = rule.KindTool
	case "UserPromptSubmit":
		ev.Kind = rule.KindPrompt
	}
	return ev
}

// Render turns an engine decision into Claude Code's wire format for eventName.
func Render(eventName string, d gate.Decision) Output {
	switch {
	case d.Verdict >= gate.Ask:
		if eventName == "UserPromptSubmit" {
			out := Output{Decision: "block", Reason: d.Reason}
			if d.Context != "" {
				out.SpecificOutput = &SpecificOutput{HookEventName: eventName, AdditionalContext: d.Context}
			}
			return out
		}
		decision := "deny"
		if d.Verdict == gate.Ask {
			decision = "ask"
		}
		return Output{SpecificOutput: &SpecificOutput{
			HookEventName:            eventName,
			PermissionDecision:       decision,
			PermissionDecisionReason: d.Reason,
			AdditionalContext:        d.Context,
		}}
	case d.Context != "":
		return Output{SpecificOutput: &SpecificOutput{HookEventName: eventName, AdditionalContext: d.Context}}
	}
	return Output{}
}

// EventName maps a paws rule event to its Claude Code hook event; anything else passes through unchanged.
func EventName(e string) string {
	switch e {
	case "pre-tool", "pre-commit", "pre-push":
		return "PreToolUse"
	case "user-prompt":
		return "UserPromptSubmit"
	}
	return e
}

// SettingsEntry is the hooks-array element `paws init` writes; Claude Code timeouts are seconds, matcher is tool-events-only.
func SettingsEntry(event, command string) map[string]any {
	inner := []any{map[string]any{"type": "command", "command": command, "timeout": 15}}
	if event == "PreToolUse" {
		return map[string]any{"matcher": "*", "hooks": inner}
	}
	return map[string]any{"hooks": inner}
}

// RawEntry is SettingsEntry marshaled, for splicing into a settings document.
func RawEntry(event, command string) json.RawMessage {
	b, _ := json.Marshal(SettingsEntry(event, command))
	return b
}
