package clientsignals

import (
	"os"
	"regexp"
	"strings"
)

const maxInvokedByLen = 64

var invokedByPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// detectAgent determines the cooperative agent marker, in precedence order:
//  1. FLY_INVOKED_BY, a self-declared tool name any agent harness can set.
//  2. The known-markers table (knownMarkers), first match wins.
//  3. The cross-tool AGENT=<name> convention.
//
// Returns ("", "") if nothing is detected.
func detectAgent() (agent, source string) {
	if v, ok := os.LookupEnv("FLY_INVOKED_BY"); ok {
		if sanitized, ok := sanitizeInvokedBy(v); ok {
			return sanitized, "env:FLY_INVOKED_BY"
		}
	}

	for _, m := range knownMarkers {
		v, present := os.LookupEnv(m.env)
		if !present {
			continue
		}

		switch m.kind {
		case presence:
			return m.agent, "env:" + m.env
		case exactValue:
			for _, want := range m.values {
				if v == want {
					return m.agent, "env:" + m.env
				}
			}
		}
	}

	if v, ok := os.LookupEnv("AGENT"); ok {
		if sanitized, ok := sanitizeInvokedBy(v); ok {
			return sanitized, "env:AGENT"
		}
	}

	return "", ""
}

// sanitizeInvokedBy validates and normalizes a self-declared agent-name
// value before it is ever emitted on the wire. Returns ok=false if the value
// doesn't look like a bare tool-name identifier — callers should treat that
// as "no valid declaration" rather than emit a rejected value.
func sanitizeInvokedBy(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" || len(v) > maxInvokedByLen {
		return "", false
	}
	if !invokedByPattern.MatchString(v) {
		return "", false
	}

	return v, true
}
