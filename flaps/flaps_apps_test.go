package flaps

import (
	"context"
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
