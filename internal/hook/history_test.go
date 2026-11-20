package hook

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/websentry-ai/paws/internal/history"
)

// writeLog writes lines as a JSONL log under dir.
func writeLog(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assistantCall(ts, id, name, input string) string {
	return `{"type":"assistant","timestamp":"` + ts + `","message":{"role":"assistant","content":[{"type":"tool_use","id":"` + id + `","name":"` + name + `","input":` + input + `}]}}`
}

func TestTranscriptsReadsOnlyToolCalls(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "-Users-me-proj", "s1.jsonl")
	writeLog(t, main,
		`{"type":"user","timestamp":"2026-11-10T10:00:00Z","message":{"role":"user","content":"deploy with token ghp_abcdefghijklmnop and tool_use"}}`,
		assistantCall("2026-11-10T10:00:01Z", "toolu_1", "Bash", `{"command":"git push origin main"}`),
		// A text block and a thinking block in the same record as a call: only the call is read.
		`{"type":"assistant","timestamp":"2026-11-10T10:00:02Z","message":{"content":[{"type":"thinking","thinking":"tool_use secret"},{"type":"text","text":"tool_use here"},{"type":"tool_use","id":"toolu_2","name":"Edit","input":{"file_path":"/Users/me/proj/main.tf","old_string":"a","new_string":"b"}}]}}`,
		`{"type":"user","timestamp":"2026-11-10T10:00:03Z","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_2","content":"tool_use output"}]}}`,
		// A request blob repeats calls already in the log.
		`{"type":"api-request-blob","timestamp":"2026-11-10T10:00:04Z","message":{"content":[{"type":"tool_use","id":"toolu_9","name":"Bash","input":{"command":"rm -rf /"}}]}}`,
		assistantCall("2026-11-10T10:00:05Z", "toolu_3", "Skill", `{"skill":"humanizer"}`),
		assistantCall("2026-11-10T10:00:06Z", "toolu_4", "mcp__claude_ai_Notion__notion-create-pages", `{"parent":{"page_id":"x"}}`),
		assistantCall("2026-09-01T10:00:00Z", "toolu_old", "Bash", `{"command":"ls"}`), // before the window
		`not json "tool_use"`,
	)
	writeLog(t, filepath.Join(dir, "-Users-me-proj", "s1", "subagents", "agent-abc.jsonl"),
		assistantCall("2026-11-11T09:00:00Z", "toolu_5", "Read", `{"file_path":"/etc/hosts"}`))
	writeLog(t, filepath.Join(dir, "-Users-me-other", "s2.jsonl"),
		`{"type":"user","timestamp":"2026-11-12T09:00:00Z","message":{"content":"hi"}}`)
	old := filepath.Join(dir, "-Users-me-proj", "s0.jsonl")
	writeLog(t, old, assistantCall("2026-11-10T10:00:00Z", "toolu_stale", "Bash", `{"command":"ls"}`))
	ago := time.Now().AddDate(0, -3, 0)
	os.Chtimes(old, ago, ago) // last written before the window: not opened

	var got []history.Call
	since := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	n, err := Transcripts{Dir: dir}.Calls(since, until, func(c history.Call) { got = append(got, c) })
	if err != nil {
		t.Fatal(err)
	}
	want := []history.Call{
		{ID: "toolu_1", Tool: "Bash", Target: "git push origin main"},
		{ID: "toolu_2", Tool: "Edit", Target: "/Users/me/proj/main.tf"},
		{ID: "toolu_3", Tool: "Skill", Target: "humanizer"},
		{ID: "toolu_4", Tool: "mcp__claude_ai_Notion__notion-create-pages", Target: `{"parent":{"page_id":"x"}}`},
		{ID: "toolu_5", Tool: "Read", Target: "/etc/hosts"},
	}
	sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
	if len(got) != len(want) {
		t.Fatalf("calls %+v, want %+v", got, want)
	}
	for i, c := range got {
		w := want[i]
		if c.Session != "s1" {
			t.Errorf("call %d session %q, want s1 (a subagent's calls count toward its parent)", i, c.Session)
		}
		if c.ID != w.ID || c.Tool != w.Tool || c.Target != w.Target || c.At.IsZero() {
			t.Errorf("call %d = %+v, want %+v", i, c, w)
		}
	}
	if n != 1 {
		t.Errorf("sessions = %d, want 1 (the subagent's log is part of s1; s2 made no call)", n)
	}
}

func TestTranscriptsWithNoProjectsFolder(t *testing.T) {
	n, err := Transcripts{Dir: filepath.Join(t.TempDir(), "missing")}.Calls(time.Time{}, time.Now(), func(history.Call) {
		t.Error("no call expected")
	})
	if err != nil || n != 0 {
		t.Fatalf("got %d, %v; want 0, nil", n, err)
	}
}

func TestTranscriptsFollowsASymlinkedFolder(t *testing.T) {
	real := t.TempDir()
	writeLog(t, filepath.Join(real, "p", "s.jsonl"), assistantCall("2026-11-10T10:00:00Z", "toolu_1", "Bash", `{"command":"ls"}`))
	link := filepath.Join(t.TempDir(), "projects")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	n, err := Transcripts{Dir: link}.Calls(time.Time{}, time.Now(), func(history.Call) {})
	if err != nil || n != 1 {
		t.Fatalf("got %d sessions, %v; want 1", n, err)
	}
}

func TestTranscriptsSkipsAHugeLine(t *testing.T) {
	dir := t.TempDir()
	huge := assistantCall("2026-11-10T10:00:00Z", "toolu_big", "Bash", `{"command":"echo `+strings.Repeat("x", maxLine)+`"}`)
	writeLog(t, filepath.Join(dir, "p", "s.jsonl"), huge, assistantCall("2026-11-10T10:00:01Z", "toolu_ok", "Bash", `{"command":"ls"}`))
	var ids []string
	Transcripts{Dir: dir}.Calls(time.Time{}, time.Now(), func(c history.Call) { ids = append(ids, c.ID) })
	if len(ids) != 1 || ids[0] != "toolu_ok" {
		t.Fatalf("ids %v, want only toolu_ok", ids)
	}
}

// A log it can't read stops the scan: a summary without it would replace the laptop's last one.
func TestTranscriptsFailsOnAnUnreadableLog(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "p", "s.jsonl")
	writeLog(t, locked, assistantCall("2026-11-10T10:00:00Z", "toolu_1", "Bash", `{"command":"ls"}`))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })
	if f, err := os.Open(locked); err == nil {
		f.Close()
		t.Skip("running as a user who can read any file")
	}
	if _, err := (Transcripts{Dir: dir}).Calls(time.Time{}, time.Now(), func(history.Call) {}); err == nil {
		t.Fatal("an unreadable log must fail the scan")
	}
}

// A recent call with no result yet may still be running, so its hook record may come after the scan: it waits.
// An old call with no result (a session that crashed) never gets one, so it counts.
func TestTranscriptsSkipCallsStillRunning(t *testing.T) {
	use := func(id, ts string) string {
		return `{"type":"assistant","timestamp":"` + ts + `","message":{"content":[{"type":"tool_use","id":"` + id + `","name":"Bash","input":{"command":"ls"}}]}}`
	}
	result := func(id string) string {
		return `{"type":"user","timestamp":"2026-11-19T11:59:00Z","message":{"content":[{"type":"tool_result","tool_use_id":"` + id + `","content":"x"}]}}`
	}
	// A result too long to decode (a large tool output) still marks its call finished.
	huge := `{"type":"user","timestamp":"2026-11-19T11:59:00Z","message":{"content":[{"tool_use_id":"big","type":"tool_result","content":"` + strings.Repeat("x", maxLine) + `"}]}}`
	log := strings.Join([]string{use("old", "2026-11-10T10:00:00Z"), use("done", "2026-11-19T11:00:00Z"), result("done"), use("big", "2026-11-19T11:10:00Z"), huge, use("running", "2026-11-19T11:30:00Z")}, "\n") + "\n"
	projects := t.TempDir()
	dir := filepath.Join(projects, "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []string
	if _, err := (Transcripts{Dir: projects}).Calls(time.Time{}, time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC), func(c history.Call) { got = append(got, c.ID) }); err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "big,done,old" {
		t.Errorf("calls %v; want big, done and old, not running", got)
	}
}
