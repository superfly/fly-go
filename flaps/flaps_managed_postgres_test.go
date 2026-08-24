package flaps

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type managedPostgresRoundTripper struct {
	req        *http.Request
	statusCode int
	body       string
}

func (t *managedPostgresRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req.Clone(req.Context())
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     make(http.Header),
	}, nil
}

func TestListManagedPostgresClusters(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body: `{"data":[{"id":"mpg-123","name":"example","status":"ready","region":"iad","plan":"basic",` +
			`"created_at":"2026-08-24T00:00:00Z","deleted_at":"","attached_apps":[{"name":"example-app"}]}]}`,
	}
	client := newTestFlapsClient(t, transport)

	clusters, err := client.ListManagedPostgresClusters(context.Background(), ListManagedPostgresClustersRequest{
		OrgSlug:        "example-org",
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("ListManagedPostgresClusters() error = %v", err)
	}
	if transport.req.Method != http.MethodGet {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodGet)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.Query().Get("org_slug"), "example-org"; got != want {
		t.Fatalf("org_slug = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.Query().Get("include_deleted"), "true"; got != want {
		t.Fatalf("include_deleted = %q, want %q", got, want)
	}
	if len(clusters) != 1 {
		t.Fatalf("cluster count = %d, want 1", len(clusters))
	}
	if got, want := clusters[0].ID, "mpg-123"; got != want {
		t.Fatalf("cluster ID = %q, want %q", got, want)
	}
	if got, want := clusters[0].AttachedApps[0].Name, "example-app"; got != want {
		t.Fatalf("attached app = %q, want %q", got, want)
	}
}

func TestListManagedPostgresClustersOmitsIncludeDeletedByDefault(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusOK, body: `{"data":[]}`}
	client := newTestFlapsClient(t, transport)

	_, err := client.ListManagedPostgresClusters(context.Background(), ListManagedPostgresClustersRequest{OrgSlug: "example-org"})
	if err != nil {
		t.Fatalf("ListManagedPostgresClusters() error = %v", err)
	}
	if transport.req.URL.Query().Has("include_deleted") {
		t.Fatalf("include_deleted unexpectedly present in query: %q", transport.req.URL.RawQuery)
	}
}

func TestListManagedPostgresClustersClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"not found"}`}
	client := newTestFlapsClient(t, transport)

	_, err := client.ListManagedPostgresClusters(context.Background(), ListManagedPostgresClustersRequest{OrgSlug: "example-org"})
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("ListManagedPostgresClusters() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestDeleteManagedPostgresCluster(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusAccepted}
	client := newTestFlapsClient(t, transport)

	if err := client.DeleteManagedPostgresCluster(context.Background(), "mpg-123"); err != nil {
		t.Fatalf("DeleteManagedPostgresCluster() error = %v", err)
	}
	if transport.req.Method != http.MethodDelete {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodDelete)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
}

func TestDeleteManagedPostgresClusterClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"not found"}`}
	client := newTestFlapsClient(t, transport)

	err := client.DeleteManagedPostgresCluster(context.Background(), "mpg-123")
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("DeleteManagedPostgresCluster() error = %v, want ErrFlapsNotFound", err)
	}
}
