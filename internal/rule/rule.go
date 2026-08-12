// Package rule parses .paws/*.md frontmatter into matchable rules. Pure — no I/O.
package rule

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule kinds: which class of agent event a rule applies to. Not exhaustive —
// lifecycle/audit-only events (session start, stop) have no kind here yet.
const (
	KindTool   = "tool"   // a pending tool call
	KindPrompt = "prompt" // a submitted user prompt
)

var validActions = map[string]bool{
	"block": true, "require-skill": true, "warn": true, "audit": true,
	"prompt-inject": true, "exec-script": true,
}

// Matcher decides whether a rule applies to a pending call.
type Matcher struct {
	Tool         string `yaml:"tool"`          // tool name, "A|B" list, or omit / quoted "*" for any
	CommandRegex string `yaml:"command_regex"` // matched against the extracted command
}

// Action is what paws does when a rule matches.
type Action struct {
	Type        string `yaml:"type"`
	Skill       string `yaml:"skill"`
	DenyMessage string `yaml:"deny_message"`
	Message     string `yaml:"message"`
	Script      string `yaml:"script"`
}

// Rule is one .paws/*.md file: frontmatter directives + a human-readable body.
type Rule struct {
	Name    string  `yaml:"name"`
	Event   string  `yaml:"event"`
	Matcher Matcher `yaml:"matcher"`
	Action  Action  `yaml:"action"`

	SourcePath string         `yaml:"-"`
	re         *regexp.Regexp `yaml:"-"`
}

// Kind maps a paws event to its rule kind; unknown events map to "".
func Kind(e string) string {
	switch e {
	case "pre-tool", "pre-commit", "pre-push":
		return KindTool
	case "user-prompt":
		return KindPrompt
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
	if Kind(r.Event) == "" {
		return nil, fmt.Errorf("missing or unknown event %q", r.Event)
	}
	if !validActions[r.Action.Type] {
		return nil, fmt.Errorf("missing or unknown action.type %q", r.Action.Type)
	}
	if r.Action.Type == "require-skill" && r.Action.Skill == "" {
		return nil, fmt.Errorf("require-skill needs a skill")
	}
	if r.Action.Type == "exec-script" {
		if r.Action.Script == "" {
			return nil, fmt.Errorf("exec-script needs a script path")
		}
		// Scripts run with full shell trust, same as the rule file itself — but the
		// path must stay inside the rule's own directory tree, not point anywhere
		// on disk, so `paws explain`/review of a .paws/ dir sees everything it runs.
		clean := filepath.Clean(r.Action.Script)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("exec-script script must be a relative path inside the rule's directory, got %q", r.Action.Script)
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
