package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

// outcomeTail bounds how much of the audit log's end the turn grader reads: a turn's records, many times over.
const outcomeTail = 4 << 20

// denySlack allows for the transcript's timestamp (when Claude wrote the call) and the deny record's (when the hook
// ran) being a second or two apart.
const denySlack = 3 * time.Second

// gradeTurn runs in the Stop hook (async: Claude isn't waiting). For each call a require-skill rule denied in the turn
// that just ended, it reads what followed in Claude Code's log and appends an outcome record: the outcome and each
// expect check's result. Never the calls themselves. Best effort: anything it can't read, it leaves ungraded.
func gradeTurn(in *hook.Input, ev gate.Event, stoppedAt time.Time) {
	if in.TranscriptPath == "" || ev.Session == "" {
		return
	}
	// The turn that stopped: a prompt the user sends while this async hook runs belongs to the next turn.
	calls, start, err := hook.TurnCalls(in.TranscriptPath, stoppedAt)
	if err != nil || len(calls) == 0 {
		return
	}
	// Stop hooks run async and can overlap: one grader at a time, so two never grade the same block.
	unlock, ok := lockGrading()
	if !ok {
		return
	}
	defer unlock()
	// From the turn's own start: a deny from the turn before isn't this turn's.
	denies, graded := turnDenies(ev.Session, start)
	if len(denies) == 0 {
		return
	}
	rules := map[string]*rule.Rule{}
	for _, r := range gate.Collect(in.CWD).Rules {
		rules[r.SourcePath+"|"+r.Name] = r
	}
	// One call can be denied by several rules (one deny names every skill): each rule claims its own blocked calls.
	used := map[string]map[int]bool{}
	seen := map[string]int{}
	for _, d := range denies {
		key := denyKey(d.Source, d.Rule, d.AgentID, blockedWhen(d.At, d.TS))
		r := rules[d.Source+"|"+d.Rule]
		if r == nil || r.Action.Type != rule.ActionRequireSkill {
			continue
		}
		ruleKey := d.Source + "|" + d.Rule
		if used[ruleKey] == nil {
			used[ruleKey] = map[int]bool{}
		}
		at, _ := time.Parse(time.RFC3339, d.TS)
		i := blockedCall(calls, d, at, used[ruleKey])
		if i < 0 {
			continue // not in this log (another agent's turn, or the log isn't flushed): left ungraded
		}
		used[ruleKey][i] = true // claimed, graded now or before, so a later deny of this rule takes the next call
		// Denies with the same key (the same rule twice in one second): the first graded[key] were graded before.
		if seen[key]++; seen[key] <= graded[key] {
			continue
		}
		res := outcome.Grade(r, calls, i)
		rec := map[string]any{"session": ev.Session, "agent": ev.Agent, "rule": d.Rule, "skill": r.Action.Skill,
			"source": d.Source, "tool": d.Tool, "outcome": res.Outcome, "checks": res.Checks, "blocked_at": d.TS}
		if d.AgentID != "" {
			rec["agent_id"] = d.AgentID
		}
		if d.At != "" {
			rec["blocked_exact"] = d.At // local, like the deny's at: "graded already" tells same-second denies apart
		}
		gate.LogOutcome(rec)
	}
}

// gradeLockStale is when a grader's lock is taken over: past the Stop hook's 15 s timeout its holder is gone.
const gradeLockStale = 16 * time.Second

// lockGrading takes ~/.paws/outcomes.lock by creating it (portable: no flock on Windows), waiting up to 12 s for
// another grader. ok is false when it couldn't: that turn stays ungraded rather than graded twice.
func lockGrading() (unlock func(), ok bool) {
	dir := gate.UserDir()
	if dir == "" {
		return nil, false
	}
	path := filepath.Join(dir, "outcomes.lock")
	for i := 0; i < 120; i++ { // up to 12 s: long enough to outlast a lock its killed holder left
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(path) }, true
		}
		if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) > gradeLockStale {
			os.Remove(path)
			continue
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, false
}

// denyKey names a deny for "graded already": its rule file and name, agent and second.
func denyKey(source, name, agent, ts string) string {
	return source + "|" + name + "|" + agent + "|" + ts
}

// blockedCall is the first refused call of d's tool and target, made by d's agent no later than the deny.
func blockedCall(calls []outcome.Call, d auditRecord, at time.Time, used map[int]bool) int {
	for i, c := range calls {
		if used[i] || !c.Denied || c.Tool != d.Tool || c.Agent != d.AgentID || c.At.After(at.Add(denySlack)) {
			continue
		}
		if gate.AuditTarget(c.Target) == d.Target {
			return i
		}
	}
	return -1
}

// turnDenies reads the end of the audit log: the session's require-skill denies since `since`, and how many blocks
// per deny key were graded already (a Stop that fires twice for one turn grades it once).
func turnDenies(session string, since time.Time) ([]auditRecord, map[string]int) {
	var denies []auditRecord
	graded := map[string]int{}
	log := gate.AuditPath()
	if log == "" {
		return nil, graded
	}
	// The log may have rotated mid-turn: the rotated file's end first, then the live one.
	for _, path := range []string{log + ".1", log} {
		readDenies(path, session, since, &denies, graded)
	}
	return denies, graded
}

// readDenies reads the end of one audit file into denies and graded (see turnDenies).
func readDenies(path, session string, since time.Time, denies *[]auditRecord, graded map[string]int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > outcomeTail {
		f.Seek(st.Size()-outcomeTail, io.SeekStart)
	}
	r := bufio.NewReaderSize(f, 1<<20)
	first := true
	for {
		line, err := r.ReadBytes('\n')
		if first && len(line) > 0 {
			first = false
			if st, _ := f.Stat(); st != nil && st.Size() > outcomeTail {
				line = nil // a partial line where the read began
			}
		}
		var rec auditRecord
		if len(line) > 0 && json.Unmarshal(line, &rec) == nil && rec.Session == session {
			switch {
			case rec.Event == "outcome":
				graded[denyKey(rec.Source, rec.Rule, rec.AgentID, blockedWhen(rec.BlockedExact, rec.BlockedAt))]++
			case rec.Verdict == "deny" && rec.Rule != "":
				if denyAfter(rec, since) {
					*denies = append(*denies, rec)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// denyAfter reports whether a deny happened at or after since: by its exact time, or, for a record from an older
// paws without one, by its second (so a deny in the turn's first second still counts).
func denyAfter(rec auditRecord, since time.Time) bool {
	if at, err := time.Parse(time.RFC3339Nano, rec.At); err == nil {
		return !at.Before(since)
	}
	ts, err := time.Parse(time.RFC3339, rec.TS)
	return err == nil && !ts.Before(since.Truncate(time.Second))
}

// blockedWhen is a deny's time for "graded already": exact when the records have it, else its second.
func blockedWhen(exact, ts string) string {
	if exact != "" {
		return exact
	}
	return ts
}
