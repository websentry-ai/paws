package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/websentry-ai/paws/internal/rule"
)

// Event is an agent-normalized pending action; adapters build it from native hook JSON.
type Event struct {
	Agent      string // which agent produced it, e.g. "claude-code"
	Kind       string // rule.KindTool or rule.KindSession
	Tool       string
	Target     string // the string rules match against (command, path, or session source)
	Session    string
	Transcript string // path to the session log, when the agent supplies one
	Cwd        string // the agent's working directory; repo conditions resolve from it
	AgentID    string // set when the call comes from a subagent rather than the main thread
	AgentType  string
}

// DefaultSkillWindow is how many of the agent's own turns back a skill load still counts for.
const DefaultSkillWindow = 20

// skillLoadSatisfied tags an audit record whose allow came from a cleared require-skill gate.
const skillLoadSatisfied = "skill-load"

// digestSatisfied tags an audit record whose require-skill passed because its digest was delivered this window.
const digestSatisfied = "digest-delivered"

// Verdict is how the matched rules want the call handled.
type Verdict int

const (
	Allow Verdict = iota
	Deny          // a required skill isn't loaded; the agent loads it and retries
)

// Decision is the engine's merged answer for one event.
type Decision struct {
	Verdict Verdict
	Reason  string // agent-facing message for a deny
	Context string // text added to the agent's context (session start)
}

// outcome is one matched rule's evaluated result.
type outcome struct {
	verdict     Verdict
	reason      string
	context     string
	fired       bool   // did the rule act — i.e. worth auditing
	satisfiedBy string // non-empty when a gate that would have denied was cleared instead
	digestErr   string // why an on_miss digest could not be used (fell back to load-required)
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

// resolve collapses matched rules into one decision: any deny wins, context concatenates, fired rules are audited.
func resolve(ev Event, matched []*rule.Rule) Decision {
	top := Decision{Verdict: Allow}
	var contexts, reasons []string
	for _, r := range matched {
		oc := evaluate(r, ev)
		if oc.fired {
			logAudit(ev, r, oc)
		}
		if oc.context != "" {
			contexts = append(contexts, oc.context)
		}
		if oc.verdict == Deny {
			top.Verdict = Deny
			reasons = append(reasons, oc.reason)
		}
	}
	// Every deny reaches the agent at once (each skill or digest it needs), so the
	// next attempt clears them all instead of discovering them one deny at a time.
	top.Reason = capText(strings.Join(reasons, "\n\n"))
	top.Context = capText(strings.Join(contexts, "\n"))
	return top
}

// maxHookText keeps a reason or context under Claude Code's 10,000-character hook output cap,
// past which it shows a file preview instead of the text.
const maxHookText = 9500

func capText(s string) string {
	if len(s) <= maxHookText {
		return s
	}
	cut := maxHookText
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n[paws: truncated; run `paws explain` for the full text]"
}

// evaluate resolves a matched rule to an outcome.
func evaluate(r *rule.Rule, ev Event) outcome {
	a := r.Action
	if a.Type != rule.ActionRequireSkill {
		return outcome{}
	}
	if ev.Kind == rule.KindTool && ev.Tool == "Skill" {
		// Loading a skill is how a require-skill gate clears; gating the Skill tool would lock
		// the agent out of the very call that satisfies the rule.
		return outcome{}
	}
	if SkillLoaded(ev.Transcript, a.Skill, DefaultSkillWindow) {
		return outcome{verdict: Allow, satisfiedBy: skillLoadSatisfied, fired: true}
	}
	digest, digestErr := loadDigest(r)
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
		return outcome{verdict: Deny, reason: digestReason(r, digest), fired: true}
	}
	return outcome{verdict: Deny, reason: msgOr(a.DenyMessage, skillReason(a)), fired: true, digestErr: digestErr}
}

// loadDigest returns the rule's digest when on_miss is digest. On a missing, unreadable or
// oversize file it returns "" and the reason, so the caller falls back to load-required and
// the audit log says why.
func loadDigest(r *rule.Rule) (string, string) {
	if r.Action.OnMiss != rule.OnMissDigest || r.Action.Digest == "" {
		return "", ""
	}
	// rule.Parse rejects absolute and escaping paths, so this stays inside the rule's directory.
	data, err := os.ReadFile(filepath.Join(filepath.Dir(r.SourcePath), r.Action.Digest))
	if err != nil {
		return "", err.Error()
	}
	if len(data) > rule.MaxDigest {
		return "", fmt.Sprintf("digest is %d bytes, over the %d-byte limit", len(data), rule.MaxDigest)
	}
	return strings.TrimSpace(string(data)), ""
}

// digestMarker tags a digest deny so the transcript shows the agent already received it.
func digestMarker(r *rule.Rule) string {
	return fmt.Sprintf("paws digest (%s):", r.Name)
}

func digestReason(r *rule.Rule, digest string) string {
	return fmt.Sprintf("%s This action requires the %q skill. Its rules follow; apply them, then retry the command (re-formed to follow them if needed).\n\n%s",
		digestMarker(r), r.Action.Skill, digest)
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
