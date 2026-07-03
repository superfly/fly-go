package clientsignals

import (
	"strings"
	"testing"
)

// clearAgentEnv unsets every env var that participates in agent detection,
// so tests get a deterministic baseline regardless of the ambient
// environment they run in (e.g. this very test suite may itself be running
// under an agent harness that sets CLAUDECODE=1).
func clearAgentEnv(t *testing.T) {
	t.Helper()

	unsetEnv(t, "FLY_INVOKED_BY")
	unsetEnv(t, "AGENT")
	for _, m := range knownMarkers {
		unsetEnv(t, m.env)
	}
}

func TestDetectAgent_None(t *testing.T) {
	clearAgentEnv(t)

	agent, source := detectAgent()
	if agent != "" || source != "" {
		t.Fatalf("expected no agent detected in clean env, got agent=%q source=%q", agent, source)
	}
}

func TestDetectAgent_FlyInvokedByTakesPrecedence(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("FLY_INVOKED_BY", "my-tool")
	t.Setenv("CLAUDECODE", "1")

	agent, source := detectAgent()
	if agent != "my-tool" || source != "env:FLY_INVOKED_BY" {
		t.Fatalf("expected FLY_INVOKED_BY to win, got agent=%q source=%q", agent, source)
	}
}

func TestDetectAgent_FlyInvokedByInvalidFallsThrough(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("FLY_INVOKED_BY", "Not Valid!")
	t.Setenv("CLAUDECODE", "1")

	agent, source := detectAgent()
	if agent != "claude-code" || source != "env:CLAUDECODE" {
		t.Fatalf("expected fallback to table match, got agent=%q source=%q", agent, source)
	}
}

func TestDetectAgent_KnownMarkersTable(t *testing.T) {
	for _, m := range knownMarkers {
		m := m
		t.Run(m.env, func(t *testing.T) {
			clearAgentEnv(t)
			switch m.kind {
			case presence:
				t.Setenv(m.env, "anything")
			case exactValue:
				t.Setenv(m.env, m.values[0])
			}

			agent, source := detectAgent()
			if agent != m.agent {
				t.Fatalf("expected agent=%q, got %q", m.agent, agent)
			}
			if source != "env:"+m.env {
				t.Fatalf("expected source=%q, got %q", "env:"+m.env, source)
			}
		})
	}
}

func TestDetectAgent_ExactValueRequiresExactMatch(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDECODE", "true")

	agent, source := detectAgent()
	if agent != "" || source != "" {
		t.Fatalf("expected no match for wrong exact value, got agent=%q source=%q", agent, source)
	}
}

func TestDetectAgent_CrossToolAgentConvention(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("AGENT", "goose")

	agent, source := detectAgent()
	if agent != "goose" || source != "env:AGENT" {
		t.Fatalf("expected cross-tool AGENT convention, got agent=%q source=%q", agent, source)
	}
}

func TestDetectAgent_CrossToolAgentInvalidIgnored(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("AGENT", "not/a-valid-name")

	agent, source := detectAgent()
	if agent != "" || source != "" {
		t.Fatalf("expected invalid AGENT value ignored, got agent=%q source=%q", agent, source)
	}
}

func TestSanitizeInvokedBy(t *testing.T) {
	maxLen := "a" + strings.Repeat("b", maxInvokedByLen-1)
	tooLong := maxLen + "c"

	cases := []struct {
		name  string
		in    string
		want  string
		valid bool
	}{
		{"simple", "claude-code", "claude-code", true},
		{"uppercase normalized", "Claude-Code", "claude-code", true},
		{"whitespace trimmed", "  claude-code  ", "claude-code", true},
		{"empty", "", "", false},
		{"only whitespace", "   ", "", false},
		{"leading hyphen rejected", "-claude", "", false},
		{"slash rejected", "not/valid", "", false},
		{"space rejected", "not valid", "", false},
		{"too long rejected", tooLong, "", false},
		{"max length ok", maxLen, maxLen, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := sanitizeInvokedBy(tc.in)
			if ok != tc.valid {
				t.Fatalf("sanitizeInvokedBy(%q) ok=%v, want %v", tc.in, ok, tc.valid)
			}
			if ok && got != tc.want {
				t.Fatalf("sanitizeInvokedBy(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
