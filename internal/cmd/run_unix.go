//go:build !windows
// +build !windows

package cmd

import "syscall"

// killProcessGroup kills a process group (Unix-like systems only)
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

// setProcAttributes sets process attributes for better process management
func setProcAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}
}
