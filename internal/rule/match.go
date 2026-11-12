package rule

import (
	"regexp"
	"strings"
)

// gitAnchor requires git to start a command: at the start or after ;/&/|/newline (optionally opening a ( subshell),
// inside $( ) (which runs even within double quotes), or as the script a shell runs (bash -c "...", sh -lc '...').
// Echoed text and commit messages can't trip it, "(" in a quoted message included.
const gitAnchor = `((^|[;&|\n])\s*\(?|\$\(|\b(ba|z|da|k)?sh(\s+-\w+)*\s+-\w*c\s+["']?)\s*`
const gitFlags = `(\s+-C\s+\S+|\s+--?[\w-]+(=\S+)?)*`

// gitEnd ends the subcommand word (space, ;, a redirect, a paren...); \b alone would let
// `commit-graph` match. `commit-tree` makes a real commit, so pre-commit names it explicitly.
const gitEnd = `($|[^\w-])`

var (
	gitCommitRe = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+commit(-tree)?` + gitEnd)
	gitPushRe   = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+push` + gitEnd)
)

// fileTools carry a file path target (file_path, notebook_path), the only tools a file_glob can match.
var fileTools = map[string]bool{"Read": true, "Write": true, "Edit": true, "NotebookEdit": true}

// Matches reports whether this rule applies to a call by tool, against target (the extracted command or prompt).
func (r *Rule) Matches(tool, target string) bool {
	if !matchTool(r.Matcher.Tool, tool) {
		return false
	}
	if r.Matcher.MCPAction != "" && !matchMCPAction(r.Matcher.MCPAction, tool) {
		return false
	}
	if r.fileRe != nil && (!fileTools[tool] || !r.fileRe.MatchString(target)) {
		return false
	}
	if r.re != nil && !r.re.MatchString(target) {
		return false
	}
	switch r.Event {
	case "pre-commit":
		return tool == "Bash" && gitCommitRe.MatchString(target)
	case "pre-push":
		return tool == "Bash" && gitPushRe.MatchString(target)
	}
	return true
}

// matchTool matches a matcher tool spec ("", "*", or a pipe list of names where * stands for any run of
// characters, such as "mcp__slack__*" or "mcp__*__get_issue") against a tool.
func matchTool(spec, tool string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "*" {
		return true
	}
	for _, p := range strings.Split(spec, "|") {
		if matchStar(strings.TrimSpace(p), tool) {
			return true
		}
	}
	return false
}

// matchStar reports whether s matches pattern p, where each * stands for any run of characters.
func matchStar(p, s string) bool {
	parts := strings.Split(p, "*")
	if len(parts) == 1 {
		return p == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	last := parts[len(parts)-1]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(s, mid)
		if i < 0 {
			return false
		}
		s = s[i+len(mid):]
	}
	return strings.HasSuffix(s, last)
}
