package clientsignals

import (
	"fmt"
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
	transport := NewClientSignalsTransport(capture, nil)

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
	transport := NewClientSignalsTransport(capture, nil)

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

type fakeLogger struct {
	lines []string
}

func (f *fakeLogger) Debugf(format string, v ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, v...))
}

func TestNewClientSignalsTransport_LogsDetectedSignalsOnce(t *testing.T) {
	resetCachedForTest()
	t.Cleanup(resetCachedForTest)

	logger := &fakeLogger{}
	NewClientSignalsTransport(&captureTripper{}, logger)

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one debug line logged at construction, got %d: %v", len(logger.lines), logger.lines)
	}
	if !strings.Contains(logger.lines[0], "client signals: enabled") {
		t.Fatalf("expected debug line to mention client signals are enabled, got %q", logger.lines[0])
	}
}

func TestNewClientSignalsTransport_NilLoggerDoesNotPanic(t *testing.T) {
	resetCachedForTest()
	t.Cleanup(resetCachedForTest)

	NewClientSignalsTransport(&captureTripper{}, nil)
}
