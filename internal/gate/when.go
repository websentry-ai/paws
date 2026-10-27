package gate

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
// devDependencies, peerDependencies, optionalDependencies), go.mod (module path, or its
// last element), Cargo.toml dependency tables, and pyproject.toml / requirements.txt
// requirement names (case-insensitive, - and _ equivalent).
func hasDependency(root, pkg string) bool {
	return packageJSONHas(filepath.Join(root, "package.json"), pkg) ||
		goModHas(filepath.Join(root, "go.mod"), pkg) ||
		cargoHas(filepath.Join(root, "Cargo.toml"), pkg) ||
		pythonHas(filepath.Join(root, "pyproject.toml"), pkg) ||
		pythonHas(filepath.Join(root, "requirements.txt"), pkg)
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

func goModHas(path, pkg string) bool {
	found := false
	eachLine(path, func(line string) bool {
		f := strings.Fields(strings.TrimPrefix(line, "require "))
		if len(f) >= 2 && strings.Contains(f[0], ".") && (f[0] == pkg || strings.HasSuffix(f[0], "/"+pkg)) {
			found = true
		}
		return !found
	})
	return found
}

func cargoHas(path, pkg string) bool {
	found, inDeps := false, false
	eachLine(path, func(line string) bool {
		if strings.HasPrefix(line, "[") {
			inDeps = strings.Contains(line, "dependencies")
			// [dependencies.serde] names the crate in the header itself
			if inDeps && strings.HasSuffix(strings.TrimSuffix(line, "]"), "."+pkg) {
				found = true
			}
			return !found
		}
		if inDeps {
			if key, _, ok := strings.Cut(line, "="); ok && strings.Trim(strings.TrimSpace(key), `"`) == pkg {
				found = true
			}
		}
		return !found
	})
	return found
}

var pyReqName = regexp.MustCompile(`^["']?([A-Za-z0-9][A-Za-z0-9._-]*)`)

func pythonHas(path, pkg string) bool {
	want := normPy(pkg)
	found := false
	eachLine(path, func(line string) bool {
		// pyproject lists ("requests>=2", ...), poetry tables (requests = "^2"), requirements lines.
		for _, part := range strings.Split(strings.Trim(line, "[],"), ",") {
			part = strings.TrimSpace(part)
			if key, _, ok := strings.Cut(part, "="); ok && !strings.ContainsAny(key, "<>!~") {
				part = key
			}
			if m := pyReqName.FindStringSubmatch(strings.TrimSpace(part)); m != nil && normPy(m[1]) == want {
				found = true
			}
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
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !fn(line) {
			return
		}
	}
}

// toolUse is one tool_use block inside an assistant transcript message.
type toolUse struct {
	Type  string         `json:"type"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

type contentMessage struct {
	Content json.RawMessage `json:"content"`
}

// editedMatching reports whether the main thread already ran Edit or Write on a path matching
// glob, anywhere in the transcript tail. No transcript means no known edit.
func editedMatching(path string, glob *regexp.Regexp) bool {
	lines, err := tailLines(path)
	if err != nil {
		return false
	}
	for _, ln := range lines {
		for _, tu := range assistantToolUses(ln) {
			if tu.Name != "Edit" && tu.Name != "Write" {
				continue
			}
			if p, _ := tu.Input["file_path"].(string); p != "" && glob.MatchString(p) {
				return true
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
		if isUserPrompt(lines[i]) {
			break
		}
		for _, tu := range assistantToolUses(lines[i]) {
			if tu.Name == "Agent" || tu.Name == "Task" {
				n++
			}
		}
	}
	return n
}

func assistantToolUses(line []byte) []toolUse {
	var e struct {
		Type        string         `json:"type"`
		IsSidechain bool           `json:"isSidechain"`
		Message     contentMessage `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.Type != "assistant" || e.IsSidechain {
		return nil
	}
	var blocks []toolUse
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return nil
	}
	out := blocks[:0]
	for _, b := range blocks {
		if b.Type == "tool_use" {
			out = append(out, b)
		}
	}
	return out
}

// isUserPrompt reports whether a transcript line is a prompt the user typed, not a tool result.
func isUserPrompt(line []byte) bool {
	var e struct {
		Type        string         `json:"type"`
		IsSidechain bool           `json:"isSidechain"`
		IsMeta      bool           `json:"isMeta"`
		Message     contentMessage `json:"message"`
	}
	if json.Unmarshal(line, &e) != nil || e.Type != "user" || e.IsSidechain || e.IsMeta {
		return false
	}
	c := strings.TrimSpace(string(e.Message.Content))
	if strings.HasPrefix(c, `"`) {
		return true
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(e.Message.Content, &blocks) != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return false
		}
	}
	return len(blocks) > 0
}
