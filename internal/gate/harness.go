package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

// Session is what the engine asks about the calling session's past. Each harness adapter answers from its own
// record of the session: Claude Code's transcript, or the ledger paws keeps for a harness without a stable one.
type Session interface {
	// SkillLoaded: the calling thread loaded skill successfully within the last window turns, with no compaction since.
	SkillLoaded(skill string, window int) bool
	// DenyDelivered: a paws deny whose reason holds marker reached the calling thread within the last window turns.
	DenyDelivered(marker string, window int) bool
	// EditedMatching: the main thread already edited or wrote a path matching glob, successfully.
	EditedMatching(glob *regexp.Regexp) bool
	// AgentsSinceUserTurn counts the subagents the main thread started since the last prompt the user typed.
	AgentsSinceUserTurn() int
	// Model is the model the calling thread runs on, "" when unknown.
	Model() string
}

// Harness is what the engine needs from the harness a call came from, beyond the call itself: where its skills live,
// and how to tell its agent to load one.
type Harness interface {
	// SkillDirs are the folders this harness loads skills from for cwd: the user's, then the project's.
	SkillDirs(cwd string) []string
	// LoadSkills ends a require-skill deny: how to load skills (each with the SKILL.md paws found, or ""), then retry
	// the call, named by noun ("command", "edit", "tool call").
	LoadSkills(skills []Skill, noun string) string
	// LoadAtStart ends the session-start context of a rule that needs skill.
	LoadAtStart(skill Skill) string
}

// Skill is a skill a deny or a session-start context names, and its SKILL.md in the harness's skill folders ("" if
// none is there: a plugin's skill, say).
type Skill struct {
	Name string
	File string
}

// TranscriptLog is the Session read from Claude Code's session log at transcript, and own, a subagent's own log when
// the call comes from one ("" for the main thread).
func TranscriptLog(transcript, own string) Session {
	return transcriptLog{transcript: transcript, own: own}
}

type transcriptLog struct{ transcript, own string }

// ownLog is the log of the thread making the call: where its own skill loads and the denies it was shown are.
// Session-wide conditions (agent-count, edits) read the transcript.
func (t transcriptLog) ownLog() string {
	if t.own != "" {
		return t.own
	}
	return t.transcript
}

func (t transcriptLog) SkillLoaded(skill string, window int) bool {
	return SkillLoaded(t.ownLog(), skill, window)
}

func (t transcriptLog) DenyDelivered(marker string, window int) bool {
	return DenyDelivered(t.ownLog(), marker, window)
}

func (t transcriptLog) EditedMatching(glob *regexp.Regexp) bool {
	return editedMatching(t.transcript, glob)
}
func (t transcriptLog) AgentsSinceUserTurn() int { return agentsSinceUserTurn(t.transcript) }
func (t transcriptLog) Model() string            { return transcriptModel(t.ownLog()) }

// SkillTool is the canonical harness: skills in Claude Code's folders, loaded with the Skill tool. An Event with no
// Harness uses it.
var SkillTool Harness = skillTool{}

type skillTool struct{}

func (skillTool) SkillDirs(cwd string) []string {
	return []string{pack.UserTarget(UserDir()).SkillsDir, pack.ProjectTarget(repoRoot(cwd)).SkillsDir}
}

func (skillTool) LoadSkills(skills []Skill, noun string) string {
	pronoun := "it"
	if len(skills) > 1 {
		pronoun = "them"
	}
	return fmt.Sprintf("Load %s with the Skill tool, then retry this exact %s.", pronoun, noun)
}

func (skillTool) LoadAtStart(Skill) string {
	return "Load it with the Skill tool before you start work."
}

// session is the Session the event's harness supplied, else Claude Code's transcript (events built without an
// adapter: tests, a hand-made event).
func (ev Event) session() Session {
	if ev.Log != nil {
		return ev.Log
	}
	return TranscriptLog(ev.Transcript, ev.Own)
}

func (ev Event) harness() Harness {
	if ev.Harness != nil {
		return ev.Harness
	}
	return SkillTool
}

// skillFile is skill's SKILL.md in the event's harness's skill folders, or "".
func (ev Event) skillFile(skill string) string {
	for _, dir := range ev.harness().SkillDirs(ev.Cwd) {
		p := filepath.Join(dir, skill, "SKILL.md")
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

// SkillRead names the skill a call loads by reading its SKILL.md: the Read tool on <dir>/skills/<skill>/SKILL.md,
// or a Bash line that does nothing but print that file (see rule.PlainRead). The file must exist; a relative path
// is taken from cwd. "" for any other call.
func SkillRead(tool, target, cwd string) string {
	var p string
	switch tool {
	case "Read":
		p = target
	case "Bash":
		p, _ = rule.PlainRead(target)
	}
	if p == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		p = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(p) && cwd != "" {
		p = filepath.Join(cwd, p)
	}
	p = filepath.Clean(p)
	dir := filepath.Dir(p)
	skill := filepath.Base(dir)
	if filepath.Base(p) != "SKILL.md" || filepath.Base(filepath.Dir(dir)) != "skills" || skill == "skills" {
		return ""
	}
	if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
		return ""
	}
	return skill
}
