package clientsignals

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

var (
	cachedOnce   sync.Once
	cachedResult Signals
)

// cached returns the process-wide signals, computed once via Detect and
// cached for the lifetime of the process. Detection involves a
// parent-process lookup and environment scanning, so it must never run per
// request — NewClientSignalsTransport calls this once at construction time,
// not from RoundTrip.
func cached() Signals {
	cachedOnce.Do(func() {
		cachedResult = Detect()
	})

	return cachedResult
}

// resetCachedForTest clears the cached signals so tests can exercise Detect
// against a freshly modified environment. Only for use in this package's
// own tests.
func resetCachedForTest() {
	cachedOnce = sync.Once{}
}

// userAgentSuffix returns the human-readable
// "(interactive=...; parent=...; agent=...)" token to append to a
// User-Agent string.
func userAgentSuffix(s Signals) string {
	suffix := fmt.Sprintf("interactive=%t; parent=%s", s.Interactive, s.Parent)
	if s.Agent != "" {
		suffix += "; agent=" + s.Agent
	}

	return "(" + suffix + ")"
}

// applyHeaders sets the Fly-Client-* headers on req from s.
func applyHeaders(req *http.Request, s Signals) {
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

	signals  Signals
	uaSuffix string
}

// debugLogger is a minimal logging interface for optional debug output.
// It matches (github.com/superfly/fly-go).Logger structurally, so callers
// can pass a fly.Logger in without this package importing it.
type debugLogger interface {
	Debugf(format string, v ...any)
}

// NewClientSignalsTransport wraps inner so that every request through it
// carries the process's client signals. Signal detection runs at most once
// per process (here, or by an earlier caller — the result is cached), never
// per request.
//
// If logger is non-nil, the detected signals are logged once, at
// construction time — never per request.
func NewClientSignalsTransport(inner http.RoundTripper, logger debugLogger) *ClientSignalsTransport {
	sig := cached()

	if logger != nil {
		logger.Debugf("client signals: enabled interactive=%t parent=%s agent=%q agent_source=%q ci=%t",
			sig.Interactive, sig.Parent, sig.Agent, sig.AgentSource, sig.CI)
	}

	return &ClientSignalsTransport{
		InnerTransport: inner,
		signals:        sig,
		uaSuffix:       userAgentSuffix(sig),
	}
}

func (t *ClientSignalsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	applyHeaders(req, t.signals)

	if ua := req.Header.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua+" "+t.uaSuffix)
	} else {
		req.Header.Set("User-Agent", t.uaSuffix)
	}

	return t.InnerTransport.RoundTrip(req)
}
