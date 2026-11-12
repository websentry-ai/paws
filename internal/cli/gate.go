// Package cli implements the paws subcommands, wiring stdin/stdout/settings to the gate engine.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/hook"
)

// Gate is the hook entrypoint: fail-open by contract — any error or panic prints "{}" and exits 0.
func Gate() (code int) {
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
	var in hook.Input
	if err := json.Unmarshal(data, &in); err != nil {
		return allow()
	}
	if in.CWD == "" {
		in.CWD, _ = os.Getwd()
	}
	ev := hook.ToEvent(&in)
	// Telemetry-only events: count and leave. No rules run on a prompt, after a tool or at turn end.
	switch in.HookEventName {
	case "Stop":
		gate.LogTelemetry("turn_end", ev, nil)
		allow()
		wrote = true       // a panic below must not print a second decision
		syncInHook(in.CWD) // the async UserPromptSubmit and Stop hooks are the only hooks that use the network
		return 0
	case "UserPromptSubmit":
		// Async, so Claude starts on the prompt without waiting. The prompt itself is never read or kept;
		// a config change this round pulls applies from the next tool call.
		allow()
		wrote = true
		syncInHook(in.CWD)
		return 0
	case "PostToolUse":
		gate.LogTelemetry("tool_result", ev, map[string]any{"tool": in.ToolName})
		gate.LogAction(ev)
		return allow()
	case "PreToolUse":
		if in.ToolName == "Skill" {
			if s, _ := in.ToolInput["skill"].(string); s != "" {
				gate.LogTelemetry("skill_invoked", ev, map[string]any{"skill": s})
			}
		}
	}
	cwd := in.CWD
	res := gate.Collect(cwd)
	out := hook.Render(in.HookEventName, gate.Decide(ev, res.Rules))
	var problems []string
	if n := len(res.Errs); n > 0 {
		problems = append(problems, fmt.Sprintf("%d rule file(s) failed to parse", n))
	}
	if n := len(res.Warns); n > 0 {
		problems = append(problems, fmt.Sprintf("%d rule file(s) have unknown keys", n))
	}
	if len(problems) > 0 {
		out.SystemMessage = "paws: " + strings.Join(problems, ", ") + " (run `unbound-paws explain` for details)"
	}
	wrote = true
	emit(out)
	if in.HookEventName == "SessionStart" {
		syncAtSessionStart(cwd, in.Source) // spawns a background sync; the gate itself stays offline
	}
	return 0
}

func emit(out hook.Output) {
	b, err := json.Marshal(out)
	if err != nil {
		os.Stdout.WriteString("{}")
		return
	}
	os.Stdout.Write(b)
}

func allow() int {
	os.Stdout.WriteString("{}")
	return 0
}
