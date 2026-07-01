package clientsignals

import (
	"os"
	"path/filepath"
	"strings"
)

// parentBucket classifies the immediate parent process into a coarse,
// finite bucket. It never returns a raw process name — always one of
// "node", "python", "shell", or "other".
func parentBucket() string {
	return classifyParentName(lookupParentName(os.Getppid()))
}

// classifyParentName maps a raw process name (possibly with a path prefix
// and/or a .exe suffix) to one of the finite approved buckets.
func classifyParentName(raw string) string {
	name := strings.ToLower(filepath.Base(raw))
	name = strings.TrimSuffix(name, ".exe")

	switch name {
	case "node":
		return "node"
	case "python", "python3", "python2":
		return "python"
	case "bash", "zsh", "fish", "sh", "dash", "ksh", "tcsh", "csh", "cmd", "powershell", "pwsh":
		return "shell"
	default:
		return "other"
	}
}
