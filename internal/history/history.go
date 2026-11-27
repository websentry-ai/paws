// Package history counts what an agent did in its past sessions, so the portal can pick skills on day one. It is
// harness-neutral: a Source reads one agent's own logs into Calls, and Summarize matches them against the catalog's
// triggers on this machine and keeps only counts. Prompts, replies, tool output, arguments and paths never leave it.
package history

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Call is one tool call an agent made, in paws's tool vocabulary: the names the gate matches rules against (Bash,
// Read, Write, Edit, mcp__<server>__<tool>, Skill...). A Source for another agent maps its own tool names to these.
type Call struct {
	// ID is the agent's own id for the call. A resumed or forked session copies earlier calls into a new log, so a
	// call seen twice counts once.
	ID     string
	Tool   string
	Target string // what a matcher runs against: the command, file path or MCP input; the skill name for Skill
	At     time.Time
	// Session is the agent's id for the session the call was made in; a subagent's calls carry their parent's.
	Session string
	// Cwd is where the session ran, when the log says: a relative path in the call is taken from it.
	Cwd string
	// Partial is set on a Read of part of a file (an offset or a limit).
	Partial bool
}

// Source reads one agent's past sessions.
type Source interface {
	// Name identifies the agent in the upload, such as "claude-code".
	Name() string
	// Calls passes every call made in [since, until) to fn and returns how many sessions had one.
	Calls(since, until time.Time, fn func(Call)) (sessions int, err error)
}

// Trigger is one catalog action: a skill and the matcher that says when it applies.
type Trigger struct {
	Skill   string
	Action  string                         // the action's id in the catalog
	Matches func(tool, target string) bool // called from several goroutines at once
}

// SkillTool is the tool a Call names when the agent loaded a skill itself.
const SkillTool = "Skill"

// Summary is what leaves the machine: counts only.
type Summary struct {
	Source     string `json:"source"`
	WindowDays int    `json:"window_days"`
	Since      string `json:"since"` // the window, as dates
	Until      string `json:"until"`
	Sessions   int    `json:"sessions"`
	ToolCalls  int    `json:"tool_calls"`
	// Skills are the catalog skills whose triggers matched, most calls first.
	Skills []SkillHits `json:"skills"`
	// Loaded are catalog skills the agent loaded without being asked. Other skill names are never sent.
	Loaded []Count `json:"loaded"`
	// Unmatched counts the calls no trigger matched, by a coarse signature (see Signature): demand for skills the
	// catalog doesn't have yet.
	Unmatched []Count `json:"unmatched"`
}

// SkillHits is one skill's matched calls. Calls counts each call once even when several of its actions match it.
type SkillHits struct {
	Skill    string   `json:"skill"`
	Calls    int      `json:"calls"`
	LastSeen string   `json:"last_seen"`
	Actions  []Action `json:"actions"`
}

// Action is how many calls one of a skill's actions matched.
type Action struct {
	ID    string `json:"id"`
	Calls int    `json:"calls"`
}

// Count is a name and how often it came up.
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// maxUnmatched bounds the signature list; the long tail says little.
const maxUnmatched = 100

// Catalog is what a scan matches against, from the catalog the portal serves.
type Catalog struct {
	Triggers []Trigger
	Skills   map[string]bool // the names a Skill call may report
	// MCPPatterns are the catalog's MCP tool patterns without their mcp__ prefix, as in <server>__<tool>
	// ("attio__create-note", "claude_ai_Gmail__*", "*azure*__storage*"). An unmatched MCP call names its server only
	// when one of them covers it, and its tool only when one names that tool exactly.
	MCPPatterns []string
}

// Summarize reads src's calls in [since, until) and counts them against cat. Each distinct (tool, target) is
// matched once, however often it ran.
func Summarize(src Source, since, until time.Time, days int, cat Catalog) (Summary, error) {
	type key struct{ tool, target string }
	type seen struct {
		n    int
		last time.Time
	}
	calls := map[key]*seen{}
	ids := map[string]bool{}
	loaded := map[string]int{}
	total := 0
	sessions, err := src.Calls(since, until, func(c Call) {
		if c.ID != "" {
			if ids[c.ID] {
				return
			}
			ids[c.ID] = true
		}
		if c.Tool == SkillTool {
			if cat.Skills[c.Target] {
				loaded[c.Target]++
			}
			return
		}
		total++
		k := key{c.Tool, c.Target}
		s := calls[k]
		if s == nil {
			s = &seen{}
			calls[k] = s
		}
		s.n++
		if c.At.After(s.last) {
			s.last = c.At
		}
	})
	sum := Summary{Source: src.Name(), WindowDays: days, Since: day(since), Until: day(until), Sessions: sessions,
		ToolCalls: total, Skills: []SkillHits{}, Loaded: counts(loaded, 0), Unmatched: []Count{}}
	if err != nil {
		return sum, err
	}
	keys := make([]key, 0, len(calls))
	for k := range calls {
		keys = append(keys, k)
	}
	// Longest first: a long command costs the most to match, and starting those early keeps the workers even.
	sort.Slice(keys, func(i, j int) bool { return len(keys[i].target) > len(keys[j].target) })
	// Matching is most of a scan's time (thousands of distinct commands against every trigger), and each command is
	// independent, so it runs on every core. Trigger.Matches must be safe to call concurrently.
	workers := min(runtime.GOMAXPROCS(0), max(1, len(keys)))
	parts := make([]partial, workers)
	var wg sync.WaitGroup
	var next atomic.Int64
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := partial{bySkill: map[string]*tally{}, unmatched: map[string]int{}}
			for i := int(next.Add(1)) - 1; i < len(keys); i = int(next.Add(1)) - 1 {
				k := keys[i]
				p.add(k.tool, k.target, calls[k].n, calls[k].last, &cat)
			}
			parts[w] = p
		}()
	}
	wg.Wait()
	bySkill, unmatched := map[string]*tally{}, map[string]int{}
	for _, p := range parts {
		for name, t := range p.bySkill {
			if bySkill[name] == nil {
				bySkill[name] = &tally{actions: map[string]int{}}
			}
			bySkill[name].merge(t)
		}
		for sig, n := range p.unmatched {
			unmatched[sig] += n
		}
	}
	for name, tl := range bySkill {
		h := SkillHits{Skill: name, Calls: tl.calls, LastSeen: day(tl.last)}
		for _, c := range counts(tl.actions, 0) {
			h.Actions = append(h.Actions, Action{ID: c.Name, Calls: c.Count})
		}
		sum.Skills = append(sum.Skills, h)
	}
	sort.Slice(sum.Skills, func(i, j int) bool {
		a, b := sum.Skills[i], sum.Skills[j]
		return a.Calls > b.Calls || a.Calls == b.Calls && a.Skill < b.Skill
	})
	sum.Unmatched = counts(unmatched, maxUnmatched)
	return sum, nil
}

// tally is one skill's counts.
type tally struct {
	calls   int
	last    time.Time
	actions map[string]int
}

func (t *tally) merge(o *tally) {
	t.calls += o.calls
	if o.last.After(t.last) {
		t.last = o.last
	}
	for id, n := range o.actions {
		t.actions[id] += n
	}
}

// partial is what one matching worker counted.
type partial struct {
	bySkill   map[string]*tally
	unmatched map[string]int
}

// add counts n calls of (tool, target), the latest at last, against every trigger.
func (p partial) add(tool, target string, n int, last time.Time, cat *Catalog) {
	matched := map[string]bool{}
	for _, t := range cat.Triggers {
		if !t.Matches(tool, target) {
			continue
		}
		tl := p.bySkill[t.Skill]
		if tl == nil {
			tl = &tally{actions: map[string]int{}}
			p.bySkill[t.Skill] = tl
		}
		tl.actions[t.Action] += n
		if !matched[t.Skill] {
			matched[t.Skill] = true
			tl.calls += n
			if last.After(tl.last) {
				tl.last = last
			}
		}
	}
	if len(matched) == 0 {
		p.unmatched[cat.Signature(tool, target)] += n
	}
}

// counts sorts m most frequent first, then by name; limit 0 keeps all.
func counts(m map[string]int, limit int) []Count {
	out := make([]Count, 0, len(m))
	for name, n := range m {
		out = append(out, Count{name, n})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || out[i].Count == out[j].Count && out[i].Name < out[j].Name
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func day(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.DateOnly)
}
