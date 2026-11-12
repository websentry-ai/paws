package catalog

import (
	"fmt"
	"sort"
)

// TriggerCases is a catalog's trigger_cases.json: for a binding, calls its trigger must fire on and near misses it
// must ignore, each a [tool, target] pair (target is the command, file path, URL, prompt, MCP input or session source
// the hook matches against).
type TriggerCases map[string]struct {
	Fires   [][2]string `json:"fires"`
	Ignores [][2]string `json:"ignores"`
}

// CheckTriggerCases pins each binding's trigger to the moments it was reviewed for. A trigger that fires when its
// skill doesn't apply blocks the call until Claude loads a skill it doesn't need, so every binding needs cases,
// except the ones in exempt (from before cases existed). It returns one problem per missing case list, case naming
// no binding, and call a trigger gets wrong.
func CheckTriggerCases(c *Catalog, cases TriggerCases, exempt map[string]bool) []string {
	byName := map[string]*Binding{}
	for _, b := range c.Bindings {
		byName[b.Name] = b
	}
	var probs []string
	for name := range byName {
		if _, ok := cases[name]; !ok && !exempt[name] {
			probs = append(probs, fmt.Sprintf("%s: add its fire and ignore cases to trigger_cases.json", name))
		}
	}
	for name := range exempt {
		if byName[name] == nil {
			probs = append(probs, fmt.Sprintf("%s: exempt from cases but no longer in the catalog; remove it from the list", name))
		} else if _, ok := cases[name]; ok {
			probs = append(probs, fmt.Sprintf("%s: has cases now; remove it from the exempt list", name))
		}
	}
	for name, tc := range cases {
		b := byName[name]
		if b == nil {
			probs = append(probs, fmt.Sprintf("%s: has cases but is not in the catalog", name))
			continue
		}
		if len(tc.Fires) == 0 || len(tc.Ignores) == 0 {
			probs = append(probs, fmt.Sprintf("%s: needs calls it fires on and near misses it ignores", name))
		}
		for _, call := range tc.Fires {
			if !b.Matches(call[0], call[1]) {
				probs = append(probs, fmt.Sprintf("%s: should fire on %s %q", name, call[0], call[1]))
			}
		}
		for _, call := range tc.Ignores {
			if b.Matches(call[0], call[1]) {
				probs = append(probs, fmt.Sprintf("%s: should ignore %s %q", name, call[0], call[1]))
			}
		}
	}
	sort.Strings(probs)
	return probs
}
