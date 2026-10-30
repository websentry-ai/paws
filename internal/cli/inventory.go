package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

// inventory is what this machine has, sent with every upload: skill directory names (user
// and project) and the paws rules in scope, each with its skill and display actions.
type inventory struct {
	Skills []string        `json:"skills"`
	Rules  []inventoryRule `json:"rules"`
}

type inventoryRule struct {
	Name    string   `json:"name"`
	Skill   string   `json:"skill"`
	Actions []string `json:"actions"`
}

func inventoryFor(cwd string) *inventory {
	inv := &inventory{Skills: []string{}, Rules: []inventoryRule{}}
	seen := map[string]bool{}
	for _, dir := range []string{pack.UserTarget(gate.UserDir()).SkillsDir, pack.ProjectTarget(projectRoot(cwd)).SkillsDir} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			name := e.Name()
			if st, err := os.Stat(filepath.Join(dir, name)); err != nil || !st.IsDir() || strings.HasPrefix(name, ".") || seen[name] {
				continue
			}
			seen[name] = true
			inv.Skills = append(inv.Skills, name)
		}
	}
	sort.Strings(inv.Skills)
	for _, r := range gate.Collect(cwd).Rules {
		inv.Rules = append(inv.Rules, inventoryRule{Name: r.Name, Skill: r.Action.Skill, Actions: []string{displayAction(r)}})
	}
	sort.SliceStable(inv.Rules, func(i, j int) bool { return inv.Rules[i].Name < inv.Rules[j].Name })
	return inv
}

// displayAction is how the portal shows what a rule runs before: a portal rule's label, else
// a short reading of its matcher ("gh pr create", "edit migrations/**", "each new session").
func displayAction(r *rule.Rule) string {
	if managed(r.SourcePath) && r.Body != "" {
		return firstLine(r.Body)
	}
	m := r.Matcher
	switch {
	case rule.Kind(r.Event) == rule.KindSession:
		return "each new session"
	case m.CommandRegex != "":
		return regexDisplay(m.CommandRegex)
	case m.FileGlob != "" && m.Tool == "Read":
		return "read " + m.FileGlob
	case m.FileGlob != "":
		return "edit " + m.FileGlob
	case strings.HasPrefix(m.Tool, "mcp__"):
		return mcpDisplay(m.Tool, m.MCPAction)
	case m.MCPAction != "":
		return "MCP " + m.MCPAction
	case m.Tool != "" && m.Tool != "*":
		return strings.ToLower(m.Tool)
	}
	return "any tool call"
}

// mcpDisplay names the first server a tool matcher lists: "gmail send", "notion create-pages".
func mcpDisplay(tool, action string) string {
	first, _, _ := strings.Cut(tool, "|")
	server, name, ok := gate.MCPParts(first)
	if !ok {
		return strings.ToLower(first)
	}
	switch {
	case action != "":
		return server + " " + action
	case name == "*":
		return server
	}
	return server + " " + name
}

var (
	// regexPrefixes are the command-start guards catalog regexes open with.
	regexPrefixes = []string{
		`(^|[;&|\n])\s*(\w+=\S*\s+)*(sudo(\s+-\S+(\s+[^-\s]\S*)?)*\s+)?`,
		`(^|[;&|\n])\s*`,
		`^`,
	}
	regexWords   = strings.NewReplacer(`\s+`, " ", `\s*`, " ", `\s`, " ", `\b`, "", `$`, "", `\.`, ".")
	plainDisplay = regexp.MustCompile(`^[A-Za-z0-9 ._/|()-]+$`)
)

// regexDisplay reads a simple command regex as words: `(^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b`
// is "gh pr (create|edit)". A regex that doesn't reduce to plain words shows as "matching command".
func regexDisplay(re string) string {
	for _, p := range regexPrefixes {
		if rest, ok := strings.CutPrefix(re, p); ok {
			re = rest
			break
		}
	}
	s := strings.Join(strings.Fields(regexWords.Replace(re)), " ")
	if s == "" || len(s) > 60 || !plainDisplay.MatchString(s) {
		return "matching command"
	}
	return s
}
