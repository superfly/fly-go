package clientsignals

import (
	"fmt"
	"net/http"
	"strconv"
)

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
// Construct one via Signals.WrapTransport. RoundTrip does no detection work
// itself — it only applies the values already computed when the transport
// was built.
type ClientSignalsTransport struct {
	InnerTransport http.RoundTripper

	signals  Signals
	uaSuffix string
}

// WrapTransport wraps inner in a *ClientSignalsTransport that attaches s to
// every request the returned transport forwards.
func (s Signals) WrapTransport(inner http.RoundTripper) *ClientSignalsTransport {
	return &ClientSignalsTransport{
		InnerTransport: inner,
		signals:        s,
		uaSuffix:       userAgentSuffix(s),
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
