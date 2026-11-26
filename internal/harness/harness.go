// Package harness is the seam between paws and the agent harnesses it runs in (Claude Code, Codex, ...). Each
// harness has one Adapter that turns its native hook stdin, session history and config files into the canonical
// form: Claude Code's hook event and tool names, gate.Event, outcome.Call. The engine (gate, outcomes, audit, sync)
// works on the canonical form only, so a new harness is a new Adapter, not a change to the engine.
package harness

import (
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/history"
	"github.com/websentry-ai/paws/internal/outcome"
)

// Adapter is one harness.
type Adapter interface {
	gate.Harness // where its skills live, and how a deny tells its agent to load one
	Codec
	Installer
	// Name is the harness id stored on every record: "claude-code", "codex".
	Name() string
	// Title is how people know the harness: "Claude Code".
	Title() string
	// History is the harness's past sessions for `paws scan`, and where they are; nil when scan can't read them.
	History() (history.Source, string)
	// SkillTrees are folders where any skills/<skill>/SKILL.md below is one of this harness's skills (Claude Code's
	// plugins), beside its SkillDirs.
	SkillTrees() []string
}

// Codec turns one hook call into the canonical form and the engine's decision back into the harness's reply.
type Codec interface {
	// Claims reports whether stdin came from this harness, for a hook command that doesn't say (no --agent).
	Claims(stdin []byte) bool
	Parse(stdin []byte) (*Hook, error)
	// Render is the hook's stdout for decision d; notice, when set, is shown to the user.
	Render(h *Hook, d gate.Decision, notice string) []byte
	// Observe runs on every hook call once paws has decided, before it replies: an adapter that keeps its own
	// record of the session writes it here.
	Observe(h *Hook, d gate.Decision)
	// Turns reads the session's turns for outcome grading; nil when the hook carries no session to read.
	Turns(h *Hook) TurnLog
}

// Installer is how paws registers its hook with the harness, and where it puts skills for it.
type Installer interface {
	// HookFile is where `paws init` registers the hook: the user's config with global, else the project's at root.
	HookFile(global bool, root string) string
	// HookFiles are every file where the harness may run a paws hook for the project at root.
	HookFiles(root string) []string
	// HookCommand is the hook command for this harness, given the command that runs `paws gate`.
	HookCommand(gate string) string
	// Events are the hook events paws registers, by their canonical names; Entry is one event's hook entry.
	Events() []string
	Entry(event, command string) map[string]any
	// SkillsDirFor is the skills folder the harness reads beside hookFile, or "" when it has none there.
	SkillsDirFor(hookFile string) string
}

// Hook is one hook call in canonical terms.
type Hook struct {
	Event string // the canonical hook event: PreToolUse, PostToolUse, SessionStart, UserPromptSubmit, Stop, PostCompact
	// Calls are what the gate decides on, one per target the call touches (a patch to several files has several);
	// always at least one. Each carries the session, its Log and the harness.
	Calls  []gate.Event
	Cwd    string
	Source string // SessionStart: startup, resume, clear or compact
	Native any    // the adapter's own reading of stdin
}

// TurnLog reads a session's tool calls by turn, for outcome grading.
type TurnLog interface {
	// LastRecordAt is when the turn that is stopping ended, at or before until; ok is false when unknown.
	LastRecordAt(until time.Time) (at time.Time, ok bool)
	// TurnCalls are the calls of the turn that holds until (every agent's, oldest first) and when the turn started.
	TurnCalls(until time.Time) ([]outcome.Call, time.Time, error)
}

// Pick is the adapter for a hook call: the one named (by `paws gate --agent`), else the first that claims stdin,
// else the last, the fallback.
func Pick(adapters []Adapter, name string, stdin []byte) Adapter {
	if a := Find(adapters, name); a != nil {
		return a
	}
	for _, a := range adapters {
		if a.Claims(stdin) {
			return a
		}
	}
	return adapters[len(adapters)-1]
}

// Find is the adapter named name, or nil.
func Find(adapters []Adapter, name string) Adapter {
	for _, a := range adapters {
		if a.Name() == name {
			return a
		}
	}
	return nil
}
