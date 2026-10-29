package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"

	"github.com/websentry-ai/paws/internal/gate"
)

// RuleStats summarises one rule's audit records.
type RuleStats struct {
	Rule string `json:"rule"`
	// Fires is every record the rule produced; the other counts split it.
	Fires int `json:"fires"`
	// Denied counts denies.
	Denied int `json:"denied"`
	// AlreadyLoaded counts allows via skill-load in sessions where this rule had not denied first:
	// the agent had the skill without being told. Over time this is internalization.
	AlreadyLoaded int `json:"already_loaded"`
	// LoadedAfterDeny counts allows via skill-load that followed a deny in the same session.
	LoadedAfterDeny int `json:"loaded_after_deny"`
	// Delivered counts digests and session-start context the agent received instead of loading a skill.
	Delivered int `json:"delivered"`
}

// Stats is the whole local summary.
type Stats struct {
	Rules       []RuleStats    `json:"rules"`
	Dormant     []string       `json:"dormant"` // rules in scope that never fired
	Skills      map[string]int `json:"skills_invoked"`
	Sessions    int            `json:"sessions"`
	Turns       int            `json:"turns"`
	ToolResults int            `json:"tool_calls"`
}

type auditRecord struct {
	TS          string `json:"ts"`
	Event       string `json:"event"`
	Rule        string `json:"rule"`
	Verdict     string `json:"verdict"`
	SatisfiedBy string `json:"satisfied_by"`
	Context     bool   `json:"context_added"`
	Session     string `json:"session"`
	Skill       string `json:"skill"`
	Source      string `json:"source"`
}

// summarize reads audit records in order (rotated file first) and counts them.
func summarize(paths []string, inScope []string) Stats {
	// Empty lists, not nil: JSON consumers (paws ui, scripts) get [] instead of null.
	st := Stats{Skills: map[string]int{}, Rules: []RuleStats{}, Dormant: []string{}}
	byRule := map[string]*RuleStats{}
	deniedIn := map[string]bool{} // rule|session pairs that saw a deny
	sessions := map[string]bool{}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		rd := bufio.NewReader(f)
		for {
			line, err := rd.ReadBytes('\n') // any length: one huge record must not end the count
			if len(line) == 0 && err != nil {
				break
			}
			var r auditRecord
			if json.Unmarshal(line, &r) != nil {
				continue
			}
			if r.Session != "" {
				sessions[r.Session] = true
			}
			switch r.Event {
			case "turn_end":
				st.Turns++
				continue
			case "tool_result":
				st.ToolResults++
				continue
			case "skill_invoked":
				st.Skills[r.Skill]++
				continue
			}
			if r.Rule == "" {
				continue
			}
			rs := byRule[r.Rule]
			if rs == nil {
				rs = &RuleStats{Rule: r.Rule}
				byRule[r.Rule] = rs
			}
			rs.Fires++
			key := r.Rule + "|" + r.Session
			switch {
			case r.Verdict == "deny":
				rs.Denied++
				deniedIn[key] = true
			case r.SatisfiedBy == "skill-load" && deniedIn[key]:
				rs.LoadedAfterDeny++
			case r.SatisfiedBy == "skill-load":
				rs.AlreadyLoaded++
			case r.SatisfiedBy == "digest-delivered" || r.Context:
				rs.Delivered++
			}
		}
		f.Close()
	}
	for _, rs := range byRule {
		st.Rules = append(st.Rules, *rs)
	}
	sort.Slice(st.Rules, func(i, j int) bool {
		return st.Rules[i].Fires > st.Rules[j].Fires || st.Rules[i].Fires == st.Rules[j].Fires && st.Rules[i].Rule < st.Rules[j].Rule
	})
	for _, name := range inScope {
		if byRule[name] == nil {
			st.Dormant = append(st.Dormant, name)
		}
	}
	sort.Strings(st.Dormant)
	st.Sessions = len(sessions)
	return st
}

// localStats summarises this user's audit log against the rules in scope for the cwd.
func localStats() (Stats, string) {
	log := gate.AuditPath()
	cwd, _ := os.Getwd()
	var names []string
	seen := map[string]bool{}
	for _, r := range gate.Collect(cwd).Rules {
		if !seen[r.Name] {
			seen[r.Name] = true
			names = append(names, r.Name)
		}
	}
	return summarize([]string{log + ".1", log}, names), log
}
