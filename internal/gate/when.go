package gate

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/websentry-ai/paws/internal/rule"
)

// Applies reports whether r matches ev and every `when:` condition holds.
func Applies(r *rule.Rule, ev Event) bool {
	if ev.Kind == "" || rule.Kind(r.Event) != ev.Kind || !r.Matches(ev.Tool, ev.Target) {
		return false
	}
	for _, c := range r.When {
		if !conditionHolds(c, ev) {
			return false
		}
	}
	return true
}

func conditionHolds(c rule.Condition, ev Event) bool {
	switch c.Kind {
	case rule.WhenDependency:
		return hasDependency(repoRoot(ev.Cwd), c.Arg)
	case rule.WhenRepoMarker:
		_, err := os.Stat(filepath.Join(repoRoot(ev.Cwd), c.Arg))
		return err == nil
	case rule.WhenNewFile:
		path := ev.Target
		if !fileTool(ev.Tool) || path == "" {
			return false
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(ev.Cwd, path)
		}
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	case rule.WhenNoEditMatching:
		return !editedMatching(ev.Transcript, c.Glob)
	case rule.WhenAgentCount:
		return agentsSinceUserTurn(ev.Transcript) > c.N
	}
	return false
}

func fileTool(tool string) bool {
	return tool == "Read" || tool == "Write" || tool == "Edit" || tool == "NotebookEdit"
}

// repoRoot is the nearest ancestor of cwd holding .git, or cwd itself.
func repoRoot(cwd string) string {
	for cur := cwd; cur != ""; {
		if hasGit(cur) {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return cwd
}

// hasDependency looks for pkg in the manifests at root: package.json (dependencies,
// devDependencies, peerDependencies, optionalDependencies), go.mod direct requires (full
// module path, or its last element without a /vN suffix), Cargo.toml dependency tables
// (crate key or its package rename), and pyproject.toml dependency tables or
// requirements.txt (case-insensitive, - _ . equivalent).
func hasDependency(root, pkg string) bool {
	return packageJSONHas(filepath.Join(root, "package.json"), pkg) ||
		goModHas(filepath.Join(root, "go.mod"), pkg) ||
		cargoHas(filepath.Join(root, "Cargo.toml"), pkg) ||
		pyprojectHas(filepath.Join(root, "pyproject.toml"), pkg) ||
		requirementsHas(filepath.Join(root, "requirements.txt"), pkg)
}

func packageJSONHas(path, pkg string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil {
		return false
	}
	for _, field := range []string{"dependencies", "devDependencies", "peerDependencies", "optionalDependencies"} {
		var deps map[string]any
		if json.Unmarshal(m[field], &deps) == nil {
			if _, ok := deps[pkg]; ok {
				return true
			}
		}
	}
	return false
}

var goMajorSuffix = regexp.MustCompile(`/v[0-9]+$`)

// goModHas checks direct requires only: indirect deps, replace and exclude lines don't count.
func goModHas(path, pkg string) bool {
	found, inRequire := false, false
	eachLine(path, func(line string) bool {
		switch {
		case line == ")":
			inRequire = false
			return true
		case strings.HasPrefix(line, "require ("):
			inRequire = true
			return true
		case strings.HasPrefix(line, "require "):
			line = strings.TrimPrefix(line, "require ")
		case !inRequire:
			return true
		}
		if strings.Contains(line, "// indirect") {
			return true
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			return true
		}
		mod := f[0]
		if mod == pkg || filepath.Base(goMajorSuffix.ReplaceAllString(mod, "")) == pkg {
			found = true
		}
		return !found
	})
	return found
}

// cargoHas checks [dependencies], [dev-dependencies], [build-dependencies] and their
// [target.*.] forms, matching the crate key or a `package = "..."` rename.
func cargoHas(path, pkg string) bool {
	var doc map[string]any
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return false
	}
	tables := []any{doc["dependencies"], doc["dev-dependencies"], doc["build-dependencies"]}
	if targets, ok := doc["target"].(map[string]any); ok {
		for _, t := range targets {
			if tm, ok := t.(map[string]any); ok {
				tables = append(tables, tm["dependencies"], tm["dev-dependencies"], tm["build-dependencies"])
			}
		}
	}
	for _, t := range tables {
		deps, _ := t.(map[string]any)
		for key, v := range deps {
			if key == pkg {
				return true
			}
			if spec, ok := v.(map[string]any); ok && spec["package"] == pkg {
				return true
			}
		}
	}
	return false
}

var pyReqName = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._-]*)`)

// pyprojectHas checks [project] dependencies and optional-dependencies, [dependency-groups],
// and poetry's dependency tables — not keywords, tool config or the project's own name.
func pyprojectHas(path, pkg string) bool {
	var doc struct {
		Project struct {
			Dependencies         []string            `toml:"dependencies"`
			OptionalDependencies map[string][]string `toml:"optional-dependencies"`
		} `toml:"project"`
		DependencyGroups map[string][]any `toml:"dependency-groups"`
		Tool             struct {
			Poetry struct {
				Dependencies    map[string]any `toml:"dependencies"`
				DevDependencies map[string]any `toml:"dev-dependencies"`
				Group           map[string]struct {
					Dependencies map[string]any `toml:"dependencies"`
				} `toml:"group"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return false
	}
	want := normPy(pkg)
	reqs := append([]string{}, doc.Project.Dependencies...)
	for _, group := range doc.Project.OptionalDependencies {
		reqs = append(reqs, group...)
	}
	for _, group := range doc.DependencyGroups {
		for _, item := range group {
			if s, ok := item.(string); ok { // {include-group = ...} tables name groups, not packages
				reqs = append(reqs, s)
			}
		}
	}
	for _, r := range reqs {
		if m := pyReqName.FindStringSubmatch(r); m != nil && normPy(m[1]) == want {
			return true
		}
	}
	poetry := []map[string]any{doc.Tool.Poetry.Dependencies, doc.Tool.Poetry.DevDependencies}
	for _, g := range doc.Tool.Poetry.Group {
		poetry = append(poetry, g.Dependencies)
	}
	for _, deps := range poetry {
		for name := range deps {
			if normPy(name) == want {
				return true
			}
		}
	}
	return false
}

func requirementsHas(path, pkg string) bool {
	want, found := normPy(pkg), false
	eachLine(path, func(line string) bool {
		if strings.HasPrefix(line, "-") { // -r other.txt, -e ., --index-url
			return true
		}
		if m := pyReqName.FindStringSubmatch(line); m != nil && normPy(m[1]) == want {
			found = true
		}
		return !found
	})
	return found
}

func normPy(s string) string {
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(s))
}

// eachLine calls fn with each trimmed, non-comment line until fn returns false.
func eachLine(path string, fn func(string) bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if !fn(line) {
			return
		}
	}
}

// toolUse is one tool_use block inside an assistant transcript message. Input decodes only
// the fields conditions read, never a Write's whole content.
type toolUse struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input struct {
		FilePath string `json:"file_path"`
	} `json:"input"`
}

// toolResult is one tool_result block inside a user transcript message.
type toolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	IsError   bool   `json:"is_error"`
}

type contentMessage struct {
	Content json.RawMessage `json:"content"`
}

// mainEntry is a main-thread transcript line, decoded once for both conditions.
type mainEntry struct {
	Type             string         `json:"type"`
	IsSidechain      bool           `json:"isSidechain"`
	IsMeta           bool           `json:"isMeta"`
	IsCompactSummary bool           `json:"isCompactSummary"`
	Message          contentMessage `json:"message"`
}

// editedMatching reports whether the main thread already edited or wrote a path matching glob,
// successfully, in the transcript tail: an Edit/Write whose tool_result is not an error. Denied
// and failed edits don't count. No transcript means no known edit.
func editedMatching(path string, glob *regexp.Regexp) bool {
	lines, err := tailLines(path)
	if err != nil {
		return false
	}
	pending := map[string]bool{} // tool_use ids of matching edits awaiting a result
	for _, ln := range lines {
		var e mainEntry
		if json.Unmarshal(ln, &e) != nil || e.IsSidechain {
			continue
		}
		switch e.Type {
		case "assistant":
			var blocks []toolUse
			if json.Unmarshal(e.Message.Content, &blocks) != nil {
				continue
			}
			for _, b := range blocks {
				if b.Type == "tool_use" && (b.Name == "Edit" || b.Name == "Write") && b.Input.FilePath != "" && glob.MatchString(b.Input.FilePath) {
					pending[b.ID] = true
				}
			}
		case "user":
			var blocks []toolResult
			if json.Unmarshal(e.Message.Content, &blocks) != nil {
				continue
			}
			for _, b := range blocks {
				if b.Type == "tool_result" && pending[b.ToolUseID] && !b.IsError {
					return true
				}
			}
		}
	}
	return false
}

// agentsSinceUserTurn counts Agent (formerly Task) spawns after the last prompt the user typed.
func agentsSinceUserTurn(path string) int {
	lines, err := tailLines(path)
	if err != nil {
		return 0
	}
	n := 0
	for i := len(lines) - 1; i >= 0; i-- {
		var e mainEntry
		if json.Unmarshal(lines[i], &e) != nil || e.IsSidechain {
			continue
		}
		if isUserPrompt(e) {
			break
		}
		if e.Type != "assistant" {
			continue
		}
		var blocks []toolUse
		if json.Unmarshal(e.Message.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" && (b.Name == "Agent" || b.Name == "Task") {
				n++
			}
		}
	}
	return n
}

// harnessPrefixes mark user-role records Claude Code writes itself: slash-command output,
// task notifications, shell escapes. They are not prompts the user typed.
var harnessPrefixes = []string{"<local-command", "<command-", "<task-notification", "<bash-", "<system-reminder"}

// isUserPrompt reports whether a transcript entry is a prompt the user typed, not a tool
// result, a compaction summary or harness output.
func isUserPrompt(e mainEntry) bool {
	if e.Type != "user" || e.IsMeta || e.IsCompactSummary {
		return false
	}
	var text string
	if json.Unmarshal(e.Message.Content, &text) == nil {
		return typed(text)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return false
		}
		if b.Type == "text" && typed(b.Text) {
			return true
		}
	}
	return false
}

func typed(text string) bool {
	t := strings.TrimSpace(text)
	for _, p := range harnessPrefixes {
		if strings.HasPrefix(t, p) {
			return false
		}
	}
	return t != ""
}
