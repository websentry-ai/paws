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
// ssh), or to a command paws doesn't know. So does a body written to a file the line then runs, sources, makes
// executable or hands to an interpreter (cat > t.sh <<EOF ... EOF; chmod +x t.sh; ./t.sh). So does every body on a
// line that defines a function of the reader's name, or runs eval, source or alias: the name may not be the program.
// In an unquoted body, $( ) still runs, so it stays too.

// Built-in git and gh commands that store or send the text they read, on stdin or as an argument, and run none of
// it. An alias can't take a built-in's name, so `git -c alias.run='!sh' run` and a gh shell alias stay out.
var (
	gitTextCmds = map[string]bool{"commit": true, "tag": true, "notes": true, "apply": true, "am": true,
		"hash-object": true, "interpret-trailers": true, "stripspace": true}
	ghTextCmds = map[string]bool{"pr": true, "issue": true, "api": true, "release": true, "gist": true,
		"label": true, "secret": true, "variable": true}
)

// sendsText reports whether the command w stores or sends what it reads.
func sendsText(w []string) bool {
	switch path.Base(w[0]) {
	case "git":
		return gitTextCmds[gitSubcommand(w)]
	case "gh":
		return len(w) > 1 && ghTextCmds[w[1]]
	}
	return false
}

// textCopiers print the text they read, so it goes wherever their output goes.
var textCopiers = map[string]bool{"cat": true, "tee": true, "echo": true, "printf": true}

// lineTree is a parsed line: each node's parent, the files the line may run, and the command names it may redefine.
type lineTree struct {
	up       map[syntax.Node]syntax.Node
	run      map[string]bool
	funcs    map[string]bool // cat() { sh; } makes cat run its input
	redefine bool            // eval, source or alias may define any function or alias
}

// dataText is line without the heredoc bodies that no shell runs. f is line parsed.
func dataText(line string, f *syntax.File) string {
	t := lineTree{up: map[syntax.Node]syntax.Node{}, run: map[string]bool{}, funcs: map[string]bool{}}
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
		case *syntax.FuncDecl:
			t.funcs[n.Name.Value] = true
		case *syntax.Stmt:
			if slices.ContainsFunc(n.Redirs, isHeredoc) {
				docs = append(docs, n)
			}
		case *syntax.CallExpr:
			var stdin []string // python3 < task.py
			if st, ok := t.up[n].(*syntax.Stmt); ok {
				for _, r := range st.Redirs {
					if r.Op == syntax.RdrIn && r.Word != nil {
						stdin = append(stdin, wordString(r.Word))
					}
				}
			}
			w := stripWrappers(callWords(n))
			t.noteRuns(w, stdin)
			if len(w) > 0 && (w[0] == "eval" || w[0] == "source" || w[0] == "." || w[0] == "alias") {
				t.redefine = true
			}
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

// interpreters run code from a file named as an argument or given on stdin.
var interpreters = map[string]bool{"python": true, "python2": true, "python3": true, "node": true, "deno": true,
	"bun": true, "tsx": true, "ts-node": true, "ruby": true, "perl": true, "php": true, "lua": true, "Rscript": true,
	"osascript": true, "go": true, "psql": true, "mysql": true, "sqlite3": true}

// noteRuns records the files a command may run: the command itself, a script a shell, source or an interpreter
// reads (as an argument or on stdin), and a file chmod changes (most often to make it executable). Files are known
// by base name, since a cd between writing and running one changes its path (cat > $S/t.go; cd $S && go run t.go).
func (t lineTree) noteRuns(w, stdin []string) {
	if len(w) == 0 {
		return
	}
	t.run[path.Base(w[0])] = true
	name := path.Base(w[0])
	if shells[name] || interpreters[name] || name == "source" || name == "." || name == "chmod" {
		for _, a := range slices.Concat(w[1:], stdin) {
			t.run[path.Base(a)] = true
		}
	}
}

// runs reports whether the line may run file.
func (t lineTree) runs(file string) bool { return t.run[path.Base(file)] }

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
	if slices.ContainsFunc(call.Args, hasProcSubst) {
		return false // tee >(sh) <<EOF
	}
	if t.redefine || t.funcs[w[0]] {
		return false // the name may not be the program paws knows
	}
	switch name := path.Base(w[0]); {
	case sendsText(w):
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
		switch kind, file := output(st); kind {
		case outFile:
			return !t.runs(file)
		case outOther:
			return false
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

// Where a command's output goes.
const (
	outInherit = iota // where the output of the command around it goes
	outFile
	outOther // a process, stderr or another descriptor: paws doesn't follow it
)

type outDest struct {
	kind int
	file string
}

// output follows st's redirections in order, as the shell does, and reports where its stdout ends up.
func output(st *syntax.Stmt) (kind int, file string) {
	fds := map[string]outDest{"1": {outInherit, ""}}
	for _, r := range st.Redirs {
		if r.Word == nil {
			continue
		}
		fd := "1"
		if r.N != nil {
			fd = r.N.Value
		}
		var d outDest
		switch r.Op {
		case syntax.RdrOut, syntax.AppOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll:
			d = outDest{outFile, wordString(r.Word)}
			if hasProcSubst(r.Word) || strings.HasPrefix(d.file, "/dev/") && d.file != "/dev/null" ||
				strings.HasPrefix(d.file, "/proc/") { // > >(sh), > /dev/stdout | sh
				d = outDest{outOther, ""}
			}
			if r.Op == syntax.RdrAll || r.Op == syntax.AppAll {
				fds["2"] = d
			}
		case syntax.DplOut: // 1>&3 copies where fd 3 goes; an unknown or closed fd isn't followed
			var ok bool
			if d, ok = fds[wordString(r.Word)]; !ok {
				d = outDest{outOther, ""}
			}
		default:
			continue
		}
		fds[fd] = d
	}
	d := fds["1"]
	return d.kind, d.file
}

func hasProcSubst(w *syntax.Word) bool {
	return slices.ContainsFunc(w.Parts, func(p syntax.WordPart) bool {
		_, ok := p.(*syntax.ProcSubst)
		return ok
	})
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
