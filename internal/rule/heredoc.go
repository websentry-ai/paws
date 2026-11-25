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
// ssh), or to a command paws doesn't know. A body written to a file is data only when every command on the line is
// one that runs no file (cd, mkdir, cat, wc, built-in git and gh commands, ...), and the file isn't one that tools
// run without being told its name (Makefile, package.json, a dotfile, a git hook). Any other command could run the
// file (./t.sh, python3 task.py, make, go run .), so the body stays. So does every body on a line that defines a
// function of the reader's name, or runs eval, source or alias: the name may not be the program. In an unquoted
// body, $( ) still runs, so it stays too.

// Built-in git and gh commands that store or send the text they read, on stdin or as an argument, and run none of
// it. An alias can't take a built-in's name, so `git -c alias.run='!sh' run` and a gh shell alias stay out.
var (
	gitTextCmds = map[string]bool{"commit": true, "tag": true, "notes": true, "hash-object": true,
		"interpret-trailers": true, "stripspace": true} // not apply or am: a patch can write a script
	ghTextCmds = map[string]bool{"pr": true, "issue": true, "api": true, "release": true, "gist": true,
		"label": true, "secret": true, "variable": true}
)

// sendsText reports whether the command w stores or sends what it reads.
func sendsText(w []string) bool {
	switch w[0] {
	case "git":
		return gitTextCmds[gitSubcommand(w)]
	case "gh":
		return len(w) > 1 && ghTextCmds[w[1]]
	}
	return false
}

// textCopiers print the text they read, so it goes wherever their output goes.
var textCopiers = map[string]bool{"cat": true, "tee": true, "echo": true, "printf": true}

// lineTree is a parsed line: each node's parent and the command names it may redefine. files collects the files the
// body being judged is written to.
type lineTree struct {
	up       map[syntax.Node]syntax.Node
	funcs    map[string]bool // cat() { sh; } makes cat run its input
	redefine bool            // eval, source or alias may define any function or alias
	runsNone bool            // every command on the line is in runsNoFile
	files    []string
}

// dataBody is a heredoc body judged to be data, and the files it is written to.
type dataBody struct {
	cuts  [][2]int
	files []string
}

// newLineTree reads f, and returns it with the statements that read a heredoc.
func newLineTree(f *syntax.File) (*lineTree, []*syntax.Stmt) {
	t := &lineTree{up: map[syntax.Node]syntax.Node{}, funcs: map[string]bool{}, runsNone: true}
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
			t.runsNone = false
		case *syntax.CoprocClause:
			t.runsNone = false
		case *syntax.Stmt:
			if slices.ContainsFunc(n.Redirs, isHeredoc) {
				docs = append(docs, n)
			}
		case *syntax.CallExpr:
			words := callWords(n)
			w := stripWrappers(words)
			if len(w) > 0 && (w[0] == "eval" || w[0] == "source" || w[0] == "." || w[0] == "alias") {
				t.redefine = true
			}
			if len(words) > 0 && (len(n.Assigns) > 0 || len(w) == 0 || !runsNoFile(w) ||
				slices.ContainsFunc(words[:len(words)-len(w)], func(a string) bool {
					return strings.Contains(a, "=") || path.Base(a) == "env" || path.Base(a) == "xargs"
				})) {
				t.runsNone = false // VAR=x before a command (GIT_SSH_COMMAND=...), env, xargs, or a command that may run a file
			}
		}
		return true
	})
	return t, docs
}

// dataText is line without the heredoc bodies that no shell runs. f is line parsed.
func dataText(line string, f *syntax.File) string {
	t, docs := newLineTree(f)
	var bodies []dataBody
	for _, st := range docs {
		t.files = nil
		if !t.takesAsData(st) {
			continue
		}
		d := dataBody{files: t.files}
		for _, r := range st.Redirs {
			if isHeredoc(r) {
				d.cuts = append(d.cuts, bodyCuts(line, r)...)
			}
		}
		bodies = append(bodies, d)
	}
	// A body written to a file may run later on the line, unless nothing on it runs a file.
	bodies = slices.DeleteFunc(bodies, func(d dataBody) bool {
		return len(d.files) > 0 && (!t.runsNone || slices.ContainsFunc(d.files, runsUnnamed))
	})
	return cut(line, bodies)
}

// cut removes the bodies' text from line. Newlines stay, so the line keeps its shape and still parses.
func cut(line string, bodies []dataBody) string {
	var cuts [][2]int
	for _, d := range bodies {
		cuts = append(cuts, d.cuts...)
	}
	if len(cuts) == 0 {
		return line
	}
	sort.Slice(cuts, func(i, j int) bool { return cuts[i][0] < cuts[j][0] })
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

// noFileCmds run no file the line may have written: they read, list, print, move or send files and text.
var noFileCmds = map[string]bool{"cd": true, "pwd": true, "mkdir": true, "touch": true, "rm": true, "rmdir": true,
	"ls": true, "wc": true, "cat": true, "tee": true, "echo": true, "printf": true, "head": true, "tail": true,
	"grep": true, "rg": true, "sort": true, "uniq": true, "diff": true, "cmp": true, "stat": true, "file": true,
	"du": true, "true": true, "false": true, ":": true, "test": true, "[": true, "sleep": true, "date": true,
	"basename": true, "dirname": true, "realpath": true, "shasum": true, "sha256sum": true, "md5": true, "jq": true}

// gitNoFileCmds are built-in git commands; the hooks they run are files runsUnnamed keeps.
var gitNoFileCmds = map[string]bool{"add": true, "status": true, "diff": true, "log": true, "show": true,
	"commit": true, "tag": true, "notes": true, "rm": true, "mv": true, "restore": true, "switch": true,
	"checkout": true, "branch": true, "fetch": true, "pull": true, "push": true, "stash": true, "reset": true,
	"rev-parse": true, "ls-files": true, "hash-object": true, "interpret-trailers": true, "stripspace": true}

// runsNoFile reports whether command w runs no file. A path (./cat) may be any program; git -c may set a command
// git runs (core.sshCommand, alias.x=!sh).
func runsNoFile(w []string) bool {
	if strings.Contains(w[0], "/") {
		return false
	}
	switch w[0] {
	case "git":
		return gitNoFileCmds[gitSubcommand(w)] && !slices.ContainsFunc(w[1:], func(a string) bool {
			return a == "-c" || a == "--config-env" || a == "--exec-path" || strings.HasPrefix(a, "--exec-path=")
		})
	case "gh":
		return len(w) > 1 && ghTextCmds[w[1]]
	}
	return noFileCmds[w[0]]
}

// runsUnnamed reports whether tools may run file as shell without being given its name: make, just, rake and task
// read their recipe files, npm its package.json scripts, docker its Dockerfile, a shell its dotfiles, git its hooks.
// A name paws can't read ($F) may be any of them.
func runsUnnamed(file string) bool {
	name := path.Base(file)
	switch name {
	case "Makefile", "makefile", "GNUmakefile", "Justfile", "justfile", "Rakefile", "Taskfile.yml", "Taskfile.yaml",
		"package.json", "Dockerfile", "Containerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml",
		"compose.yaml", "Procfile", "applypatch-msg", "pre-applypatch", "post-applypatch", "pre-commit",
		"pre-merge-commit", "prepare-commit-msg", "commit-msg", "post-commit", "pre-rebase", "post-checkout",
		"post-merge", "pre-push", "post-rewrite", "reference-transaction", "pre-auto-gc", "fsmonitor-watchman":
		return true
	}
	return strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".mk") || strings.ContainsAny(name, "$`*?[")
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

// takesAsData reports whether the text st reads, on stdin or as an argument, can't reach a shell.
func (t *lineTree) takesAsData(st *syntax.Stmt) bool {
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
	if t.redefine || t.funcs[w[0]] || strings.Contains(w[0], "/") {
		return false // the name may not be the program paws knows: cat() { sh; }, ./cat
	}
	switch name := w[0]; {
	case name == "printf" && slices.ContainsFunc(w[1:], func(a string) bool { return strings.HasPrefix(a, "-v") }):
		return false // printf -v script stores the text in a variable the line may run
	case sendsText(w):
		return t.printsAsData(st) // git stripspace <<EOF | sh prints the body
	case name == "tee":
		for _, a := range w[1:] {
			if strings.HasPrefix(a, "/dev/") && a != "/dev/null" || strings.HasPrefix(a, "/proc/") {
				return false // tee /dev/fd/3 3> >(sh)
			}
			if !strings.HasPrefix(a, "-") {
				t.files = append(t.files, a)
			}
		}
		return t.printsAsData(st)
	case textCopiers[name]:
		return t.printsAsData(st)
	}
	return false
}

// printsAsData reports whether what st prints can't reach a shell: it goes to a file (noted in t.files), to the
// terminal, or to a command that takes it as data.
func (t *lineTree) printsAsData(st *syntax.Stmt) bool {
	for {
		switch kind, file := output(st); kind {
		case outFile:
			t.files = append(t.files, file)
			return true
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
		case syntax.RdrInOut: // 1<>t.sh opens t.sh for writing too; <>f alone is stdin
			if r.N == nil {
				continue
			}
			d = outDest{outFile, wordString(r.Word)}
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
