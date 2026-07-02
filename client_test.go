package fly

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"

	genq "github.com/Khan/genqlient/graphql"
	"github.com/superfly/graphql"
)

// step describes one scripted RoundTrip outcome. If err is non-nil, the
// RoundTrip returns (nil, err); otherwise it returns a 200 response whose body
// is body.
type step struct {
	err  error
	body string
}

type scriptedTripper struct {
	mu    sync.Mutex
	calls int
	steps []step
}

func (s *scriptedTripper) RoundTrip(*http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls >= len(s.steps) {
		return nil, fmt.Errorf("scriptedTripper: unexpected call %d", s.calls+1)
	}
	st := s.steps[s.calls]
	s.calls++

	if st.err != nil {
		return nil, st.err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(st.body)),
		Header:     make(http.Header),
	}, nil
}

func (s *scriptedTripper) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// connReset returns an error that errors.Is matches as syscall.ECONNRESET,
// shaped like what the net stack actually produces.
func connReset() error {
	return fmt.Errorf("read tcp: %w", syscall.ECONNRESET)
}

func newTestClient(transport http.RoundTripper) *Client {
	return NewClientFromOptions(ClientOptions{
		Name:      "test",
		Version:   "0",
		BaseURL:   "http://example.test",
		Transport: &Transport{UnderlyingTransport: transport},
	})
}

func TestTransportSetDefaults_DoesNotOverrideFlyForceRegionFromTransport(t *testing.T) {
	t.Setenv("FLY_FORCE_REGION", "ord")

	transport := &Transport{FlyForceRegion: "iad"}
	opts := ClientOptions{Transport: transport}

	transport.setDefaults(&opts)

	if transport.FlyForceRegion != "iad" {
		t.Fatalf("expected FlyForceRegion to remain %q, got %q", "iad", transport.FlyForceRegion)
	}
}

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

func TestTransportRoundTrip_DoesNotSetFlyForceInstanceIDHeader(t *testing.T) {
	capture := &captureTripper{}
	transport := &Transport{
		UnderlyingTransport: capture,
		UserAgent:           "test/0",
		Token:               "token",
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer resp.Body.Close()

	if got := capture.req.Header.Get("Fly-Force-Instance-Id"); got != "" {
		t.Fatalf("Fly-Force-Instance-Id header = %q, want empty", got)
	}
}

func TestTransportRoundTrip_EnableClientSignalsAttachesHeaders(t *testing.T) {
	capture := &captureTripper{}
	transport := &Transport{UnderlyingTransport: capture, UserAgent: "test/0", EnableClientSignals: true}
	transport.setDefaults(&ClientOptions{})

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

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

	if ua := capture.req.Header.Get("User-Agent"); !strings.Contains(ua, "interactive=") {
		t.Fatalf("User-Agent = %q, want it to contain the client signals suffix", ua)
	}
}

func TestTransportRoundTrip_ClientSignalsDisabledByDefault(t *testing.T) {
	capture := &captureTripper{}
	transport := &Transport{UnderlyingTransport: capture, UserAgent: "test/0"}
	transport.setDefaults(&ClientOptions{})

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	defer resp.Body.Close()

	for _, h := range []string{"Fly-Client-Interactive", "Fly-Client-Parent", "Fly-Client-Agent", "Fly-Client-Agent-Source", "Fly-Client-CI"} {
		if got := capture.req.Header.Get(h); got != "" {
			t.Fatalf("%s header = %q, want empty when EnableClientSignals is not set", h, got)
		}
	}

	if ua := capture.req.Header.Get("User-Agent"); ua != "test/0" {
		t.Fatalf("User-Agent = %q, want unchanged base UA when disabled", ua)
	}
}

type fakeLogger struct {
	lines []string
}

func (f *fakeLogger) Debug(v ...any) { f.lines = append(f.lines, fmt.Sprint(v...)) }
func (f *fakeLogger) Debugf(format string, v ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, v...))
}

func TestTransportSetDefaults_LogsClientSignalsEnabled(t *testing.T) {
	logger := &fakeLogger{}
	transport := &Transport{EnableClientSignals: true}
	transport.setDefaults(&ClientOptions{Logger: logger})

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one debug line logged, got %d: %v", len(logger.lines), logger.lines)
	}
	if !strings.HasPrefix(logger.lines[0], "web: client signals: enabled") {
		t.Fatalf("expected debug line to start with the web: prefix and mention client signals are enabled, got %q", logger.lines[0])
	}
}

func TestTransportSetDefaults_LogsClientSignalsDisabled(t *testing.T) {
	logger := &fakeLogger{}
	transport := &Transport{}
	transport.setDefaults(&ClientOptions{Logger: logger})

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one debug line logged, got %d: %v", len(logger.lines), logger.lines)
	}
	if logger.lines[0] != "web: client signals: disabled" {
		t.Fatalf("expected debug line = %q, got %q", "web: client signals: disabled", logger.lines[0])
	}
}

func TestGraphQLOperationKind(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"query keyword", "query Foo { id }", "query"},
		{"mutation keyword", "mutation Foo { id }", "mutation"},
		{"subscription keyword", "subscription Foo { id }", "subscription"},
		{"anonymous selection set", "{ viewer { id } }", "query"},
		{"leading whitespace", "\n\t  query Foo { id }", "query"},
		{"leading comment", "# header\nquery Foo { id }", "query"},
		{"comment without trailing newline", "# only a comment", "unknown"},
		{"query body containing the word mutation", "query Foo { recentMutations { id } }", "query"},
		{"empty", "", "unknown"},
		{"unrecognized", "fragment Foo on Bar { id }", "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := graphQLOperationKind(tc.body)
			if got != tc.want {
				t.Fatalf("graphQLOperationKind(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestRunWithContext_RetriesQueryOnConnReset(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
		{err: connReset()},
		{body: `{"data": {}}`},
	}}
	client := newTestClient(tripper)

	req := graphql.NewRequest("query Foo { viewer { id } }")
	if _, err := client.RunWithContext(context.Background(), req); err != nil {
		t.Fatalf("RunWithContext returned error: %v", err)
	}
	if got := tripper.callCount(); got != 3 {
		t.Fatalf("call count = %d, want 3", got)
	}
}

func TestRunWithContext_GivesUpAfterMaxRetries(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
		{err: connReset()},
		{err: connReset()},
	}}
	client := newTestClient(tripper)

	req := graphql.NewRequest("query Foo { viewer { id } }")
	_, err := client.RunWithContext(context.Background(), req)
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("error chain does not contain ECONNRESET: %v", err)
	}
	if got := tripper.callCount(); got != 3 {
		t.Fatalf("call count = %d, want 3 (1 + 2 retries)", got)
	}
}

func TestRunWithContext_DoesNotRetryMutation(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
	}}
	client := newTestClient(tripper)

	req := graphql.NewRequest("mutation Foo { doThing { id } }")
	_, err := client.RunWithContext(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := tripper.callCount(); got != 1 {
		t.Fatalf("call count = %d, want 1 (no retry for mutation)", got)
	}
}

func TestRunWithContext_DoesNotRetryNonConnReset(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: errors.New("some other error")},
	}}
	client := newTestClient(tripper)

	req := graphql.NewRequest("query Foo { viewer { id } }")
	_, err := client.RunWithContext(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := tripper.callCount(); got != 1 {
		t.Fatalf("call count = %d, want 1 (no retry for non-ECONNRESET)", got)
	}
}

func TestGenqlient_RetriesQueryOnConnReset(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
		{body: `{"data": {}}`},
	}}
	client := newTestClient(tripper)

	req := &genq.Request{Query: "query Foo { viewer { id } }"}
	resp := &genq.Response{Data: &struct{}{}}
	if err := client.GenqClient().MakeRequest(context.Background(), req, resp); err != nil {
		t.Fatalf("MakeRequest returned error: %v", err)
	}
	if got := tripper.callCount(); got != 2 {
		t.Fatalf("call count = %d, want 2", got)
	}
}

func TestGenqlient_DoesNotRetryMutation(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
	}}
	client := newTestClient(tripper)

	req := &genq.Request{Query: "mutation Foo { doThing { id } }"}
	resp := &genq.Response{Data: &struct{}{}}
	if err := client.GenqClient().MakeRequest(context.Background(), req, resp); err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := tripper.callCount(); got != 1 {
		t.Fatalf("call count = %d, want 1", got)
	}
}
