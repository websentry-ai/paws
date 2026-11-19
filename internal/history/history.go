// Package history counts what an agent did in its past sessions, so the portal can pick skills on day one. It is
// harness-neutral: a Source reads one agent's own logs into Calls, and Summarize matches them against the catalog's
// triggers on this machine and keeps only counts. Prompts, replies, tool output, arguments and paths never leave it.
package history

import (
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Call is one tool call an agent made, in paws's tool vocabulary: the names the gate matches rules against (Bash,
// Read, Write, Edit, mcp__<server>__<tool>, Skill...). A Source for another agent maps its own tool names to these.
type Call struct {
	// ID is the agent's own id for the call. A resumed or forked session copies earlier calls into a new log, so a
	// call seen twice counts once.
	ID     string
	Tool   string
	Target string // what a matcher runs against: the command, file path or MCP input; the skill name for Skill
	At     time.Time
}

// Source reads one agent's past sessions.
type Source interface {
	// Name identifies the agent in the upload, such as "claude-code".
	Name() string
	// Calls passes every call made in [since, until) to fn and returns how many sessions had one.
	Calls(since, until time.Time, fn func(Call)) (sessions int, err error)
}

// Trigger is one catalog action: a skill and the matcher that says when it applies.
type Trigger struct {
	Skill   string
	Action  string                         // the action's id in the catalog
	Matches func(tool, target string) bool // called from several goroutines at once
}

// SkillTool is the tool a Call names when the agent loaded a skill itself.
const SkillTool = "Skill"

// Summary is what leaves the machine: counts only.
type Summary struct {
	Source     string `json:"source"`
	WindowDays int    `json:"window_days"`
	Since      string `json:"since"` // the window, as dates
	Until      string `json:"until"`
	Sessions   int    `json:"sessions"`
	ToolCalls  int    `json:"tool_calls"`
	// Skills are the catalog skills whose triggers matched, most calls first.
	Skills []SkillHits `json:"skills"`
	// Loaded are catalog skills the agent loaded without being asked. Other skill names are never sent.
	Loaded []Count `json:"loaded"`
	// Unmatched counts the calls no trigger matched, by a coarse signature (see Signature): demand for skills the
	// catalog doesn't have yet.
	Unmatched []Count `json:"unmatched"`
}

// SkillHits is one skill's matched calls. Calls counts each call once even when several of its actions match it.
type SkillHits struct {
	Skill    string   `json:"skill"`
	Calls    int      `json:"calls"`
	LastSeen string   `json:"last_seen"`
	Actions  []Action `json:"actions"`
}

// Action is how many calls one of a skill's actions matched.
type Action struct {
	ID    string `json:"id"`
	Calls int    `json:"calls"`
}

// Count is a name and how often it came up.
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// maxUnmatched bounds the signature list; the long tail says little.
const maxUnmatched = 100

// Summarize reads src's calls in [since, until) and counts them against triggers. catalogSkills are the names a
// Skill call may report. Each distinct (tool, target) is matched once, however often it ran.
func Summarize(src Source, since, until time.Time, days int, triggers []Trigger, catalogSkills map[string]bool) (Summary, error) {
	type key struct{ tool, target string }
	type seen struct {
		n    int
		last time.Time
	}
	calls := map[key]*seen{}
	ids := map[string]bool{}
	loaded := map[string]int{}
	total := 0
	sessions, err := src.Calls(since, until, func(c Call) {
		if c.ID != "" {
			if ids[c.ID] {
				return
			}
			ids[c.ID] = true
		}
		if c.Tool == SkillTool {
			if catalogSkills[c.Target] {
				loaded[c.Target]++
			}
			return
		}
		total++
		k := key{c.Tool, c.Target}
		s := calls[k]
		if s == nil {
			s = &seen{}
			calls[k] = s
		}
		s.n++
		if c.At.After(s.last) {
			s.last = c.At
		}
	})
	sum := Summary{Source: src.Name(), WindowDays: days, Since: day(since), Until: day(until), Sessions: sessions,
		ToolCalls: total, Skills: []SkillHits{}, Loaded: counts(loaded, 0), Unmatched: []Count{}}
	if err != nil {
		return sum, err
	}
	keys := make([]key, 0, len(calls))
	for k := range calls {
		keys = append(keys, k)
	}
	// Longest first: a long command costs the most to match, and starting those early keeps the workers even.
	sort.Slice(keys, func(i, j int) bool { return len(keys[i].target) > len(keys[j].target) })
	// Matching is most of a scan's time (thousands of distinct commands against every trigger), and each command is
	// independent, so it runs on every core. Trigger.Matches must be safe to call concurrently.
	workers := min(runtime.GOMAXPROCS(0), max(1, len(keys)))
	parts := make([]partial, workers)
	var wg sync.WaitGroup
	var next atomic.Int64
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := partial{bySkill: map[string]*tally{}, unmatched: map[string]int{}}
			for i := int(next.Add(1)) - 1; i < len(keys); i = int(next.Add(1)) - 1 {
				k := keys[i]
				p.add(k.tool, k.target, calls[k].n, calls[k].last, triggers)
			}
			parts[w] = p
		}()
	}
	wg.Wait()
	bySkill, unmatched := map[string]*tally{}, map[string]int{}
	for _, p := range parts {
		for name, t := range p.bySkill {
			if bySkill[name] == nil {
				bySkill[name] = &tally{actions: map[string]int{}}
			}
			bySkill[name].merge(t)
		}
		for sig, n := range p.unmatched {
			unmatched[sig] += n
		}
	}
	for name, tl := range bySkill {
		h := SkillHits{Skill: name, Calls: tl.calls, LastSeen: day(tl.last)}
		for _, c := range counts(tl.actions, 0) {
			h.Actions = append(h.Actions, Action{ID: c.Name, Calls: c.Count})
		}
		sum.Skills = append(sum.Skills, h)
	}
	sort.Slice(sum.Skills, func(i, j int) bool {
		a, b := sum.Skills[i], sum.Skills[j]
		return a.Calls > b.Calls || a.Calls == b.Calls && a.Skill < b.Skill
	})
	sum.Unmatched = counts(unmatched, maxUnmatched)
	return sum, nil
}

// tally is one skill's counts.
type tally struct {
	calls   int
	last    time.Time
	actions map[string]int
}

func (t *tally) merge(o *tally) {
	t.calls += o.calls
	if o.last.After(t.last) {
		t.last = o.last
	}
	for id, n := range o.actions {
		t.actions[id] += n
	}
}

// partial is what one matching worker counted.
type partial struct {
	bySkill   map[string]*tally
	unmatched map[string]int
}

// add counts n calls of (tool, target), the latest at last, against every trigger.
func (p partial) add(tool, target string, n int, last time.Time, triggers []Trigger) {
	matched := map[string]bool{}
	for _, t := range triggers {
		if !t.Matches(tool, target) {
			continue
		}
		tl := p.bySkill[t.Skill]
		if tl == nil {
			tl = &tally{actions: map[string]int{}}
			p.bySkill[t.Skill] = tl
		}
		tl.actions[t.Action] += n
		if !matched[t.Skill] {
			matched[t.Skill] = true
			tl.calls += n
			if last.After(tl.last) {
				tl.last = last
			}
		}
	}
	if len(matched) == 0 {
		p.unmatched[Signature(tool, target)] += n
	}
}

// counts sorts m most frequent first, then by name; limit 0 keeps all.
func counts(m map[string]int, limit int) []Count {
	out := make([]Count, 0, len(m))
	for name, n := range m {
		out = append(out, Count{name, n})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || out[i].Count == out[j].Count && out[i].Name < out[j].Name
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func day(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.DateOnly)
}

// Signature is a call's coarse shape, safe to send: no argument, path or input survives it.
//
//	Bash               a known program, plus its subcommands for CLIs whose subcommands are a fixed vocabulary
//	                   ("kubectl delete", "aws s3api", "rm"); a SQL verb (DELETE, DROP...); "bash other" for the rest
//	Read/Write/Edit    "read" or "edit" plus the file's extension ("edit .tf")
//	mcp__server__tool  "mcp <server> <tool>"
//	anything else      the tool name in lower case
//
// An unknown program is never named: the first word of a real command is often a path, a variable or a script.
func Signature(tool, target string) string {
	switch {
	case tool == "Bash":
		return "bash " + bashSignature(target)
	case fileTools[tool]:
		verb := "edit"
		if tool == "Read" {
			verb = "read"
		}
		ext := strings.ToLower(filepath.Ext(target))
		if !extRe.MatchString(ext) {
			return verb
		}
		return verb + " " + ext
	case strings.HasPrefix(tool, "mcp__"):
		server, name, ok := strings.Cut(strings.TrimPrefix(tool, "mcp__"), "__")
		if !ok || !wordRe.MatchString(server) || !wordRe.MatchString(name) {
			return "mcp other"
		}
		return "mcp " + strings.ToLower(server) + " " + strings.ToLower(name)
	case wordRe.MatchString(tool):
		return strings.ToLower(tool)
	}
	return "other"
}

var (
	fileTools = map[string]bool{"Read": true, "Write": true, "Edit": true, "NotebookEdit": true}
	extRe     = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)
	wordRe    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	subRe     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	// sqlVerb finds a destructive SQL statement anywhere in a command, as the gate's action keys do.
	sqlVerb   = regexp.MustCompile(`(?i:\b(delete)\s+from\b|\b(drop)\s+(?:table|database|schema|index|view)\b|\b(truncate)\s+table\b|\b(update)\s+\S+\s+set\b)`)
	chainSep  = regexp.MustCompile(`&&|\|\||[;|&\n]`)
	envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
)

func bashSignature(cmd string) string {
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
		// A bare name only: /usr/bin/rm is rm, but ./deploy.sh and ~/bin/x are scripts of the user's own.
		name := words[0]
		if strings.HasPrefix(name, "/usr/") || strings.HasPrefix(name, "/bin/") || strings.HasPrefix(name, "/opt/homebrew/") {
			name = filepath.Base(name)
		}
		n, known := programs[name]
		if !known {
			return "other"
		}
		key := []string{name}
		for _, w := range words[1:] {
			if n == 0 {
				break
			}
			if !subRe.MatchString(w) {
				break
			}
			key = append(key, w)
			n--
		}
		return strings.Join(key, " ")
	}
	return "other"
}

// programs are the commands a signature may name, with how many subcommand words it keeps. Only CLIs whose
// subcommands are their own fixed vocabulary keep any: make, npm run, yarn and npx take the user's own target,
// script or package names there.
var programs = map[string]int{
	"git": 1, "gh": 2, "glab": 2, "docker": 1, "podman": 1, "kubectl": 1, "helm": 1, "terraform": 1, "tofu": 1,
	"aws": 2, "gcloud": 2, "az": 2, "go": 1, "cargo": 1, "pip": 1, "pip3": 1, "uv": 1, "poetry": 1, "brew": 1,
	"apt": 1, "apt-get": 1, "systemctl": 1, "vault": 1, "supabase": 1, "vercel": 1, "fly": 1, "flyctl": 1,
	"heroku": 1, "wrangler": 1, "stripe": 1, "firebase": 1, "pulumi": 1, "kind": 1, "minikube": 1, "eksctl": 1,
	"npm": 0, "pnpm": 0, "yarn": 0, "bun": 0, "npx": 0, "make": 0, "rclone": 0, "gsutil": 0, "sf": 0,
	"launchctl": 0, "node": 0, "deno": 0, "python": 0, "python3": 0, "pytest": 0, "ruby": 0, "java": 0,
	"rm": 0, "rmdir": 0, "mv": 0, "cp": 0, "find": 0, "ls": 0, "cat": 0, "head": 0, "tail": 0, "grep": 0, "rg": 0,
	"sed": 0, "awk": 0, "wc": 0, "sort": 0, "jq": 0, "xargs": 0, "echo": 0, "mkdir": 0, "touch": 0, "chmod": 0,
	"chown": 0, "ln": 0, "tar": 0, "zip": 0, "unzip": 0, "curl": 0, "wget": 0, "ssh": 0, "scp": 0, "rsync": 0,
	"dd": 0, "mkfs": 0, "diskutil": 0, "kill": 0, "pkill": 0, "killall": 0, "lsof": 0, "ps": 0, "open": 0,
	"psql": 0, "pg_dump": 0, "pg_restore": 0, "mysql": 0, "mysqldump": 0, "sqlite3": 0, "mongosh": 0,
	"mongodump": 0, "redis-cli": 0, "openssl": 0, "gpg": 0, "security": 0, "defaults": 0, "crontab": 0,
	"shred": 0, "truncate": 0, "ansible": 0, "ansible-playbook": 0, "cdk": 0, "sam": 0, "serverless": 0,
}
