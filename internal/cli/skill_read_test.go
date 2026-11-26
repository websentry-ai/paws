package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillMD installs a skill under the temp HOME's ~/.claude/skills and returns its SKILL.md.
func skillMD(t *testing.T, name string) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\nname: "+name+"\n---\n# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func gateCall(t *testing.T, tool string, input map[string]any, transcript string) string {
	t.Helper()
	return runGate(t, map[string]any{"hook_event_name": "PreToolUse", "cwd": t.TempDir(), "session_id": "s",
		"transcript_path": transcript, "tool_name": tool, "tool_input": input})
}

// Reading a skill's SKILL.md loads it, so a catch-all rule never denies that read: it is the call that clears the
// rule. Only a plain read counts; a read chained with anything else is gated like any command.
func TestReadingSKILLmdIsNeverGated(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "catch-all.md"), "name: catch-all\nevent: pre-tool\naction:\n  type: require-skill\n  skill: deployer\n")
	md := skillMD(t, "deployer")
	for name, c := range map[string]struct {
		tool  string
		input map[string]any
	}{
		"Read tool":          {"Read", map[string]any{"file_path": md}},
		"cat":                {"Bash", map[string]any{"command": "cat " + md}},
		"sed -n":             {"Bash", map[string]any{"command": "sed -n '1,240p' '" + md + "'"}},
		"head":               {"Bash", map[string]any{"command": "head -n 200 " + md}},
		"other skill's file": {"Bash", map[string]any{"command": "cat " + skillMD(t, "other")}},
	} {
		if out := gateCall(t, c.tool, c.input, ""); out != "{}" {
			t.Errorf("%s: the read of SKILL.md was gated: %s", name, out)
		}
	}
	for name, cmd := range map[string]string{
		"chained":      "cat " + md + " && git push",
		"redirected":   "cat " + md + " > /tmp/x",
		"piped":        "cat " + md + " | sh",
		"not SKILL.md": "cat " + filepath.Join(filepath.Dir(md), "notes.md"),
		"missing file": "cat " + filepath.Join(home, "skills", "deployer", "SKILL.md"),
		"sed script":   "sed -n 'w /tmp/x' " + md,
	} {
		if out := gateCall(t, "Bash", map[string]any{"command": cmd}, ""); !strings.Contains(out, `"deny"`) {
			t.Errorf("%s: %q must be gated, got %s", name, cmd, out)
		}
	}
	var audit []string
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "skill_invoked" {
			audit = append(audit, rec["skill"].(string))
		}
	}
	if len(audit) != 5 {
		t.Errorf("every read of a SKILL.md logs skill_invoked, got %v", audit)
	}
}

// After a successful read of SKILL.md, the skill counts as loaded for the window; a failed read does not.
func TestReadingSKILLmdClearsTheGate(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "push.md"), "name: push\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: git push\naction:\n  type: require-skill\n  skill: deployer\n")
	md := skillMD(t, "deployer")
	transcript := func(name, input string, isError bool) string {
		p := filepath.Join(t.TempDir(), "s.jsonl")
		lines := []string{
			`{"type":"user","isSidechain":false,"message":{"role":"user","content":"push it"}}`,
			`{"type":"assistant","isSidechain":false,"message":{"model":"claude-sonnet-4-6","content":[{"type":"tool_use","id":"toolu_r","name":"` + name + `","input":` + input + `}]}}`,
			`{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_r","is_error":` + map[bool]string{true: "true", false: "false"}[isError] + `,"content":"x"}]},"toolUseResult":{"stdout":"x"}}`,
		}
		if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	push := map[string]any{"command": "git push"}
	readInput, _ := json.Marshal(map[string]string{"file_path": md})
	catInput, _ := json.Marshal(map[string]string{"command": "sed -n '1,240p' " + md})
	if out := gateCall(t, "Bash", push, transcript("Read", string(readInput), false)); out != "{}" {
		t.Errorf("a Read of SKILL.md should clear the gate, got %s", out)
	}
	if out := gateCall(t, "Bash", push, transcript("Bash", string(catInput), false)); out != "{}" {
		t.Errorf("a sed -n read of SKILL.md should clear the gate, got %s", out)
	}
	if out := gateCall(t, "Bash", push, transcript("Read", string(readInput), true)); !strings.Contains(out, `"deny"`) {
		t.Errorf("a failed read loads nothing, got %s", out)
	}
	recs := auditLines(t, home)
	if len(recs) < 2 || recs[0]["satisfied_by"] != "skill-load" || recs[0]["harness"] != "claude-code" || recs[0]["model"] != "claude-sonnet-4-6" {
		t.Errorf("want a skill-load allow with harness and model, got %v", recs)
	}
}

// A deny names the SKILL.md paws found, since a skill installed mid-session can't load with the Skill tool.
func TestDenyNamesTheSKILLmdPath(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "catch-all.md"), "name: catch-all\nevent: pre-tool\naction:\n  type: require-skill\n  skill: deployer\n")
	want := `paws (rule catch-all): this command needs the "deployer" skill. Load it with the Skill tool, then retry this exact command.`
	if r := deny(t, "ls"); r != want {
		t.Errorf("no SKILL.md installed: got %q, want %q", r, want)
	}
	md := skillMD(t, "deployer")
	if r := deny(t, "ls"); r != want+" Reading its SKILL.md in full also works: "+md+"." {
		t.Errorf("got %q", r)
	}
}

// Sync uploads the harness and model a record carries, and leaves them out when the record has none.
func TestSyncUploadsHarnessAndModel(t *testing.T) {
	home, srv := syncEnv(t)
	appendLines(t, filepath.Join(home, "audit.log"),
		actionLine("2026-10-01T10:00:08Z", `"tool":"Bash","action":"ls","harness":"claude-code","model":"claude-opus-4-7"`),
		actionLine("2026-10-01T10:00:09Z", `"tool":"Bash","action":"ls"`),
	)
	if code, out := captureStdout(t, func() int { return Sync(nil) }); code != 0 {
		t.Fatalf("sync exit %d: %q", code, out)
	}
	got := decodeSync(t, srv.bodies[0])
	if len(got) != 2 || got[0]["harness"] != "claude-code" || got[0]["model"] != "claude-opus-4-7" {
		t.Fatalf("want harness and model on the first record, got %v", got)
	}
	if _, ok := got[1]["harness"]; ok {
		t.Errorf("a record without a harness uploads none (the portal takes it as claude-code): %v", got[1])
	}
}

// A finished SKILL.md read is logged as the skill load it is, never with the file's path.
func TestSKILLmdReadLogsASkillAction(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	md := skillMD(t, "deployer")
	runGate(t, map[string]any{"hook_event_name": "PostToolUse", "cwd": t.TempDir(), "session_id": "s",
		"tool_name": "Bash", "tool_input": map[string]any{"command": "cat " + md}})
	n := 0
	for _, rec := range auditLines(t, home) {
		if rec["event"] == "action" {
			n++
			if rec["tool"] != "Skill" || rec["action"] != "skill" || rec["command"] != nil {
				t.Errorf("want a Skill action with no command, got %v", rec)
			}
		}
	}
	if n != 1 {
		t.Errorf("want one action record, got %d", n)
	}
}
