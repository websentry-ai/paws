package codex

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

// The ledger is paws's own record of a Codex session, written by the hooks, because Codex's rollout files are not a
// stable interface. One JSONL file per thread: $PAWS_HOME/sessions/codex/<session>.jsonl for the main thread and
// <session>.<agent>.jsonl for a subagent, mode 0600. The gate appends each call with paws's verdict before it replies,
// so the next call already sees it; PostToolUse adds whether the call succeeded, UserPromptSubmit a turn marker (never
// the prompt), and a compaction a compaction marker.
type record struct {
	Kind      string         `json:"kind"`          // call | result | turn | compact
	At        time.Time      `json:"at"`            // when the hook started: a call is logged before paws decides
	End       time.Time      `json:"end,omitempty"` // when paws wrote the record, after it decided
	Turn      string         `json:"turn,omitempty"`
	ID        string         `json:"id,omitempty"`   // the tool_use_id; a call to several files has one record per file
	Part      int            `json:"part,omitempty"` // which file of the call
	Tool      string         `json:"tool,omitempty"` // canonical
	Target    string         `json:"target,omitempty"`
	Text      string         `json:"text,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Verdict   string         `json:"verdict,omitempty"`    // allow | deny
	SkillLoad string         `json:"skill_load,omitempty"` // the skill whose SKILL.md the call reads
	ReadFile  string         `json:"read_file,omitempty"`  // that SKILL.md, and the lines the read prints
	ReadFrom  int            `json:"read_from,omitempty"`  // (read_to 0: to the end; read_from < 0: the last lines)
	ReadTo    int            `json:"read_to,omitempty"`
	Denied    []string       `json:"denied,omitempty"` // a deny's paragraph openings, to find a digest's marker
	OK        *bool          `json:"ok,omitempty"`     // result: the call succeeded
	Files     []string       `json:"files,omitempty"`  // result of a patch: the files it changed
	Cut       bool           `json:"cut,omitempty"`    // result: Codex showed the model only the first Shown lines
	Shown     int            `json:"shown,omitempty"`
	Agent     string         `json:"agent,omitempty"` // the subagent's id; "" on the main thread
}

// maxRecordText bounds a call's text and input in the ledger: a patch can be megabytes.
const maxRecordText = 64 << 10

// maxLedgerRead bounds how much of a ledger's end is read; older calls count as absent (a reload costs one deny).
var maxLedgerRead int64 = 16 << 20

var idPart = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// ledgerPath is the ledger of a session's thread, or "" when no user dir or session id resolves.
func ledgerPath(session, agent string) string {
	dir := gate.UserDir()
	session = idPart.ReplaceAllString(session, "")
	if dir == "" || session == "" {
		return ""
	}
	name := session
	if agent = idPart.ReplaceAllString(agent, ""); agent != "" {
		name += "." + agent
	}
	return filepath.Join(dir, "sessions", Name, name+".jsonl")
}

func appendRecords(path string, recs []record) {
	if path == "" || len(recs) == 0 || os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	end := time.Now()
	var buf bytes.Buffer
	for _, r := range recs {
		r.End = end
		// Secrets masked, as in the audit log. Masking twice changes nothing, so the grader's own masking of a
		// target still finds it.
		r.Text, r.Target = gate.AuditTarget(r.Text), gate.AuditTarget(r.Target)
		r.Input, _ = masked(r.Input).(map[string]any)
		for i, d := range r.Denied {
			r.Denied[i] = gate.AuditTarget(d)
		}
		if len(r.Text) > maxRecordText {
			r.Text = r.Text[:maxRecordText]
		}
		if b, _ := json.Marshal(r.Input); len(b) > maxRecordText {
			r.Input = nil
		}
		line, err := json.Marshal(r)
		if err != nil {
			continue
		}
		buf.Write(append(line, '\n'))
	}
	delete(ledgerCache, path)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(buf.Bytes()) // one write: parallel calls' hooks append whole lines
}

// ledgerTTL is how long a session's ledger outlives its last write.
const ledgerTTL = 7 * 24 * time.Hour

// pruneLedgers removes the ledgers beside path that nobody wrote for ledgerTTL.
func pruneLedgers(path string) {
	if path == "" {
		return
	}
	old, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.jsonl"))
	for _, f := range old {
		if st, err := os.Stat(f); err == nil && time.Since(st.ModTime()) > ledgerTTL {
			os.Remove(f)
		}
	}
}

// masked is v with every string in it masked.
func masked(v any) any {
	switch x := v.(type) {
	case string:
		return gate.AuditTarget(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[gate.AuditTarget(k)] = masked(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = masked(e)
		}
		return out
	}
	return v
}

// ledgerCache holds ledgers already read in this process (paws gate is one process per hook call).
var ledgerCache = map[string][]record{}

// readLedger is the records of the ledger at path, oldest first; nil when there is none.
func readLedger(path string) []record {
	if recs, ok := ledgerCache[path]; ok {
		return recs
	}
	recs := loadLedger(path)
	ledgerCache[path] = recs
	return recs
}

func loadLedger(path string) []record {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil
	}
	off := max(st.Size()-maxLedgerRead, 0)
	buf := make([]byte, st.Size()-off)
	if _, err := f.ReadAt(buf, off); err != nil && !errors.Is(err, io.EOF) {
		return nil
	}
	lines := bytes.Split(buf, []byte("\n"))
	if off > 0 && len(lines) > 0 {
		lines = lines[1:] // the read began mid-line
	}
	var recs []record
	for _, l := range lines {
		var r record
		if len(l) > 0 && json.Unmarshal(l, &r) == nil {
			recs = append(recs, r)
		}
	}
	return recs
}

// results is whether each call that finished succeeded, by id.
func results(recs []record) map[string]bool {
	ok := map[string]bool{}
	for _, r := range recs {
		if r.Kind == "result" && r.OK != nil {
			ok[r.ID] = *r.OK
		}
	}
	return ok
}

// session is a Codex call's gate.Session: the calling thread's ledger, and the main thread's for session-wide
// questions.
type session struct {
	own, main string
	turn      string // the calling call's turn
	model     string
	spawning  bool // the call being gated starts a subagent
}

var _ gate.Session = session{}

// window is the thread's call records of the last n turns (distinct turn ids), newest first, up to a compaction.
func (s session) window(n int) []record {
	recs := readLedger(s.own)
	turns := map[string]bool{}
	if s.turn != "" {
		turns[s.turn] = true // the turn making this call counts, whether or not it has a record yet
	}
	var out []record
	for i := len(recs) - 1; i >= 0; i-- {
		r := recs[i]
		if r.Kind == "compact" {
			break // the agent no longer has anything from before it
		}
		if r.Turn != "" && !turns[r.Turn] {
			if len(turns) == n {
				break
			}
			turns[r.Turn] = true
		}
		if r.Kind == "call" {
			out = append(out, r)
		}
	}
	return out
}

// SkillLoaded: the reads of the skill's SKILL.md in the window that succeeded print every line of it together (see
// gate.Covered). A read still running (a call made alongside it) doesn't count yet.
func (s session) SkillLoaded(skill string, window int) bool {
	if skill == "" {
		return false
	}
	recs := readLedger(s.own)
	ok := results(recs)
	cut := map[string]record{} // each result that reached the model cut short
	for _, r := range recs {
		if r.Kind == "result" && r.Cut {
			cut[r.ID] = r
		}
	}
	var reads []gate.FileRead
	for _, r := range s.window(window) {
		if r.SkillLoad == skill && r.ReadFile != "" && r.Verdict == "allow" && ok[r.ID] {
			res := cut[r.ID]
			reads = append(reads, gate.FileRead{Skill: skill, File: r.ReadFile, Lines: rule.Lines{From: r.ReadFrom, To: r.ReadTo}, At: r.At,
				Cut: res.Cut, Shown: res.Shown})
		}
	}
	return gate.Covered(reads)
}

// DenyDelivered: a paws deny in the window opened a paragraph with marker. Codex shows a PreToolUse deny to the model
// as soon as the hook replies, so a deny in the ledger was delivered.
func (s session) DenyDelivered(marker string, window int) bool {
	if marker == "" {
		return false
	}
	for _, r := range s.window(window) {
		for _, d := range r.Denied {
			if strings.HasPrefix(d, marker) {
				return true
			}
		}
	}
	return false
}

// EditedMatching: the main thread wrote or edited a matching file, the call succeeded and, for a patch, that file is
// one apply_patch says it changed.
func (s session) EditedMatching(glob *regexp.Regexp) bool {
	recs := readLedger(s.main)
	ok := results(recs)
	changed := map[string][]string{}
	for _, r := range recs {
		if r.Kind == "result" && r.Files != nil {
			changed[r.ID] = r.Files
		}
	}
	for _, r := range recs {
		if r.Kind != "call" || r.Verdict != "allow" || r.Tool != "Edit" && r.Tool != "Write" || !glob.MatchString(r.Target) || !ok[r.ID] {
			continue
		}
		if slices.Contains(changed[r.ID], filepath.Clean(r.Target)) { // only a file apply_patch says it changed
			return true
		}
	}
	return false
}

// AgentsSinceUserTurn counts the subagents the main thread spawned in the current turn, the spawn being gated
// included, as Claude Code's transcript holds it.
func (s session) AgentsSinceUserTurn() int {
	n := 0
	if s.spawning {
		n++
	}
	for _, r := range readLedger(s.main) {
		if r.Kind == "call" && r.Turn == s.turn && (r.Tool == "Agent" || r.Tool == "Task") {
			n++
		}
	}
	return n
}

func (s session) Model() string { return s.model }

// denyHeads is the opening of each paragraph of a deny reason (paws joins denies with blank lines), enough to find a
// digest's marker without keeping the digest.
func denyHeads(reason string) []string {
	var heads []string
	for _, p := range strings.Split(reason, "\n\n") {
		if len(p) > 200 {
			p = p[:200]
		}
		heads = append(heads, p)
	}
	return heads
}

// turns is a Codex session's harness.TurnLog: the ledger, by turn id.
type turns struct {
	session string
	turn    string // the turn that stopped
}

// LastRecordAt is when paws finished the last ledger record of the turn that stopped: after the denies it logged.
func (t turns) LastRecordAt(until time.Time) (time.Time, bool) {
	var at time.Time
	for _, r := range readLedger(ledgerPath(t.session, "")) {
		end := r.End
		if end.Before(r.At) {
			end = r.At
		}
		if r.Turn == t.turn && !end.After(until) && end.After(at) {
			at = end
		}
	}
	return at, !at.IsZero()
}

// TurnCalls are the calls of the turn of the main thread's last record at or before until, and its subagents' calls
// made during it.
func (t turns) TurnCalls(until time.Time) ([]outcome.Call, time.Time, error) {
	main := readLedger(ledgerPath(t.session, ""))
	// The Stop names its turn: that one, unless until is before it began (the grader asking for the turn before).
	// Otherwise the turn of the last record at or before until; append order can't be trusted alone, since an async
	// UserPromptSubmit can land late.
	turn, began := "", time.Time{}
	for _, r := range main {
		if r.Turn == t.turn && (began.IsZero() || r.At.Before(began)) {
			began = r.At
		}
	}
	if !began.IsZero() && !until.Before(began) {
		turn = t.turn
	}
	if turn == "" {
		var latest time.Time
		for _, r := range main {
			if r.Turn != "" && r.Turn != t.turn && !r.At.After(until) && !r.At.Before(latest) {
				turn, latest = r.Turn, r.At
			}
		}
	}
	if turn == "" {
		return nil, time.Time{}, errors.New("no ledger record for the turn")
	}
	var start time.Time
	var calls []outcome.Call
	for _, r := range main {
		if r.Turn != turn || r.At.After(until) {
			continue
		}
		if start.IsZero() || r.At.Before(start) {
			start = r.At
		}
		if r.Kind == "call" {
			calls = append(calls, toCall(r))
		}
	}
	subs, _ := filepath.Glob(filepath.Join(filepath.Dir(ledgerPath(t.session, "")), idPart.ReplaceAllString(t.session, "")+".*.jsonl"))
	for _, p := range subs {
		for _, r := range readLedger(p) {
			if r.Kind == "call" && !r.At.Before(start) && !r.At.After(until) {
				calls = append(calls, toCall(r))
			}
		}
	}
	sortCalls(calls)
	return calls, start, nil
}

func toCall(r record) outcome.Call {
	c := outcome.Call{ID: r.ID + "/" + strconv.Itoa(r.Part), Tool: r.Tool, Target: r.Target, Text: r.Text, Input: r.Input,
		At: r.At, Agent: r.Agent, Denied: r.Verdict == "deny", SkillLoad: r.SkillLoad}
	if server, tool, ok := gate.MCPParts(r.Tool); ok {
		c.MCP = map[string]any{"server": server, "tool": tool, "class": rule.MCPAction(r.Tool)}
	}
	return c
}

func sortCalls(calls []outcome.Call) {
	sort.SliceStable(calls, func(i, j int) bool { return calls[i].At.Before(calls[j].At) })
}
