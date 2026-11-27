package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/history"
	"github.com/websentry-ai/paws/internal/outcome"
)

// Claude is the Claude Code adapter (harness.Adapter): settings.json hooks, the session transcript, ~/.claude/skills.
type Claude struct{}

var _ harness.Adapter = Claude{}

func (Claude) Name() string  { return Agent }
func (Claude) Title() string { return "Claude Code" }

// Claims is true for any hook JSON: Claude Code is the fallback when no other adapter claims a call.
func (Claude) Claims([]byte) bool { return true }

func (Claude) Parse(stdin []byte) (*harness.Hook, error) {
	var in Input
	if err := json.Unmarshal(stdin, &in); err != nil {
		return nil, err
	}
	if in.CWD == "" {
		in.CWD, _ = os.Getwd()
	}
	return &harness.Hook{Event: in.HookEventName, Calls: []gate.Event{ToEvent(&in)}, Cwd: in.CWD, Source: in.Source, Native: &in}, nil
}

func (Claude) Render(h *harness.Hook, d gate.Decision, notice string) []byte {
	out := Render(h.Event, d)
	out.SystemMessage = notice
	b, err := json.Marshal(out)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// Observe does nothing: Claude Code's transcript already records the session.
func (Claude) Observe(*harness.Hook, gate.Decision) {}

func (Claude) Turns(h *harness.Hook) harness.TurnLog {
	in, _ := h.Native.(*Input)
	if in == nil || in.TranscriptPath == "" {
		return nil
	}
	return transcriptTurns{path: in.TranscriptPath, last: in.LastAssistantMessage}
}

// transcriptTurns is the TurnLog of Claude Code's transcript at path; last is the final message the Stop names.
type transcriptTurns struct{ path, last string }

// LastRecordAt is when the stopping turn ended: its final message, which the Stop names; without one (an older
// Claude Code), or past until, the log's last record at or before until.
func (t transcriptTurns) LastRecordAt(until time.Time) (time.Time, bool) {
	if at, ok := StopRecordAt(t.path, t.last); ok && !at.After(until) {
		return at, true
	}
	return LastRecordAt(t.path, until)
}

func (t transcriptTurns) TurnCalls(until time.Time) ([]outcome.Call, time.Time, error) {
	return TurnCalls(t.path, until)
}

func (Claude) History() (history.Source, string) {
	dir := filepath.Join(ConfigDir(), "projects")
	return Transcripts{Dir: dir}, dir
}

// SkillDirs are Claude Code's folders, then the ones paws installs Codex's skills into (~/.agents/skills, the repo's
// .agents/skills): Claude Code doesn't list a skill that is only there, but the agent can read its SKILL.md when the
// deny names the file. paws installs only for the agents set up here, so a global skill added while Claude Code ran
// paws in another repo only, or a same-name skill of the user's that paws kept for Codex, may be only there.
func (Claude) SkillDirs(cwd string) []string {
	home, _ := os.UserHomeDir()
	return append(gate.SkillTool.SkillDirs(cwd), filepath.Join(home, ".agents", "skills"), filepath.Join(gate.RepoRoot(cwd), ".agents", "skills"))
}

// SkillTrees: a plugin's skills sit at <config>/plugins/<...>/skills/<skill>/SKILL.md.
func (Claude) SkillTrees() []string { return []string{filepath.Join(ConfigDir(), "plugins")} }

// LoadSkills is the Skill tool sentence. A SKILL.md read loads a skill too (one installed mid-session isn't in the
// Skill tool's list); the deny doesn't name the path, to stay short. When a skill is only in another agent's folder,
// where the Skill tool can't find it, the deny names the files to read instead.
func (Claude) LoadSkills(skills []gate.Skill, noun string) string {
	if !slices.ContainsFunc(skills, outsideClaude) {
		return gate.SkillToolInstruction(skills, noun)
	}
	files := make([]string, len(skills))
	for i, s := range skills {
		files[i] = s.File
		if s.File == "" {
			files[i] = fmt.Sprintf("the %s skill's SKILL.md", s.Name)
		}
	}
	return fmt.Sprintf("Read %s, then %s again.", andList(files), gate.Retry(noun))
}

// outsideClaude reports whether paws found s's SKILL.md in a folder other than Claude Code's own skills folders.
func outsideClaude(s gate.Skill) bool {
	if s.File == "" {
		return false
	}
	dir := filepath.Dir(filepath.Dir(filepath.Dir(s.File))) // <dir>/skills/<skill>/SKILL.md
	return filepath.Base(dir) != ".claude" && filepath.Clean(dir) != filepath.Clean(ConfigDir())
}

func andList(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func (Claude) LoadAtStart(s gate.Skill) string {
	return gate.SkillTool.LoadAtStart(s) + readAlso([]gate.Skill{s})
}

func readAlso(skills []gate.Skill) string {
	var files []string
	for _, s := range skills {
		if s.File != "" {
			files = append(files, s.File)
		}
	}
	switch len(files) {
	case 0:
		return ""
	case 1:
		return " Reading " + files[0] + " in full also works."
	}
	return " Reading " + strings.Join(files[:len(files)-1], ", ") + " and " + files[len(files)-1] + " in full also works."
}

func (Claude) HookFile(global bool, root string) string {
	if global {
		return filepath.Join(ConfigDir(), "settings.json")
	}
	return filepath.Join(root, ".claude", "settings.json")
}

func (Claude) HookFiles(root string) []string {
	return []string{
		filepath.Join(root, ".claude", "settings.json"),
		filepath.Join(root, ".claude", "settings.local.json"),
		filepath.Join(ConfigDir(), "settings.json"),
	}
}

func (Claude) HookCommand(gate string) string { return gate }
func (Claude) InstallNote() string            { return "" }
func (Claude) Events() []string               { return RegisteredEvents }

func (Claude) Entry(event, command string) map[string]any { return SettingsEntry(event, command) }

// SkillsDirFor is the skills folder beside a settings file in a .claude folder or $CLAUDE_CONFIG_DIR; "" for a
// settings file anywhere else (a --settings path, managed settings).
func (Claude) SkillsDirFor(settingsPath string) string {
	dir := filepath.Dir(settingsPath)
	if filepath.Base(dir) != ".claude" && dir != ConfigDir() {
		return ""
	}
	return filepath.Join(dir, "skills")
}

// ConfigDir is Claude Code's user config dir: $CLAUDE_CONFIG_DIR when set, else ~/.claude.
func ConfigDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}
