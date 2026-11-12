package rule

import (
	"path"
	"regexp"
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

var shells = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true}

const maxNesting = 8 // bash -c inside eval inside bash -c ...

// maxFuncWalks caps how many function bodies one line may expand, however its functions define and call each other,
// so no line can make the hook slow.
const maxFuncWalks = 64

// parsedLine is a Bash line's commands; ok is false when the line, or a script it runs, isn't valid Bash.
type parsedLine struct {
	cmds []command
	ok   bool
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
	budget := maxFuncWalks
	p.ok = split(line, &p.cmds, 0, &budget)
	splitCache.line, splitCache.parsed = line, p
	return p
}

func split(line string, out *[]command, depth int, budget *int) bool {
	if depth > maxNesting {
		return true
	}
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil {
		return false
	}
	s := splitter{out: out, funcs: map[string]*syntax.Stmt{}, walked: map[walkKey]bool{}, budget: budget,
		piped: map[*syntax.BinaryCmd]bool{}, ok: true}
	s.walk(f, depth)
	return s.ok
}

// splitter collects the commands of one parsed line. funcs are the functions defined so far, in the order the shell
// runs the line: defining one runs nothing, so a body counts where a call runs it. A body is read once per set of
// definitions (defs counts them), so a later redefinition of a function it calls is read again, and a chain of calls
// can't blow up.
type splitter struct {
	out    *[]command
	funcs  map[string]*syntax.Stmt
	defs   int
	walked map[walkKey]bool
	budget *int                       // function-body walks left for the whole line
	piped  map[*syntax.BinaryCmd]bool // pipes already read as one pipeline
	ok     bool                       // false once a nested script didn't parse: the whole line is then read the old way
}

type walkKey struct {
	body *syntax.Stmt
	defs int
}

// nested splits a script the line runs (bash -c, a heredoc, a here-string, eval, a pipe into a shell). Its whole
// text is one more text to match, so a rule spanning commands (`curl x | sh`) sees it too.
func (s *splitter) nested(script string, depth int) {
	if !split(script, s.out, depth+1, s.budget) {
		s.ok = false
		return
	}
	*s.out = append(*s.out, command{texts: []string{script}})
}

func (s *splitter) walk(node syntax.Node, depth int) {
	syntax.Walk(node, func(n syntax.Node) bool {
		switch n := n.(type) {
		case *syntax.FuncDecl:
			s.funcs[n.Name.Value] = n.Body
			s.defs++
			return false
		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe || n.Op == syntax.PipeAll {
				s.pipedScript(n, depth)
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
							s.nested(wordString(r.Hdoc), depth)
						case r.Op == syntax.WordHdoc && r.Word != nil: // bash <<< 'git push'
							s.nested(wordString(r.Word), depth)
						}
					}
				}
			case head == "eval":
				s.nested(strings.Join(bare[1:], " "), depth)
			case s.funcs[bare[0]] != nil && !s.walked[walkKey{s.funcs[bare[0]], s.defs}] && depth < maxNesting &&
				*s.budget > 0:
				body := s.funcs[bare[0]]
				s.walked[walkKey{body, s.defs}] = true
				*s.budget--
				s.walk(body, depth+1)
			}
		}
		text := commandText(words, redirs)
		c := command{words: bare, texts: []string{text}}
		if len(bare) > 0 && len(bare) != len(words) {
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

// printed splits what a pipeline stage prints, when that can be read off the line: a heredoc, echo or printf.
func (s *splitter) printed(st *syntax.Stmt, depth int) {
	for _, r := range st.Redirs {
		switch {
		case (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil:
			s.nested(wordString(r.Hdoc), depth)
		case r.Op == syntax.WordHdoc && r.Word != nil:
			s.nested(wordString(r.Word), depth)
		}
	}
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Args) < 2 {
		return
	}
	args := make([]string, 0, len(call.Args)-1)
	for _, w := range call.Args[1:] {
		args = append(args, wordString(w))
	}
	switch wordString(call.Args[0]) {
	case "echo": // echo -e "a\nb" too
		for len(args) > 0 && (args[0] == "-e" || args[0] == "-n" || args[0] == "-E" || args[0] == "-en" || args[0] == "-ne") {
			args = args[1:]
		}
		s.nested(unescape(strings.Join(args, " ")), depth)
	case "printf": // printf '%s %s\n' git push: what it prints
		s.nested(printf(args[0], args[1:]), depth)
	}
}

var printfVerb = regexp.MustCompile(`%[-+ #0-9.]*[a-zA-Z%]`)

// printf is what `printf format args...` prints: each verb takes the next argument, %% is a %, and the format repeats
// while arguments are left, as in the shell.
func printf(format string, args []string) string {
	format = unescape(format)
	var b strings.Builder
	for {
		used := false
		b.WriteString(printfVerb.ReplaceAllStringFunc(format, func(v string) string {
			if v == "%%" {
				return "%"
			}
			if len(args) == 0 {
				return ""
			}
			used = true
			a := args[0]
			args = args[1:]
			return a
		}))
		if !used || len(args) == 0 {
			return b.String()
		}
	}
}

// unescape turns the \n and \t that echo -e and printf print into the characters.
func unescape(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\t`, "\t").Replace(s)
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
		case a == "--":
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

// stripWrappers drops the commands that run the rest of their words as a command: sudo, env, command, exec, nohup,
// time and xargs, with their flags (and the flags' values).
func stripWrappers(w []string) []string {
	for len(w) > 0 {
		switch path.Base(w[0]) {
		case "sudo":
			w = skipFlags(w[1:], "ugCDhprtU", "--user", "--group", "--chdir", "--chroot", "--host", "--prompt",
				"--role", "--type", "--other-user", "--close-from")
		case "env":
			w = envCommand(w[1:])
		case "command", "exec", "nohup", "time":
			w = skipFlags(w[1:], "a")
		case "xargs":
			w = skipFlags(w[1:], "IdEnLPsa", "--delimiter", "--eof", "--max-args", "--max-lines", "--max-procs",
				"--max-chars", "--arg-file")
		default:
			return w
		}
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
			w = append(strings.Fields(w[1]), w[2:]...)
		case strings.HasPrefix(a, "--split-string="):
			w = append(strings.Fields(strings.TrimPrefix(a, "--split-string=")), w[1:]...)
		case a == "--unset" || a == "--chdir":
			w = w[min(2, len(w)):]
		case a == "--":
			return w[1:]
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

// envShortFlags reads one cluster of env's short flags (-iS'cmd', -u NAME) and returns the words after it.
func envShortFlags(a string, rest []string) []string {
	for i := 1; i < len(a); i++ {
		attached := a[i+1:]
		switch a[i] {
		case 'S':
			if attached != "" {
				return append(strings.Fields(attached), rest...)
			}
			if len(rest) == 0 {
				return nil
			}
			return append(strings.Fields(rest[0]), rest[1:]...)
		case 'u', 'C':
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
		b.WriteString(p.Value)
	case *syntax.DblQuoted:
		for _, q := range p.Parts {
			writePartIn(b, q, true)
		}
	default:
		b.WriteString(source(p))
	}
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
