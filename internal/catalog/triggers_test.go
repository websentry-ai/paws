package catalog

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// triggerCases is testdata/trigger_cases.json: for a catalog binding, calls its trigger must fire on and near
// misses it must ignore, each a [tool, target] pair (target is the command, file path, URL, prompt, MCP input or
// session source the hook matches against).
type triggerCases map[string]struct {
	Fires   [][2]string `json:"fires"`
	Ignores [][2]string `json:"ignores"`
}

// A trigger that fires when its skill doesn't apply blocks the call until Claude loads a skill it doesn't need.
// These cases pin each listed binding's trigger to the moments it was reviewed for, so a later edit can't widen it
// unnoticed.
func TestCatalogTriggersFireOnlyWhereReviewed(t *testing.T) {
	c, probs := Load("../..")
	if len(probs) > 0 {
		t.Fatalf("catalog problems: %v", probs)
	}
	data, err := os.ReadFile("testdata/trigger_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases triggerCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	byName := map[string]*Binding{}
	for _, b := range c.Bindings {
		byName[b.Name] = b
	}
	// Every binding needs cases, except the ones from before this file, listed in without_trigger_cases.txt.
	older := map[string]bool{}
	list, err := os.ReadFile("testdata/without_trigger_cases.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(list), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			older[line] = true
		}
	}
	for name := range byName {
		if _, ok := cases[name]; !ok && !older[name] {
			t.Errorf("%s: add its fire and ignore cases to testdata/trigger_cases.json", name)
		}
	}
	for name := range older {
		if byName[name] == nil {
			t.Errorf("%s: in without_trigger_cases.txt but no longer in the catalog; remove it from the list", name)
		} else if _, ok := cases[name]; ok {
			t.Errorf("%s: has cases now; remove it from without_trigger_cases.txt", name)
		}
	}
	for name, tc := range cases {
		b := byName[name]
		if b == nil {
			t.Errorf("%s: in trigger_cases.json but not in the catalog", name)
			continue
		}
		if len(tc.Fires) == 0 || len(tc.Ignores) == 0 {
			t.Errorf("%s: needs calls it fires on and near misses it ignores", name)
		}
		for _, call := range tc.Fires {
			if !b.Matches(call[0], call[1]) {
				t.Errorf("%s: should fire on %s %q", name, call[0], call[1])
			}
		}
		for _, call := range tc.Ignores {
			if b.Matches(call[0], call[1]) {
				t.Errorf("%s: should ignore %s %q", name, call[0], call[1])
			}
		}
	}
}
