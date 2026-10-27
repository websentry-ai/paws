package rule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Condition kinds for `when:`. All of a rule's conditions must hold for it to apply.
const (
	WhenDependency     = "dependency"       // dependency:PKG — PKG is a dependency at the repo root
	WhenRepoMarker     = "repo-marker"      // repo-marker:PATH — PATH exists under the repo root
	WhenNewFile        = "new-file"         // the call's target file does not exist yet
	WhenNoEditMatching = "no-edit-matching" // no-edit-matching:GLOB — no earlier Edit/Write this session on a matching path
	WhenAgentCount     = "agent-count"      // agent-count>N — more than N Agent spawns since the last user turn
)

// Condition is one parsed `when:` entry. The gate evaluates it; parsing stays pure.
type Condition struct {
	Kind string
	Arg  string         // PKG, PATH or GLOB
	N    int            // agent-count threshold
	Glob *regexp.Regexp // compiled no-edit-matching glob
}

// When holds a rule's conditions; YAML accepts one string or a list of strings.
type When []Condition

var agentCountRe = regexp.MustCompile(`^agent-count\s*>\s*(\d+)$`)

func (w *When) UnmarshalYAML(n *yaml.Node) error {
	var items []string
	switch n.Kind {
	case yaml.ScalarNode:
		items = []string{n.Value}
	case yaml.SequenceNode:
		if err := n.Decode(&items); err != nil {
			return err
		}
	default:
		return fmt.Errorf("when: must be a string or a list of strings")
	}
	for _, it := range items {
		c, err := parseCondition(strings.TrimSpace(it))
		if err != nil {
			return err
		}
		*w = append(*w, c)
	}
	return nil
}

func parseCondition(s string) (Condition, error) {
	if m := agentCountRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return Condition{Kind: WhenAgentCount, N: n}, nil
	}
	if s == WhenNewFile {
		return Condition{Kind: WhenNewFile}, nil
	}
	kind, arg, ok := strings.Cut(s, ":")
	arg = strings.TrimSpace(arg)
	if !ok || arg == "" {
		return Condition{}, fmt.Errorf("unknown when: condition %q", s)
	}
	switch kind {
	case WhenDependency:
		return Condition{Kind: kind, Arg: arg}, nil
	case WhenRepoMarker:
		if err := RelativeInside(arg); err != nil {
			return Condition{}, fmt.Errorf("when: repo-marker %w", err)
		}
		return Condition{Kind: kind, Arg: arg}, nil
	case WhenNoEditMatching:
		re, err := globRegexp(arg)
		if err != nil {
			return Condition{}, fmt.Errorf("when: no-edit-matching: %w", err)
		}
		return Condition{Kind: kind, Arg: arg, Glob: re}, nil
	}
	return Condition{}, fmt.Errorf("unknown when: condition %q", s)
}
