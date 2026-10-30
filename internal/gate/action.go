package gate

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/websentry-ai/paws/internal/rule"
)

// ToolAction is what one finished tool call did, as an audit `action` record keeps it and
// `paws sync` uploads it. It never holds file contents, prompts or absolute paths.
type ToolAction struct {
	Tool    string   `json:"tool"`
	Action  string   `json:"action"`            // normalized key, e.g. "git commit", "edit migrations/**"
	Command string   `json:"command,omitempty"` // Bash only: redacted, at most maxCommand bytes
	File    string   `json:"file,omitempty"`    // Read/Write/Edit: relative to the repo root
	MCP     *MCPCall `json:"mcp,omitempty"`
}

// MCPCall names an MCP tool call: the server and tool as Claude Code spells them, and the
// action class (send|create|update|delete, "" for reads) that matcher.mcp_action uses.
type MCPCall struct {
	Server string `json:"server"`
	Tool   string `json:"tool"`
	Class  string `json:"class,omitempty"`
}

// maxCommand bounds an uploaded command.
const maxCommand = 500

// RedactCommand masks secrets in a command and cuts it to 500 bytes: the form a command
// leaves the machine in.
func RedactCommand(s string) string {
	s = redact(s)
	if len(s) <= maxCommand {
		return s
	}
	cut := maxCommand
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// LogAction appends an `action` record for a finished tool call. Best-effort, never blocks.
func LogAction(ev Event) {
	if DryRun {
		return
	}
	p := auditPath()
	a := DescribeAction(ev)
	if p == "" || a.Action == "" {
		return
	}
	rec := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339), "agent": ev.Agent, "event": "action",
		"session": ev.Session, "tool": a.Tool, "action": a.Action}
	if a.Command != "" {
		rec["command"] = a.Command
	}
	if a.File != "" {
		rec["file"] = a.File
	}
	if a.MCP != nil {
		rec["mcp"] = a.MCP
	}
	if ev.AgentID != "" {
		rec["agent_id"] = ev.AgentID
	}
	appendAudit(p, rec)
}

// DescribeAction normalizes a tool call into an action key (see docs/telemetry.md):
//
//	Bash               the first command of a chain: "git commit", "gh pr create", "npm test",
//	                   "ls"; or a SQL verb (DELETE, DROP, TRUNCATE, UPDATE) anywhere in it
//	Read/Write/Edit    "read" or "edit" plus the top-level directory ("edit migrations/**")
//	                   or, for a file at the repo root, its name ("edit README.md")
//	mcp__server__tool  "<server> <tool>": "gmail create-draft", "notion create-pages"
//	anything else      the tool name in lower case: "grep", "webfetch"
func DescribeAction(ev Event) ToolAction {
	a := ToolAction{Tool: ev.Tool}
	switch {
	case ev.Tool == "Bash":
		a.Action, a.Command = bashAction(ev.Target), RedactCommand(ev.Target)
	case fileTool(ev.Tool):
		verb := "edit"
		if ev.Tool == "Read" {
			verb = "read"
		}
		a.File = repoRelative(ev.Target, ev.Cwd)
		a.Action = fileAction(verb, a.File)
	default:
		if server, tool, ok := MCPParts(ev.Tool); ok {
			raw := strings.SplitN(strings.TrimPrefix(ev.Tool, "mcp__"), "__", 2)
			a.Action = server + " " + tool
			a.MCP = &MCPCall{Server: raw[0], Tool: raw[1], Class: rule.MCPAction(ev.Tool)}
		} else {
			a.Action = strings.ToLower(ev.Tool)
		}
	}
	return a
}

var (
	// sqlVerb finds a destructive SQL statement anywhere in a command (psql -c, sqlite3, a heredoc).
	// TRUNCATE alone must be upper case: lower-case `truncate -s 0 f` is coreutils.
	sqlVerb   = regexp.MustCompile(`(?i:\b(delete)\s+from\b|\b(drop)\s+(?:table|database|schema|index|view)\b|\b(truncate)\s+table\b|\b(update)\s+\S+\s+set\b)|\b(TRUNCATE)\b`)
	chainSep  = regexp.MustCompile(`&&|\|\||[;|&\n]`)
	envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	subWord   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// subcommandCLIs keep their subcommand in the key ("git commit", not "git"); gh keeps two
// ("gh pr create"). Every other program is keyed by its name alone, so a file argument
// (`cat notes.txt`) never becomes part of a key.
var subcommandCLIs = map[string]int{
	"gh": 2, "glab": 2,
	"git": 1, "npm": 1, "pnpm": 1, "yarn": 1, "bun": 1, "npx": 1, "go": 1, "cargo": 1, "make": 1,
	"docker": 1, "podman": 1, "kubectl": 1, "helm": 1, "terraform": 1, "tofu": 1, "aws": 1,
	"gcloud": 1, "gsutil": 1, "az": 1, "vercel": 1, "fly": 1, "flyctl": 1, "heroku": 1,
	"supabase": 1, "pip": 1, "pip3": 1, "uv": 1, "poetry": 1, "brew": 1, "apt": 1, "apt-get": 1,
	"systemctl": 1, "launchctl": 1, "vault": 1, "rclone": 1, "sf": 1,
}

// bashAction keys a shell command: a SQL verb if one appears, else the first command of the
// chain that isn't a `cd`, minus env assignments and sudo.
func bashAction(cmd string) string {
	if m := sqlVerb.FindStringSubmatch(cmd); m != nil {
		for _, g := range m[1:] {
			if g != "" {
				return strings.ToUpper(g)
			}
		}
	}
	for _, seg := range chainSep.Split(cmd, -1) {
		words := strings.Fields(seg)
		for len(words) > 0 && (envAssign.MatchString(words[0]) || words[0] == "sudo") {
			words = words[1:]
		}
		if len(words) == 0 || words[0] == "cd" || words[0] == "pushd" {
			continue
		}
		return commandKey(words)
	}
	return "bash"
}

// commandKey is the program's base name plus, for a CLI with subcommands, the words that
// follow it while they look like subcommands. `-C dir` and `-c k=v` are skipped (git, make).
func commandKey(words []string) string {
	name := path.Base(words[0])
	key := []string{name}
	rest := words[1:]
	for n := subcommandCLIs[name]; n > 0 && len(rest) > 0; {
		if (rest[0] == "-C" || rest[0] == "-c") && len(rest) > 1 {
			rest = rest[2:]
			continue
		}
		if !subWord.MatchString(rest[0]) {
			break
		}
		key, rest, n = append(key, rest[0]), rest[1:], n-1
	}
	return strings.Join(key, " ")
}

// repoRelative returns p relative to the repo root holding cwd (cwd itself outside a repo),
// slash-separated. A path outside that root comes back "": it is never sent absolute.
func repoRelative(p, cwd string) string {
	if p == "" || cwd == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	rel, err := filepath.Rel(repoRoot(cwd), p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// fileAction keys a file call by its top-level directory, or by name for a root file.
func fileAction(verb, rel string) string {
	if rel == "" {
		return verb
	}
	if top, _, nested := strings.Cut(rel, "/"); nested {
		return verb + " " + top + "/**"
	}
	return verb + " " + rel
}

// MCPParts splits an MCP tool name into a readable server and tool:
// mcp__claude_ai_Gmail__create_draft is ("gmail", "create-draft") and
// mcp__plugin_slack_slack__slack_send_message is ("slack", "send-message").
func MCPParts(tool string) (server, name string, ok bool) {
	raw, ok := strings.CutPrefix(tool, "mcp__")
	if !ok {
		return "", "", false
	}
	server, name, ok = strings.Cut(raw, "__")
	if !ok || server == "" || name == "" {
		return "", "", false
	}
	server = strings.ToLower(strings.TrimPrefix(server, "claude_ai_"))
	if rest, plugin := strings.CutPrefix(server, "plugin_"); plugin {
		server = rest[strings.LastIndex(rest, "_")+1:] // plugin_<plugin>_<server>
	}
	name = strings.ToLower(name)
	for _, sep := range []string{"_", "-"} {
		name = strings.TrimPrefix(name, server+sep)
	}
	return server, strings.ReplaceAll(name, "_", "-"), true
}
