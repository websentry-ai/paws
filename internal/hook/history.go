package hook

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/history"
)

// Transcripts is Claude Code's past sessions as a history.Source: the JSONL logs under <claude dir>/projects, the
// main thread's and each subagent's (<session>/subagents/agent-<id>.jsonl).
type Transcripts struct {
	Dir string // the projects directory
}

func (Transcripts) Name() string { return Agent }

// maxLine skips a log line longer than this: a tool call is small, and a pasted image or a big file read is not
// a call.
const maxLine = 16 << 20

var (
	toolUseNeedle    = []byte(`"tool_use"`)
	toolResultNeedle = []byte(`"tool_result"`)
)

// MaxCallTime bounds how long a tool call can run before it finishes (a permission prompt left open, then a long
// build or test run). A call this recent with no result yet may still be running: its hook record, and the sync that
// uploads it, may come after the scan.
const MaxCallTime = 12 * time.Hour

// Calls reads the tool_use blocks of the assistant records only. Prompts, replies, thinking and tool results sit
// in other records or other blocks and are never decoded.
func (t Transcripts) Calls(since, until time.Time, fn func(history.Call)) (int, error) {
	sessions := map[string]bool{}
	// The folder may be a symlink (a ~/.claude kept in a dotfiles repo); WalkDir doesn't follow one at the root.
	root, err := filepath.EvalSymlinks(t.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil // no sessions yet
	}
	if err != nil {
		return 0, err
	}
	// An unreadable folder or log stops the scan: a summary missing it would replace this laptop's last one.
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(since) {
			return nil // nothing in a log last written before the window can be in it
		}
		found, err := readLog(path, filepath.Base(sessionOf(root, path)), since, until, fn)
		if found {
			sessions[sessionOf(root, path)] = true
		}
		return err
	})
	return len(sessions), err
}

// sessionOf names the session a log belongs to: a subagent's log counts toward its parent session.
func sessionOf(root, path string) string {
	rel, _ := filepath.Rel(root, path)
	if dir := filepath.Dir(path); filepath.Base(dir) == "subagents" {
		rel, _ = filepath.Rel(root, filepath.Dir(dir))
	}
	return strings.TrimSuffix(rel, ".jsonl")
}

// readLog passes the log's calls in the window to fn and reports whether it had any.
func readLog(path, session string, since, until time.Time, fn func(history.Call)) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	found := false
	var recent []history.Call // made within MaxCallTime of until: counted only once their result is in the log
	finished := map[string]bool{}
	for {
		line, longIDs, err := readLine(r)
		if len(line) > 0 && bytes.Contains(line, toolUseNeedle) {
			for _, c := range logCalls(line, session) {
				switch {
				case c.At.Before(since) || !c.At.Before(until):
				case c.ID != "" && c.At.After(until.Add(-MaxCallTime)):
					recent = append(recent, c)
				default:
					fn(c)
					found = true
				}
			}
		}
		if len(line) > 0 && bytes.Contains(line, toolResultNeedle) {
			for _, id := range resultIDs(line) {
				finished[id] = true
			}
		}
		for _, id := range longIDs { // results on a line too long to decode
			finished[id] = true
		}
		if errors.Is(err, io.EOF) {
			for _, c := range recent {
				if finished[c.ID] {
					fn(c)
					found = true
				}
			}
			return found, nil
		}
		if err != nil {
			return found, fmt.Errorf("%s: %w", path, err)
		}
	}
}

// readLine returns the next line, or nil for one longer than maxLine (its rest is skipped). For a long line, ids are
// the tool_use_ids of the results it carries, found as it streams past, so no result is missed however many there are.
func readLine(r *bufio.Reader) (line []byte, ids []string, err error) {
	var buf, tail []byte
	long := false
	for {
		chunk, err := r.ReadSlice('\n')
		if !long && len(buf)+len(chunk) > maxLine {
			long, tail, buf = true, buf, nil
		}
		if long {
			// tail carries the end of the previous chunk, so an id split across two chunks is still found.
			window := append(tail, chunk...)
			for _, m := range resultIDPattern.FindAllSubmatch(window, -1) {
				ids = append(ids, string(m[1]))
			}
			if len(window) > idOverlap {
				window = window[len(window)-idOverlap:]
			}
			tail = append([]byte(nil), window...)
		} else {
			buf = append(buf, chunk...)
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			return buf, ids, err
		}
	}
}

// idOverlap is longer than any `"tool_use_id": "..."` pair, so one split across two chunks is seen whole.
const idOverlap = 512

// resultIDPattern finds a result's id in a line too long to decode (a large tool output). Inside a JSON string the
// quotes are escaped, so text in a tool's output never matches.
var resultIDPattern = regexp.MustCompile(`"tool_use_id"\s*:\s*"([^"\\]+)"`)

// logRecord is the part of an assistant record a call needs.
type logRecord struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type toolUse struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// logCalls returns an assistant record's tool calls. Request blobs repeat calls already in the log and are not
// read; user records hold prompts and results.
// resultIDs are the ids of the calls a user record returns results for. Only the ids are kept, never the results.
func resultIDs(line []byte) []string {
	var rec struct {
		Type    string `json:"type"`
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &rec) != nil || rec.Type != "user" || len(rec.Message.Content) == 0 || rec.Message.Content[0] != '[' {
		return nil
	}
	var blocks []struct {
		Type string `json:"type"`
		ID   string `json:"tool_use_id"`
	}
	if json.Unmarshal(rec.Message.Content, &blocks) != nil {
		return nil
	}
	var out []string
	for _, b := range blocks {
		if b.Type == "tool_result" && b.ID != "" {
			out = append(out, b.ID)
		}
	}
	return out
}

func logCalls(line []byte, session string) []history.Call {
	var rec logRecord
	if json.Unmarshal(line, &rec) != nil || rec.Type != "assistant" || len(rec.Message.Content) == 0 || rec.Message.Content[0] != '[' {
		return nil
	}
	var blocks []json.RawMessage
	if json.Unmarshal(rec.Message.Content, &blocks) != nil {
		return nil
	}
	var out []history.Call
	for _, raw := range blocks {
		if !bytes.Contains(raw, toolUseNeedle) {
			continue // text and thinking blocks are never decoded
		}
		var b toolUse
		if json.Unmarshal(raw, &b) != nil || b.Type != "tool_use" || b.Name == "" {
			continue
		}
		c := history.Call{ID: b.ID, Tool: b.Name, At: rec.Timestamp, Session: session}
		if b.Name == history.SkillTool {
			c.Target, _ = b.Input["skill"].(string)
		} else {
			in := Input{HookEventName: "PreToolUse", ToolName: b.Name, ToolInput: b.Input}
			c.Target = in.Command() // what the gate would match a rule against
		}
		out = append(out, c)
	}
	return out
}
