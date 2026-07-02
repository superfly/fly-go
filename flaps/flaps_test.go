package flaps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func TestNewWithOptionsSetsCookieJar(t *testing.T) {
	t.Setenv("FLY_FLAPS_BASE_URL", "http://example.com")

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if client.httpClient.Jar == nil {
		t.Fatal("httpClient.Jar is nil")
	}
}

func TestFlapsClientPersistsPathScopedCookiesPerApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/apps":
			if r.Method != http.MethodPost {
				http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
				return
			}

			var req struct {
				AppName string `json:"app_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			switch req.AppName {
			case "app-a":
				http.SetCookie(w, &http.Cookie{
					Name:  "fly_flaps_affinity",
					Value: "creator-node-a",
					Path:  "/v1/apps/app-a",
				})
			case "app-b":
				http.SetCookie(w, &http.Cookie{
					Name:  "fly_flaps_affinity",
					Value: "creator-node-b",
					Path:  "/v1/apps/app-b",
				})
			default:
				http.Error(w, "unexpected app", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/v1/apps/app-a/status":
			assertCookieHeader(w, r, "fly_flaps_affinity=creator-node-a")
		case "/v1/apps/app-b/status":
			assertCookieHeader(w, r, "fly_flaps_affinity=creator-node-b")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	ctx := context.Background()
	for _, req := range []struct {
		method string
		path   string
		body   interface{}
	}{
		{method: http.MethodPost, path: "/apps", body: map[string]string{"app_name": "app-a"}},
		{method: http.MethodPost, path: "/apps", body: map[string]string{"app_name": "app-b"}},
		{method: http.MethodGet, path: "/apps/app-a/status"},
		{method: http.MethodGet, path: "/apps/app-b/status"},
	} {
		if err := client._sendRequest(ctx, req.method, req.path, req.body, nil, nil); err != nil {
			t.Fatalf("request %s %s error = %v", req.method, req.path, err)
		}
	}
}

func assertCookieHeader(w http.ResponseWriter, r *http.Request, want string) {
	got := r.Header.Get("Cookie")
	if got != want {
		http.Error(w, "cookie header = \""+got+"\", want \""+want+"\"", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

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

type captureTripper struct {
	req *http.Request
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

func (c *captureTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.req = req.Clone(req.Context())
	c.req.Header = req.Header.Clone()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
	}, nil
}

func connReset() error {
	return fmt.Errorf("read tcp: %w", syscall.ECONNRESET)
}

func newTestFlapsClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Setenv("FLY_FLAPS_BASE_URL", "http://example.test")
	client, err := NewWithOptions(context.Background(), NewClientOpts{Transport: transport})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	return client
}

func TestFlaps_GetRetriesOnConnReset(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
		{err: connReset()},
		{body: `{}`},
	}}
	client := newTestFlapsClient(t, tripper)

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest: %v", err)
	}
	if got := tripper.callCount(); got != 3 {
		t.Fatalf("call count = %d, want 3", got)
	}
}

func TestFlaps_GetGivesUpAfterMaxRetries(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
		{err: connReset()},
		{err: connReset()},
	}}
	client := newTestFlapsClient(t, tripper)

	err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil)
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

func TestFlaps_PostDoesNotRetry(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: connReset()},
	}}
	client := newTestFlapsClient(t, tripper)

	err := client._sendRequest(context.Background(), http.MethodPost, "/apps", map[string]string{"name": "x"}, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := tripper.callCount(); got != 1 {
		t.Fatalf("call count = %d, want 1 (no retry for POST)", got)
	}
}

func TestFlaps_NewRequestSetsFlyForceInstanceIDHeaderFromOpts(t *testing.T) {
	capture := &captureTripper{}
	client, err := NewWithOptions(context.Background(), NewClientOpts{
		Transport:          capture,
		FlyForceInstanceID: "worker-2",
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest() error = %v", err)
	}

	if got := capture.req.Header.Get("Fly-Force-Instance-Id"); got != "worker-2" {
		t.Fatalf("Fly-Force-Instance-Id header = %q, want %q", got, "worker-2")
	}
}

func TestFlaps_NewRequestSetsFlyForceInstanceIDHeaderFromEnv(t *testing.T) {
	t.Setenv("FLY_FORCE_INSTANCE_ID", "worker-1")

	capture := &captureTripper{}
	client, err := NewWithOptions(context.Background(), NewClientOpts{Transport: capture})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest() error = %v", err)
	}

	if got := capture.req.Header.Get("Fly-Force-Instance-Id"); got != "worker-1" {
		t.Fatalf("Fly-Force-Instance-Id header = %q, want %q", got, "worker-1")
	}
}

func TestFlaps_EnableClientSignalsAttachesHeaders(t *testing.T) {
	capture := &captureTripper{}
	client, err := NewWithOptions(context.Background(), NewClientOpts{
		Transport:           capture,
		EnableClientSignals: true,
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest() error = %v", err)
	}

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

func TestFlaps_ClientSignalsDisabledByDefault(t *testing.T) {
	capture := &captureTripper{}
	client, err := NewWithOptions(context.Background(), NewClientOpts{Transport: capture})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest() error = %v", err)
	}

	for _, h := range []string{"Fly-Client-Interactive", "Fly-Client-Parent", "Fly-Client-Agent", "Fly-Client-Agent-Source", "Fly-Client-CI"} {
		if got := capture.req.Header.Get(h); got != "" {
			t.Fatalf("%s header = %q, want empty when EnableClientSignals is not set", h, got)
		}
	}

	if ua := capture.req.Header.Get("User-Agent"); strings.Contains(ua, "interactive=") {
		t.Fatalf("User-Agent = %q, want no client signals suffix when disabled", ua)
	}
}

func TestFlaps_GetDoesNotRetryNonConnReset(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: errors.New("some other error")},
	}}
	client := newTestFlapsClient(t, tripper)

	err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := tripper.callCount(); got != 1 {
		t.Fatalf("call count = %d, want 1 (no retry for non-ECONNRESET)", got)
	}
}

func TestSnakeCase(t *testing.T) {
	type testcase struct {
		name string
		in   string
		want string
	}

	cases := []testcase{
		{name: "case1", in: "fooBar", want: "foo_bar"},
		{name: "case2", in: appCreate.String(), want: "app_create"},
	}
	for _, tc := range cases {
		got := snakeCase(tc.in)
		if got != tc.want {
			t.Errorf("%s, got '%v', want '%v'", tc.name, got, tc.want)
		}
	}
}
