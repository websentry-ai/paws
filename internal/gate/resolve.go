package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/websentry-ai/paws/internal/rule"
)

// Event is an agent-normalized pending action; adapters build it from native hook JSON.
type Event struct {
	Agent      string // which agent produced it, e.g. "claude-code"
	Kind       string // rule.KindTool or rule.KindPrompt
	Tool       string
	Target     string // the string rules match against (command, path, or prompt)
	Session    string
	Transcript string // path to the session log, when the agent supplies one
	Cwd        string // the agent's working directory; repo conditions resolve from it
	AgentID    string // set when the call comes from a subagent rather than the main thread
	AgentType  string
	Raw        json.RawMessage // original agent event, passed through to exec-scripts
}

// DefaultSkillWindow is how many of the agent's own turns back a skill load still counts for.
const DefaultSkillWindow = 20

// skillLoadSatisfied tags an audit record whose allow came from a cleared require-skill gate.
const skillLoadSatisfied = "skill-load"

// digestSatisfied tags an audit record whose require-skill passed because its digest was delivered this window.
const digestSatisfied = "digest-delivered"

// adviceSatisfied tags an audit record whose advise was skipped: the agent already has it this window.
const adviceSatisfied = "advice-delivered"

// Verdict ranks how a matched rule wants the call handled; higher wins on conflict.
type Verdict int

const (
	Allow     Verdict = iota // audit
	Ask                      // warn
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
	verdict     Verdict
	reason      string
	context     string
	fired       bool   // did the rule act — i.e. worth auditing
	satisfiedBy string // non-empty when a gate that would have denied was cleared instead
}

// Decide filters rules to those matching the event and resolves them.
func Decide(ev Event, rules []*rule.Rule) Decision {
	var matched []*rule.Rule
	for _, r := range rules {
		if Applies(r, ev) {
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
		if SkillLoaded(ev.Transcript, a.Skill, DefaultSkillWindow) {
			return outcome{verdict: Allow, satisfiedBy: skillLoadSatisfied, fired: true}
		}
		digest := loadDigest(r)
		if digest != "" && DenyDelivered(ev.Transcript, digestMarker(r), DefaultSkillWindow) {
			// The digest already rode in a deny this window: the agent has the rules, let the retry through.
			return outcome{verdict: Allow, satisfiedBy: digestSatisfied, fired: true}
		}
		if ev.Kind == rule.KindSession {
			// A session start can't be denied: name the skill (or hand over its digest) as context instead.
			if digest != "" {
				return outcome{verdict: Allow, context: digestMarker(r) + " " + digest, fired: true}
			}
			return outcome{verdict: Allow, context: msgOr(a.Message, sessionSkillContext(a)), fired: true}
		}
		if digest != "" {
			return outcome{verdict: DenyRetry, reason: digestReason(r, digest), fired: true}
		}
		return outcome{verdict: DenyRetry, reason: msgOr(a.DenyMessage, skillReason(a)), fired: true}
	case "advise":
		marker := adviseMarker(r)
		if ContextDelivered(ev.Transcript, marker, DefaultSkillWindow) {
			return outcome{verdict: Allow, satisfiedBy: adviceSatisfied, fired: true}
		}
		return outcome{verdict: Allow, context: marker + " " + a.Message, fired: true}
	case "warn":
		return outcome{verdict: Ask, reason: msgOr(a.DenyMessage, fmt.Sprintf("paws rule %q requires approval before this runs.", r.Name)), fired: true}
	case "audit":
		return outcome{verdict: Allow, fired: true} // logged, agent sees nothing
	case "prompt-inject":
		// Deliberately not fired: this is a context nudge, not a policy event — nothing to audit.
		return outcome{verdict: Allow, context: a.Message}
	case "exec-script":
		return runScript(r, ev)
	}
	return outcome{}
}

// MaxDigest is the largest digest that rides in a deny; Claude Code caps hook output at 10k.
const MaxDigest = 7000

// loadDigest returns the rule's digest when on_miss is digest and the file is readable and
// within MaxDigest, else "" so the rule falls back to load-required.
func loadDigest(r *rule.Rule) string {
	if r.Action.OnMiss != rule.OnMissDigest || r.Action.Digest == "" {
		return ""
	}
	// rule.Parse rejects absolute and escaping paths, so this stays inside the rule's directory.
	data, err := os.ReadFile(filepath.Join(filepath.Dir(r.SourcePath), r.Action.Digest))
	if err != nil || len(data) > MaxDigest {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// digestMarker tags a digest deny so the transcript shows the agent already received it.
func digestMarker(r *rule.Rule) string {
	return fmt.Sprintf("paws digest (%s):", r.Name)
}

func digestReason(r *rule.Rule, digest string) string {
	return fmt.Sprintf("%s This action requires the %q skill. Its rules follow; apply them, then retry the command (re-formed to follow them if needed).\n\n%s",
		digestMarker(r), r.Action.Skill, digest)
}

// adviseMarker tags advise context so the transcript shows which rule already advised this window.
func adviseMarker(r *rule.Rule) string {
	return fmt.Sprintf("paws (%s):", r.Name)
}

func sessionSkillContext(a rule.Action) string {
	return fmt.Sprintf("This project requires the %q skill. Invoke it with the Skill tool before you start work.", a.Skill)
}

func skillReason(a rule.Action) string {
	return fmt.Sprintf("This action requires the %q skill. Invoke it with the Skill tool, then retry this exact command.", a.Skill)
}

func msgOr(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
