package rule

import (
	"path"
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

// splitCache holds the last line split: the gate checks every rule against the same line.
var splitCache struct {
	sync.Mutex
	line string
	cmds []command
	ok   bool
}

// commands splits line into its simple commands; ok is false when line isn't valid Bash.
func commands(line string) ([]command, bool) {
	splitCache.Lock()
	defer splitCache.Unlock()
	if splitCache.cmds != nil && splitCache.line == line {
		return splitCache.cmds, splitCache.ok
	}
	var out []command
	ok := split(line, &out, 0)
	if out == nil {
		out = []command{}
	}
	splitCache.line, splitCache.cmds, splitCache.ok = line, out, ok
	return out, ok
}

func split(line string, out *[]command, depth int) bool {
	if depth > maxNesting {
		return true
	}
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil {
		return false
	}
	syntax.Walk(f, func(n syntax.Node) bool {
		stmt, ok := n.(*syntax.Stmt)
		if !ok {
			return true
		}
		call, ok := stmt.Cmd.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		words := make([]string, len(call.Args))
		for i, w := range call.Args {
			words[i] = wordString(w)
		}
		bare := stripWrappers(words)
		if len(bare) > 0 {
			switch head := path.Base(bare[0]); {
			case shells[head]: // bash -c "script", bash <<EOF script EOF
				if script, ok := shellScript(bare[1:]); ok {
					split(script, out, depth+1)
				}
				for _, r := range stmt.Redirs {
					if (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil {
						split(wordString(r.Hdoc), out, depth+1)
					}
				}
			case head == "eval":
				split(strings.Join(bare[1:], " "), out, depth+1)
			}
		}
		// Redirections stay on the command's text ("python - <<'PY'", "cmd >log"): some triggers name them.
		var redirs []string
		for _, r := range stmt.Redirs {
			if r.Word != nil {
				redirs = append(redirs, r.Op.String()+wordString(r.Word))
			}
		}
		c := command{words: bare, texts: []string{commandText(words, redirs)}}
		if len(bare) > 0 && len(bare) != len(words) {
			c.texts = append(c.texts, commandText(bare, redirs))
		}
		*out = append(*out, c)
		return true
	})
	return true
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

// stripWrappers drops the commands that run the rest of their words as a command: sudo, env, command, exec, nohup,
// time and xargs, with their flags (and the flags' values).
func stripWrappers(w []string) []string {
	for len(w) > 0 {
		switch path.Base(w[0]) {
		case "sudo":
			w = skipFlags(w[1:], "-u", "-g", "-C", "-D", "-h", "-p", "-r", "-t", "-U")
		case "env":
			w = w[1:]
			for len(w) > 0 && (strings.HasPrefix(w[0], "-") || strings.Contains(w[0], "=")) {
				if w[0] == "-u" || w[0] == "-C" || w[0] == "-S" {
					w = w[1:]
				}
				if len(w) > 0 {
					w = w[1:]
				}
			}
		case "command", "exec", "nohup", "time":
			w = skipFlags(w[1:], "-a")
		case "xargs":
			w = skipFlags(w[1:], "-I", "-d", "-E", "-n", "-L", "-P", "-s", "-a")
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

// wordString is a word as the command sees it when it is literal (quotes removed); otherwise its source text.
func wordString(w *syntax.Word) string {
	var b strings.Builder
	for _, p := range w.Parts {
		if !literalPart(&b, p) {
			var src strings.Builder
			syntax.NewPrinter().Print(&src, w)
			return src.String()
		}
	}
	return b.String()
}

func literalPart(b *strings.Builder, p syntax.WordPart) bool {
	switch p := p.(type) {
	case *syntax.Lit:
		b.WriteString(p.Value)
	case *syntax.SglQuoted:
		b.WriteString(p.Value)
	case *syntax.DblQuoted:
		for _, q := range p.Parts {
			if !literalPart(b, q) {
				return false
			}
		}
	default:
		return false
	}
	return true
}
