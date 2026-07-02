package clientsignals

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type captureTripper struct {
	req *http.Request
}

func (c *captureTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.req = req.Clone(req.Context())
	c.req.Header = req.Header.Clone()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

func TestClientSignalsTransport_AttachesHeadersAndUserAgentSuffix(t *testing.T) {
	resetCachedForTest()
	t.Cleanup(resetCachedForTest)

	capture := &captureTripper{}
	transport := DetectOnce().WrapTransport(capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("User-Agent", "test/0")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer resp.Body.Close()

	interactive := capture.req.Header.Get("Fly-Client-Interactive")
	if interactive != "true" && interactive != "false" {
		t.Fatalf("Fly-Client-Interactive header = %q, want true or false", interactive)
	}

	switch parent := capture.req.Header.Get("Fly-Client-Parent"); parent {
	case "node", "python", "shell", "other":
	default:
		t.Fatalf("Fly-Client-Parent header = %q, want one of node/python/shell/other", parent)
	}

	if ua := capture.req.Header.Get("User-Agent"); !strings.HasPrefix(ua, "test/0 (interactive=") {
		t.Fatalf("User-Agent = %q, want it to start with the base UA followed by the client signals suffix", ua)
	}
}

func TestClientSignalsTransport_SetsUserAgentWhenNoneWasSet(t *testing.T) {
	resetCachedForTest()
	t.Cleanup(resetCachedForTest)

	capture := &captureTripper{}
	transport := DetectOnce().WrapTransport(capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer resp.Body.Close()

	if ua := capture.req.Header.Get("User-Agent"); !strings.HasPrefix(ua, "(interactive=") {
		t.Fatalf("User-Agent = %q, want it to be just the client signals suffix", ua)
	}
}

func TestSignals_WrapTransport(t *testing.T) {
	sig := Signals{
		Interactive: true,
		Parent:      "node",
		Agent:       "claude-code",
		AgentSource: "env:CLAUDECODE",
		CI:          true,
	}

	capture := &captureTripper{}
	transport := sig.WrapTransport(capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("User-Agent", "test/0")

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer resp.Body.Close()

	cases := map[string]string{
		"Fly-Client-Interactive":  "true",
		"Fly-Client-Parent":       "node",
		"Fly-Client-Agent":        "claude-code",
		"Fly-Client-Agent-Source": "env:CLAUDECODE",
		"Fly-Client-CI":           "true",
	}
	for header, want := range cases {
		if got := capture.req.Header.Get(header); got != want {
			t.Fatalf("%s header = %q, want %q", header, got, want)
		}
	}

	wantUA := "test/0 (interactive=true; parent=node; agent=claude-code)"
	if ua := capture.req.Header.Get("User-Agent"); ua != wantUA {
		t.Fatalf("User-Agent = %q, want %q", ua, wantUA)
	}
}
