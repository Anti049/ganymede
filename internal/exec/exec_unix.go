//go:build !windows

package exec

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup sets up platform-specific process group settings
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killProcessGroup kills a process group using SIGTERM or SIGKILL
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}

// killProcessGroupForce kills a process group using SIGKILL
func killProcessGroupForce(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
