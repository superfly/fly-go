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

func TestTransportSetDefaults_DoesNotOverrideFlyForceInstanceFromTransport(t *testing.T) {
	t.Setenv("FLY_FORCE_INSTANCE", "worker-1")

	transport := &Transport{FlyForceInstance: "worker-2"}
	opts := ClientOptions{Transport: transport}

	transport.setDefaults(&opts)

	if transport.FlyForceInstance != "worker-2" {
		t.Fatalf("expected FlyForceInstance to remain %q, got %q", "worker-2", transport.FlyForceInstance)
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

func TestTransportRoundTrip_SetsFlyForceInstanceHeader(t *testing.T) {
	capture := &captureTripper{}
	transport := &Transport{
		UnderlyingTransport: capture,
		UserAgent:           "test/0",
		Token:               "token",
		FlyForceInstance:    "worker-2",
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}

	if got := capture.req.Header.Get("Fly-Force-Instance"); got != "worker-2" {
		t.Fatalf("Fly-Force-Instance header = %q, want %q", got, "worker-2")
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
