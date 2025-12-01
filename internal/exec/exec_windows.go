//go:build windows

package exec

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup sets up platform-specific process group settings
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// killProcessGroup kills a process group (best effort on Windows)
func killProcessGroup(pid int) error {
	// On Windows, just kill the process itself
	// Process groups work differently and may not contain all child processes
	return nil // Process will be killed by cmd.Process.Kill()
}

// killProcessGroupForce forcefully kills a process group (best effort on Windows)
func killProcessGroupForce(pid int) error {
	// On Windows, just kill the process itself
	// Process groups work differently and may not contain all child processes
	return nil // Process will be killed by cmd.Process.Kill()
}
