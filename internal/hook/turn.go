package hook

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
//
// until is when the turn stopped: records after it (a prompt the user sent meanwhile, the next turn's calls) are
// left out. Zero reads to the end.
func TurnCalls(transcriptPath string, until time.Time) ([]outcome.Call, time.Time, error) {
	main, start, err := readTurnLog(transcriptPath, "", time.Time{}, until)
	if err != nil {
		return nil, start, err
	}
	calls := main
	logs, _ := filepath.Glob(filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents", "*.jsonl"))
	for _, p := range logs {
		agent := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(p), ".jsonl"), "agent-")
		sub, _, err := readTurnLog(p, agent, start, until)
		if err != nil {
			// A turn missing a subagent's calls would grade a retry it made as stopped: no turn rather than half.
			return nil, start, fmt.Errorf("subagent log %s: %w", filepath.Base(p), err)
		}
		calls = append(calls, sub...)
	}
	sort.SliceStable(calls, func(i, j int) bool { return calls[i].At.Before(calls[j].At) })
	return calls, start, nil
}

// lastTail is how much of a log's end LastRecordAt and StopRecordAt read: the turn's last records sit in it.
const lastTail = 1 << 20

// tailLines is the log's last lastTail bytes, split into whole lines.
func tailLines(path string) ([][]byte, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, false
	}
	off := max(st.Size()-lastTail, 0)
	buf := make([]byte, st.Size()-off)
	if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
		return nil, false
	}
	lines := bytes.Split(buf, []byte("\n"))
	if off > 0 {
		lines = lines[1:] // the first line is cut
	}
	return lines, true
}

// LastRecordAt is the timestamp of the last record in the log at or before until: the final message of the turn that is
// stopping, since Claude Code writes it before it runs the Stop hook. The Stop hook has no time of its own and starts a
// moment late, so a call the next turn makes in that gap is after this time, not part of the turn. ok is false when the
// log can't be read or holds no timestamp. StopRecordAt, which names the turn's final message, comes first.
func LastRecordAt(path string, until time.Time) (at time.Time, ok bool) {
	lines, ok := tailLines(path)
	for i := len(lines) - 1; i >= 0; i-- {
		var rec struct {
			Timestamp time.Time `json:"timestamp"`
		}
		if len(lines[i]) > 0 && json.Unmarshal(lines[i], &rec) == nil && !rec.Timestamp.IsZero() && !rec.Timestamp.After(until) {
			return rec.Timestamp, true
		}
	}
	return time.Time{}, false
}

// StopRecordAt is when the turn that stopped ended: the timestamp of its final assistant message, found by the text the
// Stop hook was given (Claude Code's last_assistant_message, the message's text blocks joined by newlines). Claude Code
// writes one record per content block, so records are grouped by message id. A message that made a tool call doesn't end
// a turn, so a turn still running when the hook reads the log can't match. ok is false when last is empty or no message
// in the log's tail has that text.
func StopRecordAt(path, last string) (at time.Time, ok bool) {
	last = strings.TrimSpace(last)
	lines, read := tailLines(path)
	if last == "" || !read {
		return time.Time{}, false
	}
	type message struct {
		text []string
		tool bool
		at   time.Time
	}
	var order []string
	byID := map[string]*message{}
	for _, line := range lines {
		if !bytes.Contains(line, []byte(`"type":"assistant"`)) {
			continue
		}
		var rec struct {
			Type      string    `json:"type"`
			Sidechain bool      `json:"isSidechain"`
			Timestamp time.Time `json:"timestamp"`
			Message   struct {
				ID      string          `json:"id"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil || rec.Type != "assistant" || rec.Sidechain || rec.Message.ID == "" {
			continue
		}
		m := byID[rec.Message.ID]
		if m == nil {
			m = &message{}
			byID[rec.Message.ID] = m
			order = append(order, rec.Message.ID)
		}
		for _, b := range contentBlocks(rec.Message.Content) {
			switch b.Type {
			case "tool_use":
				m.tool = true
			case "text":
				m.text = append(m.text, b.Text)
			}
		}
		m.at = rec.Timestamp
	}
	for i := len(order) - 1; i >= 0; i-- {
		if m := byID[order[i]]; !m.tool && strings.TrimSpace(strings.Join(m.text, "\n")) == last {
			return m.at, true
		}
	}
	return time.Time{}, false
}

// turnLine is the part of a transcript record the turn reader needs.
type turnLine struct {
	Type      string    `json:"type"`
	IsMeta    bool      `json:"isMeta"`
	Sidechain bool      `json:"isSidechain"`
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
	Text      string          `json:"text"`
}

// readTurnLog reads one log's calls. With a zero since, the turn starts at the log's last prompt (a user record
// that isn't a tool result or a meta message); a subagent log takes the main conversation's start instead.
func readTurnLog(path, agent string, since, until time.Time) ([]outcome.Call, time.Time, error) {
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
			// A sidechain record in the main log is a subagent's (see gate.foreign): its own log is read as that agent's.
			if json.Unmarshal(line, &rec) == nil && (until.IsZero() || !rec.Timestamp.After(until)) && !(agent == "" && rec.Sidechain) {
				blocks := contentBlocks(rec.Message.Content)
				switch {
				case rec.Type == "user" && since.IsZero() && gate.IsUserPromptLine(line):
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

// hookDenied: a call that never ran. Claude Code reports one a PreToolUse hook refused as "PreToolUse:<Tool> hook
// error: <reason>", and one the user rejected at the permission prompt as "The user doesn't want to proceed with
// this tool use".
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
	// Refused by a hook, or rejected by the user at the permission prompt: either way the call never ran. Anchored
	// at the start, so a command that ran and printed these words isn't read as one that didn't.
	s = strings.TrimPrefix(s, "<tool_use_error>")
	return refusedPrefix.MatchString(s) || strings.HasPrefix(s, "The user doesn't want to proceed with this tool use")
}

// refusedPrefix opens a refused call's result, as Claude Code writes it (gate.blobDenyPrefix).
var refusedPrefix = regexp.MustCompile(`^PreToolUse:\S+ hook error: `)

// CallOf is a tool call as expect checks see it, from its name and input (unbound-paws explain tries checks on one).
func CallOf(tool string, input map[string]any) outcome.Call {
	return toCall(turnBlock{Name: tool, Input: input}, time.Time{}, "")
}

func toCall(b turnBlock, at time.Time, agent string) outcome.Call {
	in := Input{HookEventName: "PreToolUse", ToolName: b.Name, ToolInput: b.Input}
	c := outcome.Call{ID: b.ID, Tool: b.Name, Target: in.Command(), Input: b.Input, At: at, Agent: agent}
	c.Text = mainText(b.Name, b.Input, c.Target)
	if b.Name == "Skill" {
		c.SkillLoad, _ = b.Input["skill"].(string)
	} else {
		c.SkillLoad = gate.SkillRead(b.Name, c.Target, "")
	}
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
