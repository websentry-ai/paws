// Package outcome grades what happened after paws blocked a call: by the end of that turn, did Claude retry the same
// call, fix it, take another route or stop, and did the rule's expect checks pass? It knows nothing about any agent:
// an adapter (hook.TurnCalls for Claude Code) turns the agent's own log of the turn into Calls.
package outcome

import (
	"regexp"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

// Call is one tool call of the turn, in the shape expect checks see (see rule.Expect).
type Call struct {
	ID     string
	Tool   string         // as the agent names it: Bash, Write, Edit, mcp__server__tool …
	Target string         // what the gate matches rules against (hook.Input.Command)
	Text   string         // the call's main text: a Bash command, written content, an Edit's new text …
	Input  map[string]any // the raw tool input
	MCP    map[string]any // server, tool and class for an MCP call, else nil
	At     time.Time
	Agent  string // "" for the main conversation, else the subagent's id
	// AgentStart is when the subagent was launched (the first record of its log); zero for the main conversation, or
	// when unknown, in which case its first call stands in.
	AgentStart time.Time
	Denied     bool // a hook (paws or another) refused it
}

// Env is the call as an expect check sees it.
func (c Call) Env() map[string]any {
	m := map[string]any{"tool": c.Tool, "text": c.Text, "input": c.Input}
	if c.Input == nil {
		m["input"] = map[string]any{}
	}
	if c.MCP != nil {
		m["mcp"] = c.MCP
	}
	return m
}

// The outcomes, in the order the portal shows them.
const (
	Fixed            = "fixed"             // retried, and every check passed
	SameRetry        = "same_retry"        // retried the same call (checks failed or none)
	ChangedFailed    = "changed_failed"    // retried a changed call, and a check failed
	ChangedUnchecked = "changed_unchecked" // retried a changed call; the rule has no checks
	OtherRoute       = "other_route"       // never retried; other calls ran
	Stopped          = "stopped"           // never retried; nothing else ran
)

// Result is one graded block.
type Result struct {
	Outcome string
	Checks  map[string]bool // expect id → passed
	Retry   *Call
}

// Grade grades the block of calls[blocked] by rule r. calls is the whole turn in time order, every agent's calls
// together. Only the blocked agent's later calls count, plus, for a block in the main conversation, subagents it
// started after the block: a subagent already running is a separate task.
func Grade(r *rule.Rule, calls []Call, blocked int) Result {
	b := calls[blocked]
	scope := inScope(calls, blocked)
	retry := -1
	for i, c := range scope {
		if !c.Denied && r.Matches(c.Tool, c.Target) {
			retry = i
			break
		}
	}
	res := Result{Checks: map[string]bool{}}
	blockedEnv := b.Env()
	seen := func(cs []Call, e *rule.Expect) bool {
		for _, c := range cs {
			if !c.Denied && e.Eval(c.Env(), blockedEnv) {
				return true
			}
		}
		return false
	}
	for i := range r.Expect {
		e := &r.Expect[i]
		var pass bool
		switch e.When {
		case rule.ExpectBefore:
			pass = retry >= 0 && seen(scope[:retry+1], e)
		case rule.ExpectRetry:
			pass = retry >= 0 && e.Eval(scope[retry].Env(), blockedEnv)
		case rule.ExpectAfter:
			pass = retry >= 0 && seen(scope[retry+1:], e)
		case rule.ExpectNever:
			pass = true
			for _, c := range scope {
				if !c.Denied && e.Hit(c.Env(), blockedEnv) {
					pass = false
					break
				}
			}
		}
		res.Checks[e.ID] = pass
	}
	if retry < 0 {
		res.Outcome = Stopped
		for _, c := range scope {
			if !c.Denied {
				res.Outcome = OtherRoute
				break
			}
		}
		return res
	}
	res.Retry = &scope[retry]
	all := len(r.Expect) > 0
	for _, ok := range res.Checks {
		all = all && ok
	}
	switch {
	case all:
		res.Outcome = Fixed
	case sameCall(scope[retry], b):
		res.Outcome = SameRetry
	case len(r.Expect) > 0:
		res.Outcome = ChangedFailed
	default:
		res.Outcome = ChangedUnchecked
	}
	return res
}

// inScope is the blocked agent's calls after the block, with, for a main-conversation block, every subagent whose
// first call came after it.
func inScope(calls []Call, blocked int) []Call {
	b := calls[blocked]
	first := map[string]time.Time{} // when each subagent started
	for _, c := range calls {
		start := c.At
		if !c.AgentStart.IsZero() {
			start = c.AgentStart
		}
		if t, ok := first[c.Agent]; !ok || start.Before(t) {
			first[c.Agent] = start
		}
	}
	var out []Call
	for _, c := range calls[blocked+1:] {
		switch {
		case c.Agent == b.Agent:
			out = append(out, c)
		case b.Agent == "" && c.Agent != "" && first[c.Agent].After(b.At):
			out = append(out, c)
		}
	}
	return out
}

// sameCall: a Bash retry is the same command once setup in front and spacing are ignored; any other call is the
// same only when its target and text match exactly (a Write's content, an Edit's new text: spacing counts there).
func sameCall(a, b Call) bool {
	if a.Tool == "Bash" {
		return Same(a.Target, b.Target)
	}
	return a.Target == b.Target && a.Text == b.Text
}

// lead is setup in front of the real command: a `cd dir &&`, an env assignment, a `(` opening a subshell.
var lead = regexp.MustCompile(`^\s*(?:\(\s*|(?:cd|pushd)\s+\S+\s*(?:&&|;)\s*|[A-Za-z_]\w*=\S*\s+)`)

// Same reports whether two calls' targets are the same once setup in front and spacing are ignored.
func Same(a, b string) bool {
	norm := func(s string) string {
		for {
			loc := lead.FindStringIndex(s)
			if loc == nil {
				return strings.Join(strings.Fields(s), " ")
			}
			s = s[loc[1]:]
		}
	}
	return norm(a) == norm(b)
}
