package rule

import (
	"regexp"
	"strings"
)

// gitAnchor requires git to start a command (or follow ;/&/|) so echoed text or commit messages can't trip it.
const gitAnchor = `(^|[;&|])\s*`
const gitFlags = `(\s+-C\s+\S+|\s+--?[\w-]+(=\S+)?)*`

var (
	gitCommitRe = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+commit\b`)
	gitPushRe   = regexp.MustCompile(gitAnchor + `git` + gitFlags + `\s+push\b`)
)

// Matches reports whether this rule applies to a call by tool, against target (the extracted command or prompt).
func (r *Rule) Matches(tool, target string) bool {
	if !matchTool(r.Matcher.Tool, tool) {
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

// matchTool matches a matcher tool spec ("", "*", or a pipe list) against a tool.
func matchTool(spec, tool string) bool {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "*" {
		return true
	}
	for _, p := range strings.Split(spec, "|") {
		if strings.TrimSpace(p) == tool {
			return true
		}
	}
	return false
}
