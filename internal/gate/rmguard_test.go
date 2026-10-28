package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/websentry-ai/paws/internal/rule"
)

// The catalog's rm-guard script reads the target from Go-marshalled JSON, where & < > arrive
// as & < >: chained commands must still split.
func TestCatalogRmGuard(t *testing.T) {
	path, _ := filepath.Abs("../../catalog/filesystem-operations/rm-guard.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := rule.Parse(data, path)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"rm -rf /":                       true,
		"cd x && rm -rf ~":               true,
		"make clean; rm -rf /usr":        true,
		"true || rm -fr ..":              true,
		`rm -rf "$OUT/"`:                 true,
		"sudo rm --recursive --force /*": true,
		"rm -rf build":                   false,
		"rm -rf build && echo ok":        false,
		"rm -f /tmp/x":                   false,
		`echo "rm -rf /"`:                false,
		"git rm -r --cached .":           false,
		"rm -rf ~/projects/old":          false,
		"rm -rf a > /dev/null 2>&1":      false,
	}
	for cmd, deny := range cases {
		raw, _ := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]string{"command": cmd}})
		oc := runScript(r, Event{Kind: rule.KindTool, Tool: "Bash", Target: cmd, Raw: raw})
		if got := oc.verdict == DenyRetry; got != deny {
			t.Errorf("%q: deny=%v, want %v (%+v)", cmd, got, deny, oc)
		}
	}
}
