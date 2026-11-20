package hook

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

// TurnCalls reads Claude Code's log of the turn that just ended: every tool call since the last prompt, in the main
// conversation (transcriptPath) and in its subagents' logs, oldest first, with whether a hook refused it. It is the
// Claude Code adapter for package outcome; another agent would read its own log into the same Calls.
func TurnCalls(transcriptPath string) ([]outcome.Call, time.Time, error) {
	main, start, err := readTurnLog(transcriptPath, "", time.Time{})
	if err != nil {
		return nil, start, err
	}
	calls := main
	logs, _ := filepath.Glob(filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents", "*.jsonl"))
	for _, p := range logs {
		agent := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(p), ".jsonl"), "agent-")
		sub, _, err := readTurnLog(p, agent, start)
		if err == nil {
			calls = append(calls, sub...)
		}
	}
	sort.SliceStable(calls, func(i, j int) bool { return calls[i].At.Before(calls[j].At) })
	return calls, start, nil
}

// turnLine is the part of a transcript record the turn reader needs.
type turnLine struct {
	Type      string    `json:"type"`
	IsMeta    bool      `json:"isMeta"`
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type turnBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     map[string]any  `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
}

// readTurnLog reads one log's calls. With a zero since, the turn starts at the log's last prompt (a user record
// that isn't a tool result or a meta message); a subagent log takes the main conversation's start instead.
func readTurnLog(path, agent string, since time.Time) ([]outcome.Call, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, since, err
	}
	defer f.Close()
	var calls []outcome.Call
	byID := map[string]int{}
	start := since
	var launched time.Time // a subagent log's first record: when the subagent started
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if agent != "" && launched.IsZero() && len(line) > 0 {
			var first struct {
				Timestamp time.Time `json:"timestamp"`
			}
			if json.Unmarshal(line, &first) == nil {
				launched = first.Timestamp
			}
		}
		if len(line) > 0 && (bytes.Contains(line, []byte(`"type":"user"`)) || bytes.Contains(line, []byte(`"type":"assistant"`))) {
			var rec turnLine
			if json.Unmarshal(line, &rec) == nil {
				blocks := contentBlocks(rec.Message.Content)
				switch {
				case rec.Type == "user" && since.IsZero() && !rec.IsMeta && isPrompt(rec.Message.Content, blocks):
					start, calls, byID = rec.Timestamp, nil, map[string]int{} // a new turn: forget the last one's calls
				case rec.Type == "assistant" && !rec.Timestamp.Before(start):
					for _, b := range blocks {
						if b.Type == "tool_use" && b.Name != "" {
							byID[b.ID] = len(calls)
							calls = append(calls, toCall(b, rec.Timestamp, agent))
						}
					}
				case rec.Type == "user":
					for _, b := range blocks {
						if i, ok := byID[b.ToolUseID]; ok && b.Type == "tool_result" && b.IsError && hookDenied(b.Content) {
							calls[i].Denied = true
						}
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	for i := range calls {
		calls[i].AgentStart = launched
	}
	return calls, start, nil
}

func contentBlocks(content json.RawMessage) []turnBlock {
	if len(content) == 0 || content[0] != '[' {
		return nil
	}
	var blocks []turnBlock
	json.Unmarshal(content, &blocks)
	return blocks
}

// isPrompt: the user typed something (a string, or text blocks), not a tool result.
func isPrompt(content json.RawMessage, blocks []turnBlock) bool {
	if len(content) > 0 && content[0] == '"' {
		return true
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return false
		}
	}
	return len(blocks) > 0
}

// hookDenied: Claude Code reports a call a PreToolUse hook refused as "PreToolUse:<Tool> hook error: <reason>".
func hookDenied(content json.RawMessage) bool {
	var s string
	if json.Unmarshal(content, &s) != nil {
		var parts []struct {
			Text string `json:"text"`
		}
		json.Unmarshal(content, &parts)
		for _, p := range parts {
			s += p.Text
		}
	}
	return strings.Contains(s, "PreToolUse:") && strings.Contains(s, " hook error: ")
}

func toCall(b turnBlock, at time.Time, agent string) outcome.Call {
	in := Input{HookEventName: "PreToolUse", ToolName: b.Name, ToolInput: b.Input}
	c := outcome.Call{ID: b.ID, Tool: b.Name, Target: in.Command(), Input: b.Input, At: at, Agent: agent}
	c.Text = mainText(b.Name, b.Input, c.Target)
	if server, tool, ok := gate.MCPParts(b.Name); ok {
		c.MCP = map[string]any{"server": server, "tool": tool, "class": rule.MCPAction(b.Name)}
	}
	return c
}

// mainText is the call's main text: what a Write writes, an Edit's new text, an MCP call's text arguments; for
// anything else the gate's own target (a Bash command, a path, a pattern).
func mainText(tool string, input map[string]any, target string) string {
	str := func(k string) string { s, _ := input[k].(string); return s }
	switch {
	case tool == "Write":
		return str("content")
	case tool == "Edit":
		return str("new_string")
	case tool == "NotebookEdit":
		return str("new_source")
	case tool == "MultiEdit":
		var parts []string
		edits, _ := input["edits"].([]any)
		for _, e := range edits {
			if m, ok := e.(map[string]any); ok {
				if s, ok := m["new_string"].(string); ok {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, "\n")
	case strings.HasPrefix(tool, "mcp__"):
		return strings.Join(strings_(input), "\n")
	}
	return target
}

// strings_ is every string value in v, keys in order, depth first.
func strings_(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			out = append(out, strings_(x[k])...)
		}
		return out
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, strings_(e)...)
		}
		return out
	}
	return nil
}
