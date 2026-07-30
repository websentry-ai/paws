package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/websentry-ai/paws/internal/rule"
)

var scriptTimeout = 5 * time.Second

// runScript: paws JSON on stdin, exit 2 = deny (stderr is the reason), else fails open — same trust model as a Husky hook.
func runScript(r *rule.Rule, ev Event) outcome {
	if DryRun {
		return outcome{}
	}
	script := r.Action.Script
	if !filepath.IsAbs(script) {
		script = filepath.Join(filepath.Dir(r.SourcePath), script)
	}
	if _, err := os.Stat(script); err != nil {
		return outcome{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", script)
	// own process group + WaitDelay: a backgrounded grandchild must not wedge the gate past the timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
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
