package flaps

import (
	"context"
	"encoding/json"
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
			`"created_at":"2026-08-24T00:00:00Z","attached_apps":[{"name":"example-app"}]}]}`,
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

func TestDeleteManagedPostgresClusterClassifiesGone(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusGone, body: `{"error":"gone"}`}
	client := newTestFlapsClient(t, transport)

	err := client.DeleteManagedPostgresCluster(context.Background(), "mpg-123")
	if !errors.Is(err, ErrFlapsGone) {
		t.Fatalf("DeleteManagedPostgresCluster() error = %v, want ErrFlapsGone", err)
	}
}

func TestListManagedPostgresDatabases(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body:       `{"data":[{"name":"app"},{"name":"reports"}]}`,
	}
	client := newTestFlapsClient(t, transport)

	databases, err := client.ListManagedPostgresDatabases(context.Background(), "mpg-123")
	if err != nil {
		t.Fatalf("ListManagedPostgresDatabases() error = %v", err)
	}
	if transport.req.Method != http.MethodGet {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodGet)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/databases"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	if len(databases) != 2 {
		t.Fatalf("database count = %d, want 2", len(databases))
	}
	if got, want := databases[0].Name, "app"; got != want {
		t.Fatalf("databases[0].Name = %q, want %q", got, want)
	}
	if got, want := databases[1].Name, "reports"; got != want {
		t.Fatalf("databases[1].Name = %q, want %q", got, want)
	}
}

func TestListManagedPostgresDatabasesEscapesQuestionMarkInClusterID(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusOK, body: `{"data":[]}`}
	client := newTestFlapsClient(t, transport)

	if _, err := client.ListManagedPostgresDatabases(context.Background(), "my?cluster"); err != nil {
		t.Fatalf("ListManagedPostgresDatabases() error = %v", err)
	}
	if got, want := transport.req.URL.EscapedPath(), "/v1/postgres/my%3Fcluster/databases"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
}

func TestListManagedPostgresDatabasesClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"Cluster not found"}`}
	client := newTestFlapsClient(t, transport)

	_, err := client.ListManagedPostgresDatabases(context.Background(), "mpg-123")
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("ListManagedPostgresDatabases() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestCreateManagedPostgresDatabase(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body:       `{"data":{"name":"reports"}}`,
	}
	client := newTestFlapsClient(t, transport)

	database, err := client.CreateManagedPostgresDatabase(context.Background(), "mpg-123", CreateManagedPostgresDatabaseRequest{Name: "reports"})
	if err != nil {
		t.Fatalf("CreateManagedPostgresDatabase() error = %v", err)
	}
	if transport.req.Method != http.MethodPost {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodPost)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/databases"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}

	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent.Name, "reports"; got != want {
		t.Fatalf("sent name = %q, want %q", got, want)
	}

	if got, want := database.Name, "reports"; got != want {
		t.Fatalf("database.Name = %q, want %q", got, want)
	}
}

func TestCreateManagedPostgresDatabaseEscapesQuestionMarkInClusterID(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusCreated, body: `{"data":{"name":"reports"}}`}
	client := newTestFlapsClient(t, transport)

	if _, err := client.CreateManagedPostgresDatabase(context.Background(), "my?cluster", CreateManagedPostgresDatabaseRequest{Name: "reports"}); err != nil {
		t.Fatalf("CreateManagedPostgresDatabase() error = %v", err)
	}
	if got, want := transport.req.URL.EscapedPath(), "/v1/postgres/my%3Fcluster/databases"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
}

func TestCreateManagedPostgresDatabaseClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"Cluster not found"}`}
	client := newTestFlapsClient(t, transport)

	_, err := client.CreateManagedPostgresDatabase(context.Background(), "mpg-123", CreateManagedPostgresDatabaseRequest{Name: "reports"})
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("CreateManagedPostgresDatabase() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestListManagedPostgresBackups(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body: `{"data":[{"id":"backup-1","status":"completed","type":"full","size_bytes":1234,` +
			`"started_at":"2026-08-26T12:00:00Z","finished_at":"2026-08-26T12:05:00Z"}]}`,
	}
	client := newTestFlapsClient(t, transport)

	backups, err := client.ListManagedPostgresBackups(context.Background(), "mpg-123")
	if err != nil {
		t.Fatalf("ListManagedPostgresBackups() error = %v", err)
	}
	if transport.req.Method != http.MethodGet {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodGet)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/backups"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
	backup := backups[0]
	if got, want := backup.ID, "backup-1"; got != want {
		t.Fatalf("backup ID = %q, want %q", got, want)
	}
	if got, want := backup.SizeBytes, int64(1234); got != want {
		t.Fatalf("backup size = %d, want %d", got, want)
	}
	if got, want := backup.StartedAt, "2026-08-26T12:00:00Z"; got != want {
		t.Fatalf("backup start = %q, want %q", got, want)
	}
}

func TestURLFromBaseURLPreservesRawPathAndQuery(t *testing.T) {
	client := newTestFlapsClient(t, &managedPostgresRoundTripper{statusCode: http.StatusOK, body: `{}`})

	got, err := client.urlFromBaseUrl("/v1/postgres/a%2Fb/backups?include_deleted=true")
	if err != nil {
		t.Fatalf("urlFromBaseUrl() error = %v", err)
	}
	if want := "/v1/postgres/a%2Fb/backups?include_deleted=true"; got.RequestURI() != want {
		t.Fatalf("request URI = %q, want %q", got.RequestURI(), want)
	}
}

func TestListManagedPostgresBackupsClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"Cluster not found"}`}
	client := newTestFlapsClient(t, transport)

	_, err := client.ListManagedPostgresBackups(context.Background(), "mpg-123")
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("ListManagedPostgresBackups() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestCreateManagedPostgresBackup(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusAccepted}
	client := newTestFlapsClient(t, transport)

	if err := client.CreateManagedPostgresBackup(context.Background(), "mpg-123", CreateManagedPostgresBackupRequest{Type: "incr"}); err != nil {
		t.Fatalf("CreateManagedPostgresBackup() error = %v", err)
	}
	if transport.req.Method != http.MethodPost {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodPost)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/backups"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent CreateManagedPostgresBackupRequest
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent.Type, "incr"; got != want {
		t.Fatalf("sent type = %q, want %q", got, want)
	}
}

func TestManagedPostgresBackupsPreserveEscapedPath(t *testing.T) {
	operations := []struct {
		name       string
		statusCode int
		body       string
		run        func(*Client, string) error
	}{
		{name: "list", statusCode: http.StatusOK, body: `{"data":[]}`, run: func(client *Client, id string) error {
			_, err := client.ListManagedPostgresBackups(context.Background(), id)
			return err
		}},
		{name: "create", statusCode: http.StatusAccepted, run: func(client *Client, id string) error {
			return client.CreateManagedPostgresBackup(context.Background(), id, CreateManagedPostgresBackupRequest{Type: "full"})
		}},
	}
	paths := []struct {
		name        string
		clusterID   string
		expectedURI string
	}{
		{name: "slash", clusterID: "a/b", expectedURI: "/v1/postgres/a%2Fb/backups"},
		{name: "slash_and_dot_segment", clusterID: "a/../b", expectedURI: "/v1/postgres/a%2F..%2Fb/backups"},
	}

	for _, operation := range operations {
		for _, path := range paths {
			t.Run(operation.name+"/"+path.name, func(t *testing.T) {
				transport := &managedPostgresRoundTripper{statusCode: operation.statusCode, body: operation.body}
				if err := operation.run(newTestFlapsClient(t, transport), path.clusterID); err != nil {
					t.Fatalf("request error = %v", err)
				}
				if got := transport.req.URL.RequestURI(); got != path.expectedURI {
					t.Fatalf("request URI = %q, want %q", got, path.expectedURI)
				}
			})
		}
	}
}

func TestCreateManagedPostgresBackupClassifiesConflict(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusConflict, body: `{"error":"backup already in progress"}`}
	client := newTestFlapsClient(t, transport)

	err := client.CreateManagedPostgresBackup(context.Background(), "mpg-123", CreateManagedPostgresBackupRequest{Type: "full"})
	var flapsErr *FlapsError
	if !errors.As(err, &flapsErr) {
		t.Fatalf("CreateManagedPostgresBackup() error = %v, want FlapsError", err)
	}
	if got, want := flapsErr.ResponseStatusCode, http.StatusConflict; got != want {
		t.Fatalf("response status = %d, want %d", got, want)
	}
	if got, want := flapsErr.Error(), "backup already in progress"; got != want {
		t.Fatalf("error message = %q, want %q", got, want)
	}
}

func TestCreateManagedPostgresBackupClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"Cluster not found"}`}
	client := newTestFlapsClient(t, transport)

	err := client.CreateManagedPostgresBackup(context.Background(), "mpg-123", CreateManagedPostgresBackupRequest{Type: "full"})
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("CreateManagedPostgresBackup() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestRestoreManagedPostgresClusterFromBackup(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body: `{"data":{"id":"mpg-restored","name":"restored-db","status":"creating",` +
			`"region":"iad","plan":"basic","disk_size_gb":25,"pg_major_version":"17"}}`,
	}
	client := newTestFlapsClient(t, transport)

	cluster, err := client.RestoreManagedPostgresCluster(context.Background(), "mpg-source", RestoreManagedPostgresClusterRequest{
		BackupID: "20260601-120000F",
		Name:     "restored-db",
	})
	if err != nil {
		t.Fatalf("RestoreManagedPostgresCluster() error = %v", err)
	}
	if transport.req.Method != http.MethodPost {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodPost)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-source/restore"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}

	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent["backup_id"], "20260601-120000F"; got != want {
		t.Fatalf("sent backup_id = %v, want %q", got, want)
	}
	if got, want := sent["name"], "restored-db"; got != want {
		t.Fatalf("sent name = %v, want %q", got, want)
	}
	if _, ok := sent["pitr_time"]; ok {
		t.Fatalf("pitr_time unexpectedly present in request: %s", body)
	}
	if got, want := cluster.ID, "mpg-restored"; got != want {
		t.Fatalf("cluster ID = %q, want %q", got, want)
	}
	if got, want := cluster.Status, "creating"; got != want {
		t.Fatalf("cluster status = %q, want %q", got, want)
	}
}

func TestRestoreManagedPostgresClusterToPointInTime(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body:       `{"data":{"id":"mpg-restored","status":"creating"}}`,
	}
	client := newTestFlapsClient(t, transport)

	_, err := client.RestoreManagedPostgresCluster(context.Background(), "mpg-source", RestoreManagedPostgresClusterRequest{
		PITRTime: "2026-06-01T12:02:30Z",
	})
	if err != nil {
		t.Fatalf("RestoreManagedPostgresCluster() error = %v", err)
	}
	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent["pitr_time"], "2026-06-01T12:02:30Z"; got != want {
		t.Fatalf("sent pitr_time = %v, want %q", got, want)
	}
	if _, ok := sent["backup_id"]; ok {
		t.Fatalf("backup_id unexpectedly present in request: %s", body)
	}
	if _, ok := sent["name"]; ok {
		t.Fatalf("name unexpectedly present in request: %s", body)
	}
}

func TestRestoreManagedPostgresClusterPreservesEscapedPath(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body:       `{"data":{"id":"mpg-restored"}}`,
	}
	client := newTestFlapsClient(t, transport)

	_, err := client.RestoreManagedPostgresCluster(context.Background(), "a/../b", RestoreManagedPostgresClusterRequest{
		BackupID: "backup-1",
	})
	if err != nil {
		t.Fatalf("RestoreManagedPostgresCluster() error = %v", err)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/a%2F..%2Fb/restore"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
}

func TestRestoreManagedPostgresClusterClassifiesNotFound(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusNotFound,
		body:       `{"error":"Backup missing not found"}`,
	}
	client := newTestFlapsClient(t, transport)

	_, err := client.RestoreManagedPostgresCluster(context.Background(), "mpg-source", RestoreManagedPostgresClusterRequest{
		BackupID: "missing",
	})
	if !errors.Is(err, ErrFlapsNotFound) {
		t.Fatalf("RestoreManagedPostgresCluster() error = %v, want ErrFlapsNotFound", err)
	}
}

func TestRestoreManagedPostgresClusterPreservesValidationError(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusUnprocessableEntity,
		body:       `{"error":"backup_id and pitr_time are mutually exclusive"}`,
	}
	client := newTestFlapsClient(t, transport)

	_, err := client.RestoreManagedPostgresCluster(context.Background(), "mpg-source", RestoreManagedPostgresClusterRequest{
		BackupID: "backup-1",
		PITRTime: "2026-06-01T12:02:30Z",
	})
	var flapsErr *FlapsError
	if !errors.As(err, &flapsErr) {
		t.Fatalf("RestoreManagedPostgresCluster() error = %v, want FlapsError", err)
	}
	if got, want := flapsErr.ResponseStatusCode, http.StatusUnprocessableEntity; got != want {
		t.Fatalf("response status = %d, want %d", got, want)
	}
	if got, want := flapsErr.Error(), "backup_id and pitr_time are mutually exclusive"; got != want {
		t.Fatalf("error message = %q, want %q", got, want)
	}
}
