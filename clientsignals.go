package fly

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/superfly/fly-go/pkg/clientsignals"
)

var (
	clientSignalsOnce   sync.Once
	clientSignalsCached clientsignals.Signals
)

// clientSignals returns the process-wide AI-agent-vs-human traffic
// classification signals, computed once and cached for the lifetime of the
// process. Detection involves a parent-process lookup and environment
// scanning, so it must never run per request — ClientSignalsTransport calls
// this once at construction time, not from RoundTrip.
func clientSignals() clientsignals.Signals {
	clientSignalsOnce.Do(func() {
		clientSignalsCached = clientsignals.Detect()
	})

	return clientSignalsCached
}

// resetClientSignalsForTest clears the cached signals so tests can exercise
// Detect() against a freshly modified environment. Only for use in this
// module's own tests.
func resetClientSignalsForTest() {
	clientSignalsOnce = sync.Once{}
}

// clientSignalsUserAgentSuffix returns the human-readable
// "(interactive=...; parent=...; agent=...)" token to append to a
// User-Agent string.
func clientSignalsUserAgentSuffix(s clientsignals.Signals) string {
	suffix := fmt.Sprintf("interactive=%t; parent=%s", s.Interactive, s.Parent)
	if s.Agent != "" {
		suffix += "; agent=" + s.Agent
	}

	return "(" + suffix + ")"
}

// applyClientSignalHeaders sets the Fly-Client-* headers on req from s.
func applyClientSignalHeaders(req *http.Request, s clientsignals.Signals) {
	req.Header.Set("Fly-Client-Interactive", strconv.FormatBool(s.Interactive))
	req.Header.Set("Fly-Client-Parent", s.Parent)
	if s.Agent != "" {
		req.Header.Set("Fly-Client-Agent", s.Agent)
		req.Header.Set("Fly-Client-Agent-Source", s.AgentSource)
	}
	if s.CI {
		req.Header.Set("Fly-Client-CI", "true")
	}
}

// ClientSignalsTransport wraps an http.RoundTripper, attaching the
// Fly-Client-* headers and appending the client-signals token to the
// existing User-Agent header on every outgoing request.
//
// Signal detection (terminal/parent-process/environment inspection) happens
// at most once per process, at construction time via
// NewClientSignalsTransport — RoundTrip does no detection work itself, it
// only applies the values already computed when the transport was built.
type ClientSignalsTransport struct {
	InnerTransport http.RoundTripper

	signals  clientsignals.Signals
	uaSuffix string
}

// NewClientSignalsTransport wraps inner so that every request through it
// carries the process's client signals. Signal detection runs at most once
// per process (here, or by an earlier caller — the result is cached), never
// per request.
func NewClientSignalsTransport(inner http.RoundTripper) *ClientSignalsTransport {
	sig := clientSignals()

	return &ClientSignalsTransport{
		InnerTransport: inner,
		signals:        sig,
		uaSuffix:       clientSignalsUserAgentSuffix(sig),
	}
}

func (t *ClientSignalsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	applyClientSignalHeaders(req, t.signals)

	if ua := req.Header.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua+" "+t.uaSuffix)
	} else {
		req.Header.Set("User-Agent", t.uaSuffix)
	}

	return t.InnerTransport.RoundTrip(req)
}
