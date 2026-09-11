package flaps

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
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
		Request:    req,
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

func TestListManagedPostgresUsers(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body:       `{"data":[{"username":"app_user","role":"writer"},{"username":"analyst","role":"schema_admin"}]}`,
	}
	client := newTestFlapsClient(t, transport)

	users, err := client.ListManagedPostgresUsers(context.Background(), "mpg-123")
	if err != nil {
		t.Fatalf("ListManagedPostgresUsers() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodGet; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/users"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresUserList; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	if len(users) != 2 {
		t.Fatalf("user count = %d, want 2", len(users))
	}
	if got, want := users[0].Username, "app_user"; got != want {
		t.Fatalf("users[0].Username = %q, want %q", got, want)
	}
	if got, want := users[0].Role, ManagedPostgresUserRoleWriter; got != want {
		t.Fatalf("users[0].Role = %q, want %q", got, want)
	}
	if got, want := users[1].Username, "analyst"; got != want {
		t.Fatalf("users[1].Username = %q, want %q", got, want)
	}
	if got, want := users[1].Role, ManagedPostgresUserRoleSchemaAdmin; got != want {
		t.Fatalf("users[1].Role = %q, want %q", got, want)
	}
}

func TestCreateManagedPostgresUser(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body:       `{"data":{"username":"reporter","role":"reader"}}`,
	}
	client := newTestFlapsClient(t, transport)

	user, err := client.CreateManagedPostgresUser(context.Background(), "mpg-123", CreateManagedPostgresUserRequest{
		Username: "reporter",
		Role:     "reader",
	})
	if err != nil {
		t.Fatalf("CreateManagedPostgresUser() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodPost; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/users"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresUserCreate; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent CreateManagedPostgresUserRequest
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent, (CreateManagedPostgresUserRequest{Username: "reporter", Role: "reader"}); got != want {
		t.Fatalf("request body = %#v, want %#v", got, want)
	}
	if got, want := user.Username, "reporter"; got != want {
		t.Fatalf("user.Username = %q, want %q", got, want)
	}
	if got, want := user.Role, ManagedPostgresUserRoleReader; got != want {
		t.Fatalf("user.Role = %q, want %q", got, want)
	}
}

func TestUpdateManagedPostgresUserRole(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	err := client.UpdateManagedPostgresUserRole(context.Background(), "mpg-123", "reporter", UpdateManagedPostgresUserRoleRequest{Role: "writer"})
	if err != nil {
		t.Fatalf("UpdateManagedPostgresUserRole() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodPatch; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/users/reporter"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresUserUpdate; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent UpdateManagedPostgresUserRoleRequest
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent.Role, "writer"; got != want {
		t.Fatalf("sent role = %q, want %q", got, want)
	}
}

func TestDeleteManagedPostgresUser(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	if err := client.DeleteManagedPostgresUser(context.Background(), "mpg-123", "reporter"); err != nil {
		t.Fatalf("DeleteManagedPostgresUser() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodDelete; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/users/reporter"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresUserDelete; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	if transport.req.Body != nil {
		t.Fatalf("request body is non-nil")
	}
}

func TestRotateManagedPostgresUserPassword(t *testing.T) {
	tests := []struct {
		name          string
		killSessions  bool
		wantKeyInJSON bool
	}{
		{
			name:          "kill_sessions_true",
			killSessions:  true,
			wantKeyInJSON: true,
		},
		{
			name:          "kill_sessions_false_omitted",
			killSessions:  false,
			wantKeyInJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{
				statusCode: http.StatusOK,
				body:       `{"data":{"username":"app_user","password":"newpassword123"}}`,
			}
			client := newTestFlapsClient(t, transport)

			creds, err := client.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "app_user", RotateManagedPostgresUserPasswordRequest{KillSessions: tt.killSessions})
			if err != nil {
				t.Fatalf("RotateManagedPostgresUserPassword() error = %v", err)
			}
			if got, want := transport.req.Method, http.MethodPost; got != want {
				t.Fatalf("request method = %q, want %q", got, want)
			}
			if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/users/app_user/rotate_password"; got != want {
				t.Fatalf("request URI = %q, want %q", got, want)
			}
			if got, want := actionFromContext(transport.req.Context()), managedPostgresUserRotatePassword; got != want {
				t.Fatalf("request action = %q, want %q", got, want)
			}
			body, err := io.ReadAll(transport.req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			var sent map[string]any
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Fatalf("decode request body %q: %v", string(body), err)
			}
			if tt.wantKeyInJSON {
				if got, ok := sent["kill_sessions"]; !ok {
					t.Fatalf("kill_sessions key unexpectedly absent in request body: %s", body)
				} else if got != true {
					t.Fatalf("kill_sessions = %v, want true", got)
				}
			} else {
				if _, ok := sent["kill_sessions"]; ok {
					t.Fatalf("kill_sessions key unexpectedly present in request body: %s", body)
				}
			}
			if got, want := creds.Username, "app_user"; got != want {
				t.Fatalf("creds.Username = %q, want %q", got, want)
			}
			if got, want := creds.Password, "newpassword123"; got != want {
				t.Fatalf("creds.Password = %q, want %q", got, want)
			}
		})
	}
}

func TestManagedPostgresUsersPreserveEscapedPaths(t *testing.T) {
	operations := []struct {
		name       string
		statusCode int
		body       string
		run        func(*Client, string, string) error
	}{
		{name: "list", statusCode: http.StatusOK, body: `{"data":[]}`, run: func(client *Client, id, _ string) error {
			_, err := client.ListManagedPostgresUsers(context.Background(), id)
			return err
		}},
		{name: "create", statusCode: http.StatusCreated, body: `{"data":{"username":"new_user","role":"reader"}}`, run: func(client *Client, id, _ string) error {
			_, err := client.CreateManagedPostgresUser(context.Background(), id, CreateManagedPostgresUserRequest{Username: "new_user", Role: "reader"})
			return err
		}},
		{name: "update", statusCode: http.StatusNoContent, run: func(client *Client, id, username string) error {
			return client.UpdateManagedPostgresUserRole(context.Background(), id, username, UpdateManagedPostgresUserRoleRequest{Role: "reader"})
		}},
		{name: "rotate", statusCode: http.StatusOK, body: `{"data":{"username":"new_user","password":"newpwd123"}}`, run: func(client *Client, id, username string) error {
			_, err := client.RotateManagedPostgresUserPassword(context.Background(), id, username, RotateManagedPostgresUserPasswordRequest{})
			return err
		}},
		{name: "delete", statusCode: http.StatusNoContent, run: func(client *Client, id, username string) error {
			return client.DeleteManagedPostgresUser(context.Background(), id, username)
		}},
	}
	paths := []struct {
		name        string
		clusterID   string
		username    string
		expectedURI func(string) string
	}{
		{name: "slash", clusterID: "a/b", username: "user/name", expectedURI: func(operation string) string {
			if operation == "list" || operation == "create" {
				return "/v1/postgres/a%2Fb/users"
			}
			if operation == "rotate" {
				return "/v1/postgres/a%2Fb/users/user%2Fname/rotate_password"
			}

			return "/v1/postgres/a%2Fb/users/user%2Fname"
		}},
		{name: "slash_and_dot_segment", clusterID: "a/../b", username: "user/../name", expectedURI: func(operation string) string {
			if operation == "list" || operation == "create" {
				return "/v1/postgres/a%2F..%2Fb/users"
			}
			if operation == "rotate" {
				return "/v1/postgres/a%2F..%2Fb/users/user%2F..%2Fname/rotate_password"
			}

			return "/v1/postgres/a%2F..%2Fb/users/user%2F..%2Fname"
		}},
	}

	for _, operation := range operations {
		for _, path := range paths {
			t.Run(operation.name+"/"+path.name, func(t *testing.T) {
				transport := &managedPostgresRoundTripper{statusCode: operation.statusCode, body: operation.body}
				if err := operation.run(newTestFlapsClient(t, transport), path.clusterID, path.username); err != nil {
					t.Fatalf("request error = %v", err)
				}
				if got, want := transport.req.URL.RequestURI(), path.expectedURI(operation.name); got != want {
					t.Fatalf("request URI = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestManagedPostgresUsersClassifyNotFound(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Client) error
	}{
		{name: "list", run: func(client *Client) error {
			_, err := client.ListManagedPostgresUsers(context.Background(), "missing")
			return err
		}},
		{name: "create", run: func(client *Client) error {
			_, err := client.CreateManagedPostgresUser(context.Background(), "missing", CreateManagedPostgresUserRequest{Username: "reporter", Role: "reader"})
			return err
		}},
		{name: "update", run: func(client *Client) error {
			return client.UpdateManagedPostgresUserRole(context.Background(), "mpg-123", "missing", UpdateManagedPostgresUserRoleRequest{Role: "reader"})
		}},
		{name: "rotate", run: func(client *Client) error {
			_, err := client.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "missing", RotateManagedPostgresUserPasswordRequest{})
			return err
		}},
		{name: "delete", run: func(client *Client) error {
			return client.DeleteManagedPostgresUser(context.Background(), "mpg-123", "missing")
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"not found"}`}
			err := operation.run(newTestFlapsClient(t, transport))
			if !errors.Is(err, ErrFlapsNotFound) {
				t.Fatalf("request error = %v, want ErrFlapsNotFound", err)
			}
			var flapsErr *FlapsError
			if !errors.As(err, &flapsErr) {
				t.Fatalf("request error = %v, want wrapped FlapsError", err)
			}
		})
	}
}

func TestManagedPostgresUsersPreserveNonNotFoundErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		run        func(*Client) error
	}{
		{name: "create_bad_request", statusCode: http.StatusBadRequest, run: func(client *Client) error {
			_, err := client.CreateManagedPostgresUser(context.Background(), "mpg-123", CreateManagedPostgresUserRequest{Username: "reporter", Role: "invalid"})
			return err
		}},
		{name: "create_conflict", statusCode: http.StatusConflict, run: func(client *Client) error {
			_, err := client.CreateManagedPostgresUser(context.Background(), "mpg-123", CreateManagedPostgresUserRequest{Username: "reporter", Role: "reader"})
			return err
		}},
		{name: "update_unprocessable", statusCode: http.StatusUnprocessableEntity, run: func(client *Client) error {
			return client.UpdateManagedPostgresUserRole(context.Background(), "mpg-123", "reporter", UpdateManagedPostgresUserRoleRequest{Role: "invalid"})
		}},
		{name: "rotate_conflict", statusCode: http.StatusConflict, run: func(client *Client) error {
			_, err := client.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "reporter", RotateManagedPostgresUserPasswordRequest{})
			return err
		}},
		// Deleting a reserved user (postgres, flypgadmin, …) is rejected with
		// 400 by the API. It must surface as a FlapsError carrying that status,
		// not be silently swallowed or misclassified as 404.
		{name: "delete_bad_request", statusCode: http.StatusBadRequest, run: func(client *Client) error {
			return client.DeleteManagedPostgresUser(context.Background(), "mpg-123", "postgres")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: test.statusCode, body: `{"error":"request rejected"}`}
			err := test.run(newTestFlapsClient(t, transport))
			if errors.Is(err, ErrFlapsNotFound) {
				t.Fatalf("request error = %v, unexpectedly classified as not found", err)
			}
			var flapsErr *FlapsError
			if !errors.As(err, &flapsErr) {
				t.Fatalf("request error = %v, want FlapsError", err)
			}
			if got, want := flapsErr.ResponseStatusCode, test.statusCode; got != want {
				t.Fatalf("response status = %d, want %d", got, want)
			}
		})
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

func TestListManagedPostgresExtensions(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body: `{"data":[` +
			`{"name":"pg_trgm","description":"text similarity","default_version":"1.6","system":false,"installed":null},` +
			`{"name":"postgis","description":"geographic objects","default_version":"3.4","system":true,"installed":{"version":"3.4.1","schema":"public"}},` +
			`{"name":"hstore","description":null,"default_version":null,"system":false,"installed":null}]}`,
	}
	client := newTestFlapsClient(t, transport)

	extensions, err := client.ListManagedPostgresExtensions(context.Background(), "mpg-123", "app")
	if err != nil {
		t.Fatalf("ListManagedPostgresExtensions() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodGet; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/databases/app/extensions"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresExtensionList; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	if len(extensions) != 3 || extensions[0].Installed != nil || extensions[1].Installed == nil || extensions[2].Installed != nil {
		t.Fatalf("extensions = %#v, want uninstalled and installed extensions", extensions)
	}
	if got, want := extensions[0].Name, "pg_trgm"; got != want {
		t.Fatalf("extensions[0].Name = %q, want %q", got, want)
	}
	if extensions[0].Description == nil || *extensions[0].Description != "text similarity" {
		t.Fatalf("extensions[0].Description = %v, want %q", extensions[0].Description, "text similarity")
	}
	if extensions[0].DefaultVersion == nil || *extensions[0].DefaultVersion != "1.6" {
		t.Fatalf("extensions[0].DefaultVersion = %v, want %q", extensions[0].DefaultVersion, "1.6")
	}
	if got, want := extensions[1].Name, "postgis"; got != want {
		t.Fatalf("installed extension name = %q, want %q", got, want)
	}
	if extensions[1].Description == nil || *extensions[1].Description != "geographic objects" {
		t.Fatalf("installed extension description = %v, want %q", extensions[1].Description, "geographic objects")
	}
	if extensions[1].DefaultVersion == nil || *extensions[1].DefaultVersion != "3.4" {
		t.Fatalf("installed extension default version = %v, want %q", extensions[1].DefaultVersion, "3.4")
	}
	if !extensions[1].System {
		t.Fatal("installed extension system = false, want true")
	}
	if got, want := *extensions[1].Installed, (ManagedPostgresInstalledExtension{Version: "3.4.1", Schema: "public"}); got != want {
		t.Fatalf("installed extension = %#v, want %#v", got, want)
	}
	if extensions[2].Description != nil || extensions[2].DefaultVersion != nil {
		t.Fatalf("extensions[2] = %#v, want nil description and default version", extensions[2])
	}
}

func TestEnableManagedPostgresExtension(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	err := client.EnableManagedPostgresExtension(context.Background(), "mpg-123", "app", EnableManagedPostgresExtensionRequest{
		Name:         "pg_trgm",
		Schema:       "extensions",
		CreateSchema: true,
	})
	if err != nil {
		t.Fatalf("EnableManagedPostgresExtension() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodPost; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/databases/app/extensions"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresExtensionEnable; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
	var sent map[string]any
	if err := json.NewDecoder(transport.req.Body).Decode(&sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	want := map[string]any{"name": "pg_trgm", "schema": "extensions", "create_schema": true}
	if !maps.Equal(sent, want) {
		t.Fatalf("request body = %#v, want %#v", sent, want)
	}
}

func TestEnableManagedPostgresExtensionOmitsEmptySchemaAndSendsFalse(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	err := client.EnableManagedPostgresExtension(context.Background(), "mpg-123", "app", EnableManagedPostgresExtensionRequest{Name: "pg_trgm"})
	if err != nil {
		t.Fatalf("EnableManagedPostgresExtension() error = %v", err)
	}
	var sent map[string]any
	if err := json.NewDecoder(transport.req.Body).Decode(&sent); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	want := map[string]any{"name": "pg_trgm", "create_schema": false}
	if !maps.Equal(sent, want) {
		t.Fatalf("request body = %#v, want %#v", sent, want)
	}
}

func TestDisableManagedPostgresExtensionWithForce(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	if err := client.DisableManagedPostgresExtension(context.Background(), "mpg-123", "app", "pg_trgm", true); err != nil {
		t.Fatalf("DisableManagedPostgresExtension() error = %v", err)
	}
	if got, want := transport.req.Method, http.MethodDelete; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/databases/app/extensions/pg_trgm?force=true"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := actionFromContext(transport.req.Context()), managedPostgresExtensionDisable; got != want {
		t.Fatalf("request action = %q, want %q", got, want)
	}
}

func TestManagedPostgresExtensionsPreserveEscapedPaths(t *testing.T) {
	operations := []struct {
		name        string
		statusCode  int
		body        string
		expectedURI string
		run         func(*Client) error
	}{
		{name: "list", statusCode: http.StatusOK, body: `{"data":[]}`, expectedURI: "/v1/postgres/a%2F..%2Fb/databases/db%2F..%2Fname/extensions", run: func(client *Client) error {
			_, err := client.ListManagedPostgresExtensions(context.Background(), "a/../b", "db/../name")
			return err
		}},
		{name: "enable", statusCode: http.StatusNoContent, expectedURI: "/v1/postgres/a%2F..%2Fb/databases/db%2F..%2Fname/extensions", run: func(client *Client) error {
			return client.EnableManagedPostgresExtension(context.Background(), "a/../b", "db/../name", EnableManagedPostgresExtensionRequest{Name: "pg_trgm"})
		}},
		{name: "disable", statusCode: http.StatusNoContent, expectedURI: "/v1/postgres/a%2F..%2Fb/databases/db%2F..%2Fname/extensions/ext%2F..%2Fname", run: func(client *Client) error {
			return client.DisableManagedPostgresExtension(context.Background(), "a/../b", "db/../name", "ext/../name", false)
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: operation.statusCode, body: operation.body}
			if err := operation.run(newTestFlapsClient(t, transport)); err != nil {
				t.Fatalf("operation error = %v", err)
			}
			if got := transport.req.URL.RequestURI(); got != operation.expectedURI {
				t.Fatalf("request URI = %q, want %q", got, operation.expectedURI)
			}
		})
	}
}

func TestDisableManagedPostgresExtensionOmitsForceWhenFalse(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	if err := client.DisableManagedPostgresExtension(context.Background(), "mpg-123", "app", "pg_trgm", false); err != nil {
		t.Fatalf("DisableManagedPostgresExtension() error = %v", err)
	}
	if got, want := transport.req.URL.RequestURI(), "/v1/postgres/mpg-123/databases/app/extensions/pg_trgm"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
}

func TestManagedPostgresExtensionsClassifyNotFound(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Client) error
	}{
		{name: "list", run: func(client *Client) error {
			_, err := client.ListManagedPostgresExtensions(context.Background(), "mpg-123", "app")
			return err
		}},
		{name: "enable", run: func(client *Client) error {
			return client.EnableManagedPostgresExtension(context.Background(), "mpg-123", "app", EnableManagedPostgresExtensionRequest{Name: "pg_trgm"})
		}},
		{name: "disable", run: func(client *Client) error {
			return client.DisableManagedPostgresExtension(context.Background(), "mpg-123", "app", "pg_trgm", false)
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"Cluster not found"}`}
			err := operation.run(newTestFlapsClient(t, transport))
			if !errors.Is(err, ErrFlapsNotFound) {
				t.Fatalf("operation error = %v, want ErrFlapsNotFound", err)
			}
		})
	}
}

func TestManagedPostgresExtensionPreservesFlapsError(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusUnprocessableEntity,
		body:       `{"error":"create_schema must be true or false"}`,
	}
	client := newTestFlapsClient(t, transport)

	err := client.EnableManagedPostgresExtension(context.Background(), "mpg-123", "app", EnableManagedPostgresExtensionRequest{Name: "pg_trgm"})
	var flapsErr *FlapsError
	if !errors.As(err, &flapsErr) {
		t.Fatalf("EnableManagedPostgresExtension() error = %v, want FlapsError", err)
	}
	if got, want := flapsErr.ResponseStatusCode, http.StatusUnprocessableEntity; got != want {
		t.Fatalf("status code = %d, want %d", got, want)
	}
	if got, want := flapsErr.ResponseBodyString(), `{"error":"create_schema must be true or false"}`; got != want {
		t.Fatalf("response body = %q, want %q", got, want)
	}
}

func TestManagedPostgresExtensionActions(t *testing.T) {
	actions := map[flapsAction]string{
		managedPostgresExtensionList:    "managedPostgresExtensionList",
		managedPostgresExtensionEnable:  "managedPostgresExtensionEnable",
		managedPostgresExtensionDisable: "managedPostgresExtensionDisable",
	}
	for action, want := range actions {
		if got := action.String(); got != want {
			t.Errorf("action string = %q, want %q", got, want)
		}
	}
}

func TestCreateManagedPostgresAttachment(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusCreated,
		body:       `{"data":{"postgres_cluster_id":"mpg-123","app_name":"my-app","attached_at":"2026-08-31T12:00:00.000000Z"}}`,
	}
	client := newTestFlapsClient(t, transport)

	attachment, err := client.CreateManagedPostgresAttachment(context.Background(), "mpg-123", CreateManagedPostgresAttachmentRequest{
		AppName: "my-app",
	})
	if err != nil {
		t.Fatalf("CreateManagedPostgresAttachment() error = %v", err)
	}
	if transport.req.Method != http.MethodPost {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodPost)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/attachments"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
	body, err := io.ReadAll(transport.req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var sent CreateManagedPostgresAttachmentRequest
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode request body %q: %v", string(body), err)
	}
	if got, want := sent.AppName, "my-app"; got != want {
		t.Fatalf("request app_name = %q, want %q", got, want)
	}
	if got, want := attachment.PostgresClusterID, "mpg-123"; got != want {
		t.Fatalf("attachment.PostgresClusterID = %q, want %q", got, want)
	}
	if got, want := attachment.AppName, "my-app"; got != want {
		t.Fatalf("attachment.AppName = %q, want %q", got, want)
	}
	if got, want := attachment.AttachedAt, "2026-08-31T12:00:00.000000Z"; got != want {
		t.Fatalf("attachment.AttachedAt = %q, want %q", got, want)
	}
}

func TestCreateManagedPostgresAttachmentAccepts200(t *testing.T) {
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusOK,
		body:       `{"data":{"postgres_cluster_id":"mpg-123","app_name":"my-app","attached_at":"2026-08-31T12:00:00.000000Z"}}`,
	}
	client := newTestFlapsClient(t, transport)

	attachment, err := client.CreateManagedPostgresAttachment(context.Background(), "mpg-123", CreateManagedPostgresAttachmentRequest{
		AppName: "my-app",
	})
	if err != nil {
		t.Fatalf("CreateManagedPostgresAttachment() error = %v", err)
	}
	if got, want := attachment.PostgresClusterID, "mpg-123"; got != want {
		t.Fatalf("attachment.PostgresClusterID = %q, want %q", got, want)
	}
}

func TestDeleteManagedPostgresAttachment(t *testing.T) {
	transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
	client := newTestFlapsClient(t, transport)

	if err := client.DeleteManagedPostgresAttachment(context.Background(), "mpg-123", "my-app"); err != nil {
		t.Fatalf("DeleteManagedPostgresAttachment() error = %v", err)
	}
	if transport.req.Method != http.MethodDelete {
		t.Fatalf("request method = %q, want %q", transport.req.Method, http.MethodDelete)
	}
	if got, want := transport.req.URL.Path, "/v1/postgres/mpg-123/attachments/my-app"; got != want {
		t.Fatalf("request path = %q, want %q", got, want)
	}
}

func TestManagedPostgresAttachmentsPreserveEscapedPaths(t *testing.T) {
	deleteOperations := []struct {
		name         string
		clusterID    string
		appName      string
		expectedPath string
	}{
		{name: "slash", clusterID: "a/b", appName: "x/y", expectedPath: "/v1/postgres/a%2Fb/attachments/x%2Fy"},
		{name: "dot_segment", clusterID: "a/../b", appName: "x/../y", expectedPath: "/v1/postgres/a%2F..%2Fb/attachments/x%2F..%2Fy"},
	}

	for _, op := range deleteOperations {
		t.Run("delete/"+op.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: http.StatusNoContent}
			client := newTestFlapsClient(t, transport)
			if err := client.DeleteManagedPostgresAttachment(context.Background(), op.clusterID, op.appName); err != nil {
				t.Fatalf("request error = %v", err)
			}
			if got := transport.req.URL.RequestURI(); got != op.expectedPath {
				t.Fatalf("request URI = %q, want %q", got, op.expectedPath)
			}
		})
	}

	// Create operations don't include app_name in the URL path (it's in the request body),
	// but the cluster ID should still be properly escaped.
	createOperations := []struct {
		name         string
		clusterID    string
		expectedPath string
	}{
		{name: "slash", clusterID: "a/b", expectedPath: "/v1/postgres/a%2Fb/attachments"},
		{name: "dot_segment", clusterID: "a/../b", expectedPath: "/v1/postgres/a%2F..%2Fb/attachments"},
	}

	for _, op := range createOperations {
		t.Run("create/"+op.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{
				statusCode: http.StatusCreated,
				body:       `{"data":{"postgres_cluster_id":"a%2Fb","app_name":"x%2Fy","attached_at":"2026-08-31T12:00:00.000000Z"}}`,
			}
			client := newTestFlapsClient(t, transport)
			if _, err := client.CreateManagedPostgresAttachment(context.Background(), op.clusterID, CreateManagedPostgresAttachmentRequest{AppName: "x/y"}); err != nil {
				t.Fatalf("request error = %v", err)
			}
			if got := transport.req.URL.RequestURI(); got != op.expectedPath {
				t.Fatalf("request URI = %q, want %q", got, op.expectedPath)
			}
		})
	}
}

func TestManagedPostgresAttachmentsClassifyNotFound(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Client) error
	}{
		{name: "create", run: func(client *Client) error {
			_, err := client.CreateManagedPostgresAttachment(context.Background(), "mpg-123", CreateManagedPostgresAttachmentRequest{AppName: "my-app"})
			return err
		}},
		{name: "delete", run: func(client *Client) error {
			return client.DeleteManagedPostgresAttachment(context.Background(), "mpg-123", "my-app")
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: http.StatusNotFound, body: `{"error":"not found"}`}
			err := operation.run(newTestFlapsClient(t, transport))
			if !errors.Is(err, ErrFlapsNotFound) {
				t.Fatalf("request error = %v, want ErrFlapsNotFound", err)
			}
			var flapsErr *FlapsError
			if !errors.As(err, &flapsErr) {
				t.Fatalf("request error = %v, want wrapped FlapsError", err)
			}
		})
	}
}

func TestManagedPostgresAttachmentsClassifyGone(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Client) error
	}{
		{name: "create", run: func(client *Client) error {
			_, err := client.CreateManagedPostgresAttachment(context.Background(), "mpg-123", CreateManagedPostgresAttachmentRequest{AppName: "my-app"})
			return err
		}},
		{name: "delete", run: func(client *Client) error {
			return client.DeleteManagedPostgresAttachment(context.Background(), "mpg-123", "my-app")
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: http.StatusGone, body: `{"error":"gone"}`}
			err := operation.run(newTestFlapsClient(t, transport))
			if !errors.Is(err, ErrFlapsGone) {
				t.Fatalf("request error = %v, want ErrFlapsGone", err)
			}
			var flapsErr *FlapsError
			if !errors.As(err, &flapsErr) {
				t.Fatalf("request error = %v, want wrapped FlapsError", err)
			}
		})
	}
}

func TestManagedPostgresAttachmentsPreserveNonNotFoundErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		run        func(*Client) error
	}{
		{name: "create_forbidden", statusCode: http.StatusForbidden, run: func(client *Client) error {
			_, err := client.CreateManagedPostgresAttachment(context.Background(), "mpg-123", CreateManagedPostgresAttachmentRequest{AppName: "my-app"})
			return err
		}},
		{name: "delete_forbidden", statusCode: http.StatusForbidden, run: func(client *Client) error {
			return client.DeleteManagedPostgresAttachment(context.Background(), "mpg-123", "my-app")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &managedPostgresRoundTripper{statusCode: test.statusCode, body: `{"error":"request rejected"}`}
			err := test.run(newTestFlapsClient(t, transport))
			if errors.Is(err, ErrFlapsNotFound) {
				t.Fatalf("request error = %v, unexpectedly classified as not found", err)
			}
			var flapsErr *FlapsError
			if !errors.As(err, &flapsErr) {
				t.Fatalf("request error = %v, want FlapsError", err)
			}
			if got, want := flapsErr.ResponseStatusCode, test.statusCode; got != want {
				t.Fatalf("response status = %d, want %d", got, want)
			}
		})
	}
}

// nonSensitiveClusterBody is a GetManagedPostgresCluster response body reused
// across the logging and retry control tests below.
const nonSensitiveClusterBody = `{"data":{"id":"mpg-123","name":"MARKER_NOT_SENSITIVE","status":"ready","region":"iad","plan":"basic","disk_size_gb":0,"cpus":0,"cpu_kind":"","memory_mb":0,"replicas":0,"pg_major_version":"","postgis_enabled":false,"endpoints":{"primary":{"direct":{"host":"","port":0},"pooler":{"host":"","port":0}}},"organization":{"name":"","slug":""},"created_at":"","attached_apps":[]}}`

// TestManagedPostgresCredentialResponsesAreRedactedFromLogs proves the debug
// logger never records a plaintext password for either credential-returning
// method, while confirming an ordinary (non-sensitive) response is still
// logged normally.
func TestManagedPostgresCredentialResponsesAreRedactedFromLogs(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		secret    string
		sensitive bool
		// call returns the field of the decoded response that should equal
		// secret on success (password for the credential-returning methods,
		// name for the non-sensitive control), so a bug that drops or
		// corrupts the response is caught here too, not just in logging.
		call func(*Client) (string, error)
	}{
		{
			name:      "rotate password",
			body:      `{"data":{"username":"appuser","password":"sekret-XYZ"}}`,
			secret:    "sekret-XYZ",
			sensitive: true,
			call: func(c *Client) (string, error) {
				creds, err := c.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "appuser", RotateManagedPostgresUserPasswordRequest{})
				return creds.Password, err
			},
		},
		{
			name:      "get credentials",
			body:      `{"data":{"username":"appuser","password":"sekret-ABC"}}`,
			secret:    "sekret-ABC",
			sensitive: true,
			call: func(c *Client) (string, error) {
				creds, err := c.GetManagedPostgresUserCredentials(context.Background(), "mpg-123", "appuser")
				return creds.Password, err
			},
		},
		{
			name:      "get cluster (non-sensitive control)",
			body:      nonSensitiveClusterBody,
			secret:    "MARKER_NOT_SENSITIVE",
			sensitive: false,
			call: func(c *Client) (string, error) {
				cluster, err := c.GetManagedPostgresCluster(context.Background(), "mpg-123")
				return cluster.Name, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &fakeLogger{}
			opts := NewClientOpts{
				Transport: &managedPostgresRoundTripper{statusCode: http.StatusOK, body: tt.body},
				Logger:    logger,
			}
			t.Setenv("FLY_FLAPS_BASE_URL", "http://example.test")
			client, err := NewWithOptions(context.Background(), opts)
			if err != nil {
				t.Fatalf("NewWithOptions: %v", err)
			}

			got, err := tt.call(client)
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
			if got != tt.secret {
				t.Fatalf("decoded value = %q, want %q", got, tt.secret)
			}

			var secretLogged, redactedLogged bool
			for _, line := range logger.lines {
				if strings.Contains(line, tt.secret) {
					secretLogged = true
				}
				if strings.Contains(line, "redacted") {
					redactedLogged = true
				}
			}

			if tt.sensitive {
				if secretLogged {
					t.Fatalf("secret %q was logged, want redacted", tt.secret)
				}
				if !redactedLogged {
					t.Fatalf("expected redaction message in logs, got: %v", logger.lines)
				}
			} else if !secretLogged {
				t.Fatalf("non-sensitive marker %q not logged, got: %v", tt.secret, logger.lines)
			}
		})
	}
}

// TestManagedPostgresRotatePasswordErrorBodyIsNotRedacted proves that
// redaction only applies to successful (2xx) responses: an error body from a
// sensitive endpoint carries no credentials and should stay visible in logs
// so failures remain debuggable.
func TestManagedPostgresRotatePasswordErrorBodyIsNotRedacted(t *testing.T) {
	logger := &fakeLogger{}
	transport := &managedPostgresRoundTripper{
		statusCode: http.StatusUnprocessableEntity,
		body:       `{"error":"ROTATE_VALIDATION_MARKER"}`,
	}
	opts := NewClientOpts{
		Transport: transport,
		Logger:    logger,
	}
	t.Setenv("FLY_FLAPS_BASE_URL", "http://example.test")
	client, err := NewWithOptions(context.Background(), opts)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	_, err = client.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "appuser", RotateManagedPostgresUserPasswordRequest{})
	if err == nil {
		t.Fatalf("RotateManagedPostgresUserPassword() expected error, got nil")
	}

	var markerLogged, redactedLogged bool
	for _, line := range logger.lines {
		if strings.Contains(line, "ROTATE_VALIDATION_MARKER") {
			markerLogged = true
		}
		if strings.Contains(line, "redacted") {
			redactedLogged = true
		}
	}

	if !markerLogged {
		t.Fatalf("error body marker not logged (should be visible, not redacted), got: %v", logger.lines)
	}
	if redactedLogged {
		t.Fatalf("error body was redacted, want it visible: %v", logger.lines)
	}
}

// TestManagedPostgresRotatePasswordSkipsAutoRetry proves rotation opts out of
// the shared transport's 502/503 retry (so a flaky gateway can't rotate the
// password twice), while an ordinary GET still retries as before.
func TestManagedPostgresRotatePasswordSkipsAutoRetry(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		call          func(*Client) error
		wantErr       bool
		wantCallCount int
	}{
		{
			name: "rotate password does not retry",
			body: `{"data":{"username":"appuser","password":"new-pw"}}`,
			call: func(c *Client) error {
				_, err := c.RotateManagedPostgresUserPassword(context.Background(), "mpg-123", "appuser", RotateManagedPostgresUserPasswordRequest{})
				return err
			},
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name: "get cluster retries (control)",
			body: nonSensitiveClusterBody,
			call: func(c *Client) error {
				_, err := c.GetManagedPostgresCluster(context.Background(), "mpg-123")
				return err
			},
			wantErr:       false,
			wantCallCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &scriptedTripper{steps: []step{
				{statusCode: http.StatusBadGateway, body: `{"error":"bad gateway"}`},
				{statusCode: http.StatusOK, body: tt.body},
			}}
			t.Setenv("FLY_FLAPS_BASE_URL", "http://example.test")
			client, err := NewWithOptions(context.Background(), NewClientOpts{Transport: transport})
			if err != nil {
				t.Fatalf("NewWithOptions: %v", err)
			}

			err = tt.call(client)
			if tt.wantErr {
				var flapsErr *FlapsError
				if !errors.As(err, &flapsErr) {
					t.Fatalf("error = %v, want FlapsError", err)
				}
				if got, want := flapsErr.ResponseStatusCode, http.StatusBadGateway; got != want {
					t.Fatalf("response status = %d, want %d", got, want)
				}
			} else if err != nil {
				t.Fatalf("call error = %v", err)
			}

			if got, want := transport.callCount(), tt.wantCallCount; got != want {
				t.Fatalf("transport called %d times, want %d", got, want)
			}
		})
	}
}

func TestManagedPostgresAttachmentActions(t *testing.T) {
	actions := map[flapsAction]string{
		managedPostgresAttachmentCreate: "managedPostgresAttachmentCreate",
		managedPostgresAttachmentDelete: "managedPostgresAttachmentDelete",
	}
	for action, want := range actions {
		if got := action.String(); got != want {
			t.Errorf("action string = %q, want %q", got, want)
		}
	}
}
