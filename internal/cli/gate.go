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
	cwd := in.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	res := gate.Collect(cwd)
	out := hook.Render(in.HookEventName, gate.Decide(hook.ToEvent(&in, data), res.Rules))
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
