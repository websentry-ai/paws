package cli

import (
	"bufio"
	"cmp"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

// outcomeTail bounds how much of the audit log's end the turn grader reads: a turn's records, many times over.
const outcomeTail = 4 << 20

// gradeTurn grades the turn a Claude Code Stop hook ended (see gradeTurnIn).
func gradeTurn(in *hook.Input, ev gate.Event, stoppedAt time.Time) {
	gradeTurnIn(hook.Claude{}.Turns(&harness.Hook{Native: in}), ev, in.CWD, stoppedAt)
}

// gradeTurnIn runs in the Stop hook (async: the agent isn't waiting). For each call a require-skill rule denied in the
// turn that just ended, it reads what followed in the harness's log of the turn and appends an outcome record: the
// outcome and each expect check's result, and the one call that answered the block (see gate.DescribeCall), never the
// rest of the turn. Best effort: anything it can't read, it leaves ungraded.
func gradeTurnIn(turns harness.TurnLog, ev gate.Event, cwd string, stoppedAt time.Time) {
	if turns == nil || ev.Session == "" {
		return
	}
	// The hook starts a moment after the harness fires Stop, and the next turn may already have made a call by then.
	// The turn ended when the harness's log says it did (Claude Code: the final message the Stop names, else the log's
	// last record).
	if at, ok := turns.LastRecordAt(stoppedAt); ok {
		stoppedAt = at
	}
	// Stop hooks run async and can overlap: one grader at a time, so two never grade the same block.
	unlock, ok := lockGrading(ev.Session)
	if !ok {
		return
	}
	defer unlock()
	// The session's denies up to the Stop, from the audit log's end. Most turns have none left to grade: then the
	// transcript, which grows with the session, isn't read at all. Denies before the last Stop graded are settled,
	// graded or never gradable (an interrupted turn, say), so only newer ones count here.
	all, allows, graded := turnDenies(ev.Session, time.Time{}, stoppedAt)
	settled := gradedThrough(ev.Session)
	if settled.After(stoppedAt) {
		settled = time.Time{} // a later Stop already ran: this one may still have its own turn's denies
	}
	// Settled through this Stop unless the turn couldn't be read in full: then its denies stay eligible, and the
	// next Stop reads them again (once: by then they're the turn before's, and the next Stop settles them).
	settle := true
	defer func() {
		if settle {
			markGraded(ev.Session, stoppedAt)
		}
	}()
	var fresh []auditRecord
	for _, d := range all {
		if denyIn(d, settled, stoppedAt) {
			fresh = append(fresh, d)
		}
	}
	if !ungraded(fresh, graded) {
		return
	}
	// The turn that stopped: a prompt the user sends while this async hook runs belongs to the next turn.
	// No calls since the last prompt: that prompt came after the Stop, and the denies before it are the turn before's,
	// graded below. A deny of this turn with no call to match is left eligible by gradeCalls.
	calls, start, err := turns.TurnCalls(stoppedAt)
	if err != nil {
		settle = false
		return
	}
	// From the turn's own start: a deny from the turn before isn't this turn's.
	var denies, earlier []auditRecord
	for _, d := range all {
		if denyIn(d, start, stoppedAt) {
			denies = append(denies, d)
		}
	}
	for _, d := range fresh {
		if !denyIn(d, start, stoppedAt) {
			earlier = append(earlier, d)
		}
	}
	// Denies left eligible by a Stop that couldn't read its turn (the log not flushed yet) belong to the turn before
	// this one: read that turn, which ended when this one began.
	if len(earlier) > 0 && !start.IsZero() {
		if prev, prevStart, err := turns.TurnCalls(start.Add(-time.Nanosecond)); err == nil && len(prev) > 0 {
			var mine []auditRecord
			for _, d := range all {
				if denyIn(d, prevStart, start.Add(-time.Nanosecond)) {
					mine = append(mine, d)
				}
			}
			gradeCalls(cwd, ev, prev, mine, allows, graded)
		}
	}
	if len(denies) == 0 {
		return
	}
	if !gradeCalls(cwd, ev, calls, denies, allows, graded) {
		settle = false
	}
}

// gradeCalls grades each deny against the calls of its turn and appends the outcome records. It reports false when a
// deny's call isn't in the calls (another agent's turn, or the log isn't flushed): that deny is left ungraded.
func gradeCalls(cwd string, ev gate.Event, calls []outcome.Call, denies, allows []auditRecord, graded map[string]int) bool {
	complete := true
	rules := map[string]*rule.Rule{}
	for _, r := range gate.Collect(cwd).Rules {
		rules[r.SourcePath+"|"+r.Name] = r
	}
	// One call can be denied by several rules (one deny names every skill): each rule claims its own blocked calls.
	used := map[string]map[int]bool{}
	seen := map[string]int{}
	for _, d := range denies {
		key := denyKey(d.Source, d.Rule, d.AgentID, blockedWhen(d.At, d.TS))
		r := rules[d.Source+"|"+d.Rule]
		// The rule as it is now: a portal sync may have replaced it since the deny. Gone, or bound to another skill,
		// it can't grade this block (left ungraded); new checks still grade what followed.
		if r == nil || r.Action.Type != rule.ActionRequireSkill || d.Skill != "" && r.Action.Skill != d.Skill {
			continue
		}
		ruleKey := d.Source + "|" + d.Rule
		if used[ruleKey] == nil {
			used[ruleKey] = map[int]bool{}
		}
		at, err := time.Parse(time.RFC3339Nano, d.At)
		if err != nil {
			at, _ = time.Parse(time.RFC3339, d.TS)
			at = at.Add(time.Second - time.Nanosecond)
		}
		i := blockedCall(calls, d, at, used[ruleKey])
		if i < 0 {
			complete = false
			continue
		}
		used[ruleKey][i] = true // claimed, graded now or before, so a later deny of this rule takes the next call
		// Denies with the same key (the same rule twice in one second): the first graded[key] were graded before.
		if seen[key]++; seen[key] <= graded[key] {
			continue
		}
		var applied func(outcome.Call) bool
		if len(r.When) > 0 {
			ids := appliedCalls(calls, allows, ruleKey)
			applied = func(c outcome.Call) bool { return ids[c.ID] }
		}
		if at.After(calls[i].DeniedAt) { // the later of the audit's deny time and the harness's own (Codex: hook end)
			calls[i].DeniedAt = at
		}
		res := outcome.GradeWhere(r, calls, i, applied)
		rec := map[string]any{"session": ev.Session, "agent": ev.Agent, "rule": d.Rule, "skill": r.Action.Skill,
			"source": d.Source, "tool": d.Tool, "outcome": res.Outcome, "checks": res.Checks, "blocked_at": d.TS}
		if ran := cmp.Or(res.Retry, res.Other); ran != nil {
			rec["retry_command"] = gate.DescribeCall(gate.Event{Tool: ran.Tool, Target: ran.Target, Cwd: cwd})
		}
		gate.Stamp(rec, ev) // the model at Stop: the fallback when the deny logged none (a session's or subagent's first call)
		if d.Model != "" {
			rec["model"] = d.Model // the model that made the blocked call (a subagent's, or before a /model switch)
		}
		if d.AgentID != "" {
			rec["agent_id"] = d.AgentID
		}
		if d.At != "" {
			rec["blocked_exact"] = d.At // local, like the deny's at: "graded already" tells same-second denies apart
		}
		gate.LogOutcome(rec)
	}
	return complete
}

// gradeLockStale is when a grader's lock is taken over: past the Stop hook's 15 s timeout its holder is gone.
const gradeLockStale = 16 * time.Second

// lockGrading takes ~/.paws/outcomes.lock by creating it (portable: no flock on Windows), waiting up to 12 s for
// another grader. ok is false when it couldn't: that turn stays ungraded rather than graded twice.
func lockGrading(session string) (unlock func(), ok bool) {
	dir := gate.UserDir()
	if dir == "" {
		return nil, false
	}
	// One lock per session: graders of different sessions never touch the same denies, so they never wait on each
	// other. A grader takes well under a second, so the wait is short and leaves the Stop hook time to sync.
	path := filepath.Join(dir, "outcomes-"+lockName(session)+".lock")
	for i := 0; i < 20; i++ { // up to 2 s
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

// lockName keeps a session id's letters, digits and dashes, so it is a safe file name.
func lockName(session string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, session)
}

// denyKey names a deny for "graded already": its rule file and name, agent and second.
func denyKey(source, name, agent, ts string) string {
	return source + "|" + name + "|" + agent + "|" + ts
}

// blockedCall finds the call a deny refused: of the refused calls this rule hasn't claimed with the deny's tool,
// agent and target, the latest at or before the deny. A call is logged before its hooks run, on the same clock, so
// an earlier refusal of the same command (another hook's) is never picked over the one paws refused. A record with
// only ts counts to the end of its second.
func blockedCall(calls []outcome.Call, d auditRecord, at time.Time, used map[int]bool) int {
	best := -1
	for i, c := range calls {
		if used[i] || !c.Denied || c.Tool != d.Tool || c.Agent != d.AgentID || c.At.After(at) {
			continue
		}
		if gate.AuditTarget(c.Target) == d.Target && (best < 0 || !c.At.Before(calls[best].At)) {
			best = i
		}
	}
	return best
}

// turnDenies reads the end of the audit log: the session's require-skill denies and allows since `since`, and how
// many blocks per deny key were graded already (a Stop that fires twice for one turn grades it once).
func turnDenies(session string, since, until time.Time) (denies, allows []auditRecord, graded map[string]int) {
	var fired []auditRecord
	graded = map[string]int{}
	log := gate.AuditPath()
	if log == "" {
		return nil, nil, graded
	}
	// The log may have rotated mid-turn: the rotated file's end first, then the live one.
	for _, path := range []string{log + ".1", log} {
		readDenies(path, session, since, until, &fired, graded)
	}
	for _, rec := range fired {
		if rec.Verdict == "deny" {
			denies = append(denies, rec)
		} else {
			allows = append(allows, rec)
		}
	}
	return denies, allows, graded
}

// appliedCalls is the calls a rule applied to and let through, by tool_use id: for each allow it logged, the latest
// call that ran with its tool, agent and target at or before it.
func appliedCalls(calls []outcome.Call, allows []auditRecord, ruleKey string) map[string]bool {
	ids := map[string]bool{}
	for _, a := range allows {
		if a.Source+"|"+a.Rule != ruleKey {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, a.At)
		if err != nil {
			at, _ = time.Parse(time.RFC3339, a.TS)
			at = at.Add(time.Second - time.Nanosecond)
		}
		best := -1
		for i, c := range calls {
			if c.Denied || ids[c.ID] || c.Tool != a.Tool || c.Agent != a.AgentID || c.At.After(at) ||
				gate.AuditTarget(c.Target) != a.Target {
				continue
			}
			if best < 0 || !c.At.Before(calls[best].At) {
				best = i
			}
		}
		if best >= 0 {
			ids[calls[best].ID] = true
		}
	}
	return ids
}

// readDenies reads the end of one audit file: its fired-rule records into fired, outcomes into graded.
func readDenies(path, session string, since, until time.Time, fired *[]auditRecord, graded map[string]int) {
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
			case rec.Verdict != "" && rec.Rule != "": // a fired rule: a deny, or an allow
				if denyIn(rec, since, until) {
					*fired = append(*fired, rec)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// denyIn reports whether a deny happened from since to until: by its exact time, or, for a record from an older
// paws without one, by its second (so a deny in the turn's first or last second still counts).
func denyIn(rec auditRecord, since, until time.Time) bool {
	if at, err := time.Parse(time.RFC3339Nano, rec.At); err == nil {
		return !at.Before(since) && !at.After(until)
	}
	ts, err := time.Parse(time.RFC3339, rec.TS)
	return err == nil && !ts.Before(since.Truncate(time.Second)) && !ts.After(until)
}

// gradedThrough is when this session's last Stop was graded (zero if never); markGraded records it, and drops the
// marks of sessions idle for a week.
func gradedThrough(session string) time.Time {
	data, err := os.ReadFile(gradedMark(session))
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	return t
}

func markGraded(session string, at time.Time) {
	path := gradedMark(session)
	if path == "" || gradedThrough(session).After(at) {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	os.WriteFile(path, []byte(at.UTC().Format(time.RFC3339Nano)), 0o600)
	if old, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*")); len(old) > 0 {
		for _, f := range old {
			if st, err := os.Stat(f); err == nil && time.Since(st.ModTime()) > 7*24*time.Hour {
				os.Remove(f)
			}
		}
	}
}

func gradedMark(session string) string {
	dir := gate.UserDir()
	if dir == "" || lockName(session) == "" {
		return ""
	}
	return filepath.Join(dir, "graded", lockName(session))
}

// ungraded reports whether any deny has no outcome record yet.
func ungraded(denies []auditRecord, graded map[string]int) bool {
	seen := map[string]int{}
	for _, d := range denies {
		key := denyKey(d.Source, d.Rule, d.AgentID, blockedWhen(d.At, d.TS))
		if seen[key]++; seen[key] > graded[key] {
			return true
		}
	}
	return false
}

// blockedWhen is a deny's time for "graded already": exact when the records have it, else its second.
func blockedWhen(exact, ts string) string {
	if exact != "" {
		return exact
	}
	return ts
}
