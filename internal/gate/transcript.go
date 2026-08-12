package gate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
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
	if path == "" || skill == "" || window < 1 {
		return false
	}
	lines, err := tailLines(path)
	if err != nil {
		return false
	}
	turns := 0
	for i := len(lines) - 1; i >= 0; i-- {
		var e transcriptEntry
		if json.Unmarshal(lines[i], &e) != nil {
			continue
		}
		// Subagents write their own transcript files, so sidechain records should not
		// appear here at all; skipping them keeps that guarantee if that ever changes.
		if e.IsSidechain {
			continue
		}
		switch {
		case e.Type == "system" && e.Subtype == "compact_boundary":
			return false
		case e.Type == "assistant":
			turns++
			if turns > window {
				return false
			}
		case e.Type == "user" && len(e.ToolUseResult) > 0 && e.ToolUseResult[0] == '{':
			var sr skillResult
			if json.Unmarshal(e.ToolUseResult, &sr) == nil && sr.Success && sr.CommandName == skill {
				return true
			}
		}
	}
	return false
}

// tailLines returns the last tailBytes of path split into whole lines.
func tailLines(path string) ([][]byte, error) {
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
