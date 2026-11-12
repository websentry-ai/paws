package rule

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/syntax"
)

// A Bash command line is split into the simple commands the shell would run, and a rule's command_regex is also
// matched against each one. So `bash -c "git commit"`, `sudo -iu x git push`, `xargs git commit` and `cat <<EOF |
// bash` are each seen as the command they run. Matching only adds: what the whole-line regexes matched before still
// matches. A line, or a script it runs, that doesn't parse adds nothing.

// command is one simple command: its words with quotes removed, as the program receives them.
type command struct {
	words []string
	// texts are what a command_regex sees: the words joined by spaces, as written and (when it differs) without
	// wrappers such as sudo or env; ; & | and newlines inside an argument become spaces, so they can't start a command.
	texts []string
}

var shells = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true, "ash": true, "mksh": true}

const maxNesting = 8 // bash -c inside eval inside bash -c ...

// limits bound the work one line can cause, however it nests, so no line can make the hook slow: how many function
// bodies it may expand, and how many bytes of nested script (bash -c, heredocs, what printf prints...) it may parse.
type limits struct {
	funcWalks, scriptBytes int
}

const (
	maxFuncWalks   = 64
	maxScriptBytes = 256 << 10
)

// parsedLine is the commands the parser found in a Bash line. A line, or a script it runs, that isn't valid Bash
// adds none: the whole-line regexes still decide it.
type parsedLine struct {
	cmds []command
}

// splitCache holds the last line read: the gate checks every rule against the same line.
var splitCache struct {
	sync.Mutex
	line   string
	parsed *parsedLine
}

func parseLine(line string) *parsedLine {
	splitCache.Lock()
	defer splitCache.Unlock()
	if splitCache.parsed != nil && splitCache.line == line {
		return splitCache.parsed
	}
	p := &parsedLine{cmds: []command{}}
	split(line, &p.cmds, 0, &limits{funcWalks: maxFuncWalks, scriptBytes: maxScriptBytes}, nil, nil)
	splitCache.line, splitCache.parsed = line, p
	return p
}

// split adds line's commands to out. sc is the shell's function scope: eval runs in the same shell and shares it; a
// new shell (bash -c) starts with none (nil).
// owner is the splitter of the shell this line runs in when that shell is the caller's (eval, source): the line's
// loops and traps are then replayed when that shell's whole line is done, not when this part of it is.
func split(line string, out *[]command, depth int, lim *limits, sc *scope, owner *splitter) bool {
	if depth > maxNesting {
		return true
	}
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil {
		return false
	}
	if sc == nil {
		sc = &scope{funcs: map[string][]*syntax.Stmt{}}
	}
	s := splitter{out: out, sc: sc, walked: map[walkKey]bool{}, lim: lim, piped: map[*syntax.BinaryCmd]bool{},
		bg: map[*syntax.Stmt]bool{}, seenLoop: map[syntax.Node]bool{}, rewalked: map[syntax.Node]bool{}, owner: owner}
	s.walk(f, depth)
	if owner != nil {
		return true // the enclosing shell replays this line's loops and traps with its own
	}
	// A loop body runs again and again: a call early in it may run a function defined later in it, or after it in an
	// earlier pass. Each loop's body is read once more now that every definition is known (within the line's budget).
	// Each loop is read in the scope it sits in: a loop inside ( ) sees that subshell's functions.
	for i := 0; i < len(s.loops) && s.lim.funcWalks > 0; i++ {
		lp := s.loops[i]
		if s.rewalked[lp.node] {
			continue
		}
		s.rewalked[lp.node] = true
		s.lim.funcWalks--
		outer := s.sc
		s.sc = lp.sc
		for _, st := range loopBody(lp.node) {
			s.walk(st, depth)
		}
		s.sc = outer
	}
	// The shell runs EXIT and other handlers after the line's commands, with every function defined by then. Each is
	// read in its own copy of that scope: which handler runs, and in what order, depends on signals, so one handler's
	// definitions must not hide a call in another.
	for _, t := range s.traps { // in the scope the trap was set in: a trap inside ( ) sees that subshell's functions
		s.nestedIn(t.action, depth, t.sc.child())
	}
	return true
}

// splitter collects the commands of one parsed line. sc holds the functions defined so far, in the order the shell
// runs the line: defining one runs nothing, so a body counts where a call runs it. A body is read once per scope and
// set of definitions, so a later redefinition of a function it calls is read again, and a chain of calls can't blow
// up.
type splitter struct {
	out    *[]command
	sc     *scope
	walked map[walkKey]bool
	lim    *limits                    // what the whole line may still cost
	piped  map[*syntax.BinaryCmd]bool // pipes already read as one pipeline
	bg     map[*syntax.Stmt]bool      // background statements already entered
	traps  []trapAt                   // trap handlers, read again once the line is done
	// loops are the while, until, for and select loops met, read once more once the line is done.
	loops              []loopAt
	seenLoop, rewalked map[syntax.Node]bool
	owner              *splitter // the enclosing shell's splitter, for a line eval or source runs in it
}

// scope is one shell's functions, and how many definitions it has seen (a walked body is keyed on it). A name keeps
// every body defined for it so far: a definition inside if, case or a loop may not run, so a call is read against
// each of them, which only ever adds matches.
type scope struct {
	funcs map[string][]*syntax.Stmt
	defs  int
}

// child is the scope of a subshell: ( ), $( ), <( ). It starts with the parent's functions; its own don't leak back.
func (sc *scope) child() *scope {
	c := &scope{funcs: make(map[string][]*syntax.Stmt, len(sc.funcs)), defs: sc.defs}
	for k, v := range sc.funcs {
		c.funcs[k] = append([]*syntax.Stmt(nil), v...)
	}
	return c
}

type walkKey struct {
	body *syntax.Stmt
	sc   *scope
	defs int
}

// nested splits a script a new shell runs (bash -c, a heredoc, a here-string, a pipe into a shell). Its whole text
// is one more text to match, so a rule spanning commands (`curl x | sh`) sees it too.
func (s *splitter) nested(script string, depth int) { s.nestedIn(script, depth, nil) }

// nestedIn splits a script run with these functions defined; eval passes its own shell's.
// root is the splitter of the shell s runs in: itself, or the enclosing one when s reads an eval or source line.
func (s *splitter) root() *splitter {
	for s.owner != nil {
		s = s.owner
	}
	return s
}

func (s *splitter) nestedIn(script string, depth int, sc *scope) {
	if len(script) > s.lim.scriptBytes {
		return // past the line's budget: the whole-line regexes still apply
	}
	s.lim.scriptBytes -= len(script)
	var owner *splitter
	if sc == s.sc { // eval and source run in this shell: their loops and traps wait for this shell's line
		owner = s.root()
	}
	if split(script, s.out, depth+1, s.lim, sc, owner) {
		*s.out = append(*s.out, command{texts: []string{script}})
	}
}

// trapAt is a trap handler and the scope of the shell that set it.
type trapAt struct {
	action string
	sc     *scope
}

// loopAt is a loop and the scope it runs in.
type loopAt struct {
	node syntax.Node
	sc   *scope
}

// loopBody is the statements a while, until, for or select loop runs each time round, a while or until condition
// included.
func loopBody(n syntax.Node) []*syntax.Stmt {
	switch n := n.(type) {
	case *syntax.WhileClause: // the condition runs again before every pass, too
		return append(append([]*syntax.Stmt{}, n.Cond...), n.Do...)
	case *syntax.ForClause:
		return n.Do
	}
	return nil
}

func subshellStmts(n syntax.Node) []*syntax.Stmt {
	switch n := n.(type) {
	case *syntax.Subshell:
		return n.Stmts
	case *syntax.CmdSubst:
		return n.Stmts
	case *syntax.ProcSubst:
		return n.Stmts
	}
	return nil
}

func (s *splitter) walk(node syntax.Node, depth int) {
	syntax.Walk(node, func(n syntax.Node) bool {
		if st, ok := n.(*syntax.Stmt); ok && st.Background && !s.bg[st] {
			// cmd & runs in a subshell: functions it defines don't reach the line.
			s.bg[st] = true
			outer := s.sc
			s.sc = outer.child()
			s.walk(st, depth)
			s.sc = outer
			return false
		}
		switch n := n.(type) {
		case *syntax.FuncDecl:
			// `f() { ...; } | cat` parses as a function whose body is the pipe; bash runs a pipeline whose first
			// stage defines f in a subshell. The definition stays there, and the rest of the pipe runs.
			if b, ok := n.Body.Cmd.(*syntax.BinaryCmd); ok && (b.Op == syntax.Pipe || b.Op == syntax.PipeAll) {
				outer := s.sc
				s.sc = outer.child()
				s.walk(b.Y, depth)
				s.sc = outer
				return false
			}
			s.sc.funcs[n.Name.Value] = append(s.sc.funcs[n.Name.Value], n.Body)
			s.sc.defs++
			return false
		case *syntax.Subshell, *syntax.CmdSubst, *syntax.ProcSubst:
			// ( ), $( ) and <( ) run in a copy of the shell: functions they define don't reach the line.
			outer := s.sc
			s.sc = outer.child()
			for _, st := range subshellStmts(n) {
				s.walk(st, depth)
			}
			s.sc = outer
			return false
		case *syntax.CoprocClause: // coproc runs its command in a subshell too
			outer := s.sc
			s.sc = outer.child()
			s.walk(n.Stmt, depth)
			s.sc = outer
			return false
		case *syntax.WhileClause, *syntax.ForClause:
			if r := s.root(); !r.seenLoop[n] {
				r.seenLoop[n] = true
				r.loops = append(r.loops, loopAt{n, s.sc})
			}
		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe || n.Op == syntax.PipeAll {
				// Each stage of a pipeline runs in a subshell: functions it defines don't reach the line.
				s.pipedScript(n, depth)
				for _, stage := range []*syntax.Stmt{n.X, n.Y} {
					outer := s.sc
					s.sc = outer.child()
					s.walk(stage, depth)
					s.sc = outer
				}
				return false
			}
		case *syntax.Stmt:
			s.stmt(n, depth)
		}
		return true
	})
}

func (s *splitter) stmt(stmt *syntax.Stmt, depth int) {
	// Redirections stay on the command's text ("python - <<'PY'", "cmd >log"): some triggers name them.
	var redirs []string
	for _, r := range stmt.Redirs {
		if r.Word != nil {
			redirs = append(redirs, r.Op.String()+wordString(r.Word))
			// echo 'git push' > >(bash): the shell in >( ) runs what the command prints
			if (r.Op == syntax.RdrOut || r.Op == syntax.AppOut || r.Op == syntax.ClbOut) && readsShellStdinAll(procSubst(r.Word, syntax.CmdOut)) {
				s.printed(stmt, depth)
			}
		}
	}
	switch cmd := stmt.Cmd.(type) {
	case *syntax.DeclClause: // export X=1, declare, local, readonly
		words := []string{cmd.Variant.Value}
		for _, a := range cmd.Args {
			words = append(words, source(a))
		}
		*s.out = append(*s.out, command{words: words, texts: []string{commandText(words, redirs)}})
	case *syntax.CallExpr:
		var assigns []string
		for _, a := range cmd.Assigns {
			assigns = append(assigns, source(a))
		}
		if len(cmd.Args) == 0 {
			if len(assigns) > 0 {
				*s.out = append(*s.out, command{texts: []string{strings.Join(assigns, " ")}})
			}
			return
		}
		words := make([]string, len(cmd.Args))
		for i, w := range cmd.Args {
			words[i] = wordString(w)
		}
		bare := stripWrappers(words)
		if len(bare) > 0 {
			switch head := path.Base(bare[0]); {
			case shells[head]: // bash -c "script", bash <<EOF script EOF
				if script, ok := shellScript(bare[1:]); ok {
					s.nested(script, depth)
				} else if readsStdin(bare[1:]) {
					for _, r := range stmt.Redirs {
						switch {
						case (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil:
							s.nested(hdocString(r), depth)
						case r.Op == syntax.WordHdoc && r.Word != nil: // bash <<< 'git push'
							s.nested(wordString(r.Word), depth)
						case r.Op == syntax.RdrIn && r.Word != nil: // bash < <(echo 'git push')
							s.printedAll(procSubst(r.Word, syntax.CmdIn), depth)
						}
					}
				} else {
					s.procScripts(cmd.Args[1:], depth, nil) // bash <(echo 'git push'): a new shell
				}
			case head == "source" || head == ".": // . <(echo 'git push') runs in this shell, with its functions
				s.procScripts(cmd.Args[1:], depth, s.sc)
			case head == "eval":
				args := bare[1:]
				if len(args) > 0 && args[0] == "--" {
					args = args[1:]
				}
				s.nestedIn(strings.Join(args, " "), depth, s.sc) // eval runs in this shell
			case head == "trap": // trap 'git push' EXIT runs its action later, in this shell
				if action, ok := trapAction(bare[1:]); ok {
					// Read now in a copy of the scope (a signal may come at any time; what the handler defines
					// stays out of the line), and again once the line is done, with every function defined by then.
					s.nestedIn(action, depth, s.sc.child())
					r := s.root()
					r.traps = append(r.traps, trapAt{action, s.sc})
				}
			case len(s.sc.funcs[bare[0]]) > 0 && depth < maxNesting:
				for _, body := range s.sc.funcs[bare[0]] {
					key := walkKey{body, s.sc, s.sc.defs}
					if s.walked[key] || s.lim.funcWalks <= 0 {
						continue
					}
					s.walked[key] = true
					s.lim.funcWalks--
					s.walk(body, depth+1)
				}
			}
		}
		text := commandText(words, redirs)
		c := command{words: bare, texts: []string{text}}
		if len(bare) > 0 && !slices.Equal(bare, words) { // env -S 'git push x' keeps its word count
			c.texts = append(c.texts, commandText(bare, redirs))
		}
		if len(assigns) > 0 { // HUSKY=0 git commit: a trigger may name the variable
			c.texts = append(c.texts, strings.Join(assigns, " ")+" "+text)
		}
		*s.out = append(*s.out, c)
	}
}

// pipedScript reads scripts piped into a shell: `cat <<EOF | bash`, `echo "git push" | sh | tee log`. Each stage of
// the pipeline that is a shell reading stdin runs what the stage before it prints.
func (s *splitter) pipedScript(pipe *syntax.BinaryCmd, depth int) {
	if s.piped[pipe] {
		return
	}
	var stages []*syntax.Stmt
	var flatten func(st *syntax.Stmt)
	flatten = func(st *syntax.Stmt) {
		if b, ok := st.Cmd.(*syntax.BinaryCmd); ok && (b.Op == syntax.Pipe || b.Op == syntax.PipeAll) && len(st.Redirs) == 0 {
			s.piped[b] = true
			flatten(b.X)
			flatten(b.Y)
			return
		}
		stages = append(stages, st)
	}
	s.piped[pipe] = true
	flatten(pipe.X)
	flatten(pipe.Y)
	for i := 1; i < len(stages); i++ {
		if readsShellStdin(stages[i]) {
			s.printed(stages[i-1], depth)
		}
	}
}

// readsShellStdin reports whether st is a shell (behind any wrapper) that runs the script on its stdin.
func readsShellStdin(st *syntax.Stmt) bool {
	sh, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(sh.Args) == 0 {
		return false
	}
	words := make([]string, len(sh.Args))
	for i, w := range sh.Args {
		words[i] = wordString(w)
	}
	bare := stripWrappers(words)
	return len(bare) > 0 && shells[path.Base(bare[0])] && readsStdin(bare[1:])
}

// printed splits what st prints, when that can be read off the line.
func (s *splitter) printed(st *syntax.Stmt, depth int) {
	if out, ok := printedBy(st); ok {
		s.nested(out, depth)
	}
}

// printedAll splits what a process substitution prints, when that can be read off the line.
func (s *splitter) printedAll(ps *syntax.ProcSubst, depth int) { s.printedAllIn(ps, depth, nil) }

// printedAllIn is printedAll for a script run with these functions defined: source runs in this shell (s.sc).
func (s *splitter) printedAllIn(ps *syntax.ProcSubst, depth int, sc *scope) {
	if ps == nil {
		return
	}
	if out, ok := printedByAll(ps.Stmts); ok {
		s.nestedIn(out, depth, sc)
	}
}

// procScripts splits the scripts a shell or source reads from <( ): bash <(echo 'git push').
func (s *splitter) procScripts(args []*syntax.Word, depth int, sc *scope) {
	for _, w := range args {
		s.printedAllIn(procSubst(w, syntax.CmdIn), depth, sc)
	}
}

// procSubst is w when it is just a process substitution of this kind, else nil.
func procSubst(w *syntax.Word, op syntax.ProcOperator) *syntax.ProcSubst {
	if w != nil && len(w.Parts) == 1 {
		if ps, ok := w.Parts[0].(*syntax.ProcSubst); ok && ps.Op == op {
			return ps
		}
	}
	return nil
}

// readsShellStdinAll reports whether >( ) runs a shell that reads its stdin as a script.
func readsShellStdinAll(ps *syntax.ProcSubst) bool {
	return ps != nil && len(ps.Stmts) == 1 && readsShellStdin(ps.Stmts[0])
}

// printedBy is what st prints, when the line shows it: echo or printf (behind any wrapper), a heredoc or here-string
// it reads (cat <<EOF), or the output of the commands in a { } or ( ) group.
func printedBy(st *syntax.Stmt) (string, bool) {
	switch cmd := st.Cmd.(type) {
	case *syntax.CallExpr:
		if out, ok := echoed(cmd); ok {
			return out, true
		}
	case *syntax.Block:
		if out, ok := printedByAll(cmd.Stmts); ok {
			return out, true
		}
	case *syntax.Subshell:
		if out, ok := printedByAll(cmd.Stmts); ok {
			return out, true
		}
	}
	var b strings.Builder
	ok := false
	for _, r := range st.Redirs {
		switch {
		case (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil:
			b.WriteString(hdocString(r))
			ok = true
		case r.Op == syntax.WordHdoc && r.Word != nil:
			b.WriteString(wordString(r.Word) + "\n")
			ok = true
		}
	}
	return b.String(), ok
}

// printedByAll is what a list of commands prints, joined, when the line shows what at least one of them prints.
func printedByAll(stmts []*syntax.Stmt) (string, bool) {
	var b strings.Builder
	ok := false
	for _, st := range stmts {
		if out, k := printedBy(st); k {
			b.WriteString(out)
			ok = true
		}
		if b.Len() > maxScriptBytes {
			break
		}
	}
	return b.String(), ok
}

var echoFlags = regexp.MustCompile(`^-[neE]+$`)

// echoed is what an echo or printf command prints.
func echoed(call *syntax.CallExpr) (string, bool) {
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		words[i] = wordString(w)
	}
	bare := stripWrappers(words)
	if len(bare) == 0 {
		return "", false
	}
	args := bare[1:]
	switch path.Base(bare[0]) {
	case "echo":
		// bash's echo decodes escapes with -e, zsh's and sh's by default; reading them errs on the safe side. The
		// last of -e and -E wins.
		decode, newline := true, true
		for len(args) > 0 && echoFlags.MatchString(args[0]) {
			if i := strings.LastIndexAny(args[0], "eE"); i >= 0 {
				decode = args[0][i] == 'e'
			}
			newline = newline && !strings.Contains(args[0], "n")
			args = args[1:]
		}
		out := strings.Join(args, " ")
		if decode {
			var stop bool
			if out, stop = decodeEscapes(out, escEcho); stop {
				return out, true // \c ends the output
			}
		}
		if newline {
			out += "\n"
		}
		return out, true
	case "printf": // printf '%s %s\n' git push: what it prints
		return printf(args), true
	}
	return "", false
}

// printfVerb is one conversion: flags, a width and a precision (either may be * to take an argument), and a letter.
// maxPrintfWidth bounds a printf field width (%99999s), so a crafted width can't grow the output.
const maxPrintfWidth = 1024

var printfVerb = regexp.MustCompile(`%[-+ #0]*(\*|[0-9]*)(\.(\*|[0-9]*))?[a-zA-Z%]`)

// printf is what `printf [--] format args...` prints: each verb takes the next argument, %% is a %, and the format
// repeats while arguments are left, as in the shell. %b decodes its argument's escapes and its \c ends the output;
// a precision cuts %s and %b; -v sets a variable and prints nothing.
func printf(args []string) string {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 || args[0] == "-v" {
		return ""
	}
	format, _ := decodeEscapes(args[0], escFormat)
	args = args[1:]
	var b strings.Builder
	for {
		used := false
		take := func() string {
			if len(args) == 0 {
				return ""
			}
			used = true
			a := args[0]
			args = args[1:]
			return a
		}
		last := 0
		for _, m := range printfVerb.FindAllStringSubmatchIndex(format, -1) {
			b.WriteString(format[last:m[0]])
			last = m[1]
			conv := format[m[1]-1]
			if conv == '%' {
				b.WriteByte('%')
				continue
			}
			width := 0 // %5s pads to five characters: printf 'git%5s' push prints "git push"
			if w := format[m[2]:m[3]]; w == "*" {
				width, _ = strconv.Atoi(take())
			} else if w != "" {
				width, _ = strconv.Atoi(w)
			}
			width = min(max(width, -maxPrintfWidth), maxPrintfWidth)
			leftAlign := strings.Contains(format[m[0]+1:m[2]], "-") || width < 0
			if width < 0 {
				width = -width
			}
			prec := -1
			if m[4] >= 0 {
				p := format[m[6]:m[7]]
				if p == "*" {
					p = take()
				}
				prec, _ = strconv.Atoi(p)
			}
			a := take()
			stop := false
			switch conv {
			case 'b':
				a, stop = decodeEscapes(a, escB)
			case 'c':
				a = a[:min(1, len(a))]
			case 'd', 'i', 'o', 'u', 'x', 'X', 'e', 'E', 'f', 'F', 'g', 'G', 'a', 'A':
				// Numbers print as numbers: printf 'git %xommit' 12 prints "git commit". The width is applied here
				// with the flags (0-padding, +, #), so it isn't applied again below.
				a = printfNumber(format[m[0]+1:m[2]], width, leftAlign, prec, conv, a)
				width = 0
			}
			if (conv == 's' || conv == 'b') && prec >= 0 && prec < len(a) {
				a = a[:prec]
			}
			if pad := width - len(a); pad > 0 {
				if leftAlign {
					a += strings.Repeat(" ", pad)
				} else {
					a = strings.Repeat(" ", pad) + a
				}
			}
			b.WriteString(a)
			if stop {
				return b.String()
			}
		}
		b.WriteString(format[last:])
		if !used || len(args) == 0 || b.Len() > maxScriptBytes {
			return b.String()
		}
	}
}

// printfNumber formats arg the way printf's numeric conversions do. Like bash it reads 0x, 0 (octal) and 'c (the
// character's code); an argument that isn't a number prints as 0.
func printfNumber(flags string, width int, leftAlign bool, prec int, conv byte, arg string) string {
	arg = strings.TrimSpace(arg)
	spec := "%" + strings.ReplaceAll(flags, "-", "")
	if leftAlign {
		spec += "-"
	}
	if width > 0 {
		spec += strconv.Itoa(width)
	}
	if prec >= 0 {
		spec += "." + strconv.Itoa(min(prec, maxPrintfWidth))
	}
	switch conv {
	case 'e', 'E', 'f', 'F', 'g', 'G', 'a', 'A':
		f, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			f = float64(printfInt(arg))
		}
		verb := map[byte]string{'a': "e", 'A': "E", 'F': "f"}[conv]
		if verb == "" {
			verb = string(conv)
		}
		return fmt.Sprintf(spec+verb, f)
	}
	n := printfInt(arg)
	switch conv {
	case 'o', 'x', 'X':
		return fmt.Sprintf(spec+string(conv), uint64(n))
	case 'u':
		return fmt.Sprintf(spec+"d", uint64(n))
	}
	return fmt.Sprintf(spec+"d", n)
}

// printfInt reads a printf integer argument: decimal, 0x hex, 0 octal, or 'c / "c for c's character code.
func printfInt(arg string) int64 {
	if len(arg) >= 2 && (arg[0] == '\'' || arg[0] == '"') {
		r := []rune(arg[1:])
		return int64(r[0])
	}
	n, err := strconv.ParseInt(arg, 0, 64)
	if err != nil {
		return 0
	}
	return n
}

// trapAction is the command `trap [-lpP] [--] action signal...` sets; trap - and a single operand reset instead.
func trapAction(args []string) (string, bool) {
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" {
		dash := args[0] == "--"
		args = args[1:]
		if dash {
			break
		}
	}
	if len(args) < 2 || args[0] == "-" {
		return "", false
	}
	return args[0], true
}

func commandText(words, redirs []string) string {
	parts := make([]string, 0, len(words)+len(redirs))
	for i, w := range words {
		if i > 0 {
			w = strings.Map(func(r rune) rune {
				if r == ';' || r == '&' || r == '|' || r == '\n' {
					return ' '
				}
				return r
			}, w)
		}
		parts = append(parts, w)
	}
	return strings.Join(append(parts, redirs...), " ")
}

// shellArgs reads a shell's arguments the way bash does: short flags combine (-euo pipefail -c), o and O take the
// next word, -c runs its first operand as the script, and otherwise the first operand is a script file. stdin is
// whether the shell runs what it reads on stdin: no -c and no script file, or -s.
func shellArgs(args []string) (script string, isC, stdin bool) {
	readStdin := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--rcfile" || a == "--init-file":
			i++
		case a == "--" || a == "-": // bash - and sh -: the options end and the script comes on stdin
			if i+1 < len(args) {
				if isC {
					return args[i+1], true, false
				}
				return "", false, readStdin
			}
			return "", isC, !isC
		case strings.HasPrefix(a, "--"):
		case len(a) > 1 && (a[0] == '-' || a[0] == '+'):
			for _, ch := range a[1:] {
				switch ch {
				case 'c':
					isC = true
				case 's':
					readStdin = true
				case 'o', 'O':
					i++ // -o pipefail, -euo pipefail
				}
			}
		default:
			if isC {
				return a, true, false
			}
			return "", false, readStdin // a script file; stdin is its input unless -s
		}
	}
	return "", isC, !isC
}

// shellScript is the script of `bash [flags] -c script`.
func shellScript(args []string) (string, bool) {
	script, isC, _ := shellArgs(args)
	return script, isC && script != ""
}

// readsStdin reports whether a shell with these arguments runs the script it reads on stdin.
func readsStdin(args []string) bool {
	_, _, stdin := shellArgs(args)
	return stdin
}

// stripWrappers drops the commands that run the rest of their words as a command: sudo, doas, env, command, builtin,
// exec, nohup, setsid, nice, timeout, stdbuf, time and xargs, with their flags (and the flags' values).
func stripWrappers(w []string) []string {
	for len(w) > 0 {
		switch path.Base(w[0]) {
		case "sudo": // sudo [flags] [VAR=value...] command
			w = skipAssigns(skipFlags(w[1:], "aCcDghpRrTtUu", "--user", "--group", "--chdir", "--chroot", "--host",
				"--prompt", "--role", "--type", "--other-user", "--close-from", "--command-timeout", "--auth-type",
				"--login-class"))
		case "doas":
			w = skipFlags(w[1:], "aCu")
		case "env":
			w = envCommand(w[1:])
		case "command", "builtin", "exec", "nohup", "setsid":
			w = skipFlags(w[1:], "a")
		case "nice":
			w = skipFlags(w[1:], "n", "--adjustment")
		case "timeout": // timeout [flags] duration command
			w = skipFlags(w[1:], "ks", "--kill-after", "--signal")
			w = w[min(1, len(w)):]
		case "stdbuf":
			w = skipFlags(w[1:], "eio", "--error", "--input", "--output")
		case "time": // the time program; the time keyword is parsed as such
			w = skipFlags(w[1:], "fo", "--format", "--output")
		case "xargs": // -e, -i and -l take a value only when attached
			w = skipFlags(w[1:], "adEIJLnPRsS", "--arg-file", "--delimiter", "--max-args", "--max-procs",
				"--max-chars", "--process-slot-var")
		default:
			return w
		}
	}
	return w
}

var assignWord = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// skipAssigns drops leading VAR=value words.
func skipAssigns(w []string) []string {
	for len(w) > 0 && assignWord.MatchString(w[0]) {
		w = w[1:]
	}
	return w
}

// envCommand is the command `env [assignments] [flags] command...` runs. -S (--split-string) holds a command line of
// its own: env -S 'git push' runs git push.
func envCommand(w []string) []string {
	for len(w) > 0 {
		a := w[0]
		switch {
		case strings.Contains(a, "=") && !strings.HasPrefix(a, "-"):
			w = w[1:]
		case a == "--split-string":
			if len(w) < 2 {
				return nil
			}
			w = append(envSplitString(w[1]), w[2:]...)
		case strings.HasPrefix(a, "--split-string="):
			w = append(envSplitString(strings.TrimPrefix(a, "--split-string=")), w[1:]...)
		case a == "--unset" || a == "--chdir":
			w = w[min(2, len(w)):]
		case a == "--":
			return w[1:]
		case a == "-": // env - is env -i
			w = w[1:]
		case strings.HasPrefix(a, "--"):
			w = w[1:]
		case strings.HasPrefix(a, "-") && a != "-":
			w = envShortFlags(a, w[1:])
		default:
			return w
		}
	}
	return w
}

// envSplitString splits an env -S operand the way GNU env does, which is not bash's grammar: whitespace separates
// words; '...' is literal except \\ and \'; "..." and unquoted text take the escapes \" \' \\ \# \$ \t \n \r
// \f \v; \_ is a word break outside quotes and a space inside "..."; \c ends the string; # starting a word starts a
// comment. ${VAR} stays as written.
func envSplitString(str string) []string {
	var words []string
	var cur strings.Builder
	inWord := false
	flush := func() {
		if inWord {
			words = append(words, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(str); i++ {
		c := str[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v':
			flush()
		case c == '#' && !inWord:
			return words
		case c == '\'':
			inWord = true
			for i++; i < len(str) && str[i] != '\''; i++ {
				if str[i] == '\\' && i+1 < len(str) && (str[i+1] == '\\' || str[i+1] == '\'') {
					i++
				}
				cur.WriteByte(str[i])
			}
		case c == '"':
			inWord = true
			for i++; i < len(str) && str[i] != '"'; i++ {
				if str[i] == '\\' && i+1 < len(str) {
					i++
					if str[i] == 'c' {
						flush()
						return words
					}
					cur.WriteString(envEscape(str[i], true))
					continue
				}
				cur.WriteByte(str[i])
			}
		case c == '\\' && i+1 < len(str):
			i++
			switch str[i] {
			case '_':
				flush()
			case 'c':
				flush()
				return words
			default:
				inWord = true
				cur.WriteString(envEscape(str[i], false))
			}
		default:
			inWord = true
			cur.WriteByte(c)
		}
	}
	flush()
	return words
}

// envEscape is what env -S makes of \c for one escape letter c.
func envEscape(c byte, quoted bool) string {
	switch c {
	case 't':
		return "\t"
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 'f':
		return "\f"
	case 'v':
		return "\v"
	case '_':
		if quoted {
			return " "
		}
	}
	return string(c)
}

// envShortFlags reads one cluster of env's short flags (-iS'cmd', -u NAME) and returns the words after it.
func envShortFlags(a string, rest []string) []string {
	for i := 1; i < len(a); i++ {
		attached := a[i+1:]
		switch a[i] {
		case 'S':
			if attached != "" {
				return append(envSplitString(attached), rest...)
			}
			if len(rest) == 0 {
				return nil
			}
			return append(envSplitString(rest[0]), rest[1:]...)
		case 'u', 'C', 'P':
			if attached != "" || len(rest) == 0 {
				return rest
			}
			return rest[1:]
		}
	}
	return rest
}

// skipFlags drops leading flags. In a cluster of short flags (-iu), a letter in shortValue takes a value: the rest of
// the word when there is one (-uroot), else the next word (-iu deploy). A long flag in longValue takes the next word
// unless written with =. -- ends the flags.
func skipFlags(w []string, shortValue string, longValue ...string) []string {
	for len(w) > 0 && strings.HasPrefix(w[0], "-") && w[0] != "-" {
		a := w[0]
		w = w[1:]
		switch {
		case a == "--":
			return w
		case strings.HasPrefix(a, "--"):
			if !strings.Contains(a, "=") && len(w) > 0 {
				for _, f := range longValue {
					if a == f {
						w = w[1:]
						break
					}
				}
			}
		default:
			for i := 1; i < len(a); i++ {
				if strings.IndexByte(shortValue, a[i]) >= 0 {
					if i == len(a)-1 && len(w) > 0 {
						w = w[1:]
					}
					break
				}
			}
		}
	}
	return w
}

// gitSubcommand is the subcommand of a git command ("commit" for `git -C repo -c a=b commit -m x`), or "".
func gitSubcommand(words []string) string {
	if len(words) == 0 || path.Base(words[0]) != "git" {
		return ""
	}
	for i := 1; i < len(words); i++ {
		switch a := words[i]; {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" || a == "--namespace" ||
			a == "--exec-path" || a == "--config-env" || a == "--super-prefix":
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return a
		}
	}
	return ""
}

// wordString is a word as the command sees it, quotes removed; an expansion ($VAR, $(cmd)) stays as its source text.
func wordString(w *syntax.Word) string {
	var b strings.Builder
	for _, p := range w.Parts {
		writePart(&b, p)
	}
	return b.String()
}

func writePart(b *strings.Builder, p syntax.WordPart) { writePartIn(b, p, false) }

// writePartIn writes one part; inDouble is whether it sits inside double quotes, where a backslash escapes only
// $ ` " \ and a newline. Outside quotes a backslash escapes any character (\git is git).
func writePartIn(b *strings.Builder, p syntax.WordPart, inDouble bool) {
	switch p := p.(type) {
	case *syntax.Lit:
		b.WriteString(unbackslash(p.Value, inDouble))
	case *syntax.SglQuoted:
		if p.Dollar { // $'git\tpush'
			b.WriteString(ansiC(p.Value))
		} else {
			b.WriteString(p.Value)
		}
	case *syntax.DblQuoted:
		for _, q := range p.Parts {
			writePartIn(b, q, true)
		}
	default:
		b.WriteString(source(p))
	}
}

// ansiC decodes the escapes of a $'...' string.
func ansiC(s string) string {
	out, _ := decodeEscapes(s, escDollar)
	return out
}

// escMode is where escapes are read; each place reads a few of them its own way.
type escMode int

const (
	escDollar escMode = iota // $'...': \cX is a control character
	escFormat                // a printf format: \c is just \c
	escEcho                  // echo -e: octal is \0 and up to three digits; \c ends the output
	escB                     // printf %b: octal is \0NNN or \NNN; \c ends the output
)

var simpleEscapes = map[byte]byte{'a': 7, 'b': 8, 'e': 0x1b, 'E': 0x1b, 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t',
	'v': '\v', '\\': '\\'}

// decodeEscapes decodes s's backslash escapes as the shell does in mode: \a \b \e \f \n \r \t \v \\, \xHH, \uHHHH,
// \UHHHHHHHH, octal, \c, and in $'...' and printf formats \' \" \?. Any other escape stays as written. stop is
// whether \c ended the output.
func decodeEscapes(s string, mode escMode) (out string, stop bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		c := s[i]
		if r, ok := simpleEscapes[c]; ok {
			b.WriteByte(r)
			continue
		}
		switch {
		case c == '\'' || c == '"' || c == '?':
			if mode == escEcho || mode == escB {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		case c == 'x' || c == 'u' || c == 'U':
			digits := map[byte]int{'x': 2, 'u': 4, 'U': 8}[c]
			n, j := 0, i+1
			for ; j < len(s) && j <= i+digits && strings.IndexByte("0123456789abcdefABCDEF", s[j]) >= 0; j++ {
				n = n*16 + strings.IndexByte("0123456789abcdef", s[j]|0x20)
			}
			switch {
			case j == i+1:
				b.WriteByte('\\')
				b.WriteByte(c)
				continue
			case c == 'x':
				b.WriteByte(byte(n))
			default:
				b.WriteRune(rune(n))
			}
			i = j - 1
		case c == 'c':
			switch {
			case mode == escEcho || mode == escB:
				return b.String(), true
			case mode == escDollar && i+1 < len(s): // \cJ is a newline
				i++
				b.WriteByte(s[i] & 0x1f)
			default:
				b.WriteString(`\c`)
			}
		case c >= '0' && c <= '7':
			start := i
			switch {
			case mode == escEcho && c != '0':
				b.WriteByte('\\')
				b.WriteByte(c)
				continue
			case (mode == escEcho || mode == escB) && c == '0':
				start = i + 1 // \0 then up to three digits
			}
			n, j := 0, start
			for ; j < len(s) && j < start+3 && s[j] >= '0' && s[j] <= '7'; j++ {
				n = n*8 + int(s[j]-'0')
			}
			b.WriteByte(byte(n))
			i = j - 1
		default:
			b.WriteByte('\\')
			b.WriteByte(c)
		}
	}
	return b.String(), false
}

// hdocString is a heredoc's body as the shell reads it: as written when the delimiter is quoted (<<'EOF'), else with
// each backslash kept except before $ ` and \ (the parser already joins \-newline continuations).
func hdocString(r *syntax.Redirect) string {
	quoted := false
	if r.Word != nil {
		for _, p := range r.Word.Parts {
			if l, ok := p.(*syntax.Lit); !ok || strings.Contains(l.Value, `\`) {
				quoted = true
			}
		}
	}
	var b strings.Builder
	for _, p := range r.Hdoc.Parts {
		l, ok := p.(*syntax.Lit)
		switch {
		case ok && quoted:
			b.WriteString(l.Value)
		case ok:
			for i := 0; i < len(l.Value); i++ {
				if l.Value[i] == '\\' && i+1 < len(l.Value) && strings.IndexByte("$`\\", l.Value[i+1]) >= 0 {
					i++
				}
				b.WriteByte(l.Value[i])
			}
		default:
			b.WriteString(source(p))
		}
	}
	return b.String()
}

func unbackslash(s string, inDouble bool) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (!inDouble || strings.IndexByte("$`\"\\\n", s[i+1]) >= 0) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// source is a node's text as written.
func source(n syntax.Node) string {
	var b strings.Builder
	syntax.NewPrinter().Print(&b, n)
	return b.String()
}
