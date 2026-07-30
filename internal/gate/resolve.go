package gate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/websentry-ai/paws/internal/rule"
)

// Event is an agent-normalized pending action; adapters build it from native hook JSON.
type Event struct {
	Agent   string // which agent produced it, e.g. "claude-code"
	Kind    string // rule.KindTool or rule.KindPrompt
	Tool    string
	Target  string // the string rules match against (command, path, or prompt)
	Session string
	Raw     json.RawMessage // original agent event, passed through to exec-scripts
}

// Verdict ranks how a matched rule wants the call handled; higher wins on conflict.
type Verdict int

const (
	Allow     Verdict = iota // audit / warn / prompt-inject
	Ask                      // require-approval
	DenyRetry                // require-skill / exec-script deny (has a retry path)
	Deny                     // block (hard)
)

// Decision is the engine's merged answer for one event.
type Decision struct {
	Verdict Verdict
	Reason  string // agent-facing message for ask/deny
	Context string // advisory text to surface to the agent
}

// outcome is one matched rule's evaluated result.
type outcome struct {
	verdict Verdict
	reason  string
	context string
	fired   bool // did the rule act — i.e. worth auditing
}

// Decide filters rules to those matching the event and resolves them.
func Decide(ev Event, rules []*rule.Rule) Decision {
	var matched []*rule.Rule
	for _, r := range rules {
		if ev.Kind == "" || rule.Kind(r.Event) != ev.Kind {
			continue
		}
		if r.Matches(ev.Tool, ev.Target) {
			matched = append(matched, r)
		}
	}
	return resolve(ev, matched)
}

// resolve collapses matched rules into one decision: deny > ask > allow wins, context concatenates, fired rules are audited.
func resolve(ev Event, matched []*rule.Rule) Decision {
	top := Decision{Verdict: Allow}
	var contexts []string
	for _, r := range matched {
		oc := evaluate(r, ev)
		if oc.fired {
			logAudit(ev, r, oc)
		}
		if oc.context != "" {
			contexts = append(contexts, oc.context)
		}
		if oc.verdict > top.Verdict {
			top.Verdict, top.Reason = oc.verdict, oc.reason
		}
	}
	top.Context = strings.Join(contexts, "\n")
	return top
}

// evaluate resolves a matched rule to an outcome. Only exec-script runs a subprocess.
func evaluate(r *rule.Rule, ev Event) outcome {
	a := r.Action
	switch a.Type {
	case "block":
		return outcome{verdict: Deny, reason: msgOr(a.DenyMessage, fmt.Sprintf("Blocked by paws rule %q.", r.Name)), fired: true}
	case "require-skill":
		// No compliance tracking yet: this denies every retry unconditionally, even after the skill is loaded.
		return outcome{verdict: DenyRetry, reason: msgOr(a.DenyMessage,
			fmt.Sprintf("This action requires the %q skill. Load it (read its SKILL.md) for context — paws still denies an automated retry of this exact command; a human needs to run it directly.", a.Skill)), fired: true}
	case "require-approval":
		return outcome{verdict: Ask, reason: msgOr(a.DenyMessage, fmt.Sprintf("paws rule %q requires approval before this runs.", r.Name)), fired: true}
	case "warn":
		return outcome{verdict: Allow, context: msgOr(a.Message, fmt.Sprintf("paws: rule %q applies to this action.", r.Name)), fired: true}
	case "prompt-inject":
		return outcome{verdict: Allow, context: a.Message, fired: true}
	case "audit":
		return outcome{verdict: Allow, fired: true} // logged, agent sees nothing
	case "exec-script":
		return runScript(r, ev)
	}
	return outcome{}
}

func msgOr(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
