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

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: unbound-paws explain [--tool T] [--event E] <command|path|url|...>\n       echo '<event json>' | unbound-paws explain -")
		return 2
	}
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
				} else if in.HookEventName = hook.EventName(args[i+1]); in.HookEventName != "PreToolUse" && in.HookEventName != "SessionStart" {
					fmt.Fprintf(os.Stderr, "paws: unknown event %q (want pre-tool, pre-commit, pre-push or session-start)\n", args[i+1])
					return 2
				}
				i++
			default:
				words = append(words, args[i])
			}
		}
		text := strings.Join(words, " ")
		switch in.HookEventName {
		case "SessionStart":
			in.ToolName, in.Source = "", text
		default:
			in.ToolInput = map[string]any{}
			if field := hook.InputField(in.ToolName); field != "" {
				in.ToolInput[field] = text
			} else if json.Unmarshal([]byte(text), &in.ToolInput) != nil {
				// An MCP tool matches on its whole input: pass it as JSON, or this text stands in for it.
				in.ToolInput = map[string]any{"input": text}
			}
		}
	}

	cwd, _ := os.Getwd()
	if in.CWD == "" {
		in.CWD = cwd
	}
	res := gate.Collect(cwd)
	ev := hook.ToEvent(in)

	fmt.Printf("event : %s\ntool  : %s\ninput : %s\n.paws : %s\n",
		in.HookEventName, in.ToolName, ev.Target, strings.Join(tildeAll(res.Dirs), ", "))
	if len(res.Dirs) == 0 {
		fmt.Println("        (none found — no .paws directory in scope)")
	}
	for _, fe := range res.Errs {
		fmt.Printf("  ! skipped %s: %v\n", fe.Path, fe.Err)
	}
	for _, fe := range res.Warns {
		fmt.Printf("  ! unknown key in %s (ignored): %s\n", fe.Path, oneLine(fe.Err))
	}
	seen := map[string]string{}
	for _, r := range res.Rules {
		if first, ok := seen[r.Name]; ok {
			fmt.Printf("  ! two rules named %q (%s, %s): both apply, but the audit log and fire counts can't tell them apart\n", r.Name, first, r.SourcePath)
		} else {
			seen[r.Name] = r.SourcePath
		}
	}

	fmt.Println("\nrules:")
	for _, r := range res.Rules {
		mark := "  "
		if gate.Applies(r, ev) {
			mark = "→ "
		}
		fmt.Printf("  %s%-24s event=%-12s action=%-14s %s\n", mark, r.Name, r.Event, r.Action.Type, tilde(r.SourcePath))
		if mark == "→ " {
			if p := gate.DigestProblem(r); p != "" {
				fmt.Printf("      ! digest not used, the deny names the skill instead: %s\n", p)
			}
		}
	}
	if len(res.Rules) == 0 {
		fmt.Println("  (no rules loaded)")
	}

	printDecision(gate.Decide(ev, res.Rules))
	return 0
}

// oneLine flattens a multi-line YAML error ("unmarshal errors:\n  line 6: field x not found").
func oneLine(err error) string {
	return strings.Join(strings.Fields(err.Error()), " ")
}

func printDecision(d gate.Decision) {
	fmt.Println("\ndecision:")
	switch d.Verdict {
	case gate.Deny:
		fmt.Printf("  DENY — %s\n", d.Reason)
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
