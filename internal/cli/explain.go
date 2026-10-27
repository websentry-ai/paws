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

// Explain is the local dry-run: show which .paws rules are in scope, which match, and the resulting decision.
func Explain(args []string) int {
	gate.DryRun = true
	in := &hook.Input{HookEventName: "PreToolUse", ToolName: "Bash"}

	if len(args) == 1 && args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: read stdin: %v\n", err)
			return 2
		}
		if err := json.Unmarshal(data, in); err != nil {
			fmt.Fprintf(os.Stderr, "paws: bad event json: %v\n", err)
			return 2
		}
	} else {
		var words []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--tool", "--event":
				if i+1 >= len(args) {
					fmt.Fprintf(os.Stderr, "paws: %s needs a value\n", args[i])
					return 2
				}
				if args[i] == "--tool" {
					in.ToolName = args[i+1]
				} else {
					in.HookEventName = hook.EventName(args[i+1])
				}
				i++
			default:
				words = append(words, args[i])
			}
		}
		text := strings.Join(words, " ")
		switch in.HookEventName {
		case "UserPromptSubmit":
			in.Prompt = text
		case "SessionStart":
			in.ToolName, in.Source = "", text
		default:
			in.ToolInput = map[string]any{}
			switch in.ToolName {
			case "Read", "Write", "Edit":
				in.ToolInput["file_path"] = text
			default:
				in.ToolInput["command"] = text
			}
		}
	}

	cwd, _ := os.Getwd()
	if in.CWD == "" {
		in.CWD = cwd
	}
	res := gate.Collect(cwd)
	ev := hook.ToEvent(in, nil)

	fmt.Printf("event : %s\ntool  : %s\ninput : %s\n.paws : %s\n",
		in.HookEventName, in.ToolName, ev.Target, strings.Join(res.Dirs, ", "))
	if len(res.Dirs) == 0 {
		fmt.Println("        (none found — no .paws directory in scope)")
	}
	for _, fe := range res.Errs {
		fmt.Printf("  ! skipped %s: %v\n", fe.Path, fe.Err)
	}

	fmt.Println("\nrules:")
	for _, r := range res.Rules {
		mark := "  "
		if gate.Applies(r, ev) {
			mark = "→ "
		}
		fmt.Printf("  %s%-24s event=%-12s action=%-14s [%s]\n", mark, r.Name, r.Event, r.Action.Type, r.SourcePath)
	}
	if len(res.Rules) == 0 {
		fmt.Println("  (no rules loaded)")
	}

	printDecision(gate.Decide(ev, res.Rules))
	return 0
}

func printDecision(d gate.Decision) {
	fmt.Println("\ndecision:")
	switch d.Verdict {
	case gate.Deny, gate.DenyRetry:
		fmt.Printf("  DENY — %s\n", d.Reason)
	case gate.Ask:
		fmt.Printf("  ASK — %s\n", d.Reason)
	default:
		if d.Context != "" {
			fmt.Printf("  allow + context: %s\n", d.Context)
			return
		}
		fmt.Println("  allow (no matching gate)")
	}
	if d.Context != "" && d.Verdict > gate.Allow {
		fmt.Printf("  context: %s\n", d.Context)
	}
}
