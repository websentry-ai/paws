package rule

import (
	"regexp"
	"strings"
)

// gitAnchor requires git to start a command (or follow ;/&/|/newline) so echoed text or commit messages can't trip it.
const gitAnchor = `(^|[;&|\n])\s*`
const gitFlags = `(\s+-C\s+\S+|\s+--?[\w-]+(=\S+)?)*`

var (
	gitCommitRe = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+commit\b`)
	gitPushRe   = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+push\b`)
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

// matchTool matches a matcher tool spec ("", "*", or a pipe list of names and "prefix*" patterns) against a tool.
func matchTool(spec, tool string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "*" {
		return true
	}
	for _, p := range strings.Split(spec, "|") {
		p = strings.TrimSpace(p)
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(tool, prefix) {
				return true
			}
		} else if p == tool {
			return true
		}
	}
	return false
}
