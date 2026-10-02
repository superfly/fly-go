package flaps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// statusRoundTripper answers every request with a canned status and body,
// which is all AppNameAvailable looks at.
type statusRoundTripper struct {
	status int
	body   string
}

type listAppsRoundTripper struct {
	bodies []string
	// statuses optionally sets the status code of each response. Missing
	// entries default to 200.
	statuses []int
	requests []*http.Request
}

func (t *listAppsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(t.requests) >= len(t.bodies) {
		return nil, fmt.Errorf("unexpected request %d", len(t.requests)+1)
	}
	t.requests = append(t.requests, req.Clone(req.Context()))
	i := len(t.requests) - 1
	status := http.StatusOK
	if i < len(t.statuses) && t.statuses[i] != 0 {
		status = t.statuses[i]
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(t.bodies[i])),
		Header:     make(http.Header),
	}, nil
}

func TestListAppsTraversesPages(t *testing.T) {
	transport := &listAppsRoundTripper{bodies: []string{
		`{"total_apps":3,"apps":[{"name":"one"}],"next_cursor":"next-page"}`,
		`{"total_apps":3,"apps":[{"name":"two"},{"name":"three"}]}`,
	}}
	client := newTestClient(t, transport)

	apps, err := client.ListApps(t.Context(), ListAppsRequest{
		OrgSlug: "acme",
		AppRole: "builder",
	})
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if len(apps) != 3 || apps[0].Name != "one" || apps[1].Name != "two" || apps[2].Name != "three" {
		t.Fatalf("ListApps() apps = %#v, want one, two, three", apps)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(transport.requests))
	}

	firstQuery := transport.requests[0].URL.Query()
	if got := firstQuery.Get("org_slug"); got != "acme" {
		t.Fatalf("first request org_slug = %q, want acme", got)
	}
	if got := firstQuery.Get("app_role"); got != "builder" {
		t.Fatalf("first request app_role = %q, want builder", got)
	}
	if got := firstQuery.Get("limit"); got != "5000" {
		t.Fatalf("first request limit = %q, want 5000", got)
	}
	if got := firstQuery.Get("cursor"); got != "" {
		t.Fatalf("first request cursor = %q, want empty", got)
	}

	secondQuery := transport.requests[1].URL.Query()
	if got := secondQuery.Get("limit"); got != "5000" {
		t.Fatalf("second request limit = %q, want 5000", got)
	}
	if got := secondQuery.Get("cursor"); got != "next-page" {
		t.Fatalf("second request cursor = %q, want next-page", got)
	}
}

func TestListAppsRetriesFailedPage(t *testing.T) {
	transport := &listAppsRoundTripper{
		bodies: []string{
			`{"total_apps":2,"apps":[{"name":"one"}],"next_cursor":"next-page"}`,
			`{"error":"internal error"}`,
			`{"total_apps":2,"apps":[{"name":"two"}]}`,
		},
		statuses: []int{0, http.StatusInternalServerError},
	}
	client := newTestClient(t, transport)

	apps, err := client.ListApps(t.Context(), ListAppsRequest{OrgSlug: "acme"})
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if len(apps) != 2 || apps[0].Name != "one" || apps[1].Name != "two" {
		t.Fatalf("ListApps() apps = %#v, want one, two", apps)
	}
	if len(transport.requests) != 3 {
		t.Fatalf("request count = %d, want 3", len(transport.requests))
	}
	if got := transport.requests[2].URL.Query().Get("cursor"); got != "next-page" {
		t.Fatalf("retried request cursor = %q, want next-page", got)
	}
}

func TestListAppsDoesNotRetryClientErrors(t *testing.T) {
	transport := &listAppsRoundTripper{
		bodies:   []string{`{"error":"cursor is expired"}`},
		statuses: []int{http.StatusBadRequest},
	}
	client := newTestClient(t, transport)

	_, err := client.ListApps(t.Context(), ListAppsRequest{OrgSlug: "acme"})
	var ferr *FlapsError
	if !errors.As(err, &ferr) || ferr.ResponseStatusCode != http.StatusBadRequest {
		t.Fatalf("ListApps() error = %v, want a 400 FlapsError", err)
	}
	if len(transport.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(transport.requests))
	}
}

func TestListAppsRejectsRepeatedCursor(t *testing.T) {
	transport := &listAppsRoundTripper{bodies: []string{
		`{"total_apps":2,"apps":[{"name":"one"}],"next_cursor":"same-page"}`,
		`{"total_apps":2,"apps":[{"name":"two"}],"next_cursor":"same-page"}`,
	}}
	client := newTestClient(t, transport)

	if _, err := client.ListApps(t.Context(), ListAppsRequest{OrgSlug: "acme"}); err == nil {
		t.Fatal("ListApps() error = nil, want an error for a repeated cursor")
	}
	if len(transport.requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(transport.requests))
	}
}

func (s *statusRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

func TestAppNameAvailable(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		wantOK  bool
		wantErr bool
	}{
		{
			// The regression case: the app is ours and GetApp succeeds, so
			// there is no error to classify.
			name:   "app exists",
			status: http.StatusOK,
			body:   `{"id":"app-id","name":"taken-app","status":"deployed"}`,
		},
		{
			// The app exists in an org we cannot see into. The API looks the
			// name up globally and then refuses the read, so this is a 403 —
			// and the name is taken.
			name:   "forbidden",
			status: http.StatusForbidden,
			body:   `{"error":"unauthorized"}`,
		},
		{
			// Nobody asked the question successfully, so this says nothing
			// about the name and must not be reported as taken.
			name:    "unauthenticated",
			status:  http.StatusUnauthorized,
			body:    `{"error":"invalid token"}`,
			wantErr: true,
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   `{"error":"app not found"}`,
			wantOK: true,
		},
		{
			// Anything else says nothing about the name, so it propagates.
			name:    "server error",
			status:  http.StatusInternalServerError,
			body:    `{"error":"something went wrong"}`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, &statusRoundTripper{status: tc.status, body: tc.body})

			ok, err := client.AppNameAvailable(context.Background(), "some-app")

			if tc.wantErr {
				if err == nil {
					t.Fatal("AppNameAvailable() error = nil, want non-nil")
				}
			} else if err != nil {
				t.Fatalf("AppNameAvailable() error = %v, want nil", err)
			}

			if ok != tc.wantOK {
				t.Fatalf("AppNameAvailable() ok = %v, want %v", ok, tc.wantOK)
			}
		})
	}
}
