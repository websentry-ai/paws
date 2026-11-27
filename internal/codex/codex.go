// Package codex is the Codex CLI adapter (harness.Adapter): hooks.json, the hook stdin Codex sends, ~/.agents/skills,
// and a ledger paws keeps of each session in place of Codex's rollout files.
package codex

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/history"
	"github.com/websentry-ai/paws/internal/rule"
)

// Name is the harness id on every record.
const Name = "codex"

// Adapter is the Codex adapter.
type Adapter struct{}

var _ harness.Adapter = Adapter{}

func (Adapter) Name() string  { return Name }
func (Adapter) Title() string { return "Codex" }

// SkillTrees is nil: Codex's skills sit directly in its SkillDirs.
func (Adapter) SkillTrees() []string { return nil }

// History is nil: Codex's rollout format isn't stable, so scan doesn't read it.
func (Adapter) History() (history.Source, string) { return nil, "" }

// Input is the JSON Codex pipes to a hook: Claude Code's fields plus turn_id and model on every event.
type Input struct {
	SessionID      string          `json:"session_id"`
	TurnID         string          `json:"turn_id"`
	TranscriptPath string          `json:"transcript_path"`
	CWD            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	Model          string          `json:"model"`
	ToolName       string          `json:"tool_name"`
	ToolInput      map[string]any  `json:"tool_input"`
	ToolUseID      string          `json:"tool_use_id"`
	ToolResponse   json.RawMessage `json:"tool_response"`
	Source         string          `json:"source"` // SessionStart: startup|resume|clear|compact|fork
	AgentID        string          `json:"agent_id"`
	AgentType      string          `json:"agent_type"`
}

// parsed is the adapter's reading of one hook call, kept in harness.Hook.Native.
type parsed struct {
	in    Input
	at    time.Time // when the hook started: a call is logged before paws decides on it, as Claude Code logs it
	calls []call
}

// call is one canonical call and the input expect checks see for it.
type call struct {
	tool, target, text string
	input              map[string]any
}

// Claims a hook call that names a turn and a model (Claude Code sends neither on a tool call), or whose transcript
// is a Codex rollout: Codex can run hooks it imported from Claude Code's settings, without --agent.
func (Adapter) Claims(stdin []byte) bool {
	var in Input
	if json.Unmarshal(stdin, &in) != nil {
		return false
	}
	sessions := filepath.Join(Home(), "sessions") + string(filepath.Separator)
	return in.TurnID != "" && in.Model != "" || in.TranscriptPath != "" && strings.HasPrefix(in.TranscriptPath, sessions)
}

func (a Adapter) Parse(stdin []byte) (*harness.Hook, error) {
	p := &parsed{at: time.Now()}
	if err := json.Unmarshal(stdin, &p.in); err != nil {
		return nil, err
	}
	in := &p.in
	if in.CWD == "" {
		in.CWD, _ = os.Getwd()
	}
	p.calls = canonical(in)
	base := gate.Event{Agent: Name, Session: in.SessionID, Cwd: in.CWD, AgentID: in.AgentID, AgentType: in.AgentType,
		Log:     session{own: ledgerPath(in.SessionID, in.AgentID), main: ledgerPath(in.SessionID, ""), turn: in.TurnID, model: in.Model},
		Harness: a}
	h := &harness.Hook{Event: in.HookEventName, Cwd: in.CWD, Source: in.Source, Native: p}
	for _, c := range p.calls {
		ev := base
		ev.Tool, ev.Target = c.tool, c.target
		if s, ok := ev.Log.(session); ok && c.tool == "Agent" {
			s.spawning = true // this call is a spawn the ledger doesn't hold yet
			ev.Log = s
		}
		switch in.HookEventName {
		case "PreToolUse":
			ev.Kind = rule.KindTool
			ev.SkillLoad = gate.SkillRead(c.tool, c.target, in.CWD)
		case "PostToolUse":
			ev.SkillLoad = gate.SkillRead(c.tool, c.target, in.CWD)
		case "SessionStart":
			ev.Kind, ev.Target = rule.KindSession, in.Source
		}
		h.Calls = append(h.Calls, ev)
	}
	return h, nil
}

// canonical maps a Codex tool call to canonical calls: Bash as is, apply_patch to a Write (new file) or Edit per
// file, spawn_agent to Agent, MCP tools as is. A hook call without a tool (a prompt, a stop) is one call with none.
func canonical(in *Input) []call {
	ti := in.ToolInput
	str := func(k string) string { s, _ := ti[k].(string); return s }
	switch {
	case in.ToolName == "":
		return []call{{}}
	case in.ToolName == "Bash":
		// The command is always gated as Bash. A patch inside it (apply_patch <<'EOF' ... EOF through the shell)
		// also edits files: those are gated too, as Codex's apply_patch tool is.
		cmd := str("command")
		calls := []call{{tool: "Bash", target: cmd, text: cmd, input: map[string]any{"command": cmd}}}
		if before, _, ok := strings.Cut(cmd, "*** Begin Patch"); ok && applyPatchCmd.MatchString(before) {
			for _, sp := range shellPatches(cmd, in.CWD) {
				calls = append(calls, patchCalls(sp.text, sp.dir)...)
			}
		}
		return calls
	case in.ToolName == "apply_patch":
		patch := cmp.Or(str("command"), str("input"), str("patch"))
		if calls := patchCalls(patch, in.CWD); len(calls) > 0 {
			return calls
		}
		return []call{{tool: "apply_patch", target: patch, text: patch, input: ti}}
	case in.ToolName == "spawn_agent":
		prompt := ""
		for _, k := range []string{"prompt", "message", "task", "instructions"} {
			if prompt = str(k); prompt != "" {
				break
			}
		}
		if prompt == "" {
			b, _ := json.Marshal(ti)
			prompt = string(b)
		}
		return []call{{tool: "Agent", target: prompt, text: prompt, input: map[string]any{"prompt": prompt}}}
	case strings.HasPrefix(in.ToolName, "mcp__"):
		b, _ := json.Marshal(ti)
		return []call{{tool: in.ToolName, target: string(b), text: strings.Join(stringsIn(ti), "\n"), input: ti}}
	}
	return []call{{tool: in.ToolName, target: in.ToolName, text: in.ToolName, input: ti}}
}

// cdStep is one `cd <dir>` command of a chain.
var cdStep = regexp.MustCompile(`^cd\s+(?:--\s+)?(?:'([^']*)'|"([^"]*)"|(\S+))$`)

// chainSep splits a command line into its commands.
var chainSep = regexp.MustCompile(`&&|\|\||[;\n]`)

// shellPatch is one apply_patch run in a shell command: its patch text and the folder it runs in.
type shellPatch struct{ text, dir string }

// applyPatchCmd finds apply_patch (or its applypatch alias) where a command starts: at the line's start, after ; & | or (, as a whole word,
// behind any of the wrappers that run the next word as the command (command apply_patch, exec, env, nohup, time) and
// any FOO=bar assignments.
var applyPatchCmd = regexp.MustCompile(`(?m)(?:^|[;&|(])[ \t]*(?:(?:command|builtin|exec|env|nohup|time|sudo|[A-Za-z_][A-Za-z0-9_]*=\S*)[ \t]+)*(apply_?patch)(?:[ \t<]|$)`)

// shellPatches are the apply_patch runs of a shell command, each with the folder the commands before it moved to.
func shellPatches(cmd, cwd string) []shellPatch {
	var out []shellPatch
	var starts []int // where each apply_patch command begins: not "apply_patch" inside a file name
	for _, m := range applyPatchCmd.FindAllStringSubmatchIndex(cmd, -1) {
		starts = append(starts, m[2])
	}
	for k, at := range starts {
		end := len(cmd)
		if k+1 < len(starts) {
			end = starts[k+1]
		}
		out = append(out, shellPatch{text: cmd[at:end], dir: patchDir(cmd[:at], cwd)})
	}
	return out
}

// patchDir is where a shell patch runs, given the command before it: the hook's cwd, moved by each `cd`, so a
// relative path in `ls && cd migrations && apply_patch ...` is a file in migrations/. A cd inside a subshell that
// closes before apply_patch moves nothing.
func patchDir(before, cwd string) string {
	var outer []string // the folder each open subshell started in
	for _, part := range chainSep.Split(before, -1) {
		part = strings.TrimSpace(part)
		for strings.HasPrefix(part, "(") {
			outer, part = append(outer, cwd), strings.TrimSpace(part[1:])
		}
		inner := strings.TrimRight(part, ") ")
		if m := cdStep.FindStringSubmatch(inner); m != nil {
			cwd = cdTo(m[1]+os.ExpandEnv(m[2]+m[3]), cwd) // "$DIR" and $DIR from the hook's environment; '$DIR' is literal
		}
		for closes := strings.Count(part[len(inner):], ")"); closes > 0 && len(outer) > 0; closes-- {
			cwd, outer = outer[len(outer)-1], outer[:len(outer)-1] // the subshell ended: back where it began
		}
	}
	return cwd
}

// cdTo is the folder `cd dir` moves to from cwd.
func cdTo(dir, cwd string) string {
	if rest, ok := strings.CutPrefix(dir, "~/"); ok {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}
	return filepath.Clean(dir)
}

// patchCalls are a patch's files as canonical calls: a Write for a new file, an Edit for any other, relative paths
// taken from cwd.
func patchCalls(patch, cwd string) []call {
	var calls []call
	for _, f := range parsePatch(patch) {
		path := f.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		c := call{tool: "Edit", target: path, text: f.Added, input: map[string]any{"file_path": path, "new_string": f.Added}}
		if f.New {
			c.tool, c.input = "Write", map[string]any{"file_path": path, "content": f.Added}
		}
		calls = append(calls, c)
	}
	return calls
}

// stringsIn is every string value in v, keys in order, depth first: an MCP call's text, as expect checks see it.
func stringsIn(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			out = append(out, stringsIn(x[k])...)
		}
		return out
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, stringsIn(e)...)
		}
		return out
	}
	return nil
}

type output struct {
	SystemMessage  string          `json:"systemMessage,omitempty"`
	SpecificOutput *specificOutput `json:"hookSpecificOutput,omitempty"`
}

type specificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
	AdditionalContext        string `json:"additionalContext,omitempty"`
}

// Render is Claude Code's reply shape, which Codex reads too. Codex shows a deny as "Command blocked by PreToolUse
// hook: <reason>. Command: <command>", dropping the reason's trailing period itself; paws's own Codex wording ends
// without one, and a rule's deny_message goes as written.
func (Adapter) Render(h *harness.Hook, d gate.Decision, notice string) []byte {
	out := output{SystemMessage: notice}
	switch {
	case h.Event != "PreToolUse" && h.Event != "SessionStart":
	case d.Verdict == gate.Deny:
		out.SpecificOutput = &specificOutput{HookEventName: h.Event, PermissionDecision: "deny",
			PermissionDecisionReason: d.Reason, AdditionalContext: d.Context}
	case d.Context != "":
		out.SpecificOutput = &specificOutput{HookEventName: h.Event, AdditionalContext: d.Context}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// Observe writes the ledger: each call with paws's verdict, each result, each turn and compaction.
func (Adapter) Observe(h *harness.Hook, d gate.Decision) {
	p, _ := h.Native.(*parsed)
	if p == nil {
		return
	}
	in := &p.in
	own := ledgerPath(in.SessionID, in.AgentID)
	switch {
	case in.HookEventName == "PreToolUse":
		verdict, heads := "allow", []string(nil)
		if d.Verdict == gate.Deny {
			verdict, heads = "deny", denyHeads(d.Reason)
		}
		var recs []record
		for i, c := range p.calls {
			rec := record{Kind: "call", At: p.at, Turn: in.TurnID, ID: in.ToolUseID, Part: i, Tool: c.tool,
				Target: c.target, Text: c.text, Input: c.input, Verdict: verdict, SkillLoad: h.Calls[i].SkillLoad,
				Denied: heads, Agent: in.AgentID}
			if r, ok := gate.ReadOf(c.tool, c.target, c.input, in.CWD, true); ok {
				rec.ReadFile, rec.ReadFrom, rec.ReadTo = r.File, r.Lines.From, r.Lines.To
			}
			recs = append(recs, rec)
		}
		appendRecords(own, recs)
	case in.HookEventName == "PostToolUse":
		ok := resultOK(in.ToolResponse)
		var files []string
		if in.ToolName == "apply_patch" || in.ToolName == "Bash" && len(p.calls) > 1 {
			// A patch: the files it changed are the ones apply_patch lists after its success line. Through the shell
			// there is no exit code, so that line is the only sign it applied.
			runs := shellPatches(cmdOf(in), in.CWD)
			if in.ToolName == "apply_patch" {
				str := func(k string) string { s, _ := in.ToolInput[k].(string); return s }
				runs = []shellPatch{{text: cmp.Or(str("command"), str("input"), str("patch")), dir: in.CWD}} // the tool runs in the hook's cwd
			}
			files = patchedFiles(in.ToolResponse, runs)
			if in.ToolName == "Bash" {
				ok = len(files) > 0
			}
		}
		var out string
		if json.Unmarshal(in.ToolResponse, &out) != nil {
			out = string(in.ToolResponse)
		}
		c, shown := gate.ShownLines(out) // what the model got of a read
		appendRecords(own, []record{{Kind: "result", At: p.at, Turn: in.TurnID, ID: in.ToolUseID, OK: &ok, Files: files,
			Cut: c, Shown: shown, Agent: in.AgentID}})
	case in.HookEventName == "UserPromptSubmit":
		appendRecords(own, []record{{Kind: "turn", At: p.at, Turn: in.TurnID}})
	case in.HookEventName == "PostCompact", in.HookEventName == "SessionStart" && in.Source == "compact":
		appendRecords(own, []record{{Kind: "compact", At: p.at, Turn: in.TurnID}})
	case in.HookEventName == "SessionStart":
		pruneLedgers(own)
	}
}

// cmdOf is a Bash call's command, "" for any other tool.
func cmdOf(in *Input) string {
	if in.ToolName != "Bash" {
		return ""
	}
	s, _ := in.ToolInput["command"].(string)
	return s
}

// patchedFiles are the files apply_patch says it changed: the "A path", "M path" and "D path" lines after its
// "Success. Updated the following files:" line, absolute.
func patchedFiles(raw json.RawMessage, runs []shellPatch) []string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		s = string(raw)
	}
	// A listed path is relative to the folder its own run ran in: the run whose patch names that file. Pairing by
	// name, not by position, keeps a failed run (it lists nothing) from shifting the rest.
	var files []string
	for _, list := range strings.Split(s, "Success. Updated the following files:")[1:] {
		for _, line := range strings.Split(list, "\n") {
			op, path, ok := strings.Cut(strings.TrimSpace(line), " ")
			if !ok || len(op) != 1 || !strings.Contains("AMD", op) {
				continue
			}
			if filepath.IsAbs(path) {
				files = append(files, filepath.Clean(path))
				continue
			}
			// When runs in different folders each name this path, the list can't say which one changed it: credit
			// none, so a no-edit-matching rule still applies rather than passing on an edit that may not have happened.
			var owners []string
			for _, run := range runs {
				for _, f := range parsePatch(run.text) {
					if filepath.Clean(f.Path) == filepath.Clean(path) && !slices.Contains(owners, filepath.Join(run.dir, path)) {
						owners = append(owners, filepath.Join(run.dir, path))
					}
				}
			}
			if len(owners) == 1 {
				files = append(files, owners[0])
			}
		}
	}
	return files
}

// resultOK reads a PostToolUse tool_response: apply_patch's starts "Exit code: N"; a Bash result is the output alone,
// so a read that failed shows as the reader's error on the first line ("sed: <path>: No such file or directory"). The
// rest of the output is the file, which may mention any error.
func resultOK(raw json.RawMessage) bool {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		b, _ := json.Marshal(raw)
		s = string(b)
	}
	if rest, ok := strings.CutPrefix(s, "Exit code: "); ok {
		return strings.HasPrefix(rest, "0\n") || rest == "0"
	}
	first, _, _ := strings.Cut(s, "\n")
	return !readError.MatchString(first)
}

var readError = regexp.MustCompile(`(^|: )(No such file or directory|Permission denied|Is a directory)$`)

// Turns is the ledger of the session, from the Stop's turn id.
func (Adapter) Turns(h *harness.Hook) harness.TurnLog {
	p, _ := h.Native.(*parsed)
	if p == nil || p.in.SessionID == "" || p.in.TurnID == "" {
		return nil
	}
	return turns{session: p.in.SessionID, turn: p.in.TurnID}
}

// SkillDirs are where Codex finds skills: ~/.agents/skills, the repo's .agents/skills and $CODEX_HOME/skills; then
// Claude Code's folders, whose skills the agent can read just the same when paws names the file.
func (Adapter) SkillDirs(cwd string) []string {
	return append([]string{userSkills(), filepath.Join(gate.RepoRoot(cwd), ".agents", "skills"), filepath.Join(Home(), "skills")},
		gate.SkillTool.SkillDirs(cwd)...)
}

// LoadSkills tells Codex to read each SKILL.md in full and follow it, then make the call again, changed if the skill
// says so. Codex has no Skill tool, and told to retry "this exact command" it re-runs the blocked call unchanged.
func (Adapter) LoadSkills(skills []gate.Skill, noun string) string {
	it, says := "it", "the skill says"
	if len(skills) > 1 {
		it, says = "them", "the skills say"
	}
	verb := "make the " + noun
	if noun == "command" {
		verb = "run the command"
	}
	// "On its own": a read chained with other commands isn't a load, and Codex likes to chain.
	return fmt.Sprintf("Read %s in full on its own, follow %s, then %s again, changed if %s so", readList(skills), it, verb, says)
}

func (Adapter) LoadAtStart(s gate.Skill) string {
	return fmt.Sprintf("Read %s in full on its own and follow it before you start work.", readList([]gate.Skill{s}))
}

func readList(skills []gate.Skill) string {
	files := make([]string, len(skills))
	for i, s := range skills {
		files[i] = s.File
		if s.File == "" {
			files[i] = fmt.Sprintf("the %s skill's SKILL.md (in .agents/skills)", s.Name)
		}
	}
	if len(files) == 1 {
		return files[0]
	}
	return strings.Join(files[:len(files)-1], ", ") + " and " + files[len(files)-1]
}

// Home is Codex's config dir: $CODEX_HOME, else ~/.codex.
func Home() string {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

func userSkills() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agents", "skills")
}

func (Adapter) HookFile(global bool, root string) string {
	if global {
		return filepath.Join(Home(), "hooks.json")
	}
	return filepath.Join(root, ".codex", "hooks.json")
}

func (a Adapter) HookFiles(root string) []string {
	return []string{a.HookFile(false, root), a.HookFile(true, "")}
}

func (Adapter) HookCommand(gate string) string { return gate + " --agent " + Name }

// Events are the hooks paws registers in Codex: Claude Code's five, plus PostCompact for the ledger.
func (Adapter) Events() []string {
	return []string{"PreToolUse", "SessionStart", "UserPromptSubmit", "PostToolUse", "Stop", "PostCompact"}
}

// Entry is a hooks.json entry: an explicit 15 s timeout (Codex's default is 600 s), async where nothing waits on the
// answer, and matcher * on tool events. PostToolUse and PostCompact are not async: the ledger must hold a call's
// result, or a compaction, before the next call is gated.
func (Adapter) Entry(event, command string) map[string]any {
	h := map[string]any{"type": "command", "command": command, "timeout": 15}
	switch event {
	case "UserPromptSubmit", "Stop":
		h["async"] = true
	}
	if event == "PreToolUse" || event == "PostToolUse" {
		return map[string]any{"matcher": "*", "hooks": []any{h}}
	}
	return map[string]any{"hooks": []any{h}}
}

// SkillsDirFor: Codex reads ~/.agents/skills for the user's hooks.json, and <repo>/.agents/skills for a project's
// .codex/hooks.json.
func (Adapter) SkillsDirFor(hookFile string) string {
	dir := filepath.Dir(hookFile)
	switch {
	case filepath.Clean(dir) == filepath.Clean(Home()):
		return userSkills()
	case filepath.Base(dir) == ".codex":
		return filepath.Join(filepath.Dir(dir), ".agents", "skills")
	}
	return ""
}

// InstallNote: Codex runs a hook only once the user trusts it.
func (Adapter) InstallNote() string {
	return "Codex runs a new or changed hook only after you trust it: start Codex, run /hooks and trust the paws hook."
}
