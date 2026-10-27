package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

var scriptTimeout = 5 * time.Second

// runScript: paws JSON on stdin, exit 2 = deny (stderr is the reason), else fails open — same trust model as a Husky hook.
func runScript(r *rule.Rule, ev Event) outcome {
	if DryRun {
		return outcome{}
	}
	// rule.Parse already rejects absolute paths and ".." escapes, so this stays inside the rule's own directory tree.
	script := filepath.Join(filepath.Dir(r.SourcePath), r.Action.Script)
	if _, err := os.Stat(script); err != nil {
		return outcome{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptShell, script)
	// own process group (where the OS has them) + WaitDelay: a backgrounded grandchild must not wedge the gate past the timeout
	isolate(cmd)
	cmd.WaitDelay = time.Second

	payload, _ := json.Marshal(map[string]any{
		"agent": ev.Agent, "event": r.Event, "tool": ev.Tool, "target": ev.Target, "raw": ev.Raw,
	})
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	if err == nil {
		return outcome{verdict: Allow}
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 2 {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = strings.TrimSpace(stdout.String())
		}
		if reason == "" {
			reason = "Blocked by paws exec-script rule " + r.Name + "."
		}
		return outcome{verdict: DenyRetry, reason: reason, fired: true}
	}
	return outcome{} // timeout / other error → fail-open
}
