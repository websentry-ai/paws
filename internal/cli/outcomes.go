package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
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
func gradeTurn(in *hook.Input, ev gate.Event) {
	if in.TranscriptPath == "" || ev.Session == "" {
		return
	}
	calls, start, err := hook.TurnCalls(in.TranscriptPath)
	if err != nil || len(calls) == 0 {
		return
	}
	denies, graded := turnDenies(ev.Session, start.Add(-denySlack))
	if len(denies) == 0 {
		return
	}
	rules := map[string]*rule.Rule{}
	for _, r := range gate.Collect(in.CWD).Rules {
		rules[r.SourcePath+"|"+r.Name] = r
	}
	used := map[int]bool{}
	for _, d := range denies {
		if graded[d.Rule+"|"+d.TS] {
			continue
		}
		r := rules[d.Source+"|"+d.Rule]
		if r == nil || r.Action.Type != rule.ActionRequireSkill {
			continue
		}
		at, _ := time.Parse(time.RFC3339, d.TS)
		i := blockedCall(calls, d, at, used)
		if i < 0 {
			continue // not in this log (another agent's turn, or the log isn't flushed): left ungraded
		}
		used[i] = true
		res := outcome.Grade(r, calls, i)
		rec := map[string]any{"session": ev.Session, "agent": ev.Agent, "rule": d.Rule, "skill": r.Action.Skill,
			"source": d.Source, "tool": d.Tool, "outcome": res.Outcome, "checks": res.Checks, "blocked_at": d.TS}
		if d.AgentID != "" {
			rec["agent_id"] = d.AgentID
		}
		gate.LogOutcome(rec)
	}
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

// turnDenies reads the end of the audit log: the session's require-skill denies since `since`, and the blocks
// already graded (a Stop that fires twice for one turn grades it once).
func turnDenies(session string, since time.Time) ([]auditRecord, map[string]bool) {
	var denies []auditRecord
	graded := map[string]bool{}
	path := gate.AuditPath()
	f, err := os.Open(path)
	if err != nil {
		return nil, graded
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
				graded[rec.Rule+"|"+rec.BlockedAt] = true
			case rec.Verdict == "deny" && rec.Rule != "":
				if ts, err := time.Parse(time.RFC3339, rec.TS); err == nil && !ts.Before(since) {
					denies = append(denies, rec)
				}
			}
		}
		if err != nil {
			break
		}
	}
	return denies, graded
}
