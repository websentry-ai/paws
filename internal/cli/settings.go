package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/websentry-ai/paws/internal/hook"
)

// settingsDoc keeps every top-level value as raw bytes so foreign keys survive a rewrite verbatim.
type settingsDoc map[string]json.RawMessage

func readSettings(path string) (settingsDoc, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settingsDoc{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc settingsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc == nil {
		doc = settingsDoc{}
	}
	return doc, nil
}

// writeSettings writes atomically: temp file in the same dir, then rename.
func writeSettings(path string, doc settingsDoc) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	mode := os.FileMode(0o600)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	if err := os.Chmod(tmp, mode); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ensureHooks adds (or repoints, if the binary moved) a paws entry per registered event; idempotent, other hooks untouched.
func ensureHooks(doc settingsDoc, command string) (bool, error) {
	hooks, err := hooksBlock(doc)
	if err != nil {
		return false, err
	}
	changed := false
	for _, event := range hook.RegisteredEvents {
		var list []json.RawMessage
		if raw, ok := hooks[event]; ok {
			if err := json.Unmarshal(raw, &list); err != nil {
				return false, fmt.Errorf("hooks.%s: %w", event, err)
			}
		}
		desired := hook.RawEntry(event, command)
		idx := pawsEntryIndex(list)
		switch {
		case idx < 0:
			list = append(list, desired)
		case jsonEqual(list[idx], desired):
			continue
		default:
			list[idx] = desired
		}
		changed = true
		b, err := json.Marshal(list)
		if err != nil {
			return false, err
		}
		hooks[event] = b
	}
	if changed {
		return true, setHooksBlock(doc, hooks)
	}
	return false, nil
}

// removeHooks strips paws entries from every event list.
func removeHooks(doc settingsDoc) (bool, error) {
	hooks, err := hooksBlock(doc)
	if err != nil {
		return false, err
	}
	changed := false
	for event, raw := range hooks {
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			continue // not an entry list; leave it alone
		}
		var kept []json.RawMessage
		for _, entry := range list {
			if isPawsEntry(entry) {
				changed = true
				continue
			}
			kept = append(kept, entry)
		}
		if len(kept) == len(list) {
			continue
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		b, err := json.Marshal(kept)
		if err != nil {
			return false, err
		}
		hooks[event] = b
	}
	if changed {
		if len(hooks) == 0 {
			delete(doc, "hooks")
			return true, nil
		}
		return true, setHooksBlock(doc, hooks)
	}
	return false, nil
}

func hooksBlock(doc settingsDoc) (map[string]json.RawMessage, error) {
	hooks := map[string]json.RawMessage{}
	if raw, ok := doc["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			return nil, fmt.Errorf("hooks block: %w", err)
		}
	}
	return hooks, nil
}

func setHooksBlock(doc settingsDoc, hooks map[string]json.RawMessage) error {
	b, err := json.Marshal(hooks)
	if err != nil {
		return err
	}
	doc["hooks"] = b
	return nil
}

// pawsEntryIndex finds our entry in an event list, tolerating older formats so re-running init upgrades, not duplicates.
func pawsEntryIndex(list []json.RawMessage) int {
	for i, entry := range list {
		if isPawsEntry(entry) {
			return i
		}
	}
	return -1
}

func isPawsEntry(entry json.RawMessage) bool {
	var e struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if json.Unmarshal(entry, &e) != nil {
		return false
	}
	for _, h := range e.Hooks {
		if isPawsCommand(h.Command) {
			return true
		}
	}
	return false
}

// Exact match, not substring — "paws" anywhere in an unrelated command shouldn't count as ours.
func isPawsCommand(command string) bool {
	const suffix = " gate"
	if !strings.HasSuffix(command, suffix) {
		return false
	}
	bin := strings.Trim(strings.TrimSuffix(command, suffix), `'"`)
	return filepath.Base(bin) == "paws"
}

func jsonEqual(a, b json.RawMessage) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
