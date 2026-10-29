// Package cli implements the paws subcommands, wiring stdin/stdout/settings to the gate engine.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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
	// Telemetry-only events: count and leave. No rules run after a tool or at turn end.
	switch in.HookEventName {
	case "Stop":
		gate.LogTelemetry("turn_end", ev, nil)
		return allow()
	case "PostToolUse":
		gate.LogTelemetry("tool_result", ev, map[string]any{"tool": in.ToolName})
		return allow()
	case "UserPromptSubmit":
		// Retired event: a settings entry from an older build may still call us until `paws init` reruns.
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
	if n := len(res.Errs); n > 0 {
		out.SystemMessage = fmt.Sprintf("paws: %d rule file(s) failed to parse (run `paws explain` for details)", n)
	}
	wrote = true
	emit(out)
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
