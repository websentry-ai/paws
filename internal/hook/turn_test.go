package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTurnCallsReadsTheLastTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	lines := []string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"first prompt"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"old","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:01:00Z","message":{"content":[{"type":"text","text":"second prompt"}]}}`,
		`{"type":"user","isMeta":true,"timestamp":"2026-11-20T10:01:00Z","message":{"content":"<system-reminder>not a prompt</system-reminder>"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:01:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"rm -rf build"}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:01:02Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","is_error":true,"content":"PreToolUse:Bash hook error: paws: load coreutils"}]}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:01:03Z","message":{"content":[{"type":"tool_use","id":"b","name":"Write","input":{"file_path":"/x/a.go","content":"package a"}}]}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:01:04Z","message":{"content":[{"type":"tool_use","id":"c","name":"mcp__notion__create_pages","input":{"title":"T","parent":{"page_id":"p"}}}]}}`,
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
	sub := filepath.Join(dir, "s", "subagents")
	os.MkdirAll(sub, 0o700)
	os.WriteFile(filepath.Join(sub, "agent-x1.jsonl"), []byte(strings.Join([]string{
		`{"type":"assistant","timestamp":"2026-11-20T10:00:30Z","message":{"content":[{"type":"tool_use","id":"s0","name":"Bash","input":{"command":"before the turn"}}]}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:01:05Z","message":{"content":[{"type":"tool_use","id":"s1","name":"Bash","input":{"command":"go test ./..."}}]}}`,
	}, "\n")+"\n"), 0o600)

	calls, start, err := TurnCalls(path, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if start.Format("15:04:05") != "10:01:00" {
		t.Errorf("turn start %v, want the second prompt (a meta message isn't one)", start)
	}
	var got []string
	for _, c := range calls {
		got = append(got, c.ID+":"+c.Agent)
	}
	if strings.Join(got, ",") != "a:,b:,c:,s1:x1" {
		t.Fatalf("calls %v; want this turn's main calls and the subagent's call in it", got)
	}
	if !calls[0].Denied || calls[1].Denied {
		t.Error("the hook-refused call should be marked denied, the others not")
	}
	if calls[1].Text != "package a" || calls[2].Text != "p\nT" || calls[2].MCP["server"] != "notion" {
		t.Errorf("text/mcp: %q %q %v", calls[1].Text, calls[2].Text, calls[2].MCP)
	}
}

// A prompt sent while the async Stop hook runs belongs to the next turn: grading stops at the Stop.
func TestTurnCallsStopsAtTheStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(path, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"first"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"ls"}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:09Z","message":{"content":"next prompt, typed fast"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:10Z","message":{"content":[{"type":"tool_use","id":"b","name":"Bash","input":{"command":"pwd"}}]}}`,
	}, "\n")+"\n"), 0o600)
	calls, _, err := TurnCalls(path, time.Date(2026, 11, 20, 10, 0, 5, 0, time.UTC))
	if err != nil || len(calls) != 1 || calls[0].ID != "a" {
		t.Fatalf("calls %v, %v; want the stopped turn's call a only", calls, err)
	}
}
