//go:build windows

package gate

import "os/exec"

// Claude Code on Windows runs hooks under Git Bash, which puts sh on PATH. Without it the script
// fails to start and the gate fails open.
const scriptShell = "sh"

// isolate keeps the default cancel (kill the direct child); Windows has no process groups to signal.
func isolate(cmd *exec.Cmd) {}
