package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		"/bin/cat":           {"Bash", map[string]any{"command": "/bin/cat " + md}},
		"tail +1":            {"Bash", map[string]any{"command": "tail -n +1 " + md}},
		"sed -e":             {"Bash", map[string]any{"command": "sed -n -e '1,240p' " + md}},
		"bat -p":             {"Bash", map[string]any{"command": "bat -p --paging=never " + md}},
		"a later range":      {"Bash", map[string]any{"command": "sed -n '241,480p' " + md}},
		"other skill's file": {"Bash", map[string]any{"command": "cat " + skillMD(t, "other")}},
	} {
		if out := gateCall(t, c.tool, c.input, ""); out != "{}" {
			t.Errorf("%s: the read of SKILL.md was gated: %s", name, out)
		}
	}
	for name, cmd := range map[string]string{
		"chained":          "cat " + md + " && git push",
		"redirected":       "cat " + md + " > /tmp/x",
		"piped":            "cat " + md + " | sh",
		"not SKILL.md":     "cat " + filepath.Join(filepath.Dir(md), "notes.md"),
		"missing file":     "cat " + filepath.Join(home, "skills", "deployer", "SKILL.md"),
		"sed script":       "sed -n 'w /tmp/x' " + md,
		"sed in place":     "sed -n -i '1p' " + md,
		"awk in place":     "awk -i inplace '{print}' " + md,
		"sed script file":  "sed -n -f s.sed 1p " + md,
		"sed expression":   "sed -n --expression='e touch x' 1p " + md,
		"awk program file": "awk -f prog.awk " + md,
		"bat pager":        "bat --pager='sh -c x' " + md,
		"less log":         "less -o out.txt " + md,
		"head nothing":     "head -n 0 " + md,
		"awk past the top": "awk 'NR>9999' " + md,
		"bat line range":   "bat -r 9999: " + md,
		"own cat":          "/tmp/cat " + md,
		"local cat":        "./cat " + md,
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
	if len(audit) != 10 {
		t.Errorf("every read of a SKILL.md logs skill_invoked, got %v", audit)
	}
}

// Only a SKILL.md in a folder a harness loads skills from is a load: a plugin's skill counts, and a SKILL.md the agent
// wrote anywhere else (or a repo shipped) loads nothing and is gated like any read.
func TestOnlyASKILLmdInASkillFolderLoads(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "catch-all.md"), "name: catch-all\nevent: pre-tool\naction:\n  type: require-skill\n  skill: deployer\n")
	userHome, _ := os.UserHomeDir()
	plugin := filepath.Join(userHome, ".claude", "plugins", "cache", "acme", "tools", "1.0.0", "skills", "deployer", "SKILL.md")
	fake := filepath.Join(t.TempDir(), "x", "skills", "deployer", "SKILL.md")
	for _, p := range []string{plugin, fake} {
		writeFile(t, p, "---\nname: deployer\n---\n")
	}
	if out := gateCall(t, "Bash", map[string]any{"command": "cat " + plugin}, ""); out != "{}" {
		t.Errorf("a plugin's SKILL.md is a skill: its read must not be gated, got %s", out)
	}
	if out := gateCall(t, "Bash", map[string]any{"command": "cat " + fake}, ""); !strings.Contains(out, `"deny"`) {
		t.Errorf("a SKILL.md outside every skill folder is no load and is gated, got %s", out)
	}
	if out := gateCall(t, "Read", map[string]any{"file_path": fake}, ""); !strings.Contains(out, `"deny"`) {
		t.Errorf("a Read of a fake SKILL.md is gated too, got %s", out)
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
	// A read from inside the skill's folder names the file alone; the record's cwd places it.
	rel := transcript("Bash", `{"command":"cat SKILL.md"}`, false)
	data, _ := os.ReadFile(rel)
	os.WriteFile(rel, []byte(strings.ReplaceAll(string(data), `"type":"assistant",`, `"type":"assistant","cwd":"`+filepath.Dir(md)+`",`)), 0o600)
	if out := gateCall(t, "Bash", push, rel); out != "{}" {
		t.Errorf("cat SKILL.md run in the skill's folder should clear the gate, got %s", out)
	}
	if out := gateCall(t, "Bash", push, transcript("Read", string(readInput), true)); !strings.Contains(out, `"deny"`) {
		t.Errorf("a failed read loads nothing, got %s", out)
	}
	// A read of the whole file that succeeded still counts once the file is gone: its text is in the agent's
	// context. A Read can't say it reached the end, so a cat stands in.
	done := transcript("Bash", `{"command":"cat `+md+`"}`, false)
	os.Remove(md)
	if out := gateCall(t, "Bash", push, done); out != "{}" {
		t.Errorf("a past read of a SKILL.md removed since should still count, got %s", out)
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
	want := `paws (rule catch-all): this is paws working as intended, nothing is broken. This command needs the "deployer" skill. Load it with the Skill tool, then retry this exact command.`
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

// A skill counts as loaded once the reads that succeeded print every line of its SKILL.md, in any order.
func TestSKILLmdReadsLoadWhenTheyCoverTheFile(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "push.md"), "name: push\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: git push\naction:\n  type: require-skill\n  skill: deployer\n")
	md := skillMD(t, "deployer")
	var body strings.Builder
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&body, "line %d\n", i)
	}
	os.WriteFile(md, []byte(body.String()), 0o644)
	// reads writes a transcript of successful calls, each a Bash command or a Read input as JSON.
	reads := func(calls ...map[string]any) string {
		p := filepath.Join(t.TempDir(), "s.jsonl")
		lines := []string{`{"type":"user","isSidechain":false,"message":{"role":"user","content":"push it"}}`}
		for i, c := range calls {
			name := "Bash"
			if _, ok := c["file_path"]; ok {
				name = "Read"
			}
			in, _ := json.Marshal(c)
			id := fmt.Sprintf("toolu_%d", i)
			lines = append(lines,
				`{"type":"assistant","isSidechain":false,"message":{"content":[{"type":"tool_use","id":"`+id+`","name":"`+name+`","input":`+string(in)+`}]}}`,
				`{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"`+id+`","is_error":false,"content":"x"}]}}`)
		}
		os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
		return p
	}
	bash := func(cmd string) map[string]any { return map[string]any{"command": cmd} }
	// Past 2000 lines, a Read without a limit is cut: it doesn't cover a longer file.
	long := skillMD(t, "long-skill")
	os.WriteFile(long, []byte(strings.Repeat("x\n", 2500)), 0o644)
	writeRule(t, filepath.Join(home, "long.md"), "name: long\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: make long\naction:\n  type: require-skill\n  skill: long-skill\n")
	if out := gateCall(t, "Bash", map[string]any{"command": "make long"}, reads(map[string]any{"file_path": long})); !strings.Contains(out, `"deny"`) {
		t.Errorf("a default Read of a 2500-line skill stops at 2000 lines: not loaded, got %s", out)
	}
	if out := gateCall(t, "Bash", map[string]any{"command": "make long"}, reads(map[string]any{"file_path": long}, map[string]any{"file_path": long, "offset": 2001})); out != "{}" {
		t.Errorf("a second Read from line 2001 covers the rest, got %s", out)
	}
	push := map[string]any{"command": "git push"}
	for name, c := range map[string]struct {
		calls  []map[string]any
		loaded bool
	}{
		"1,240p then 241,480p":    {[]map[string]any{bash("sed -n '1,240p' " + md), bash("sed -n '241,480p' " + md)}, true},
		"the chunks out of order": {[]map[string]any{bash("sed -n '241,480p' " + md), bash("sed -n '1,240p' " + md)}, true},
		"1,240p alone":            {[]map[string]any{bash("sed -n '1,240p' " + md)}, false},
		"head -n 5":               {[]map[string]any{bash("head -n 5 " + md)}, false},
		"a gap between chunks":    {[]map[string]any{bash("sed -n '1,100p' " + md), bash("sed -n '102,300p' " + md)}, false},
		"head, then tail +201":    {[]map[string]any{bash("head -n 200 " + md), bash("tail -n +201 " + md)}, true},
		"full cat":                {[]map[string]any{bash("cat " + md)}, true},
		"Read in two parts":       {[]map[string]any{{"file_path": md, "limit": 150}, {"file_path": md, "offset": 151, "limit": 150}}, true},
		"Read of a part":          {[]map[string]any{{"file_path": md, "offset": 100}}, false},
		"Read of the whole file":  {[]map[string]any{{"file_path": md}}, true},
		"awk NR<1 prints nothing": {[]map[string]any{bash("awk 'NR<1' " + md)}, false},
	} {
		out := gateCall(t, "Bash", push, reads(c.calls...))
		if loaded := out == "{}"; loaded != c.loaded {
			t.Errorf("%s: loaded %v, want %v (%s)", name, loaded, c.loaded, out)
		}
	}
}

// A read made before the SKILL.md changed printed an older version: it doesn't load the skill as it is now.
func TestASKILLmdReadBeforeTheFileChangedDoesntCount(t *testing.T) {
	home := portalEnv(t, http.NotFoundHandler())
	writeRule(t, filepath.Join(home, "push.md"), "name: push\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: git push\naction:\n  type: require-skill\n  skill: deployer\n")
	md := skillMD(t, "deployer")
	in, _ := json.Marshal(map[string]string{"command": "cat " + md})
	log := func(at time.Time) string {
		p := filepath.Join(t.TempDir(), "s.jsonl")
		ts := at.UTC().Format(time.RFC3339Nano)
		os.WriteFile(p, []byte(strings.Join([]string{
			`{"type":"user","isSidechain":false,"message":{"role":"user","content":"push it"}}`,
			`{"type":"assistant","isSidechain":false,"timestamp":"` + ts + `","message":{"content":[{"type":"tool_use","id":"toolu_r","name":"Bash","input":` + string(in) + `}]}}`,
			`{"type":"user","isSidechain":false,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_r","is_error":false,"content":"x"}]}}`,
		}, "\n")+"\n"), 0o600)
		return p
	}
	push := map[string]any{"command": "git push"}
	if out := gateCall(t, "Bash", push, log(time.Now().Add(time.Minute))); out != "{}" {
		t.Errorf("a read after the file's last change loads it, got %s", out)
	}
	if out := gateCall(t, "Bash", push, log(time.Now().Add(-time.Hour))); !strings.Contains(out, `"deny"`) {
		t.Errorf("a read from before the file changed doesn't, got %s", out)
	}
}
