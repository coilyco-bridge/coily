//go:build windows

package main

import "syscall"

// Windows process-creation flags. DETACHED_PROCESS gives the child no
// console, CREATE_NEW_PROCESS_GROUP makes it its own group so a Ctrl
// event aimed at coily's console does not reach it. Together they are
// the Windows analog of Unix setsid (coilysiren/coily#302).
const (
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
)

// detachSysProcAttr returns the SysProcAttr that detaches the headless
// dispatch child on Windows.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
}
