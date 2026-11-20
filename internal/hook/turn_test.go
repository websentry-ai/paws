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

// A compaction summary, a task notification or a system reminder mid-turn is not a new prompt.
func TestTurnCallsKeepsTheTurnAcrossHarnessRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(path, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"clean up"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"rm -rf build"}}]}}`,
		`{"type":"user","isCompactSummary":true,"timestamp":"2026-11-20T10:00:02Z","message":{"content":"This session is being continued..."}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:03Z","message":{"content":"<task-notification>done</task-notification>"}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:04Z","message":{"content":[{"type":"text","text":"<system-reminder>x</system-reminder>"}]}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:05Z","message":{"content":[{"type":"tool_use","id":"b","name":"Bash","input":{"command":"ls"}}]}}`,
	}, "\n")+"\n"), 0o600)
	calls, start, err := TurnCalls(path, time.Time{})
	if err != nil || len(calls) != 2 || start.Format("15:04:05") != "10:00:00" {
		t.Fatalf("calls %d from %v, %v; want both calls of the one turn", len(calls), start, err)
	}
}

// A sidechain record in the main log is a subagent's call, not the main agent's.
func TestTurnCallsSkipsSidechainRecordsInTheMainLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(path, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"clean up"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"rm -rf build"}}]}}`,
		`{"type":"assistant","isSidechain":true,"timestamp":"2026-11-20T10:00:02Z","message":{"content":[{"type":"tool_use","id":"s","name":"Bash","input":{"command":"ls build"}}]}}`,
	}, "\n")+"\n"), 0o600)
	calls, _, err := TurnCalls(path, time.Time{})
	if err != nil || len(calls) != 1 || calls[0].ID != "a" {
		t.Fatalf("calls %v, %v; want the main agent's call a only", calls, err)
	}
}

// A call the user rejected at the permission prompt never ran, like a refused one.
func TestTurnCallsMarksRejectedCallsAsNotRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(path, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"clean up"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"rm -rf build"}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:02Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","is_error":true,"content":"The user doesn't want to proceed with this tool use. The tool use was rejected."}]}}`,
	}, "\n")+"\n"), 0o600)
	calls, _, err := TurnCalls(path, time.Time{})
	if err != nil || len(calls) != 1 || !calls[0].Denied {
		t.Fatalf("calls %+v, %v; want the rejected call marked as not run", calls, err)
	}
}

// A command that ran and failed, printing a refusal's words, isn't a refused call.
func TestTurnCallsDoesNotReadOutputAsARefusal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(path, []byte(strings.Join([]string{
		`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"run the tests"}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:01Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"go test ./..."}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:02Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","is_error":true,"content":"Exit code 1\n--- FAIL: want PreToolUse:Bash hook error: paws"}]}}`,
		`{"type":"assistant","timestamp":"2026-11-20T10:00:03Z","message":{"content":[{"type":"tool_use","id":"b","name":"Bash","input":{"command":"rm -rf build"}}]}}`,
		`{"type":"user","timestamp":"2026-11-20T10:00:04Z","message":{"content":[{"type":"tool_result","tool_use_id":"b","is_error":true,"content":"<tool_use_error>PreToolUse:Bash hook error: paws: load coreutils</tool_use_error>"}]}}`,
	}, "\n")+"\n"), 0o600)
	calls, _, err := TurnCalls(path, time.Time{})
	if err != nil || len(calls) != 2 || calls[0].Denied || !calls[1].Denied {
		t.Fatalf("calls %+v, %v; want only b refused", calls, err)
	}
}

// A subagent log that can't be read fails the turn: grading half a turn could call a retry made there "stopped".
func TestTurnCallsFailsOnAnUnreadableSubagentLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	os.WriteFile(path, []byte(`{"type":"user","timestamp":"2026-11-20T10:00:00Z","message":{"content":"go"}}`+"\n"), 0o600)
	sub := filepath.Join(dir, "s", "subagents", "agent-a.jsonl")
	os.MkdirAll(filepath.Dir(sub), 0o700)
	os.WriteFile(sub, []byte("{}\n"), 0o000) // there, but not readable
	if _, _, err := TurnCalls(path, time.Time{}); err == nil {
		t.Error("an unreadable subagent log was skipped")
	}
}
