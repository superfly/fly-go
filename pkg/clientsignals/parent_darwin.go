//go:build darwin

package clientsignals

import (
	"bytes"

	"golang.org/x/sys/unix"
)

// lookupParentName resolves the process name for ppid via the kern.proc.pid
// sysctl. No subprocess is spawned. Returns "" on any failure.
func lookupParentName(ppid int) string {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", ppid)
	if err != nil {
		return ""
	}

	// P_comm is a fixed-size buffer; the kernel does not guarantee bytes
	// after the NUL terminator are zeroed, so truncate at the first NUL
	// rather than trimming trailing NULs.
	comm := kp.Proc.P_comm[:]
	if i := bytes.IndexByte(comm, 0); i >= 0 {
		comm = comm[:i]
	}

	return string(comm)
}
