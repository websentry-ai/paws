package rule

import (
	"path"
	"regexp"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/syntax"
)

// A Bash line is split into the simple commands the shell would run, and a rule's command_regex is also matched
// against each one: `cd web && npm install x` holds `npm install x`, and `bash -c "git push"`, `sudo git push` and
// `echo "git push" | sh` each hold `git push`. Matching only adds: what the whole-line regexes matched before still
// matches. A line, or a script it runs, that doesn't parse adds nothing.

// command is one simple command.
type command struct {
	words []string // quotes removed and wrappers (sudo, env, ...) dropped, as the program receives them
	texts []string // what a command_regex sees: the command as written, and without its wrappers
}

var shells = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true, "ash": true, "mksh": true}

const (
	maxNesting     = 8        // bash -c inside eval inside bash -c ...
	maxScriptBytes = 64 << 10 // nested script text one line may parse
)

// splitCache holds the last line read: the gate checks every rule against the same line.
var splitCache struct {
	sync.Mutex
	line string
	cmds []command
}

func parseLine(line string) []command {
	splitCache.Lock()
	defer splitCache.Unlock()
	if splitCache.cmds != nil && splitCache.line == line {
		return splitCache.cmds
	}
	sp := &splitter{cmds: []command{}, walked: map[*syntax.Stmt]bool{}, budget: maxScriptBytes}
	sp.split(line, 0, map[string][]*syntax.Stmt{})
	splitCache.line, splitCache.cmds = line, sp.cmds
	return sp.cmds
}

// splitter collects the commands of one line and of the scripts it runs.
type splitter struct {
	cmds   []command
	walked map[*syntax.Stmt]bool // function bodies already read: each is read once per line
	budget int                   // nested script bytes still allowed
}

// split adds a script's commands. funcs holds the functions of the shell it runs in: eval shares its caller's, a new
// shell starts with none. It reports whether the script parsed.
func (sp *splitter) split(script string, depth int, funcs map[string][]*syntax.Stmt) bool {
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(script), "")
	if err != nil {
		return false
	}
	sp.walk(f, depth, funcs)
	return true
}

// nested splits a script the line runs (bash -c, eval, a heredoc or pipe into a shell). Its whole text is one more
// text to match, so a rule spanning commands (`curl x | sh`) sees it too.
func (sp *splitter) nested(script string, depth int, funcs map[string][]*syntax.Stmt) {
	if script == "" || depth >= maxNesting || len(script) > sp.budget {
		return // the whole-line regexes still apply
	}
	sp.budget -= len(script)
	if funcs == nil {
		funcs = map[string][]*syntax.Stmt{}
	}
	if sp.split(script, depth+1, funcs) {
		sp.cmds = append(sp.cmds, command{texts: []string{script}})
	}
}

func (sp *splitter) walk(node syntax.Node, depth int, funcs map[string][]*syntax.Stmt) {
	syntax.Walk(node, func(n syntax.Node) bool {
		switch n := n.(type) {
		case *syntax.FuncDecl:
			// Defining a function runs nothing: its body counts where the line calls it. A name keeps every body
			// given to it, since a definition inside if or case may not run.
			funcs[n.Name.Value] = append(funcs[n.Name.Value], n.Body)
			return false
		case *syntax.BinaryCmd:
			if n.Op == syntax.Pipe || n.Op == syntax.PipeAll {
				// echo "git push" | sh: a shell reading stdin runs what the stage before it prints.
				if next := pipeEnd(n.Y, false); readsStdin(shellWords(next)) {
					if out, ok := printed(pipeEnd(n.X, true)); ok {
						sp.nested(out, depth, nil)
					}
				}
			}
		case *syntax.Stmt:
			sp.stmt(n, depth, funcs)
		}
		return true
	})
}

func (sp *splitter) stmt(st *syntax.Stmt, depth int, funcs map[string][]*syntax.Stmt) {
	// Redirections stay on the command's text ("python - <<'PY'", "cmd >log"): some triggers name them.
	var redirs []string
	for _, r := range st.Redirs {
		if r.Word != nil {
			redirs = append(redirs, r.Op.String()+wordString(r.Word))
		}
	}
	switch cmd := st.Cmd.(type) {
	case *syntax.DeclClause: // export X=1, declare, local, readonly
		words := []string{cmd.Variant.Value}
		for _, a := range cmd.Args {
			words = append(words, source(a))
		}
		sp.cmds = append(sp.cmds, command{words: words, texts: []string{commandText(words, redirs)}})
	case *syntax.CallExpr:
		var assigns []string
		for _, a := range cmd.Assigns {
			assigns = append(assigns, source(a))
		}
		words := callWords(cmd)
		if len(words) == 0 {
			if len(assigns) > 0 {
				sp.cmds = append(sp.cmds, command{texts: []string{strings.Join(assigns, " ")}})
			}
			return
		}
		bare := stripWrappers(words)
		sp.runs(st, bare, depth, funcs)
		text := commandText(words, redirs)
		c := command{words: bare, texts: []string{text}}
		if len(bare) > 0 && len(bare) < len(words) {
			c.texts = append(c.texts, commandText(bare, redirs))
		}
		if len(assigns) > 0 { // HUSKY=0 git commit: a trigger may name the variable
			c.texts = append(c.texts, strings.Join(assigns, " ")+" "+text)
		}
		sp.cmds = append(sp.cmds, c)
	}
}

// runs splits the script a command runs: bash -c, a heredoc or here-string fed to a shell, eval, or the body of a
// function it calls.
func (sp *splitter) runs(st *syntax.Stmt, words []string, depth int, funcs map[string][]*syntax.Stmt) {
	if len(words) == 0 {
		return
	}
	switch name := path.Base(words[0]); {
	case shells[name]:
		script, isC, stdin := shellArgs(words[1:])
		switch {
		case isC:
			sp.nested(script, depth, nil)
		case stdin: // bash <<EOF, bash <<< 'git push'
			if out, ok := stdinText(st); ok {
				sp.nested(out, depth, nil)
			}
		}
	case name == "eval": // eval runs in this shell, with its functions
		args := words[1:]
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		sp.nested(strings.Join(args, " "), depth, funcs)
	case depth < maxNesting:
		for _, body := range funcs[words[0]] {
			if !sp.walked[body] {
				sp.walked[body] = true
				sp.walk(body, depth+1, funcs)
			}
		}
	}
}

// pipeEnd is the last (or first) command on one side of a pipe: a | b | c nests as pipes within pipes.
func pipeEnd(st *syntax.Stmt, last bool) *syntax.Stmt {
	for {
		b, ok := st.Cmd.(*syntax.BinaryCmd)
		if !ok || (b.Op != syntax.Pipe && b.Op != syntax.PipeAll) {
			return st
		}
		if last {
			st = b.Y
		} else {
			st = b.X
		}
	}
}

var echoFlags = regexp.MustCompile(`^-[neE]+$`)

// printed is what st prints when the line shows it: echo's arguments, or the heredoc or here-string it reads
// (cat <<EOF). Escapes are left as written.
func printed(st *syntax.Stmt) (string, bool) {
	if call, ok := st.Cmd.(*syntax.CallExpr); ok {
		if w := stripWrappers(callWords(call)); len(w) > 0 && path.Base(w[0]) == "echo" {
			args := w[1:]
			for len(args) > 0 && echoFlags.MatchString(args[0]) {
				args = args[1:]
			}
			return strings.Join(args, " "), true
		}
	}
	return stdinText(st)
}

// stdinText is the heredoc or here-string a command reads on stdin.
func stdinText(st *syntax.Stmt) (string, bool) {
	for _, r := range st.Redirs {
		switch {
		case (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil:
			var b strings.Builder
			for _, p := range r.Hdoc.Parts {
				if l, ok := p.(*syntax.Lit); ok {
					b.WriteString(l.Value)
				} else {
					b.WriteString(source(p))
				}
			}
			return b.String(), true
		case r.Op == syntax.WordHdoc && r.Word != nil:
			return wordString(r.Word), true
		}
	}
	return "", false
}

// shellWords is the words of st after any wrapper, when st is a shell.
func shellWords(st *syntax.Stmt) []string {
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil
	}
	if w := stripWrappers(callWords(call)); len(w) > 0 && shells[path.Base(w[0])] {
		return w
	}
	return nil
}

// readsStdin reports whether a shell command (words from shellWords) runs the script on its stdin.
func readsStdin(words []string) bool {
	if len(words) == 0 {
		return false
	}
	_, _, stdin := shellArgs(words[1:])
	return stdin
}

// shellArgs reads a shell's arguments as bash does: short flags combine (-euo pipefail -c), -o and -O take the next
// word, and -c runs the first operand as the script. Without -c, the shell runs its stdin when it is given no script
// file, or -s.
func shellArgs(args []string) (script string, isC, stdin bool) {
	s := false
	i := 0
loop:
	for ; i < len(args); i++ {
		switch a := args[i]; {
		case a == "-" || a == "--": // the options end
			i++
			break loop
		case a == "--rcfile" || a == "--init-file":
			i++
		case strings.HasPrefix(a, "--"):
		case len(a) > 1 && (a[0] == '-' || a[0] == '+'):
			for _, ch := range a[1:] {
				switch ch {
				case 'c':
					isC = true
				case 's':
					s = true
				case 'o', 'O':
					i++ // -o pipefail
				}
			}
		default:
			break loop
		}
	}
	if isC {
		if i < len(args) {
			return args[i], true, false
		}
		return "", true, false
	}
	return "", false, s || i >= len(args)
}

// wrapper is a command that runs the rest of its words as a command, and its flags that take a value: short letters,
// and long names (which take the next word unless written with =).
type wrapper struct {
	short string
	long  []string
}

var wrappers = map[string]wrapper{
	"sudo":    {"aCcDghpRrTtUu", []string{"--user", "--group", "--chdir", "--chroot", "--host", "--prompt", "--role", "--type", "--other-user", "--close-from", "--command-timeout", "--auth-type", "--login-class"}},
	"doas":    {"aCu", nil},
	"env":     {"CPSu", []string{"--chdir", "--unset", "--split-string"}},
	"command": {}, "builtin": {}, "exec": {"a", nil}, "nohup": {}, "setsid": {},
	"nice":    {"n", []string{"--adjustment"}},
	"timeout": {"ks", []string{"--kill-after", "--signal"}},
	"stdbuf":  {"eio", []string{"--error", "--input", "--output"}},
	"time":    {"fo", []string{"--format", "--output"}}, // the time program; the time keyword parses as such
	"xargs":   {"adEIJLnPRsS", []string{"--arg-file", "--delimiter", "--max-args", "--max-procs", "--max-chars", "--process-slot-var"}},
}

var assignWord = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// stripWrappers drops leading wrappers with their flags, flag values and VAR=value words: `sudo -u deploy git push`
// runs git push.
func stripWrappers(w []string) []string {
	for len(w) > 0 {
		name := path.Base(w[0])
		wr, ok := wrappers[name]
		if !ok {
			return w
		}
		w = skipFlags(w[1:], wr.short, wr.long...)
		for len(w) > 0 && assignWord.MatchString(w[0]) {
			w = w[1:]
		}
		if name == "timeout" && len(w) > 0 { // timeout 30 git push
			w = w[1:]
		}
	}
	return w
}

// skipFlags drops leading flags. In a cluster of short flags (-iu), a letter in shortValue takes a value: the rest of
// the word when there is one (-uroot), else the next word (-iu deploy). -- ends the flags.
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

// commandText joins words for a regex. ; & | and newlines inside an argument become spaces, so text quoted in an
// argument (git commit -m "a; npm install x") can't look like the start of a command.
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

func callWords(call *syntax.CallExpr) []string {
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		words[i] = wordString(w)
	}
	return words
}

// wordString is a word as the command sees it, quotes removed; an expansion ($VAR, $(cmd)) stays as its source text.
func wordString(w *syntax.Word) string {
	var b strings.Builder
	for _, p := range w.Parts {
		writePart(&b, p, false)
	}
	return b.String()
}

// writePart writes one part of a word; inDouble is whether it sits inside double quotes, where a backslash escapes
// only $ ` " \ and a newline. Outside quotes a backslash escapes any character (\git is git).
func writePart(b *strings.Builder, p syntax.WordPart, inDouble bool) {
	switch p := p.(type) {
	case *syntax.Lit:
		b.WriteString(unbackslash(p.Value, inDouble))
	case *syntax.SglQuoted:
		b.WriteString(p.Value)
	case *syntax.DblQuoted:
		for _, q := range p.Parts {
			writePart(b, q, true)
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
