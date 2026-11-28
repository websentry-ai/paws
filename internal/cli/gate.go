// Package cli implements the paws subcommands, wiring stdin/stdout/settings to the gate engine.
package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/codex"
	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/hook"
)

// adapters are the harnesses paws runs in; the last is the fallback for a hook call no other one claims.
var adapters = []harness.Adapter{codex.Adapter{}, hook.Claude{}}

// A SKILL.md read loads a skill only from a folder some harness loads skills from.
func init() {
	gate.SkillRoots = func(cwd string) (dirs, trees []string) {
		for _, a := range adapters {
			dirs = append(dirs, a.SkillDirs(cwd)...)
			trees = append(trees, a.SkillTrees()...)
		}
		return dirs, trees
	}
}

// Gate is the hook entrypoint for a hook command that names no harness.
func Gate() int { return GateFor(nil) }

// GateFor is `paws gate [--agent <harness>]`, the hook entrypoint: fail-open by contract — any error or panic
// prints "{}" and exits 0.
func GateFor(args []string) (code int) {
	wrote := false
	defer func() {
		if recover() != nil {
			if !wrote {
				os.Stdout.WriteString("{}")
			}
			code = 0
		}
	}()

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return allow()
	}
	a := harness.Pick(adapters, agentArg(args), data)
	h, err := a.Parse(data)
	if err != nil || len(h.Calls) == 0 {
		return allow()
	}
	ev := h.Calls[0]
	// Telemetry-only events: count and leave. No rules run on a prompt, after a tool or at turn end.
	switch h.Event {
	case "Stop":
		stoppedAt := time.Now()
		gate.LogTelemetry("turn_end", ev, nil)
		a.Observe(h, gate.Decision{})
		allow()
		wrote = true                                  // a panic below must not print a second decision
		gradeTurnIn(a.Turns(h), ev, h.Cwd, stoppedAt) // what followed each block this turn; before the sync, so it goes up with it
		syncInHook(h.Cwd)                             // the async UserPromptSubmit and Stop hooks are the only hooks that use the network
		return 0
	case "UserPromptSubmit":
		// Async, so the agent starts on the prompt without waiting. The prompt itself is never read or kept;
		// a config change this round pulls applies from the next tool call.
		a.Observe(h, gate.Decision{})
		allow()
		wrote = true
		syncInHook(h.Cwd)
		return 0
	case "PostToolUse":
		// A failed call (a Codex command that exited non-zero, a patch that didn't apply) logs nothing.
		if h.Ran == nil || slices.Contains(h.Ran, true) {
			gate.LogTelemetry("tool_result", ev, map[string]any{"tool": ev.Tool})
		}
		for i, c := range h.Calls {
			if h.Ran == nil || h.Ran[i] {
				gate.LogAction(c)
			}
		}
		a.Observe(h, gate.Decision{})
		return allow()
	case "PreToolUse":
		if ev.SkillLoad != "" && ev.SkillWhole { // a chunk of a SKILL.md loads nothing alone: not an invocation
			skills := ev.SkillLoads
			if len(skills) == 0 {
				skills = []string{ev.SkillLoad}
			}
			for _, s := range skills {
				gate.LogTelemetry("skill_invoked", ev, map[string]any{"skill": s})
			}
		}
	}
	res := gate.Collect(h.Cwd)
	d := gate.DecideAll(h.Calls, res.Rules)
	var problems []string
	if n := len(res.Errs); n > 0 {
		problems = append(problems, fmt.Sprintf("%d rule file(s) failed to parse", n))
	}
	if n := len(res.Warns); n > 0 {
		problems = append(problems, fmt.Sprintf("%d rule file(s) have warnings (unknown keys or expect checks)", n))
	}
	notice := ""
	if len(problems) > 0 {
		notice = "paws: " + strings.Join(problems, ", ") + " (run `unbound-paws explain` for details)"
	}
	if h.Event == "SessionStart" {
		// A portal skill paws left out because the user has their own: say so where the user looks, once a session.
		for _, c := range loadPortalState().Clashes {
			notice = strings.TrimPrefix(notice+"\n", "\n") + "paws: " + c
		}
	}
	a.Observe(h, d)
	out := a.Render(h, d, notice)
	wrote = true
	os.Stdout.Write(out)
	if h.Event == "SessionStart" {
		syncAtSessionStart(h.Cwd, h.Source) // spawns a background sync; the gate itself stays offline
	}
	return 0
}

// agentArg is the harness `paws gate --agent <name>` names, or "".
func agentArg(args []string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--agent" {
			return args[i+1]
		}
	}
	return ""
}

func allow() int {
	os.Stdout.WriteString("{}")
	return 0
}
