package hook

import (
	"encoding/json"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

// Agent identifies this adapter in audit records.
const Agent = "claude-code"

// RegisteredEvents are the Claude Code hook events `paws init` subscribes to.
var RegisteredEvents = []string{"PreToolUse", "SessionStart", "PostToolUse", "Stop"}

// RetiredEvents were registered by earlier builds; `paws init` removes paws entries from them.
var RetiredEvents = []string{"UserPromptSubmit"}

// ToEvent normalizes a Claude Code hook event into the engine's terms.
func ToEvent(in *Input) gate.Event {
	ev := gate.Event{
		Agent:      Agent,
		Tool:       in.ToolName,
		Target:     in.MatchTarget(),
		Session:    in.SessionID,
		Transcript: in.TranscriptPath,
		Cwd:        in.CWD,
		AgentID:    in.AgentID,
		AgentType:  in.AgentType,
	}
	switch in.HookEventName {
	case "PreToolUse":
		ev.Kind = rule.KindTool
	case "SessionStart":
		ev.Kind = rule.KindSession
	}
	return ev
}

// Render turns an engine decision into Claude Code's wire format for eventName.
func Render(eventName string, d gate.Decision) Output {
	switch {
	case d.Verdict == gate.Deny:
		return Output{SpecificOutput: &SpecificOutput{
			HookEventName:            eventName,
			PermissionDecision:       "deny",
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
	case "session-start":
		return "SessionStart"
	}
	return e
}

// SettingsEntry is the hooks-array element `paws init` writes; Claude Code timeouts are seconds, matcher is tool-events-only.
func SettingsEntry(event, command string) map[string]any {
	inner := []any{map[string]any{"type": "command", "command": command, "timeout": 15}}
	if event == "PostToolUse" || event == "Stop" {
		// Telemetry only: nothing waits on the answer, so the agent never pays for it.
		inner[0].(map[string]any)["async"] = true
	}
	if event == "PreToolUse" || event == "PostToolUse" {
		return map[string]any{"matcher": "*", "hooks": inner}
	}
	return map[string]any{"hooks": inner}
}

// RawEntry is SettingsEntry marshaled, for splicing into a settings document.
func RawEntry(event, command string) json.RawMessage {
	b, _ := json.Marshal(SettingsEntry(event, command))
	return b
}
