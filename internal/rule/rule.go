// Package rule parses .paws/*.md frontmatter into matchable rules. Pure — no I/O.
package rule

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule kinds: which class of agent event a rule applies to.
const (
	KindTool    = "tool"    // a pending tool call
	KindPrompt  = "prompt"  // a submitted user prompt
	KindSession = "session" // a session starting (startup, resume, clear, compact)
)

var validActions = map[string]bool{
	"block": true, "require-skill": true, "warn": true, "audit": true,
	"prompt-inject": true, "exec-script": true, "advise": true,
}

// sessionActions are the actions that make sense when nothing can be denied: a session is starting.
var sessionActions = map[string]bool{"require-skill": true, "advise": true, "audit": true}

// Matcher decides whether a rule applies to a pending call.
type Matcher struct {
	Tool         string `yaml:"tool"`          // tool name, "A|B" list, "mcp__server__*" prefix, or omit / quoted "*" for any
	CommandRegex string `yaml:"command_regex"` // matched against the extracted command
	FileGlob     string `yaml:"file_glob"`     // matched against file_path on Read/Write/Edit
	MCPAction    string `yaml:"mcp_action"`    // send|create|update|delete, or an "A|B" list; MCP tools only
}

// on_miss values: what a require-skill deny carries when the skill isn't loaded.
const (
	OnMissLoadRequired = "load-required" // name the skill; the agent invokes it (default)
	OnMissDigest       = "digest"        // the binding's compiled digest rides in the deny
)

// MaxDigest is the largest digest, in bytes, that rides in a deny. Claude Code caps hook
// output at 10,000 characters; the deny's own wording needs the rest.
const MaxDigest = 7000

// Action is what paws does when a rule matches.
type Action struct {
	Type        string `yaml:"type"`
	Skill       string `yaml:"skill"`
	DenyMessage string `yaml:"deny_message"`
	Message     string `yaml:"message"`
	Script      string `yaml:"script"`
	OnMiss      string `yaml:"on_miss"`
	Digest      string `yaml:"digest"` // path to the digest file, relative to the rule's directory
}

// Upstream pins the third-party skill a catalog binding installs. The runner never reads it.
type Upstream struct {
	Repo        string `yaml:"repo"`
	Path        string `yaml:"path"`
	Ref         string `yaml:"ref"`
	ContentHash string `yaml:"content_hash"`
}

// Provenance is catalog metadata about an upstream skill, shown on the catalog page.
type Provenance struct {
	Stars    int    `yaml:"stars"`
	Installs int    `yaml:"installs"`
	License  string `yaml:"license"`
	LastPush string `yaml:"last_push"`
	Fetched  string `yaml:"fetched"`
}

// Rule is one .paws/*.md file: frontmatter directives + a human-readable body.
// A catalog binding is the same file with upstream/provenance/tags filled in;
// unknown keys are ignored so the two formats stay one.
type Rule struct {
	Name       string     `yaml:"name"`
	Event      string     `yaml:"event"`
	Matcher    Matcher    `yaml:"matcher"`
	Action     Action     `yaml:"action"`
	When       When       `yaml:"when"`
	Upstream   Upstream   `yaml:"upstream"`
	Provenance Provenance `yaml:"provenance"`
	Tags       []string   `yaml:"tags"`

	SourcePath string         `yaml:"-"`
	Body       string         `yaml:"-"` // the markdown after the frontmatter: the rule's README
	re         *regexp.Regexp `yaml:"-"`
	fileRe     *regexp.Regexp `yaml:"-"`
}

// Kind maps a paws event to its rule kind; unknown events map to "".
func Kind(e string) string {
	switch e {
	case "pre-tool", "pre-commit", "pre-push":
		return KindTool
	case "user-prompt":
		return KindPrompt
	case "session-start":
		return KindSession
	}
	return ""
}

// Parse reads one markdown file's frontmatter into a validated Rule.
func Parse(data []byte, path string) (*Rule, error) {
	fm, err := frontmatter(data)
	if err != nil {
		return nil, err
	}
	var r Rule
	if err := yaml.Unmarshal(fm, &r); err != nil {
		return nil, fmt.Errorf("bad frontmatter: %w", err)
	}
	r.SourcePath = path
	r.Body = body(data)
	if Kind(r.Event) == "" {
		return nil, fmt.Errorf("missing or unknown event %q", r.Event)
	}
	if !validActions[r.Action.Type] {
		return nil, fmt.Errorf("missing or unknown action.type %q", r.Action.Type)
	}
	if r.Action.Type == "require-skill" && r.Action.Skill == "" {
		return nil, fmt.Errorf("require-skill needs a skill")
	}
	switch r.Action.OnMiss {
	case "", OnMissLoadRequired:
	case OnMissDigest:
		if r.Action.Digest == "" {
			return nil, fmt.Errorf("on_miss: digest needs a digest file")
		}
	default:
		return nil, fmt.Errorf("unknown action.on_miss %q (want %s or %s)", r.Action.OnMiss, OnMissLoadRequired, OnMissDigest)
	}
	if r.Action.OnMiss != "" && r.Action.Type != "require-skill" {
		return nil, fmt.Errorf("on_miss only applies to require-skill")
	}
	if r.Action.Digest != "" {
		if err := RelativeInside(r.Action.Digest); err != nil {
			return nil, fmt.Errorf("digest %w", err)
		}
	}
	if r.Action.Type == "exec-script" {
		if r.Action.Script == "" {
			return nil, fmt.Errorf("exec-script needs a script path")
		}
		// Scripts run with full shell trust, same as the rule file itself — but the
		// path must stay inside the rule's own directory tree, not point anywhere
		// on disk, so `paws explain`/review of a .paws/ dir sees everything it runs.
		if err := RelativeInside(r.Action.Script); err != nil {
			return nil, fmt.Errorf("exec-script script %w", err)
		}
	}
	if Kind(r.Event) == KindSession {
		if !sessionActions[r.Action.Type] {
			return nil, fmt.Errorf("session-start supports require-skill, advise and audit, not %s", r.Action.Type)
		}
		if r.Matcher.Tool != "" || r.Matcher.FileGlob != "" || r.Matcher.MCPAction != "" {
			return nil, fmt.Errorf("session-start has no tool call; only matcher.command_regex (against the start source) applies")
		}
	}
	if r.Action.Type == "advise" {
		if Kind(r.Event) == KindPrompt {
			return nil, fmt.Errorf("advise is for tool and session events (use prompt-inject on user-prompt)")
		}
		if r.Action.Message == "" {
			return nil, fmt.Errorf("advise needs a message")
		}
	}
	if r.Matcher.MCPAction != "" {
		for _, a := range strings.Split(r.Matcher.MCPAction, "|") {
			if !mcpActions[strings.TrimSpace(a)] {
				return nil, fmt.Errorf("unknown matcher.mcp_action %q (want send, create, update or delete)", a)
			}
		}
	}
	if r.Matcher.FileGlob != "" {
		re, err := globRegexp(r.Matcher.FileGlob)
		if err != nil {
			return nil, fmt.Errorf("bad file_glob: %w", err)
		}
		r.fileRe = re
	}
	for _, c := range r.When {
		if c.Kind == WhenNewFile && Kind(r.Event) != KindTool {
			return nil, fmt.Errorf("when: new-file needs a tool event")
		}
	}
	if r.Action.Type == "warn" && Kind(r.Event) == KindPrompt {
		return nil, fmt.Errorf("warn is not supported on user-prompt (use block, audit, or prompt-inject)")
	}
	if r.Action.Type == "audit" && r.Action.Message != "" {
		return nil, fmt.Errorf("audit is silent by design and does not support message: (use prompt-inject to surface text to the agent)")
	}
	if r.Action.Type == "prompt-inject" {
		if Kind(r.Event) != KindPrompt {
			return nil, fmt.Errorf("prompt-inject requires event: user-prompt")
		}
		if r.Action.Message == "" {
			return nil, fmt.Errorf("prompt-inject needs a message")
		}
	}
	if (r.Event == "pre-commit" || r.Event == "pre-push") && !matchTool(r.Matcher.Tool, "Bash") {
		return nil, fmt.Errorf("%s rules only fire on Bash; matcher.tool %q can never match", r.Event, r.Matcher.Tool)
	}
	if r.Matcher.CommandRegex != "" {
		re, err := regexp.Compile(r.Matcher.CommandRegex)
		if err != nil {
			return nil, fmt.Errorf("bad command_regex: %w", err)
		}
		r.re = re
	}
	return &r, nil
}

// IsRuleFile reports whether a path relative to a .paws or catalog root is a rule/binding file.
// READMEs, skills (skills/**, SKILL.md) and digests (digests/**) live beside rules and are not rules.
func IsRuleFile(rel string) bool {
	if filepath.Ext(rel) != ".md" {
		return false
	}
	if base := filepath.Base(rel); base == "README.md" || base == "SKILL.md" {
		return false
	}
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/") {
		if seg == "skills" || seg == "digests" {
			return false
		}
	}
	return true
}

// CheckKnownFields re-decodes the frontmatter strictly: a misspelled key (comand_regex) is an error.
// Parse stays lenient so a newer rule file never breaks an older runner; catalog validation uses this.
func CheckKnownFields(data []byte) error {
	fm, err := frontmatter(data)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(fm))
	dec.KnownFields(true)
	var r Rule
	if err := dec.Decode(&r); err != nil && err != io.EOF {
		return fmt.Errorf("frontmatter: %w", err)
	}
	return nil
}

// RelativeInside rejects a path that is absolute or escapes the rule's directory.
func RelativeInside(p string) error {
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("must be a relative path inside the rule's directory, got %q", p)
	}
	return nil
}

// body returns the text after the closing frontmatter fence, trimmed.
func body(data []byte) string {
	s := bytes.TrimLeft(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), " \t\r\n")
	if !bytes.HasPrefix(s, []byte("---")) {
		return ""
	}
	i := bytes.Index(s[3:], []byte("\n---"))
	if i < 0 {
		return ""
	}
	rest := s[3+i+4:]
	if j := bytes.IndexByte(rest, '\n'); j >= 0 {
		rest = rest[j+1:]
	} else {
		rest = nil
	}
	return strings.TrimSpace(string(rest))
}

// frontmatter returns the bytes between the leading "---" fence and its close.
func frontmatter(data []byte) ([]byte, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	s := bytes.TrimLeft(data, " \t\r\n")
	if !bytes.HasPrefix(s, []byte("---")) {
		return nil, fmt.Errorf("no frontmatter fence")
	}
	s = s[3:]
	if i := bytes.Index(s, []byte("\n---")); i >= 0 {
		return s[:i], nil
	}
	return nil, fmt.Errorf("unterminated frontmatter")
}
