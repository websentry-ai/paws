package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const pawsCmd = pawsMarker + `'/opt/paws' gate`

func TestEnsureHooksMergesAndIsIdempotent(t *testing.T) {
	doc := settingsDoc{
		"env":   json.RawMessage(`{"FOO":"bar"}`),
		"hooks": json.RawMessage(`{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"\"/x/unbound.py\"","timeout":15000}]}]}`),
	}

	changed, err := ensureHooks(doc, pawsCmd)
	if err != nil || !changed {
		t.Fatalf("first ensureHooks should report a change: %v %v", changed, err)
	}
	changed, err = ensureHooks(doc, pawsCmd)
	if err != nil || changed {
		t.Fatalf("second ensureHooks should be a no-op: %v %v", changed, err)
	}

	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	if err := json.Unmarshal(hooks["PreToolUse"], &pre); err != nil {
		t.Fatal(err)
	}
	if len(pre) != 2 {
		t.Fatalf("expected 2 PreToolUse entries, got %d", len(pre))
	}
	if !containsCommand(pre, `"/x/unbound.py"`) {
		t.Error("existing unbound.py entry was dropped")
	}
	if !containsCommand(pre, pawsCmd) {
		t.Error("paws entry was not added")
	}
	if string(doc["env"]) != `{"FOO":"bar"}` {
		t.Error("env block was mutated")
	}
}

func TestEnsureHooksFromEmpty(t *testing.T) {
	doc := settingsDoc{}
	if changed, err := ensureHooks(doc, pawsCmd); err != nil || !changed {
		t.Fatalf("should add hooks to empty settings: %v %v", changed, err)
	}
	hooks, err := hooksBlock(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Fatal("PreToolUse not created")
	}
}

func TestEnsureHooksRepointsMovedBinary(t *testing.T) {
	doc := settingsDoc{}
	if _, err := ensureHooks(doc, pawsMarker+`'/old/path/paws' gate`); err != nil {
		t.Fatal(err)
	}
	changed, err := ensureHooks(doc, pawsMarker+`'/new/path/paws' gate`)
	if err != nil || !changed {
		t.Fatalf("repointing to a new path should report a change: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 {
		t.Fatalf("moved binary should repoint the existing entry, not duplicate it; got %d entries", len(pre))
	}
	if !containsCommand(pre, pawsMarker+`'/new/path/paws' gate`) {
		t.Error("entry was not repointed to the new path")
	}
}

func TestIsPawsCommandRejectsLookalikes(t *testing.T) {
	lookalikes := []string{
		`'/opt/paws' gate`,                  // right shape, no marker — pre-fix this matched via basename
		`'/opt/other/paws-utils/check' gate`, // "paws" only as a path component
		`echo paws && something gate`,
	}
	for _, c := range lookalikes {
		if isPawsCommand(c) {
			t.Errorf("isPawsCommand(%q) = true, want false", c)
		}
	}
	if !isPawsCommand(pawsCmd) {
		t.Errorf("isPawsCommand(%q) = false, want true", pawsCmd)
	}
}

func TestRemoveHooksStripsOnlyPaws(t *testing.T) {
	doc := settingsDoc{
		"hooks": json.RawMessage(`{"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"\"/x/unbound.py\"","timeout":15000}]}]}`),
	}
	if _, err := ensureHooks(doc, pawsCmd); err != nil {
		t.Fatal(err)
	}
	changed, err := removeHooks(doc)
	if err != nil || !changed {
		t.Fatalf("removeHooks should report a change: %v %v", changed, err)
	}
	hooks, _ := hooksBlock(doc)
	var pre []json.RawMessage
	json.Unmarshal(hooks["PreToolUse"], &pre)
	if len(pre) != 1 || containsCommand(pre, pawsCmd) {
		t.Fatalf("expected only the unbound.py entry left, got %s", hooks["PreToolUse"])
	}
}

func TestReadSettingsMalformedDoesNotDestroy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{invalid`), 0o644)

	if _, err := readSettings(path); err == nil {
		t.Fatal("malformed settings.json should error, not silently reset")
	}
	data, _ := os.ReadFile(path)
	if string(data) != `{invalid` {
		t.Error("readSettings must not modify the file on parse failure")
	}
}

func TestReadSettingsMissingIsEmpty(t *testing.T) {
	doc, err := readSettings(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || len(doc) != 0 {
		t.Fatalf("missing file should yield an empty doc, got %v %v", doc, err)
	}
}

func TestWriteSettingsRoundTripsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	doc := settingsDoc{"someOtherPlugin": json.RawMessage(`{"n":123456789012345}`)}
	if err := writeSettings(path, doc); err != nil {
		t.Fatal(err)
	}
	got, err := readSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	var v struct{ N int64 }
	if err := json.Unmarshal(got["someOtherPlugin"], &v); err != nil || v.N != 123456789012345 {
		t.Errorf("large integer lost precision, got %s", got["someOtherPlugin"])
	}
}

func containsCommand(list []json.RawMessage, command string) bool {
	for _, entry := range list {
		var e struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		}
		if json.Unmarshal(entry, &e) != nil {
			continue
		}
		for _, h := range e.Hooks {
			if h.Command == command {
				return true
			}
		}
	}
	return false
}
