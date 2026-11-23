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
// Bash command, what a Write writes, an Edit's new text, an MCP call's text arguments), commands (for Bash, each
// command the line runs: see Commands), input (the raw tool input, any field by path: call.input.parent) and, for
// MCP calls, mcp.server, mcp.tool and mcp.class.
//
// A check can instead have cases, one per way of doing the thing (git, the gh CLI, an MCP tool): the first case
// whose `match` holds on the retry (on the blocked call when there is none) gives the `if`. When no case matches,
// the check doesn't apply to this block and is left out of its grade, rather than passing.
//
//   - id: described
//     when: retry
//     cases:
//   - match: call.tool == "Bash" && any(call.commands, # matches "^gh pr create")
//     if: call.text matches "--body"
//   - match: call.tool endsWith "create_pull_request"
//     if: len(trim(call.input.body ?? "")) >= 20
//
// `when` says which calls the check looks at, all within the turn and the blocked agent:
//
//	before  a call between the block and the retry, the retry included, passes
//	retry   the retried call (the next call the rule matches and lets through) passes
//	after   a call after the retry passes
//	never   no call after the block passes (the check holds when nothing matches)
type Expect struct {
	ID    string       `yaml:"id" json:"id"`
	Label string       `yaml:"label" json:"label"`
	When  string       `yaml:"when" json:"when"`
	If    string       `yaml:"if,omitempty" json:"if,omitempty"`
	Cases []ExpectCase `yaml:"cases,omitempty" json:"cases,omitempty"`

	prog *vm.Program
}

// ExpectCase is one way a check can be met: its `if` applies to calls its `match` holds on.
type ExpectCase struct {
	Match string `yaml:"match" json:"match"`
	If    string `yaml:"if" json:"if"`

	match, prog *vm.Program
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
	if (e.If == "") == (len(e.Cases) == 0) {
		return fmt.Errorf("expect %s: needs an if, or cases", e.ID)
	}
	if e.If != "" {
		prog, err := compileExpr(e.If)
		if err != nil {
			return fmt.Errorf("expect %s: bad if: %v", e.ID, err)
		}
		e.prog = prog
		return nil
	}
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Match == "" || c.If == "" {
			return fmt.Errorf("expect %s: case %d needs a match and an if", e.ID, i+1)
		}
		var err error
		if c.match, err = compileExpr(c.Match); err != nil {
			return fmt.Errorf("expect %s: case %d: bad match: %v", e.ID, i+1, err)
		}
		if c.prog, err = compileExpr(c.If); err != nil {
			return fmt.Errorf("expect %s: case %d: bad if: %v", e.ID, i+1, err)
		}
	}
	return nil
}

func compileExpr(src string) (*vm.Program, error) {
	return expr.Compile(src, expr.Env(expectEnv), expr.AsBool(), expr.AllowUndefinedVariables())
}

// For is the check as it applies to a block whose retry (or, with none, blocked call) is subject: the check itself,
// or for a check with cases, the first case whose match holds. ok is false when no case matches: the check doesn't
// apply. A match that fails to run doesn't hold.
func (e *Expect) For(subject, blocked map[string]any) (_ *Expect, ok bool) {
	if e.prog == nil && (len(e.Cases) == 0 || e.Cases[0].prog == nil) {
		if e.compile() != nil {
			return e, true // runs as a fail, as before
		}
	}
	if len(e.Cases) == 0 {
		return e, true
	}
	env := map[string]any{"call": subject, "blocked": blocked}
	for _, c := range e.Cases {
		if out, err := expr.Run(c.match, env); err == nil && out == true {
			return &Expect{ID: e.ID, Label: e.Label, When: e.When, If: c.If, prog: c.prog}, true
		}
	}
	return nil, false
}

// Eval runs the check's `if` on a call; a runtime error (a missing field compared as a number, say) is a fail.
func (e *Expect) Eval(call, blocked map[string]any) bool {
	ok, err := e.run(call, blocked)
	return err == nil && ok
}

// Hit is Eval for a `never` check: a runtime error counts as a hit, since it can't show the call didn't happen.
// Guard fields with the tool first (`call.tool == "Bash" && ...`): && stops before the field is read.
func (e *Expect) Hit(call, blocked map[string]any) bool {
	ok, err := e.run(call, blocked)
	return err != nil || ok
}

func (e *Expect) run(call, blocked map[string]any) (bool, error) {
	if e.prog == nil {
		if err := e.compile(); err != nil {
			return false, err
		}
		if e.prog == nil {
			return false, fmt.Errorf("expect %s: has cases; run the one For picks", e.ID)
		}
	}
	out, err := expr.Run(e.prog, map[string]any{"call": call, "blocked": blocked})
	ok, isBool := out.(bool)
	if err == nil && !isBool {
		err = fmt.Errorf("expect %s: not a bool", e.ID)
	}
	return ok, err
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
