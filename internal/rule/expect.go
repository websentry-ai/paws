package rule

import (
	"fmt"
	"regexp"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Expect is one check of what should follow when the rule blocks a call: did Claude do what the skill asks? It
// runs locally after the turn, on the turn's own tool calls, and only its result (pass or fail) leaves the laptop.
//
//	expect:
//	  - id: listed-first
//	    label: Listed what it would delete first
//	    when: before
//	    if: call.tool == "Bash" && call.text matches "^(ls|du|stat)\\b"
//
// `if` is an expr expression (https://expr-lang.org) over two calls, `call` (the one being checked) and `blocked`
// (the call the rule blocked), each a map: tool (Bash, Write, Edit, Read, mcp__…), text (the call's main text: a
// Bash command, what a Write writes, an Edit's new text, an MCP call's text arguments), input (the raw tool input,
// any field by path: call.input.parent) and, for MCP calls, mcp.server, mcp.tool and mcp.class.
//
// `when` says which calls the check looks at, all within the turn and the blocked agent:
//
//	before  a call between the block and the retry, the retry included, passes
//	retry   the retried call (the next call the rule matches and lets through) passes
//	after   a call after the retry passes
//	never   no call after the block passes (the check holds when nothing matches)
type Expect struct {
	ID    string `yaml:"id" json:"id"`
	Label string `yaml:"label" json:"label"`
	When  string `yaml:"when" json:"when"`
	If    string `yaml:"if" json:"if"`

	prog *vm.Program
}

// The positions an expect check can take.
const (
	ExpectBefore = "before"
	ExpectRetry  = "retry"
	ExpectAfter  = "after"
	ExpectNever  = "never"
)

var expectID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)

// expectEnv is the shape `if` is compiled against; maps, so any input field is reachable.
var expectEnv = map[string]any{"call": map[string]any{}, "blocked": map[string]any{}}

func (e *Expect) compile() error {
	if !expectID.MatchString(e.ID) {
		return fmt.Errorf("expect id %q: want lower-case letters, digits and -", e.ID)
	}
	switch e.When {
	case ExpectBefore, ExpectRetry, ExpectAfter, ExpectNever:
	default:
		return fmt.Errorf("expect %s: unknown when %q (want before, retry, after or never)", e.ID, e.When)
	}
	if e.If == "" {
		return fmt.Errorf("expect %s: needs an if", e.ID)
	}
	prog, err := expr.Compile(e.If, expr.Env(expectEnv), expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return fmt.Errorf("expect %s: bad if: %v", e.ID, err)
	}
	e.prog = prog
	return nil
}

// Eval runs the check's `if` on a call; a runtime error (a missing field compared as a number, say) is a fail.
func (e *Expect) Eval(call, blocked map[string]any) bool {
	if e.prog == nil {
		if e.compile() != nil {
			return false
		}
	}
	out, err := expr.Run(e.prog, map[string]any{"call": call, "blocked": blocked})
	ok, isBool := out.(bool)
	return err == nil && isBool && ok
}

// UsableExpect is the checks that compile, in order, first of each id: a portal action keeps its good checks when
// one can't be read (a newer catalog's syntax, say).
func UsableExpect(checks []Expect) []Expect {
	var out []Expect
	seen := map[string]bool{}
	for _, c := range checks {
		if c.compile() == nil && !seen[c.ID] {
			seen[c.ID] = true
			out = append(out, c)
		}
	}
	return out
}

// validateExpect compiles every check and refuses duplicate ids.
func validateExpect(checks []Expect) error {
	seen := map[string]bool{}
	for i := range checks {
		if err := checks[i].compile(); err != nil {
			return err
		}
		if seen[checks[i].ID] {
			return fmt.Errorf("expect id %q repeats", checks[i].ID)
		}
		seen[checks[i].ID] = true
	}
	return nil
}
