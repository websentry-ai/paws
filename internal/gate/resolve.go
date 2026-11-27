package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/websentry-ai/paws/internal/rule"
)

// Event is an agent-normalized pending action; adapters build it from native hook JSON, in the canonical vocabulary
// (Claude Code's tool names: Bash, Read, Write, Edit, Agent, mcp__<server>__<tool>, Skill).
type Event struct {
	Agent      string // the harness that produced it, the id stored on every record: "claude-code", "codex"
	Kind       string // rule.KindTool or rule.KindSession
	Tool       string
	Target     string // the string rules match against (command, path, or session source)
	Session    string
	Transcript string // path to Claude Code's session log, when the event has no Log
	Own        string // the calling thread's own log when it isn't Transcript: a subagent's
	Cwd        string // the agent's working directory; repo conditions resolve from it
	AgentID    string // set when the call comes from a subagent rather than the main thread
	AgentType  string
	// SkillLoad names the skill this call loads (the Skill tool, or a read of the skill's SKILL.md); such a call is
	// never gated, since it is the call that clears a require-skill gate.
	SkillLoad string
	// SkillWhole is set when this call loads the skill on its own: the Skill tool, or a read of the whole SKILL.md
	// (a chunk of it is still a skill read, never gated, but loads nothing alone).
	SkillWhole bool
	Log        Session // the session's past; nil reads Transcript and Own as Claude Code logs
	Harness   Harness // skill folders and deny wording; nil is SkillTool
}

// DefaultSkillWindow is how many turns back a skill load still counts for: the user's last 20 messages
// and the agent's work on each.
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
	skill       string // set on a deny that uses the default message: resolve merges these into one
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

// DecideAll is the decision for a call that touches several targets (a patch to several files), one event each:
// any deny wins, and each distinct reason and context is kept once.
func DecideAll(evs []Event, rules []*rule.Rule) Decision {
	if len(evs) == 1 {
		return Decide(evs[0], rules)
	}
	top := Decision{Verdict: Allow}
	var reasons, contexts []string
	for _, ev := range evs {
		d := Decide(ev, rules)
		if d.Verdict == Deny {
			top.Verdict = Deny
			reasons = appendNew(reasons, d.Reason)
		}
		if d.Context != "" {
			contexts = appendNew(contexts, d.Context)
		}
	}
	top.Reason = capText(strings.Join(reasons, "\n\n"))
	top.Context = capText(strings.Join(contexts, "\n"))
	return top
}

// resolve collapses matched rules into one decision: any deny wins, context concatenates, fired rules are audited.
func resolve(ev Event, matched []*rule.Rule) Decision {
	top := Decision{Verdict: Allow}
	var contexts, reasons, names, skills []string
	for _, r := range matched {
		oc := evaluate(r, ev)
		if oc.fired {
			logAudit(ev, r, oc)
		}
		if oc.context != "" {
			contexts = append(contexts, oc.context)
		}
		if oc.verdict != Deny {
			continue
		}
		top.Verdict = Deny
		if oc.skill == "" {
			reasons = append(reasons, oc.reason)
			continue
		}
		names = appendNew(names, r.Name)
		skills = appendNew(skills, oc.skill)
	}
	if len(skills) > 0 {
		// One sentence names every skill the default message would, instead of repeating it per rule.
		reasons = append([]string{skillReason(names, skills, ev)}, reasons...)
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
	return s[:cut] + "\n[paws: truncated; run `unbound-paws explain` for the full text]"
}

// evaluate resolves a matched rule to an outcome.
func evaluate(r *rule.Rule, ev Event) outcome {
	a := r.Action
	if a.Type != rule.ActionRequireSkill {
		return outcome{}
	}
	if ev.Kind == rule.KindTool && (ev.SkillLoad != "" || ev.Tool == "Skill") {
		// Loading a skill is how a require-skill gate clears; gating the Skill tool, or the read of a
		// SKILL.md, would lock the agent out of the very call that satisfies the rule.
		return outcome{}
	}
	if ev.session().SkillLoaded(a.Skill, DefaultSkillWindow) {
		return outcome{verdict: Allow, satisfiedBy: skillLoadSatisfied, fired: true}
	}
	digest, digestErr := loadDigest(r)
	if digest == "" && portalSkillMissing(r, ev) {
		// A portal rule whose skill isn't on this laptop (deleted by hand, or not installed
		// yet): the agent has nothing to load, so denying would be a dead end. The next config
		// check reinstalls the skill.
		return outcome{}
	}
	if digest != "" && ev.session().DenyDelivered(digestMarker(r), DefaultSkillWindow) {
		// The digest already rode in a deny this window: the agent has the rules, let the retry through.
		return outcome{verdict: Allow, satisfiedBy: digestSatisfied, fired: true}
	}
	if ev.Kind == rule.KindSession {
		// A session start can't be denied: name the skill (or hand over its digest) as context instead.
		if digest != "" {
			return outcome{verdict: Allow, context: digestMarker(r) + " " + digest, fired: true}
		}
		return outcome{verdict: Allow, context: msgOr(a.Message, sessionSkillContext(r, ev)), fired: true}
	}
	if digest != "" {
		return outcome{verdict: Deny, reason: digestReason(r, digest), fired: true}
	}
	if a.DenyMessage != "" {
		return outcome{verdict: Deny, reason: a.DenyMessage, fired: true, digestErr: digestErr}
	}
	return outcome{verdict: Deny, skill: a.Skill, fired: true, digestErr: digestErr}
}

// portalSkillMissing reports whether r is a portal rule (under UserDir()/portal) whose skill
// has no SKILL.md in the harness's user skills directory or cwd's project. paws installs every portal
// skill there, so unlike a local rule (whose skill may come from a plugin) absence is certain.
func portalSkillMissing(r *rule.Rule, ev Event) bool {
	user := UserDir()
	if user == "" {
		return false
	}
	root := user // loadDir resolves symlinks, so SourcePath is under the resolved dir
	if resolved, err := filepath.EvalSymlinks(user); err == nil {
		root = resolved
	}
	if !strings.HasPrefix(r.SourcePath, filepath.Join(root, "portal")+string(filepath.Separator)) {
		return false
	}
	return ev.skillFile(r.Action.Skill) == ""
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

// DigestProblem says why r's on_miss digest can't be used, or "" when it can (or r has none).
// The gate then falls back to naming the skill; explain shows this so the fallback isn't silent.
func DigestProblem(r *rule.Rule) string {
	_, problem := loadDigest(r)
	return problem
}

// digestMarker tags a digest deny so the transcript shows the agent already received it.
func digestMarker(r *rule.Rule) string {
	return fmt.Sprintf("paws digest (%s):", r.Name)
}

func digestReason(r *rule.Rule, digest string) string {
	return fmt.Sprintf("%s This action requires the %q skill. Its rules follow; apply them, then retry the command (re-formed to follow them if needed).\n\n%s",
		digestMarker(r), r.Action.Skill, digest)
}

func sessionSkillContext(r *rule.Rule, ev Event) string {
	skill := Skill{Name: r.Action.Skill, File: ev.skillFile(r.Action.Skill)}
	return fmt.Sprintf("paws (rule %s): this project needs the %q skill. %s", r.Name, r.Action.Skill, ev.harness().LoadAtStart(skill))
}

// skillReason is the default require-skill deny: it says paws sent it and which rules, so a person
// reading the chat knows where the block came from, and names every missing skill in one sentence.
// It opens by saying nothing is broken, because Claude Code labels every deny "hook error".
// e.g. paws (rule git--git): this is paws working as intended, nothing is broken. This command needs the "git" skill. Load it with the Skill tool, then retry this exact command.
func skillReason(names, skills []string, ev Event) string {
	label := "rule " + names[0]
	if len(names) > 1 {
		label = "rules " + strings.Join(names, ", ")
	}
	quoted := make([]string, len(skills))
	refs := make([]Skill, len(skills))
	for i, s := range skills {
		quoted[i] = fmt.Sprintf("%q", s)
		refs[i] = Skill{Name: s, File: ev.skillFile(s)}
	}
	need := fmt.Sprintf("the %s skill", quoted[0])
	if len(quoted) > 1 {
		need = fmt.Sprintf("the %s and %s skills", strings.Join(quoted[:len(quoted)-1], ", "), quoted[len(quoted)-1])
	}
	noun := actionNoun(ev)
	return fmt.Sprintf("paws (%s): this is paws working as intended, nothing is broken. This %s needs %s. %s", label, noun, need, ev.harness().LoadSkills(refs, noun))
}

// actionNoun names the call a deny blocks: Claude retries "this exact command" for Bash, and the
// same wording for an edit or a tool call.
func actionNoun(ev Event) string {
	switch ev.Tool {
	case "Bash":
		return "command"
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		return "edit"
	}
	return "tool call"
}

func appendNew(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func msgOr(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
