package gate

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// tailBytes is how much of a transcript's end the `when` conditions read.
const tailBytes = 1 << 20

// maxWindowBytes caps how far back a skill-window scan reads. 20 user messages span 4 to 16 MB
// in real sessions; a load further back than this counts as not loaded, which costs one reload.
var maxWindowBytes int64 = 64 << 20

// windowChunk is how much a window scan reads at a time, walking back from the end of the file.
const windowChunk = 1 << 20

// transcriptEntry is the subset of a transcript line the skill scan needs.
type transcriptEntry struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	IsSidechain bool   `json:"isSidechain"`
	// Polymorphic in the log: an object for tool results, a bare string for hook denials.
	ToolUseResult json.RawMessage `json:"toolUseResult"`

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

// skillResult is the agent-written result of a skill invocation. Only the agent
// harness produces this record, so model-authored text cannot forge one.
type skillResult struct {
	CommandName string `json:"commandName"`
	Success     bool   `json:"success"`
}

// SkillLoaded reports whether skill was invoked successfully within the last `window`
// turns of the transcript at path: with the Skill tool, or by reading its SKILL.md (see
// SkillRead) with a result that isn't an error. A turn is one message the user typed and
// everything the agent did to answer it, however many tool calls that took.
//
// False is the answer for every uncertain case — no transcript, unreadable file,
// window exhausted, or a context compaction between the load and now — because
// the caller gates on false. A compaction ends the search: the skill's text was
// dropped from the agent's context, so the load no longer means it has the guidance.
func SkillLoaded(path, skill string, window int) bool { return skillLoaded(path, skill, window, "") }

// skillLoaded is SkillLoaded with the session's cwd, for a relative SKILL.md read in a record that names none.
func skillLoaded(path, skill string, window int, cwd string) bool {
	if skill == "" {
		return false
	}
	lines := windowLines(path, window)
	calls := skillCalls(lines, path, skill)
	// Only a line with the skill's result record or one of its calls' ids (a blob answering it) can match.
	name, _ := json.Marshal(skill)
	needles := [][]byte{append([]byte(`"commandName":`), name...)}
	for id := range calls {
		needles = append(needles, []byte(id))
	}
	loaded := scanLines(lines, path, needles, func(e transcriptEntry) bool {
		if e.Type == requestBlob {
			// A tool_result answering a Skill tool_use for this skill, not an error. The pairing
			// by id is what keeps a command that prints "Launching skill: x" from counting.
			for _, b := range e.blocks() {
				if b.Type == "tool_result" && !b.IsError && calls[b.ToolUseID] {
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
	if loaded {
		return true
	}
	// Reads of the skill's SKILL.md load it once those that succeeded cover the whole file.
	reads := readCalls(lines, path, skill, cwd)
	return len(reads) > 0 && Covered(succeededReads(lines, path, reads))
}

// succeededReads are the reads whose tool_result, paired by id, is not an error: in a user record or a request blob.
func succeededReads(lines [][]byte, path string, reads map[string]FileRead) []FileRead {
	var needles [][]byte
	for id := range reads {
		needles = append(needles, []byte(id))
	}
	var ok []FileRead
	done := map[string]bool{}
	scanLines(lines, path, needles, func(e transcriptEntry) bool {
		if e.Type != "user" && e.Type != requestBlob {
			return false
		}
		for _, b := range e.blocks() {
			if r, isRead := reads[b.ToolUseID]; isRead && b.Type == "tool_result" && !b.IsError && !done[b.ToolUseID] {
				done[b.ToolUseID] = true
				ok = append(ok, r)
			}
		}
		return false // read every line: the reads add up
	})
	return ok
}

var skillToolName = []byte(`"name":"Skill"`)

// skillCalls returns the ids of the log's own Skill tool_use blocks that invoke skill.
func skillCalls(lines [][]byte, path, skill string) map[string]bool {
	calls := map[string]bool{}
	name, _ := json.Marshal(skill)
	input := append([]byte(`"skill":`), name...) // skips the huge tool-definition lines cheaply
	for _, l := range lines {
		var e transcriptEntry
		if !bytes.Contains(l, input) || !bytes.Contains(l, skillToolName) || json.Unmarshal(l, &e) != nil || foreign(e, path) || (e.Type != "assistant" && e.Type != requestBlob) {
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

// readCalls returns the log's own Read and Bash tool_use blocks that read skill's SKILL.md, by id, with the lines
// each one printed.
func readCalls(lines [][]byte, path, skill, cwd string) map[string]FileRead {
	calls := map[string]FileRead{}
	file := []byte("SKILL.md") // skips every line that can't name the file; a relative `cat SKILL.md` names it too
	for _, l := range lines {
		var e transcriptEntry
		if !bytes.Contains(l, file) || !bytes.Contains(l, toolUseType) || json.Unmarshal(l, &e) != nil || foreign(e, path) || (e.Type != "assistant" && e.Type != requestBlob) {
			continue
		}
		var m struct {
			Cwd       string    `json:"cwd"`
			Timestamp time.Time `json:"timestamp"`
			Message   struct {
				Content []struct {
					Type  string         `json:"type"`
					ID    string         `json:"id"`
					Name  string         `json:"name"`
					Input map[string]any `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(l, &m) != nil {
			continue
		}
		for _, b := range m.Message.Content {
			key := "command"
			if b.Name == "Read" {
				key = "file_path"
			}
			target, _ := b.Input[key].(string)
			// The file may be gone since: a read that succeeded put the skill in the agent's context all the same.
			if r, ok := ReadOf(b.Name, target, b.Input, cmp.Or(m.Cwd, cwd), false); b.Type == "tool_use" && ok && r.Skill == skill {
				r.At = m.Timestamp
				calls[b.ID] = r
			}
		}
	}
	return calls
}

var toolUseType = []byte(`"type":"tool_use"`)

// transcriptModel is the model of the last assistant message in the log at path: Claude Code names it on each one,
// so after a /model switch it is one response behind at most. "" when the tail has none.
func transcriptModel(path string) string {
	lines, err := tailLines(path)
	if err != nil {
		return ""
	}
	for i := len(lines) - 1; i >= 0; i-- {
		l := lines[i]
		if !bytes.Contains(l, assistantType) || !bytes.Contains(l, modelKey) {
			continue
		}
		var e struct {
			Type        string `json:"type"`
			IsSidechain bool   `json:"isSidechain"`
			Message     struct {
				Model string `json:"model"`
			} `json:"message"`
		}
		if json.Unmarshal(l, &e) != nil || e.Type != "assistant" || e.IsSidechain && !subagentLog(path) {
			continue
		}
		if m := e.Message.Model; m != "" && m != "<synthetic>" { // <synthetic>: a message Claude Code wrote itself
			return m
		}
	}
	return ""
}

var (
	assistantType = []byte(`"type":"assistant"`)
	modelKey      = []byte(`"model":"`)
)

// hookDenyPrefix is how Claude Code records a PreToolUse deny in toolUseResult:
// "Error: PreToolUse:<Tool> hook error: <reason>". A failed command is recorded as
// "Error: Exit code N\n<output>", so anchoring here keeps tool output from forging a deny.
var hookDenyPrefix = regexp.MustCompile(`^Error: PreToolUse:\S+ hook error: `)

// blobDenyPrefix is the same deny as a request blob records it: the tool_result content,
// without the "Error: " that toolUseResult adds.
var blobDenyPrefix = regexp.MustCompile(`^PreToolUse:\S+ hook error: `)

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
	return scanLines(windowLines(path, window), path, [][]byte{[]byte(" hook error: ")}, func(e transcriptEntry) bool {
		if e.Type == requestBlob {
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

// scanLines reports whether hit matches one of the log's own entries among lines. Only lines
// holding one of needles are parsed: a window can be megabytes, and most of it can't match.
func scanLines(lines [][]byte, path string, needles [][]byte, hit func(transcriptEntry) bool) bool {
	for _, l := range lines {
		if !containsAny(l, needles) {
			continue
		}
		var e transcriptEntry
		if json.Unmarshal(l, &e) != nil {
			continue
		}
		e.raw = l
		if !foreign(e, path) && hit(e) {
			return true
		}
	}
	return false
}

func containsAny(l []byte, needles [][]byte) bool {
	for _, n := range needles {
		if bytes.Contains(l, n) {
			return true
		}
	}
	return false
}

// windowCache holds windows already read in this process. paws gate is one process per hook
// call, so several rules scanning the same transcript read it once.
var windowCache = map[string][][]byte{}

// windowLines returns the lines of the last `window` turns of the transcript at path, newest
// first. It reads back from the end of the file until the window-th message the user typed, a
// compaction boundary, the start of the file or maxWindowBytes, whichever comes first.
func windowLines(path string, window int) [][]byte {
	if path == "" || window < 1 {
		return nil
	}
	key := strconv.Itoa(window) + "\x00" + path
	if lines, ok := windowCache[key]; ok {
		return lines
	}
	lines := readWindow(path, window)
	windowCache[key] = lines
	return lines
}

func readWindow(path string, window int) [][]byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil
	}
	var out [][]byte
	turns, end := 0, fi.Size()
	var carry []byte // a line whose start is in the chunk before the one just read
	for end > 0 {
		n := min(int64(windowChunk), end, maxWindowBytes-(fi.Size()-end))
		if n <= 0 {
			break // the read limit: anything older counts as not loaded
		}
		buf := make([]byte, n, n+int64(len(carry)))
		if _, err := f.ReadAt(buf, end-n); err != nil && !errors.Is(err, io.EOF) {
			return nil
		}
		end -= n
		lines := bytes.Split(append(buf, carry...), []byte("\n"))
		if end > 0 {
			carry, lines = lines[0], lines[1:] // the chunk started mid-line
		}
		for i := len(lines) - 1; i >= 0; i-- {
			l := lines[i]
			if len(l) == 0 {
				continue
			}
			m := turnMark(l, path)
			if m == compactionMark {
				return out // a compaction: the agent no longer has anything from before it
			}
			out = append(out, l)
			if m == userTurn {
				if turns++; turns == window {
					return out
				}
			}
		}
	}
	return out
}

var (
	userType   = []byte(`"type":"user"`)
	compaction = []byte(`"subtype":"compact_boundary"`)
	// Keys of user records that are never a typed message. Inside a JSON string a quote is escaped,
	// so text that quotes one of these can't match.
	notTyped = [][]byte{[]byte(`"toolUseResult":`), []byte(`"isMeta":true`), []byte(`"isCompactSummary":true`)}
)

type mark int

const (
	noMark         mark = iota
	userTurn            // a message the user typed: a turn starts here
	compactionMark      // a compaction boundary: the window ends here
)

// turnMark says whether l opens a turn or is a compaction boundary. The byte checks keep large
// tool results, pasted images and skill text unparsed.
func turnMark(l []byte, path string) mark {
	isCompaction := bytes.Contains(l, compaction)
	if !isCompaction && (!bytes.Contains(l, userType) || containsAny(l, notTyped)) {
		return noMark
	}
	var e struct {
		mainEntry
		Subtype string `json:"subtype"`
		Origin  *struct {
			Kind string `json:"kind"` // "human" for a typed message; others (task-notification) are injected
		} `json:"origin"`
		ToolUseResult json.RawMessage `json:"toolUseResult"`
	}
	if json.Unmarshal(l, &e) != nil || e.IsSidechain && !subagentLog(path) {
		return noMark
	}
	switch {
	case e.Type == "system" && e.Subtype == "compact_boundary":
		return compactionMark
	case len(e.ToolUseResult) == 0 && (e.Origin == nil || e.Origin.Kind == "human") && isUserPrompt(e.mainEntry):
		// isUserPrompt is the same test agent-count uses: it also skips records Claude Code writes
		// itself (slash-command output, shell escapes), which carry no origin in older logs.
		return userTurn
	}
	return noMark
}

// foreign reports whether e belongs to another thread than the log's own. Claude Code marks every record
// of a subagent's log (<session>/subagents/agent-<id>.jsonl) isSidechain, and keeps the main log's own
// records unmarked; a sidechain record anywhere else is a subagent's and says nothing about the main thread.
func foreign(e transcriptEntry, path string) bool {
	return e.IsSidechain && !subagentLog(path)
}

func subagentLog(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "subagents" && strings.HasPrefix(filepath.Base(path), "agent-")
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
