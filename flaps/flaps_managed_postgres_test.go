package flaps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateManagedPostgresCluster(t *testing.T) {
	var got CreateManagedPostgresClusterRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/v1/postgres" {
			t.Errorf("path = %s, want /v1/postgres", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decoding request body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data":{"id":"abc123","name":"my-db","status":"creating"}}`)
	}))
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	want := CreateManagedPostgresClusterRequest{
		OrgSlug:        "my-org",
		Name:           "my-db",
		Region:         "iad",
		Plan:           "basic",
		DiskSizeGB:     10,
		PGMajorVersion: "16",
	}

	cluster, err := client.CreateManagedPostgresCluster(context.Background(), want)
	if err != nil {
		t.Fatalf("CreateManagedPostgresCluster() error = %v", err)
	}

	if got != want {
		t.Fatalf("request body = %+v, want %+v", got, want)
	}
	if cluster.ID != "abc123" {
		t.Fatalf("cluster id = %s, want abc123", cluster.ID)
	}
	if cluster.Status != "creating" {
		t.Fatalf("cluster status = %s, want creating", cluster.Status)
	}
}

func TestGetManagedPostgresCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/v1/postgres/abc123" {
			t.Errorf("path = %s, want /v1/postgres/abc123", r.URL.Path)
		}

		fmt.Fprint(w, `{"data":{"id":"abc123","status":"ready","endpoints":{"primary":{"direct":{"host":"direct.test","port":5432},"pooler":{"host":"pooler.test","port":5433}}}}}`)
	}))
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	cluster, err := client.GetManagedPostgresCluster(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetManagedPostgresCluster() error = %v", err)
	}

	if cluster.Status != ManagedPostgresStatusReady {
		t.Fatalf("cluster status = %s, want %s", cluster.Status, ManagedPostgresStatusReady)
	}

	pooler := cluster.Endpoints.Primary.Pooler
	if pooler.Host != "pooler.test" {
		t.Fatalf("pooler host = %s, want pooler.test", pooler.Host)
	}
	if pooler.Port != 5433 {
		t.Fatalf("pooler port = %d, want 5433", pooler.Port)
	}
}

func TestGetManagedPostgresUserCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if want := "/v1/postgres/abc123/users/fly-user/credentials"; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}

		fmt.Fprint(w, `{"data":{"username":"fly-user","password":"s3cret"}}`)
	}))
	defer server.Close()

	t.Setenv("FLY_FLAPS_BASE_URL", server.URL)

	client, err := NewWithOptions(context.Background(), NewClientOpts{})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}

	creds, err := client.GetManagedPostgresUserCredentials(context.Background(), "abc123", "fly-user")
	if err != nil {
		t.Fatalf("GetManagedPostgresUserCredentials() error = %v", err)
	}

	if creds.Username != "fly-user" {
		t.Fatalf("username = %s, want fly-user", creds.Username)
	}
	if creds.Password != "s3cret" {
		t.Fatalf("password = %s, want s3cret", creds.Password)
	}
}
