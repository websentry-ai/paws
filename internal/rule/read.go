package rule

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// PlainRead returns the one file a Bash line does nothing but print: cat, head, tail, nl, less, more or bat on it,
// a sed -n print or an awk print of it, alone on the line. A line that chains, pipes, redirects, assigns or expands
// anything is not a plain read, so `cat f && git push` never passes for one.
func PlainRead(line string) (string, bool) {
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil || len(f.Stmts) != 1 {
		return "", false
	}
	st := f.Stmts[0]
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || st.Negated || st.Background || st.Coprocess || len(st.Redirs) > 0 || len(call.Assigns) > 0 || len(call.Args) < 2 {
		return "", false
	}
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		if words[i], ok = literalWord(w); !ok {
			return "", false
		}
	}
	prog := words[0]
	if strings.Contains(prog, "/") && !systemBin.MatchString(prog) {
		return "", false // ./cat or /tmp/cat may be anything; only the system's readers are known to just read
	}
	return readTarget(path.Base(prog), words[1:])
}

// systemBin is where the system's own readers live.
var systemBin = regexp.MustCompile(`^/(usr/(local/)?|opt/homebrew/)?bin/[a-z]+$`)

// literalWord is w's text when it holds no expansion: plain text and quotes only.
func literalWord(w *syntax.Word) (string, bool) {
	var b strings.Builder
	for _, p := range w.Parts {
		switch p := p.(type) {
		case *syntax.Lit:
			b.WriteString(unbackslash(p.Value, false))
		case *syntax.SglQuoted:
			if p.Dollar {
				return "", false
			}
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, q := range p.Parts {
				lit, ok := q.(*syntax.Lit)
				if !ok {
					return "", false
				}
				b.WriteString(unbackslash(lit.Value, true))
			}
		default:
			return "", false
		}
	}
	return b.String(), true
}

var (
	// sedPrint is a sed -n script that prints the file from its first line: p, 1,240p, 1,$p. A range that starts
	// later can print nothing.
	sedPrint = regexp.MustCompile(`^\s*(1\s*,\s*([1-9]\d*|\$)\s*)?p\s*$`)
	// awkPrint is an awk program that prints the file from its first line: 1, {print}, {print $0}, NR<=200,
	// NR<=200{print}.
	awkPrint = regexp.MustCompile(`^\s*(1|\{\s*print(\s+\$0)?\s*;?\s*\}|NR\s*(<=\s*[1-9]\d*|<\s*([2-9]|[1-9]\d+))(\s*\{\s*print(\s+\$0)?\s*;?\s*\})?)\s*$`)
	// valueFlags take the next word as their value, per reader.
	valueFlags = map[string]map[string]bool{
		"head": {"-n": true, "-c": true},
		"tail": {"-n": true, "-c": true},
		"bat":  {"-l": true, "--language": true},
		"sed":  {"-e": true},
		"awk":  {"-F": true},
	}
	// readFlags are the only flags each reader may carry: ones that only change how the file prints. Anything else
	// can write (sed -i, gawk -i inplace, less -o), run a program (bat --pager, sed -f) or print nothing (head -n 0).
	readFlags = map[string]*regexp.Regexp{
		"cat":  regexp.MustCompile(`^-[nbAeEstTuv]+$`),
		"nl":   regexp.MustCompile(`^-ba$`),
		"head": regexp.MustCompile(`^(-[nc]|-[qv]|-[1-9]\d*|-[nc][1-9]\d*|--(lines|bytes)=[1-9]\d*)$`),
		"tail": regexp.MustCompile(`^(-[nc]|-[qv]|-[1-9]\d*|-[nc]\+?[1-9]\d*|-[nc]\+0|--(lines|bytes)=\+?[1-9]\d*)$`),
		"less": regexp.MustCompile(`^-[RNSXF]+$`),
		"more": regexp.MustCompile(`^-[RNSXF]+$`),
		"bat":  regexp.MustCompile(`^(-[pnA]+|--(plain|number|show-all)|--paging=never|--(style|color)=[a-z,-]+|-l|--language(=\S+)?)$`),
		"sed":  regexp.MustCompile(`^(-[nEr]+|-[esu]|--(quiet|silent))$`),
		"awk":  regexp.MustCompile(`^-F$`),
	}
	// countValue is a head or tail -n/-c value that prints something: not 0.
	countValue = regexp.MustCompile(`^\+?[1-9]\d*$|^\+0$`)
	readers    = map[string]bool{"cat": true, "head": true, "tail": true, "nl": true, "less": true, "more": true,
		"bat": true, "batcat": true, "sed": true, "awk": true}
)

// quiet is a sed flag that holds -n (alone or with -E/-r), or its long form: sed prints only what the script says.
func quiet(flag string) bool {
	return flag == "--quiet" || flag == "--silent" || !strings.HasPrefix(flag, "--") && strings.Contains(flag, "n")
}

// readTarget is the file a reader's arguments print, when they print exactly one file and nothing else.
func readTarget(prog string, args []string) (string, bool) {
	if !readers[prog] {
		return "", false
	}
	if prog == "batcat" {
		prog = "bat"
	}
	var operands, flags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			operands = append(operands, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "-") && a != "-":
			if !readFlags[prog].MatchString(a) {
				return "", false
			}
			flags = append(flags, a)
			if valueFlags[prog][a] {
				if i+1 >= len(args) {
					return "", false
				}
				i++
				switch prog {
				case "sed":
					operands = append(operands, args[i]) // the script, checked below like a bare one
				case "head", "tail":
					if !countValue.MatchString(args[i]) || prog == "head" && strings.HasPrefix(args[i], "+") {
						return "", false
					}
				}
			}
		default:
			operands = append(operands, a)
		}
	}
	switch prog {
	case "sed":
		if !slices.ContainsFunc(flags, quiet) || len(operands) != 2 || !sedPrint.MatchString(operands[0]) {
			return "", false
		}
		operands = operands[1:]
	case "awk":
		if len(operands) != 2 || !awkPrint.MatchString(operands[0]) {
			return "", false
		}
		operands = operands[1:]
	}
	if len(operands) != 1 || operands[0] == "-" {
		return "", false
	}
	return operands[0], true
}
