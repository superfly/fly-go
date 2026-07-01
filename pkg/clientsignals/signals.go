// Package clientsignals computes coarse, privacy-safe signals that help
// estimate whether a CLI process is being driven by a human or an AI agent.
//
// This package is intentionally self-contained (no imports of the parent
// fly-go module, and only golang.org/x/sys as an external dependency, used
// for syscall-based parent-process lookup on Darwin/Windows) so it can later
// be extracted into its own standalone library with minimal friction.
package clientsignals

// Signals is the set of coarse, privacy-safe traffic-classification signals
// computed once per process.
type Signals struct {
	// Interactive is true if the process's stdout appears to be attached to
	// a terminal.
	Interactive bool

	// Parent is a coarse bucket describing the immediate parent process.
	// Always one of "node", "python", "shell", or "other" — never a raw
	// process name.
	Parent string

	// Agent is the cooperative agent marker, e.g. "claude-code". Empty if no
	// agent was declared or detected.
	Agent string

	// AgentSource identifies how Agent was determined, e.g.
	// "env:FLY_INVOKED_BY" or "env:CLAUDECODE" — the matched variable name,
	// never its value. Empty if and only if Agent is empty.
	AgentSource string

	// CI is true when a CI environment is detected.
	CI bool
}

// Detect computes the current process's client signals fresh from the
// environment and file descriptors. It is pure and side-effect free (aside
// from reading process state); it does not cache its result — callers that
// want a single value for the lifetime of a process should cache it
// themselves.
func Detect() Signals {
	agent, source := detectAgent()

	return Signals{
		Interactive: isInteractive(),
		Parent:      parentBucket(),
		Agent:       agent,
		AgentSource: source,
		CI:          isCI(),
	}
}
