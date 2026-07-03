//go:build windows

package clientsignals

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// lookupParentName resolves the process name for ppid by walking a
// Toolhelp32 process snapshot. No subprocess is spawned. Returns "" on any
// failure or if the PID can't be found.
func lookupParentName(ppid int) string {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return ""
	}
	for {
		if entry.ProcessID == uint32(ppid) {
			return windows.UTF16ToString(entry.ExeFile[:])
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			return ""
		}
	}
}
