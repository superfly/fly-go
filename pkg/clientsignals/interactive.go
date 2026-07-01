package clientsignals

import "os"

// isInteractive reports whether the process's stdout appears to be attached
// to a terminal. This is a coarser heuristic than golang.org/x/term.IsTerminal
// (no ioctl/GetConsoleMode, just the ModeCharDevice bit from Stat), traded
// off deliberately to keep this package dependency-free. Acceptable here
// because Interactive is a coarse traffic-classification bit, not a UX gate.
func isInteractive() bool {
	return isInteractiveFile(os.Stdout)
}

func isInteractiveFile(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}
