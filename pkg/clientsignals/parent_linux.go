//go:build linux

package clientsignals

import (
	"fmt"
	"os"
	"strings"
)

// lookupParentName resolves the process name for ppid by reading
// /proc/<ppid>/comm. Returns "" on any failure.
func lookupParentName(ppid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", ppid))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}
