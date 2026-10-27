//go:build !windows

package gate

import (
	"os/exec"
	"syscall"
)

const scriptShell = "/bin/sh"

// isolate puts the script in its own process group so a timeout kills every descendant.
func isolate(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}
