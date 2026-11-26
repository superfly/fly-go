package flaps

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type readTransport func(*http.Request) (*http.Response, error)

func (f readTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMachineHistoryPreservesAffinity(t *testing.T) {
	const base = "https://flaps.example"
	t.Setenv("FLY_FLAPS_BASE_URL", base)
	store := &memAffinityStore{}
	expires := time.Now().Add(time.Minute)
	first := testAffinityToken(t, "app-a", "first", expires)
	next := testAffinityToken(t, "app-a", "next", expires)
	store.StoreAffinity(base, "app-a", first, expires)
	calls := 0
	transport := readTransport(func(r *http.Request) (*http.Response, error) {
		want := first
		if calls > 0 {
			want = next
		}
		if got := r.Header.Get(AffinityHeader); got != want {
			t.Fatalf("unexpected affinity on read %d", calls+1)
		}
		path, body := "/v1/apps/app-a/machines/m?version=v", `{"id":"m"}`
		if calls == 1 {
			path, body = "/v1/apps/app-a/machines/m/events?limit=20", `[{"id":"exit"}]`
		}
		if r.Method != http.MethodGet || r.URL.RequestURI() != path {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		calls++
		headers := http.Header{}
		headers.Set(AffinityHeader, next)
		return &http.Response{StatusCode: 200, Header: headers, Request: r, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client, err := NewWithOptions(t.Context(), NewClientOpts{AffinityStore: store, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	var machine struct{ ID string }
	if err := client.GetMachineVersion(t.Context(), "app-a", "m", "v", &machine); err != nil || machine.ID != "m" {
		t.Fatalf("version read: %+v, %v", machine, err)
	}
	var events []struct{ ID string }
	if err := client.GetMachineEvents(t.Context(), "app-a", "m", 20, &events); err != nil || len(events) != 1 || events[0].ID != "exit" {
		t.Fatalf("events read: %+v, %v", events, err)
	}
}

func TestMachineHistoryRefusesRedirect(t *testing.T) {
	calls := 0
	client, err := NewWithOptions(t.Context(), NewClientOpts{Transport: readTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			t.Fatal("followed Machine history redirect")
		}
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"/elsewhere"}}, Request: r, Body: http.NoBody}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := client.GetMachineVersion(t.Context(), "app-a", "m", "v", &out); err == nil || calls != 1 {
		t.Fatalf("redirect read: calls=%d, err=%v", calls, err)
	}
}
