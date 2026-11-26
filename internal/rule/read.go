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
	return readTarget(path.Base(words[0]), words[1:])
}

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
	// sedPrint is a sed -n script that only prints a line range: p, 5p, 1,240p, 10,$p.
	sedPrint = regexp.MustCompile(`^\s*(\d+|\$)?\s*(,\s*(\d+|\$))?\s*p\s*$`)
	// awkPrint is an awk program that only prints: 1, {print}, {print $0}, NR<=200, NR<=200{print}.
	awkPrint = regexp.MustCompile(`^\s*(1|\{\s*print(\s+\$0)?\s*;?\s*\}|NR\s*(<|<=|>|>=|==)\s*\d+(\s*\{\s*print(\s+\$0)?\s*;?\s*\})?)\s*$`)
	// valueFlags take the next word as their value, per reader.
	valueFlags = map[string]map[string]bool{
		"head": {"-n": true, "-c": true},
		"tail": {"-n": true, "-c": true},
		"bat": {"-r": true, "-l": true, "-H": true, "-m": true, "--line-range": true, "--language": true,
			"--highlight-line": true, "--style": true, "--theme": true, "--paging": true, "--wrap": true, "--tabs": true,
			"--color": true, "--decorations": true, "--terminal-width": true, "--map-syntax": true},
		"awk": {"-F": true},
	}
	readers = map[string]bool{"cat": true, "head": true, "tail": true, "nl": true, "less": true, "more": true,
		"bat": true, "batcat": true, "sed": true, "awk": true}
)

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
			flags = append(flags, a)
			if valueFlags[prog][a] || prog == "sed" && a == "-e" {
				if i+1 >= len(args) {
					return "", false
				}
				i++
				if prog == "sed" {
					operands = append(operands, args[i]) // the script, checked below like a bare one
				}
			}
		default:
			operands = append(operands, a)
		}
	}
	switch prog {
	case "sed":
		if !slices.Contains(flags, "-n") || len(operands) != 2 || !sedPrint.MatchString(operands[0]) {
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
