package flaps

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type roundTripRecorder struct {
	req *http.Request
}

func (r *roundTripRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(http.NoBody),
		Header:     make(http.Header),
	}, nil
}

func newTestClient(t *testing.T, transport http.RoundTripper) *Client {
	t.Helper()

	baseURL, err := url.Parse("https://machines.test")
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}

	return &Client{
		baseUrl:    baseURL,
		httpClient: &http.Client{Transport: transport},
		userAgent:  "fly-go-test",
	}
}

func TestWaitDefaults(t *testing.T) {
	recorder := &roundTripRecorder{}
	client := newTestClient(t, recorder)

	err := client.Wait(context.Background(), "my-app", "m123")
	if err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	if recorder.req == nil {
		t.Fatal("expected a request to be sent")
	}

	if recorder.req.Method != http.MethodGet {
		t.Fatalf("method = %s, want %s", recorder.req.Method, http.MethodGet)
	}

	if recorder.req.URL.Path != "/v1/apps/my-app/machines/m123/wait" {
		t.Fatalf("path = %s, want %s", recorder.req.URL.Path, "/v1/apps/my-app/machines/m123/wait")
	}

	query := recorder.req.URL.Query()
	if got := query.Get("state"); got != "started" {
		t.Fatalf("state = %q, want %q", got, "started")
	}
	if got := query.Get("timeout"); got != "60" {
		t.Fatalf("timeout = %q, want %q", got, "60")
	}
	if got := query.Get("version"); got != "" {
		t.Fatalf("version = %q, want empty", got)
	}
	if got := query.Get("from_event_id"); got != "" {
		t.Fatalf("from_event_id = %q, want empty", got)
	}
}

func TestWaitOptions(t *testing.T) {
	recorder := &roundTripRecorder{}
	client := newTestClient(t, recorder)

	err := client.Wait(
		context.Background(),
		"my-app",
		"m123",
		WithWaitStates("stopped", "suspended"),
		WithWaitTimeout(30*time.Second),
		WithWaitVersion("01JC1W3CW1VEKNMHX7MJ70DE1D"),
		WithWaitFromEventID("01JC1V3CW1VEKNMHX7MJ70DE1D"),
	)
	if err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	query := recorder.req.URL.Query()
	states := query["state"]
	if len(states) != 2 || states[0] != "stopped" || states[1] != "suspended" {
		t.Fatalf("state = %v, want %v", states, []string{"stopped", "suspended"})
	}
	if got := query.Get("timeout"); got != "30" {
		t.Fatalf("timeout = %q, want %q", got, "30")
	}
	if got := query.Get("version"); got != "01JC1W3CW1VEKNMHX7MJ70DE1D" {
		t.Fatalf("version = %q, want %q", got, "01JC1W3CW1VEKNMHX7MJ70DE1D")
	}
	if got := query.Get("from_event_id"); got != "01JC1V3CW1VEKNMHX7MJ70DE1D" {
		t.Fatalf("from_event_id = %q, want %q", got, "01JC1V3CW1VEKNMHX7MJ70DE1D")
	}
}

func TestWaitTimeoutClamp(t *testing.T) {
	t.Run("clamps to max", func(t *testing.T) {
		recorder := &roundTripRecorder{}
		client := newTestClient(t, recorder)

		err := client.Wait(context.Background(), "my-app", "m123", WithWaitTimeout(2*time.Minute))
		if err != nil {
			t.Fatalf("Wait returned error: %v", err)
		}

		if got := recorder.req.URL.Query().Get("timeout"); got != "60" {
			t.Fatalf("timeout = %q, want %q", got, "60")
		}
	})

	t.Run("clamps to min", func(t *testing.T) {
		recorder := &roundTripRecorder{}
		client := newTestClient(t, recorder)

		err := client.Wait(context.Background(), "my-app", "m123", WithWaitTimeout(500*time.Millisecond))
		if err != nil {
			t.Fatalf("Wait returned error: %v", err)
		}

		query := recorder.req.URL.Query()
		if got := query.Get("timeout"); got != "1" {
			t.Fatalf("timeout = %q, want %q", got, "1")
		}
		states := query["state"]
		if len(states) != 1 || states[0] != "started" {
			t.Fatalf("state = %v, want %v", states, []string{"started"})
		}
	})
}

func TestWaitStatesNoCommaSplitting(t *testing.T) {
	recorder := &roundTripRecorder{}
	client := newTestClient(t, recorder)

	err := client.Wait(context.Background(), "my-app", "m123", WithWaitStates("started,stopped"))
	if err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}

	states := recorder.req.URL.Query()["state"]
	if len(states) != 1 || states[0] != "started,stopped" {
		t.Fatalf("state = %v, want %v", states, []string{"started,stopped"})
	}
}
