//go:build windows
// +build windows

package cmd

import "syscall"

// killProcessGroup kills a process group (not supported on Windows)
func killProcessGroup(pid int) error {
	return nil // Not supported on Windows
}

// setProcAttributes sets process attributes for better process management
func setProcAttributes() *syscall.SysProcAttr {
	return nil // Windows doesn't support process groups the same way
}