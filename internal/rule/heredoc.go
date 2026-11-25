package rule

import (
	"path"
	"slices"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// A heredoc's body is part of the Bash line, so the whole-line regexes read it as commands: notes written with
// `cat <<'EOF' > notes.md` that mention git push, or a PR body sent with `gh pr create --body-file - <<EOF` that
// mentions rm -rf, were stopped as if they ran them. dataText cuts the bodies no shell runs out of the line before
// those regexes read it.
//
// A body is data when the command reading it only stores or sends text (git commit -F -, gh --body-file -), or copies
// it (cat, tee, echo, printf) to a file, to the terminal, or on to such a command, through a pipe or a $( ) argument.
// Every other body stays: one fed to a shell (bash <<EOF, cat <<EOF | sh), to an interpreter (psql, python, node,
// ssh), or to a command paws doesn't know. So does a body written to a file the line then runs, sources or makes
// executable (cat > t.sh <<EOF ... EOF; chmod +x t.sh; ./t.sh). In an unquoted body, $( ) still runs, so it stays too.

// textSenders store or send the text they read, on stdin or as an argument, and run none of it.
var textSenders = map[string]bool{"git": true, "gh": true}

// textCopiers print the text they read, so it goes wherever their output goes.
var textCopiers = map[string]bool{"cat": true, "tee": true, "echo": true, "printf": true}

// lineTree is a parsed line: each node's parent, and the files the line may run.
type lineTree struct {
	up  map[syntax.Node]syntax.Node
	run map[string]bool
}

// dataText is line without the heredoc bodies that no shell runs. f is line parsed.
func dataText(line string, f *syntax.File) string {
	t := lineTree{up: map[syntax.Node]syntax.Node{}, run: map[string]bool{}}
	var stack []syntax.Node
	var docs []*syntax.Stmt
	syntax.Walk(f, func(n syntax.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			t.up[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		switch n := n.(type) {
		case *syntax.Stmt:
			if slices.ContainsFunc(n.Redirs, isHeredoc) {
				docs = append(docs, n)
			}
		case *syntax.CallExpr:
			t.noteRuns(stripWrappers(callWords(n)))
		}
		return true
	})
	var cuts [][2]int
	for _, st := range docs {
		if !t.takesAsData(st) {
			continue
		}
		for _, r := range st.Redirs {
			if isHeredoc(r) {
				cuts = append(cuts, bodyCuts(line, r)...)
			}
		}
	}
	if len(cuts) == 0 {
		return line
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i][0] < cuts[j][0] })
	// Newlines stay, so the line keeps its shape and still parses.
	var b strings.Builder
	at := 0
	for _, c := range cuts {
		if c[0] < at {
			continue
		}
		b.WriteString(line[at:c[0]])
		b.WriteString(strings.Repeat("\n", strings.Count(line[c[0]:c[1]], "\n")))
		at = c[1]
	}
	b.WriteString(line[at:])
	return b.String()
}

// bodyCuts are the parts of a heredoc's body that are plain text. $( ) in an unquoted body runs, so it stays.
func bodyCuts(line string, r *syntax.Redirect) [][2]int {
	doc := r.Hdoc
	// The body's last part runs on through the closing delimiter's line; the cut stops before that line.
	start, end := int(doc.Pos().Offset()), int(doc.End().Offset())
	if start < 0 || end > len(line) || start > end {
		return nil
	}
	if nl := strings.LastIndexByte(line[start:end], '\n'); nl >= 0 && strings.TrimLeft(line[start+nl+1:end], "\t") == wordString(r.Word) {
		end = start + nl + 1
	}
	var cuts [][2]int
	for i, p := range doc.Parts {
		if _, ok := p.(*syntax.Lit); !ok {
			continue
		}
		s, e := int(p.Pos().Offset()), min(int(p.End().Offset()), end)
		if i+1 < len(doc.Parts) {
			e = min(e, int(doc.Parts[i+1].Pos().Offset()))
		}
		if s < e {
			cuts = append(cuts, [2]int{s, e})
		}
	}
	return cuts
}

func isHeredoc(r *syntax.Redirect) bool {
	return (r.Op == syntax.Hdoc || r.Op == syntax.DashHdoc) && r.Hdoc != nil
}

// noteRuns records the files a command may run: the command itself, a script a shell or source reads, and a file
// chmod changes (most often to make it executable). Paths are cleaned, so ./t.sh and t.sh are one file.
func (t lineTree) noteRuns(w []string) {
	if len(w) == 0 {
		return
	}
	t.run[path.Clean(w[0])] = true
	if name := path.Base(w[0]); shells[name] || name == "source" || name == "." || name == "chmod" {
		for _, a := range w[1:] {
			t.run[path.Clean(a)] = true
		}
	}
}

// runs reports whether the line may run file.
func (t lineTree) runs(file string) bool { return t.run[path.Clean(file)] }

// takesAsData reports whether the text st reads, on stdin or as an argument, can't reach a shell.
func (t lineTree) takesAsData(st *syntax.Stmt) bool {
	if st.Cmd == nil {
		return t.printsAsData(st) // `> notes.md <<EOF` only writes a file
	}
	call, ok := st.Cmd.(*syntax.CallExpr)
	if !ok {
		return false // while read ...; done <<EOF, { ...; } and other compound commands
	}
	words := callWords(call)
	if len(words) == 0 {
		return false // X=$(cat <<EOF): the variable may run later
	}
	w := stripWrappers(words)
	if len(w) == 0 || slices.ContainsFunc(words[:len(words)-len(w)], func(a string) bool { return path.Base(a) == "xargs" }) {
		return false // xargs git <<EOF: the body becomes the command's arguments
	}
	switch name := path.Base(w[0]); {
	case textSenders[name]:
		return true
	case name == "tee" && slices.ContainsFunc(w[1:], t.runs):
		return false // tee t.sh <<EOF ... EOF; bash t.sh
	case textCopiers[name]:
		return t.printsAsData(st)
	}
	return false
}

// printsAsData reports whether what st prints can't reach a shell: it goes to a file the line doesn't run, to the
// terminal, or to a command that takes it as data.
func (t lineTree) printsAsData(st *syntax.Stmt) bool {
	for {
		if file, ok := output(st); ok {
			return file != "" && !t.runs(file)
		}
		switch p := t.up[st].(type) {
		case *syntax.File:
			return true // the terminal: Claude reads it, nothing runs it
		case *syntax.BinaryCmd:
			if (p.Op == syntax.Pipe || p.Op == syntax.PipeAll) && p.X == st {
				return t.takesAsData(pipeEnd(p.Y, false))
			}
		case *syntax.CmdSubst, *syntax.ProcSubst: // an argument (or a file) of the command around it
			outer := enclosingStmt(t.up[p], t.up)
			return outer != nil && t.takesAsData(outer)
		case *syntax.FuncDecl:
			return false // the caller's output: paws doesn't follow it
		}
		// Inside { }, ( ), if, && ...: the output is the enclosing command's.
		if st = enclosingStmt(t.up[st], t.up); st == nil {
			return false
		}
	}
}

// output reports whether st redirects its output, and to which file: "" for a process (> >(sh)).
func output(st *syntax.Stmt) (file string, ok bool) {
	for _, r := range st.Redirs {
		switch r.Op {
		case syntax.RdrOut, syntax.AppOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll:
			if r.N != nil && r.N.Value != "1" || r.Word == nil {
				continue
			}
			for _, p := range r.Word.Parts {
				if _, ok := p.(*syntax.ProcSubst); ok {
					return "", true
				}
			}
			return wordString(r.Word), true
		}
	}
	return "", false
}

// enclosingStmt is n when it is a statement, else the nearest statement around it.
func enclosingStmt(n syntax.Node, up map[syntax.Node]syntax.Node) *syntax.Stmt {
	for n != nil {
		if st, ok := n.(*syntax.Stmt); ok {
			return st
		}
		n = up[n]
	}
	return nil
}
