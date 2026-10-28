package gate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

// tailBytes is how much of a transcript's end is read. Sessions reach tens of MB,
// and a window's worth of turns lives in the last few hundred KB of them.
const tailBytes = 1 << 20

// transcriptEntry is the subset of a transcript line the skill scan needs.
type transcriptEntry struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	IsSidechain bool   `json:"isSidechain"`
	// Polymorphic in the log: an object for tool results, a bare string for hook denials.
	ToolUseResult json.RawMessage `json:"toolUseResult"`
	Attachment    *attachment     `json:"attachment"`
	Message       struct {
		ID string `json:"id"`
	} `json:"message"`

	raw []byte // the whole line, for the few records whose content blocks are read
}

// requestBlob is the type of the records Claude Code writes as it sends each API request:
// the new messages of that request, content-addressed. They reach the file when the request
// goes out, while the matching user/assistant records can be flushed later, after the next
// tool call's hook has already run. So a skill result or a delivered deny may exist only as
// a blob when the retry is gated. Subagent blobs go to the subagent's own file.
const requestBlob = "api-request-blob"

// block is one content block of a message: a tool_use, a tool_result or text.
type block struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input struct {
		Skill string `json:"skill"`
	} `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
	Text      string          `json:"text"`
}

// blocks parses the entry's message content blocks; nil when the content is a bare string.
func (e transcriptEntry) blocks() []block {
	var m struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	var bs []block
	if json.Unmarshal(e.raw, &m) != nil || len(m.Message.Content) == 0 || m.Message.Content[0] != '[' ||
		json.Unmarshal(m.Message.Content, &bs) != nil {
		return nil
	}
	return bs
}

// text is a tool_result's content as text: a bare string, or its text blocks joined.
func (b block) text() string {
	var s string
	if json.Unmarshal(b.Content, &s) == nil {
		return s
	}
	var parts []block
	if json.Unmarshal(b.Content, &parts) != nil {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.Text)
	}
	return strings.Join(out, "\n")
}

// attachment is a harness-written record; hook_additional_context holds what a hook's
// additionalContext delivered to the agent.
type attachment struct {
	Type    string   `json:"type"`
	Content []string `json:"content"`
}

// skillResult is the agent-written result of a skill invocation. Only the agent
// harness produces this record, so model-authored text cannot forge one.
type skillResult struct {
	CommandName string `json:"commandName"`
	Success     bool   `json:"success"`
}

// SkillLoaded reports whether skill was invoked successfully within the last
// `window` agent turns of the transcript at path.
//
// False is the answer for every uncertain case — no transcript, unreadable file,
// window exhausted, or a context compaction between the load and now — because
// the caller gates on false. A compaction ends the search: the skill's text was
// dropped from the agent's context, so the load no longer means it has the guidance.
func SkillLoaded(path, skill string, window int) bool {
	if skill == "" {
		return false
	}
	var calls map[string]bool // built on the first blob tool_result: most scans end before one
	return scanWindow(path, window, func(e transcriptEntry) bool {
		if e.Type == requestBlob {
			// A tool_result answering a Skill tool_use for this skill, not an error. The pairing
			// by id is what keeps a command that prints "Launching skill: x" from counting.
			if !bytes.Contains(e.raw, []byte(`"tool_result"`)) {
				return false
			}
			for _, b := range e.blocks() {
				if b.Type != "tool_result" || b.IsError {
					continue
				}
				if calls == nil {
					calls = skillCalls(path, skill)
				}
				if calls[b.ToolUseID] {
					return true
				}
			}
			return false
		}
		if e.Type != "user" || len(e.ToolUseResult) == 0 || e.ToolUseResult[0] != '{' {
			return false
		}
		var sr skillResult
		return json.Unmarshal(e.ToolUseResult, &sr) == nil && sr.Success && sr.CommandName == skill
	})
}

var skillToolName = []byte(`"name":"Skill"`)

// skillCalls returns the ids of main-thread Skill tool_use blocks that invoke skill.
func skillCalls(path, skill string) map[string]bool {
	calls := map[string]bool{}
	lines, err := tailLines(path)
	if err != nil {
		return calls
	}
	name, _ := json.Marshal(skill)
	input := append([]byte(`"skill":`), name...) // skips the huge tool-definition lines cheaply
	for _, l := range lines {
		var e transcriptEntry
		if !bytes.Contains(l, input) || !bytes.Contains(l, skillToolName) || json.Unmarshal(l, &e) != nil || e.IsSidechain || (e.Type != "assistant" && e.Type != requestBlob) {
			continue
		}
		e.raw = l
		for _, b := range e.blocks() {
			if b.Type == "tool_use" && b.Name == "Skill" && b.Input.Skill == skill {
				calls[b.ID] = true
			}
		}
	}
	return calls
}

// ContextDelivered reports whether a hook's additionalContext containing marker reached the
// agent within the last `window` turns. Same window and compaction rules as SkillLoaded.
func ContextDelivered(path, marker string, window int) bool {
	if marker == "" {
		return false
	}
	return scanWindow(path, window, func(e transcriptEntry) bool {
		if e.Type == requestBlob {
			// The blob carries the context as text: "PreToolUse:Bash hook additional context: ...".
			if !bytes.Contains(e.raw, []byte(hookContextLabel)) {
				return false
			}
			for _, b := range e.blocks() {
				if i := strings.Index(b.Text, hookContextLabel); b.Type == "text" && i >= 0 && strings.Contains(b.Text[i:], marker) {
					return true
				}
			}
			return false
		}
		if e.Type != "attachment" || e.Attachment == nil || e.Attachment.Type != "hook_additional_context" {
			return false
		}
		for _, c := range e.Attachment.Content {
			if strings.Contains(c, marker) {
				return true
			}
		}
		return false
	})
}

// hookDenyPrefix is how Claude Code records a PreToolUse deny in toolUseResult:
// "Error: PreToolUse:<Tool> hook error: <reason>". A failed command is recorded as
// "Error: Exit code N\n<output>", so anchoring here keeps tool output from forging a deny.
var hookDenyPrefix = regexp.MustCompile(`^Error: PreToolUse:\S+ hook error: `)

// blobDenyPrefix is the same deny as a request blob records it: the tool_result content,
// without the "Error: " that toolUseResult adds.
var blobDenyPrefix = regexp.MustCompile(`^PreToolUse:\S+ hook error: `)

// hookContextLabel precedes a hook's additionalContext in the text a request blob records.
const hookContextLabel = " hook additional context: "

// DenyDelivered reports whether a paws deny whose reason starts with marker reached the
// agent within the last `window` turns.
func DenyDelivered(path, marker string, window int) bool {
	if marker == "" {
		return false
	}
	// paws joins retryable denies with blank lines, so the marker opens the reason or a paragraph.
	denied := func(result string, prefix *regexp.Regexp) bool {
		loc := prefix.FindStringIndex(result)
		if loc == nil {
			return false
		}
		reason := result[loc[1]:]
		return strings.HasPrefix(reason, marker) || strings.Contains(reason, "\n\n"+marker)
	}
	return scanWindow(path, window, func(e transcriptEntry) bool {
		if e.Type == requestBlob {
			if !bytes.Contains(e.raw, []byte(" hook error: ")) {
				return false
			}
			for _, b := range e.blocks() {
				if b.Type == "tool_result" && b.IsError && denied(b.text(), blobDenyPrefix) {
					return true
				}
			}
			return false
		}
		if e.Type != "user" || len(e.ToolUseResult) == 0 || e.ToolUseResult[0] != '"' {
			return false
		}
		var result string
		if json.Unmarshal(e.ToolUseResult, &result) != nil {
			return false
		}
		return denied(result, hookDenyPrefix)
	})
}

// scanWindow walks the transcript backwards and reports whether hit matches an entry before
// the window of agent turns runs out or a compaction boundary is crossed.
func scanWindow(path string, window int, hit func(transcriptEntry) bool) bool {
	if path == "" || window < 1 {
		return false
	}
	lines, err := tailLines(path)
	if err != nil {
		return false
	}
	turns, lastID := 0, ""
	for i := len(lines) - 1; i >= 0; i-- {
		var e transcriptEntry
		if json.Unmarshal(lines[i], &e) != nil {
			continue
		}
		e.raw = lines[i]
		// Subagents write their own transcript files, so sidechain records should not
		// appear here at all; skipping them keeps that guarantee if that ever changes.
		if e.IsSidechain {
			continue
		}
		switch {
		case e.Type == "system" && e.Subtype == "compact_boundary":
			return false
		case e.Type == "assistant":
			// Claude Code writes one entry per content block (thinking, text, each tool_use),
			// all sharing the response's message id. A turn is one response, not one entry.
			if e.Message.ID == "" || e.Message.ID != lastID {
				turns++
			}
			lastID = e.Message.ID
			if turns > window {
				return false
			}
		case hit(e):
			return true
		}
	}
	return false
}

// tailCache holds tails already read in this process. paws gate is one process per hook
// call, so several rules scanning the same transcript read and split it once.
var tailCache = map[string][][]byte{}

// tailLines returns the last tailBytes of path split into whole lines, cached per path.
func tailLines(path string) ([][]byte, error) {
	if lines, ok := tailCache[path]; ok {
		return lines, nil
	}
	lines, err := readTail(path)
	if err == nil {
		tailCache[path] = lines
	}
	return lines, err
}

func readTail(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, errors.New("empty transcript")
	}
	off, n := int64(0), size
	if size > tailBytes {
		off, n = size-tailBytes, tailBytes
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, off); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	lines := bytes.Split(buf, []byte("\n"))
	if off > 0 && len(lines) > 0 {
		lines = lines[1:] // the read started mid-object
	}
	return lines, nil
}
