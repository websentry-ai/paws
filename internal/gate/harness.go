package gate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

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
	// the call, named by noun ("command", "edit", "tool call"), and then Reassurance.
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
// the call comes from one ("" for the main thread). cwd is the session's working directory: a relative SKILL.md read
// in a record that names no cwd (a request blob) is taken from it.
func TranscriptLog(transcript, own, cwd string) Session {
	return transcriptLog{transcript: transcript, own: own, cwd: cwd}
}

type transcriptLog struct{ transcript, own, cwd string }

// ownLog is the log of the thread making the call: where its own skill loads and the denies it was shown are.
// Session-wide conditions (agent-count, edits) read the transcript.
func (t transcriptLog) ownLog() string {
	if t.own != "" {
		return t.own
	}
	return t.transcript
}

func (t transcriptLog) SkillLoaded(skill string, window int) bool {
	return skillLoaded(t.ownLog(), skill, window, t.cwd)
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
	return fmt.Sprintf("Load %s with the Skill tool, then retry this exact %s. %s.", pronoun, noun, Reassurance)
}

// Reassurance ends a require-skill deny, after its instruction: the harness shows a deny as an error.
const Reassurance = "This is paws working as intended, nothing is broken"

func (skillTool) LoadAtStart(Skill) string {
	return "Load it with the Skill tool before you start work."
}

// session is the Session the event's harness supplied, else Claude Code's transcript (events built without an
// adapter: tests, a hand-made event).
func (ev Event) session() Session {
	if ev.Log != nil {
		return ev.Log
	}
	return TranscriptLog(ev.Transcript, ev.Own, ev.Cwd)
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

// SkillRoots are where a SKILL.md read counts as a skill load, for cwd: dirs are skills folders (the file is
// <dir>/<skill>/SKILL.md), trees are folders where any skills/<skill>/SKILL.md below counts (Claude Code's plugins).
// The CLI sets it from every harness it knows; the default is Claude Code's folders.
var SkillRoots = func(cwd string) (dirs, trees []string) {
	dirs = SkillTool.SkillDirs(cwd)
	return dirs, []string{filepath.Join(filepath.Dir(dirs[0]), "plugins")}
}

// SkillRead names the skill a call loads by reading its SKILL.md: the Read tool on <dir>/skills/<skill>/SKILL.md,
// or a Bash line that does nothing but print that file (see rule.PlainRead). The file must exist and sit under one
// of the SkillRoots, so a SKILL.md the agent wrote somewhere else loads nothing. A relative path is taken from cwd,
// or from the hook's own working directory (the session's) when the record names none. "" for any other call.
// Such a read is never gated, whatever part of the file it prints; the skill counts as loaded once successful
// reads cover the whole file (Covered).
func SkillRead(tool, target, cwd string) string {
	r, _ := ReadOf(tool, target, nil, cwd, true)
	return r.Skill
}

// PastSkillRead is SkillRead for a call already made: the file may be gone since, and the read still put the skill
// in the agent's context.
func PastSkillRead(tool, target, cwd string) string {
	r, _ := ReadOf(tool, target, nil, cwd, false)
	return r.Skill
}

// FileRead is a read of a skill's SKILL.md: the skill, the file, the lines the read printed, and when (zero if
// unknown).
type FileRead struct {
	Skill string
	File  string
	Lines rule.Lines
	At    time.Time
	// Cut is set when the harness showed the agent only the first Shown lines of the read's output (too long).
	Cut   bool
	Shown int
}

// cutMark is how a harness marks output it shortened before the agent saw it, on a line of its own: Codex's
// "…N tokens truncated…", Claude Code's "... [N lines truncated] ...". A skill's text that merely mentions one in a
// sentence isn't a cut.
var cutMark = regexp.MustCompile(`(?m)^[ \t]*(?:…|\.\.\.)?[ \t]*\[?\d+ (?:tokens|lines|chars|characters|bytes) truncated\]?[ \t]*(?:…|\.\.\.)?[ \t]*$`)

// ShownLines reports whether a read's output reached the agent cut short, and how many whole lines from its start
// did. Claude Code puts a <persisted-output> note with a 2KB preview in place of a long output; Codex cuts out the
// middle and marks it.
func ShownLines(output string) (cut bool, shown int) {
	if rest, ok := strings.CutPrefix(output, "<persisted-output>"); ok {
		_, preview, _ := strings.Cut(rest, "Preview (first ")
		_, preview, _ = strings.Cut(preview, "\n")
		return true, strings.Count(preview, "\n") // a line the preview cuts in two doesn't count
	}
	if loc := cutMark.FindStringIndex(output); loc != nil {
		return true, strings.Count(output[:loc[0]], "\n")
	}
	return false, 0
}

// readLimit is how many lines Claude Code's Read prints when the call sets no limit.
const readLimit = 2000

// ReadOf is the SKILL.md read a call makes (see SkillRead), with the lines it prints: a Read's offset and limit from
// input, a Bash reader's range. With exists false the file may be gone since.
func ReadOf(tool, target string, input map[string]any, cwd string, exists bool) (FileRead, bool) {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	var p string
	var lines rule.Lines
	switch tool {
	case "Read":
		p = target
		// Without a limit, a Read prints readLimit lines: Claude Code cuts a longer file there.
		offset, _ := input["offset"].(float64)
		limit, _ := input["limit"].(float64)
		if limit <= 0 {
			limit = readLimit
		}
		lines = rule.Lines{From: max(int(offset), 1)}
		lines.To = lines.From + int(limit) - 1
	case "Bash":
		p, lines, _ = rule.PlainRead(target)
	}
	if p == "" {
		return FileRead{}, false
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return FileRead{}, false
		}
		p = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(p) && cwd != "" {
		p = filepath.Join(cwd, p)
	}
	p = filepath.Clean(p)
	dir := filepath.Dir(p)
	skill := filepath.Base(dir)
	if filepath.Base(p) != "SKILL.md" || filepath.Base(filepath.Dir(dir)) != "skills" || skill == "skills" ||
		strings.ContainsAny(skill, "*?[") { // skills/*/SKILL.md is many files, no one skill

		return FileRead{}, false
	}
	if exists {
		if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
			return FileRead{}, false
		}
	}
	if !underSkillRoot(filepath.Dir(dir), cwd) {
		return FileRead{}, false
	}
	return FileRead{Skill: skill, File: p, Lines: lines}, true
}

// Covered reports whether successful reads together printed every line of one SKILL.md, in any order: 1,240p then
// 241,480p of a 300-line file covers it, 1,240p alone doesn't. A read made before the file last changed printed an
// older version, so it doesn't count. A file gone since counts as covered only by a read of the whole of it.
func Covered(reads []FileRead) bool {
	byFile := map[string][]rule.Lines{}
	for _, r := range reads {
		if r.Cut { // only the first lines of the output reached the agent
			if r.Lines.From < 0 { // tail -n N: its output starts N lines from the end
				n, ok := lineCount(r.File)
				if !ok {
					continue
				}
				r.Lines.From = max(n+r.Lines.From+1, 1)
			}
			if r.Shown < 1 {
				continue
			}
			r.Lines.To = r.Lines.From + r.Shown - 1
		}
		if st, err := os.Stat(r.File); err == nil && !r.At.IsZero() && r.At.Before(st.ModTime()) {
			continue // both times come from this machine's clock
		}
		byFile[r.File] = append(byFile[r.File], r.Lines)
	}
	for file, ranges := range byFile {
		if n, ok := lineCount(file); covers(ranges, n, ok) {
			return true
		}
	}
	return false
}

// covers reports whether ranges hold lines 1 through n. With n unknown (the file is gone), only a read to the end
// of the file from its first line does: a bounded one, a Read's 2000 lines included, may have missed the rest.
func covers(ranges []rule.Lines, n int, known bool) bool {
	if !known {
		return slices.ContainsFunc(ranges, func(l rule.Lines) bool { return l.From == 1 && l.To == 0 })
	}
	type span struct{ from, to int }
	var spans []span
	for _, l := range ranges {
		from, to := l.From, l.To
		if from < 0 {
			from = max(n+from+1, 1) // the last -From lines
		}
		if to == 0 || to > n {
			to = n
		}
		spans = append(spans, span{from, to})
	}
	slices.SortFunc(spans, func(a, b span) int { return a.from - b.from })
	next := 1 // the first line not yet printed
	for _, sp := range spans {
		if sp.from > next {
			break
		}
		next = max(next, sp.to+1)
	}
	return next > n
}

// lineCount is how many lines the file at p has; ok is false when it can't be read.
func lineCount(p string) (int, bool) {
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, false
	}
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n, true
}

// underSkillRoot reports whether skills, the folder holding a skill's folder, is one of the SkillRoots dirs or lies
// under one of its trees, as written or with symlinks resolved.
func underSkillRoot(skills, cwd string) bool {
	dirs, trees := SkillRoots(cwd)
	for _, p := range forms(skills) {
		for _, d := range dirs {
			for _, r := range forms(d) {
				if p == r {
					return true
				}
			}
		}
		for _, t := range trees {
			for _, r := range forms(t) {
				if strings.HasPrefix(p, r+string(filepath.Separator)) {
					return true
				}
			}
		}
	}
	return false
}

// forms is p cleaned, and with its symlinks resolved when that differs (macOS's /var is /private/var).
func forms(p string) []string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil && r != p {
		return []string{p, r}
	}
	return []string{p}
}
