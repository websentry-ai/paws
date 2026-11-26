package rule

import (
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Lines is the part of a file a read prints, by line number from 1: From through To, To 0 meaning to the end. A
// negative From is the file's last -From lines (tail -n N).
type Lines struct{ From, To int }

// AllLines is a read of the whole file.
var AllLines = Lines{From: 1}

// PlainRead returns the one file a Bash line does nothing but print, and the lines it prints: cat, head, tail, nl,
// less, more or bat on it, a sed -n print of a line range or an awk print of it, alone on the line. A line that
// chains, pipes, redirects, assigns or expands anything is not a plain read, so `cat f && git push` never passes for
// one; nor is a read whose lines can't be told (head -c, an awk program that filters).
func PlainRead(line string) (string, Lines, bool) {
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(strings.NewReader(line), "")
	if err != nil || len(f.Stmts) != 1 {
		return "", Lines{}, false
	}
	st := f.Stmts[0]
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || st.Negated || st.Background || st.Coprocess || len(st.Redirs) > 0 || len(call.Assigns) > 0 || len(call.Args) < 2 {
		return "", Lines{}, false
	}
	words := make([]string, len(call.Args))
	for i, w := range call.Args {
		if words[i], ok = literalWord(w); !ok {
			return "", Lines{}, false
		}
	}
	prog := words[0]
	if strings.Contains(prog, "/") && !systemBin.MatchString(prog) {
		return "", Lines{}, false // ./cat or /tmp/cat may be anything; only the system's readers are known to just read
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
	// sedPrint is a sed -n script that prints one line range: p, 5p, 241,480p, 10,$p.
	sedPrint = regexp.MustCompile(`^\s*(?:([1-9]\d*)\s*(?:,\s*([1-9]\d*|\$))?)?\s*p\s*$`)
	// awkPrint is an awk program that prints from the first line: 1, {print}, {print $0}, NR<=200, NR<200{print}.
	awkPrint = regexp.MustCompile(`^\s*(?:1|\{\s*print(?:\s+\$0)?\s*;?\s*\}|NR\s*(<=?)\s*([1-9]\d*)(?:\s*\{\s*print(?:\s+\$0)?\s*;?\s*\})?)\s*$`)
	// valueFlags take the next word as their value, per reader.
	valueFlags = map[string]map[string]bool{
		"head": {"-n": true},
		"tail": {"-n": true},
		"bat":  {"-l": true, "--language": true},
		"sed":  {"-e": true},
		"awk":  {"-F": true},
	}
	// readFlags are the only flags each reader may carry: ones that only change how the file prints. Anything else
	// can write (sed -i, gawk -i inplace, less -o), run a program (bat --pager, sed -f) or print lines that can't be
	// told (head -c).
	readFlags = map[string]*regexp.Regexp{
		"cat":  regexp.MustCompile(`^-[nbAeEstTuv]+$`),
		"nl":   regexp.MustCompile(`^-ba$`),
		"head": regexp.MustCompile(`^(-n|-[qv]|-\d+|-n\d+|--lines=\d+)$`),
		"tail": regexp.MustCompile(`^(-n|-[qv]|-\d+|-n\+?\d+|--lines=\+?\d+)$`),
		"less": regexp.MustCompile(`^-[RNSXF]+$`),
		"more": regexp.MustCompile(`^-[RNSXF]+$`),
		"bat":  regexp.MustCompile(`^(-[pnA]+|--(plain|number|show-all)|--paging=never|--(style|color)=[a-z,-]+|-l|--language(=\S+)?)$`),
		"sed":  regexp.MustCompile(`^(-[nEr]+|-[esu]|--(quiet|silent))$`),
		"awk":  regexp.MustCompile(`^-F$`),
	}
	// lineCount is a head or tail line count: N, or for tail +K (from line K on).
	lineCount = regexp.MustCompile(`^(\+?)(\d+)$`)
	readers   = map[string]bool{"cat": true, "head": true, "tail": true, "nl": true, "less": true, "more": true,
		"bat": true, "batcat": true, "sed": true, "awk": true}
)

// quiet is a sed flag that holds -n (alone or with -E/-r), or its long form: sed prints only what the script says.
func quiet(flag string) bool {
	return flag == "--quiet" || flag == "--silent" || !strings.HasPrefix(flag, "--") && strings.Contains(flag, "n")
}

// readTarget is the file a reader's arguments print and its lines, when they print exactly one file and nothing else.
func readTarget(prog string, args []string) (string, Lines, bool) {
	if !readers[prog] {
		return "", Lines{}, false
	}
	if prog == "batcat" {
		prog = "bat"
	}
	var operands, flags []string
	count := "10" // head's and tail's default
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			operands = append(operands, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "-") && a != "-":
			if !readFlags[prog].MatchString(a) {
				return "", Lines{}, false
			}
			flags = append(flags, a)
			if valueFlags[prog][a] {
				if i+1 >= len(args) {
					return "", Lines{}, false
				}
				i++
				if prog == "sed" {
					operands = append(operands, args[i]) // the script, checked below like a bare one
				} else {
					count = args[i]
				}
				continue
			}
			if prog == "head" || prog == "tail" {
				if n := strings.TrimLeft(strings.TrimPrefix(strings.TrimPrefix(a, "--lines="), "-n"), "-"); n != "" && n != "q" && n != "v" {
					count = n
				}
			}
		default:
			operands = append(operands, a)
		}
	}
	lines := AllLines
	switch prog {
	case "sed":
		m := sedPrint.FindStringSubmatch(firstOr(operands))
		if !slices.ContainsFunc(flags, quiet) || len(operands) != 2 || m == nil {
			return "", Lines{}, false
		}
		lines = sedLines(m)
		operands = operands[1:]
	case "awk":
		m := awkPrint.FindStringSubmatch(firstOr(operands))
		if len(operands) != 2 || m == nil {
			return "", Lines{}, false
		}
		if m[2] != "" {
			n, _ := strconv.Atoi(m[2])
			if m[1] == "<" {
				n--
			}
			lines = Lines{From: 1, To: n}
		}
		operands = operands[1:]
	case "head", "tail":
		m := lineCount.FindStringSubmatch(count)
		if m == nil {
			return "", Lines{}, false
		}
		n, err := strconv.Atoi(m[2])
		switch {
		case err != nil || n == 0 && m[1] == "":
			return "", Lines{}, false // nothing printed, or a count too big to be one
		case prog == "head" && m[1] == "":
			lines = Lines{From: 1, To: n}
		case prog == "head":
			return "", Lines{}, false
		case m[1] == "+":
			lines = Lines{From: max(n, 1)}
		default:
			lines = Lines{From: -n}
		}
	}
	if len(operands) != 1 || operands[0] == "-" || lines.To != 0 && lines.To < lines.From {
		return "", Lines{}, false
	}
	return operands[0], lines, true
}

// sedLines is the range a sedPrint match prints.
func sedLines(m []string) Lines {
	if m[1] == "" {
		return AllLines
	}
	from, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "":
		return Lines{From: from, To: from}
	case "$":
		return Lines{From: from}
	}
	to, _ := strconv.Atoi(m[2])
	return Lines{From: from, To: to}
}

func firstOr(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}
