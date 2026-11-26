package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/websentry-ai/paws/internal/harness"
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
	if len(bytes.TrimSpace(data)) == 0 {
		return settingsDoc{}, nil // an empty file is an empty object, e.g. one just created by hand
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc == nil {
		doc = settingsDoc{}
	}
	return doc, nil
}

// writeSettings writes atomically: temp file in the same dir, then rename.
// marshalPlain is json.Marshal (or MarshalIndent, with indent) that leaves < > & as they are. Go escapes them as
// \u003c \u003e \u0026 by default, which makes the guard command unreadable in a committed settings.json. Every
// step that wraps a hook command in JSON must use it, because each json.Marshal of a RawMessage escapes it again.
func marshalPlain(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func writeSettings(path string, doc settingsDoc) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := marshalPlain(doc, "  ")
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

// ensureHooks is ensureHooksFor Claude Code.
func ensureHooks(doc settingsDoc, command string, guard bool) (bool, error) {
	return ensureHooksFor(hook.Claude{}, doc, command, guard)
}

// ensureHooksFor adds (or repoints, if the binary moved) a paws entry per event the harness registers; idempotent,
// other hooks untouched.
func ensureHooksFor(a harness.Adapter, doc settingsDoc, command string, guard bool) (bool, error) {
	hooks, err := hooksBlock(doc)
	if err != nil {
		return false, err
	}
	changed := false
	for _, event := range a.Events() {
		var list []json.RawMessage
		if raw, ok := hooks[event]; ok {
			if err := json.Unmarshal(raw, &list); err != nil {
				return false, fmt.Errorf("hooks.%s: %w", event, err)
			}
		}
		entry := a.Entry(event, command)
		if guard {
			entry = a.Entry(event, guardedCommand(command, event))
		}
		desired, err := json.Marshal(entry)
		if err != nil {
			return false, err
		}
		idx := pawsEntryIndex(list)
		switch {
		case idx < 0:
			list = append(list, desired)
		case jsonEqual(list[idx], desired):
			continue
		default:
			// Repoint our hook; a hook someone else added to the same entry stays where it is.
			if rest, _ := withoutPawsHooks(list[idx]); rest == nil {
				list[idx] = desired
			} else {
				list[idx] = rest
				list = append(list, desired)
			}
		}
		changed = true
		b, err := marshalPlain(list, "")
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
		kept, stripped := stripPaws(list)
		if !stripped {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		b, err := marshalPlain(kept, "")
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
	b, err := marshalPlain(hooks, "")
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

// stripPaws removes paws's hooks from an event's entry list, dropping entries left empty.
func stripPaws(list []json.RawMessage) (kept []json.RawMessage, stripped bool) {
	for _, entry := range list {
		rest, had := withoutPawsHooks(entry)
		stripped = stripped || had
		if rest != nil {
			kept = append(kept, rest)
		}
	}
	return kept, stripped
}

// withoutPawsHooks returns entry minus paws's own hook objects and whether it had any; rest is
// nil when nothing else was in it. A hook a user added beside ours in the same entry survives.
func withoutPawsHooks(entry json.RawMessage) (rest json.RawMessage, had bool) {
	var e map[string]json.RawMessage
	var hooks []json.RawMessage
	if json.Unmarshal(entry, &e) != nil || json.Unmarshal(e["hooks"], &hooks) != nil {
		return entry, false
	}
	var kept []json.RawMessage
	for _, h := range hooks {
		var c struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(h, &c) == nil && isPawsCommand(c.Command) {
			continue
		}
		kept = append(kept, h)
	}
	if len(kept) == len(hooks) {
		return entry, false
	}
	if len(kept) == 0 {
		return nil, true
	}
	e["hooks"], _ = marshalPlain(kept, "")
	rest, _ = marshalPlain(e, "")
	return rest, true
}

// missingMsg is what a teammate sees when a committed hook runs on a machine without paws.
const missingMsg = "paws is not installed; install it with npm i -g unbound-paws (or see https://paws.getunbound.ai)"

// guardMid is the middle of a guarded hook command. exit is 2 for PreToolUse, where Claude Code blocks the call and
// shows stderr to Claude, and 1 for the other events, where it only shows the error: exit 2 on UserPromptSubmit or
// Stop would block the prompt or the end of the turn.
func guardMid(exit string) string {
	return " >/dev/null 2>&1 || { echo '" + missingMsg + "' >&2; exit " + exit + "; }; "
}

// guardedCommand makes a project hook fail loudly where the binary is missing, instead of a vague "command not
// found". It is for sh, which Claude Code runs hooks with on macOS and Linux.
func guardedCommand(command, event string) string {
	exit := "1"
	if event == "PreToolUse" {
		exit = "2"
	}
	return "command -v " + strings.TrimSuffix(command, " gate") + guardMid(exit) + command
}

// unguard returns the plain `<prog> gate` inside a command guardedCommand wrote, or command itself.
func unguard(command string) string {
	rest, ok := strings.CutPrefix(command, "command -v ")
	if !ok {
		return command
	}
	for _, exit := range []string{"1", "2"} {
		mid := guardMid(exit)
		if i := strings.Index(rest, mid); i >= 0 {
			if prog, plain := rest[:i], rest[i+len(mid):]; plain == prog+" gate" {
				return plain
			}
		}
	}
	return command
}

// legacyMarker prefixed hook commands written by paws 0.3.0 and earlier; still recognised so
// init upgrades and uninstall removes those entries.
const legacyMarker = "PAWS_MANAGED=1 "

// isPawsCommand recognises exactly the commands init writes: `paws gate`, `"$HOME/<path>" gate` or a
// shellQuote'd path then ` gate`, the binary named paws (paws.exe), or any such single path
// after 0.3.0's marker. Nothing that can expand or chain (`"$(x)/paws"`, `scan; paws gate`) counts.
func isPawsCommand(command string) bool {
	command = unguard(command)
	rest, marked := strings.CutPrefix(command, legacyMarker)
	prog, ok := strings.CutSuffix(rest, " gate")
	if !ok {
		return false
	}
	if prog == cliName {
		return true
	}
	var path string
	switch {
	case strings.HasPrefix(prog, `"$HOME/`) && strings.HasSuffix(prog, `"`) && len(prog) > len(`"$HOME/"`):
		path = prog[len(`"$HOME/`) : len(prog)-1]
		if strings.ContainsAny(path, "\"$`\\") {
			return false
		}
	case strings.HasPrefix(prog, "'") && len(prog) >= 2:
		path = strings.ReplaceAll(prog[1:len(prog)-1], `'\''`, "'")
		if shellQuote(path) != prog {
			return false
		}
	default:
		return false
	}
	if marked {
		return true // only paws wrote the marker; before it, the copy kept its build name (paws-dev)
	}
	base := path[strings.LastIndexAny(path, `/\`)+1:]
	return base == "paws" || base == "paws.exe"
}

// hasPawsHook reports whether the settings file at path registers paws for any event.
func hasPawsHook(path string) bool {
	doc, err := readSettings(path)
	if err != nil {
		return false
	}
	hooks, err := hooksBlock(doc)
	if err != nil {
		return false
	}
	for _, raw := range hooks {
		var list []json.RawMessage
		if json.Unmarshal(raw, &list) == nil && pawsEntryIndex(list) >= 0 {
			return true
		}
	}
	return false
}

func jsonEqual(a, b json.RawMessage) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
