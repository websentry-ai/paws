package rule

import (
	"path"
	"regexp"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/syntax"
)

// A Bash command line is split into the simple commands the shell would run, and a rule's command_regex is matched
// against each one rather than against the whole line. So `bash -c "git commit"`, `(git commit)`, `sudo env X=1 git
// commit` and `xargs git commit` are each seen as git commit, while the same words inside an argument
// (`echo "a; npm install x"`, a commit message) are not a command. A line that doesn't parse falls back to matching
// the whole line, as before.

// command is one simple command: its words with quotes removed, as the program receives them.
type command struct {
	words []string
	// texts are what a command_regex sees: the words joined by spaces, as written and (when it differs) without
	// wrappers such as sudo or env; ; & | and newlines inside an argument become spaces, so they can't start a command.
	texts []string
}

var shells = map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true}

const maxNesting = 8 // bash -c inside eval inside bash -c ...

// parsedLine is a Bash line read for matching: its commands, and the line itself with every ; & | and newline that
// sits inside quotes turned into a space (flat). A command_regex matches if it matches flat or any command: flat keeps
// rules that span commands (`curl ... | sh`) or name a heredoc's text working as before, and the commands catch what
// runs inside bash -c, eval, wrappers and piped scripts.
type parsedLine struct {
	cmds []command
	flat string
	ok   bool // false when the line, or a script it runs, isn't valid Bash
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
	p.ok = split(line, &p.cmds, 0)
	if p.ok {
		p.flat = flatten(line)
	}
	splitCache.line, splitCache.parsed = line, p
	return p
}

// flatten blanks the command separators inside quoted text, so `echo "a; npm install x"` holds no second command.
func flatten(line string) string {
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil {
		return line
	}
	b := []byte(line)
	syntax.Walk(f, func(n syntax.Node) bool {
		switch n.(type) {
		case *syntax.SglQuoted, *syntax.DblQuoted:
			for i := n.Pos().Offset(); i < n.End().Offset() && int(i) < len(b); i++ {
				if c := b[i]; c == ';' || c == '&' || c == '|' || c == '\n' {
					b[i] = ' '
				}
			}
		}
		return true
	})
	return string(b)
}

func split(line string, out *[]command, depth int) bool {
	if depth > maxNesting {
		return true
	}
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil {
		return false
	}
	s := splitter{out: out, funcs: map[string]*syntax.Stmt{}, walked: map[*syntax.Stmt]bool{}, ok: true}
	s.walk(f, depth)
	return s.ok
}

// splitter collects the commands of one parsed line. funcs are the functions defined so far, in the order the shell
// runs the line: defining one runs nothing, so a body counts where a call runs it, and each body is read once.
type splitter struct {
	out    *[]command
	funcs  map[string]*syntax.Stmt
	walked map[*syntax.Stmt]bool
	ok     bool // false once a nested script didn't parse: the whole line is then read the old way
}

// nested splits a script the line runs (bash -c, a heredoc, eval, a pipe into a shell). The script whole, with
// quoted separators blanked, is one more text: a rule spanning commands (`curl x | sh`) still sees it inside bash -c.
func (s *splitter) nested(script string, depth int) {
	if !split(script, s.out, depth+1) {
		s.ok = false
		return
	}
	*s.out = append(*s.out, command{texts: []string{flatten(script)}})
}

func (s *splitter) walk(node syntax.Node, depth int) {
	syntax.Walk(node, func(n syntax.Node) bool {
		switch n := n.(type) {
		case *syntax.FuncDecl:
			s.funcs[n.Name.Value] = n.Body
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
						if (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil {
							s.nested(wordString(r.Hdoc), depth)
						}
					}
				}
			case head == "eval":
				s.nested(strings.Join(bare[1:], " "), depth)
			case s.funcs[bare[0]] != nil && !s.walked[s.funcs[bare[0]]] && depth < maxNesting:
				body := s.funcs[bare[0]]
				s.walked[body] = true
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

// pipedScript reads a script piped into a shell: `cat <<EOF | bash`, `echo "git push" | sh`.
func (s *splitter) pipedScript(pipe *syntax.BinaryCmd, depth int) {
	sh, ok := pipe.Y.Cmd.(*syntax.CallExpr)
	if !ok || len(sh.Args) == 0 {
		return
	}
	words := make([]string, len(sh.Args))
	for i, w := range sh.Args {
		words[i] = wordString(w)
	}
	bare := stripWrappers(words)
	if len(bare) == 0 || !shells[path.Base(bare[0])] || !readsStdin(bare[1:]) {
		return
	}
	left := pipe.X
	for {
		inner, ok := left.Cmd.(*syntax.BinaryCmd)
		if !ok {
			break
		}
		left = inner.Y
	}
	for _, r := range left.Redirs {
		if (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil {
			s.nested(wordString(r.Hdoc), depth)
		}
	}
	if call, ok := left.Cmd.(*syntax.CallExpr); ok && len(call.Args) > 1 {
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

// shellScript finds the script of `bash [flags] -c script`: -c may be combined (-ce, -euxc); -o takes a value.
func shellScript(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "+o" || a == "--rcfile" || a == "--init-file":
			i++
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-") && strings.Contains(a, "c"):
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		case !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "+"):
			return "", false // a script file, not -c
		}
	}
	return "", false
}

// readsStdin reports whether a shell with these arguments runs the script it reads on stdin: not with -c (the
// script is an argument) or a script file (stdin is that script's input), unless -s says to read stdin anyway.
func readsStdin(args []string) bool {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "+o" || a == "--rcfile" || a == "--init-file":
			i++
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-") || strings.HasPrefix(a, "+"):
			if strings.Contains(a, "s") {
				return true
			}
			if strings.Contains(a, "c") {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// stripWrappers drops the commands that run the rest of their words as a command: sudo, env, command, exec, nohup,
// time and xargs, with their flags (and the flags' values).
func stripWrappers(w []string) []string {
	for len(w) > 0 {
		switch path.Base(w[0]) {
		case "sudo":
			w = skipFlags(w[1:], "-u", "-g", "-C", "-D", "-h", "-p", "-r", "-t", "-U", "--user", "--group", "--chdir",
				"--chroot", "--host", "--prompt", "--role", "--type", "--other-user", "--close-from")
		case "env":
			w = w[1:]
			for len(w) > 0 && (strings.HasPrefix(w[0], "-") || strings.Contains(w[0], "=")) {
				if w[0] == "-u" || w[0] == "-C" || w[0] == "-S" || w[0] == "--unset" || w[0] == "--chdir" ||
					w[0] == "--split-string" {
					w = w[1:]
				}
				if len(w) > 0 {
					w = w[1:]
				}
			}
		case "command", "exec", "nohup", "time":
			w = skipFlags(w[1:], "-a")
		case "xargs":
			w = skipFlags(w[1:], "-I", "-d", "-E", "-n", "-L", "-P", "-s", "-a", "--delimiter", "--eof", "--max-args",
				"--max-lines", "--max-procs", "--max-chars", "--arg-file")
		default:
			return w
		}
	}
	return w
}

// skipFlags drops leading flags; a flag in withValue also drops the word after it.
func skipFlags(w []string, withValue ...string) []string {
	for len(w) > 0 && strings.HasPrefix(w[0], "-") {
		takes := false
		for _, f := range withValue {
			takes = takes || w[0] == f
		}
		w = w[1:]
		if takes && len(w) > 0 {
			w = w[1:]
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

func writePart(b *strings.Builder, p syntax.WordPart) {
	switch p := p.(type) {
	case *syntax.Lit:
		b.WriteString(p.Value)
	case *syntax.SglQuoted:
		b.WriteString(p.Value)
	case *syntax.DblQuoted:
		for _, q := range p.Parts {
			writePart(b, q)
		}
	default:
		b.WriteString(source(p))
	}
}

// source is a node's text as written.
func source(n syntax.Node) string {
	var b strings.Builder
	syntax.NewPrinter().Print(&b, n)
	return b.String()
}
