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
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/superfly/client-signals/go"
	"golang.org/x/net/http2"
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

func TestNewWithOptions_UserAgentFallsBackToFlyGoVersion(t *testing.T) {
	t.Setenv("FLY_FLAPS_BASE_URL", "http://example.com")

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if !strings.HasPrefix(client.userAgent, "fly-go/") {
		t.Fatalf("expected userAgent to start with %q, got %q", "fly-go/", client.userAgent)
	}
}

func TestNewWithOptions_UserAgentUsesOptsWhenProvided(t *testing.T) {
	t.Setenv("FLY_FLAPS_BASE_URL", "http://example.com")

	client, err := NewWithOptions(context.Background(), NewClientOpts{UserAgent: "flyctl/1.2.3"})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if want := "flyctl/1.2.3"; client.userAgent != want {
		t.Fatalf("expected userAgent %q, got %q", want, client.userAgent)
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
// RoundTrip returns (nil, err); otherwise it returns a response with the
// specified statusCode (or 200 if not set) and the given body.
type step struct {
	err        error
	body       string
	statusCode int
}

type scriptedTripper struct {
	mu    sync.Mutex
	calls int
	steps []step
}

type captureTripper struct {
	req *http.Request
}

func (s *scriptedTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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

	statusCode := st.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(st.body)),
		Header:     make(http.Header),
		Request:    req,
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

func TestFlaps_GetRetriesOnEOF(t *testing.T) {
	tripper := &scriptedTripper{steps: []step{
		{err: io.EOF},
		{err: io.EOF},
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

func TestFlaps_GetRetriesHTTP2InternalError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("protocol = %s, want HTTP/2", r.Proto)
		}
		if calls.Add(1) == 1 {
			panic(http.ErrAbortHandler) // Send RST_STREAM(INTERNAL_ERROR) before response headers.
		}
		fmt.Fprint(w, `{"id":"m1"}`)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)
	client, err := NewWithOptions(t.Context(), NewClientOpts{Transport: server.Client().Transport})
	if err != nil {
		t.Fatal(err)
	}
	m, err := client.Get(t.Context(), "app", "m1")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "m1" || calls.Load() != 2 {
		t.Fatalf("machine = %q, calls = %d", m.ID, calls.Load())
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

func TestFlaps_ClientSignalsAttachesHeaders(t *testing.T) {
	capture := &captureTripper{}
	client, err := NewWithOptions(context.Background(), NewClientOpts{
		Transport:     capture,
		ClientSignals: &clientsignals.Signals{Interactive: true, Parent: "shell"},
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if err := client._sendRequest(context.Background(), http.MethodGet, "/apps", nil, nil, nil); err != nil {
		t.Fatalf("_sendRequest() error = %v", err)
	}

	if got := capture.req.Header.Get("Fly-Client-Interactive"); got != "true" {
		t.Fatalf("Fly-Client-Interactive header = %q, want %q", got, "true")
	}
	if got := capture.req.Header.Get("Fly-Client-Parent"); got != "shell" {
		t.Fatalf("Fly-Client-Parent header = %q, want %q", got, "shell")
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
			t.Fatalf("%s header = %q, want empty when ClientSignals is not set", h, got)
		}
	}

	if ua := capture.req.Header.Get("User-Agent"); strings.Contains(ua, "interactive=") {
		t.Fatalf("User-Agent = %q, want no client signals suffix when disabled", ua)
	}
}

type fakeLogger struct {
	lines []string
}

func (f *fakeLogger) Debug(v ...any) { f.lines = append(f.lines, fmt.Sprint(v...)) }
func (f *fakeLogger) Debugf(format string, v ...any) {
	f.lines = append(f.lines, fmt.Sprintf(format, v...))
}

func TestFlaps_LogsClientSignalsEnabled(t *testing.T) {
	logger := &fakeLogger{}
	_, err := NewWithOptions(context.Background(), NewClientOpts{
		Transport:     &captureTripper{},
		ClientSignals: &clientsignals.Signals{},
		Logger:        logger,
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one debug line logged, got %d: %v", len(logger.lines), logger.lines)
	}
	if !strings.HasPrefix(logger.lines[0], "flaps: client signals: enabled") {
		t.Fatalf("expected debug line to start with the flaps: prefix and mention client signals are enabled, got %q", logger.lines[0])
	}
}

func TestFlaps_LogsClientSignalsDisabled(t *testing.T) {
	logger := &fakeLogger{}
	_, err := NewWithOptions(context.Background(), NewClientOpts{
		Transport: &captureTripper{},
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one debug line logged, got %d: %v", len(logger.lines), logger.lines)
	}
	if logger.lines[0] != "flaps: client signals: disabled" {
		t.Fatalf("expected debug line = %q, got %q", "flaps: client signals: disabled", logger.lines[0])
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

func TestPeerInternalResetRejectsOtherErrors(t *testing.T) {
	peer := errors.New("received from peer")
	for _, err := range []error{
		errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer"),
		http2.StreamError{StreamID: 1, Code: http2.ErrCodeCancel, Cause: peer},
		http2.StreamError{StreamID: 1, Code: http2.ErrCodeInternal},
		http2.StreamError{StreamID: 1, Code: http2.ErrCodeInternal, Cause: context.Canceled},
	} {
		if isPeerInternalReset(err) {
			t.Errorf("unexpected retry for %T: %v", err, err)
		}
	}
}

func TestPeerInternalResetAttemptLimits(t *testing.T) {
	for _, tc := range []struct {
		method   string
		attempts int32
	}{{"GET", 3}, {"POST", 1}, {"PUT", 1}, {"PATCH", 1}, {"DELETE", 1}} {
		t.Run(tc.method, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); panic(http.ErrAbortHandler) }))
			srv.EnableHTTP2 = true
			srv.StartTLS()
			defer srv.Close()
			t.Setenv("FLY_FLAPS_BASE_URL", srv.URL)
			client, err := NewWithOptions(t.Context(), NewClientOpts{Transport: srv.Client().Transport})
			if err != nil {
				t.Fatal(err)
			}
			err = client._sendRequest(t.Context(), tc.method, "/apps/probe", nil, nil, nil)
			if !isPeerInternalReset(err) || calls.Load() != tc.attempts {
				t.Fatalf("%s: attempts=%d, error=%v", tc.method, calls.Load(), err)
			}
		})
	}
}
