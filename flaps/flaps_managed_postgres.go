package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type ManagedPostgresEndpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ManagedPostgresEndpoints struct {
	Primary struct {
		Direct ManagedPostgresEndpoint `json:"direct"`
		Pooler ManagedPostgresEndpoint `json:"pooler"`
	} `json:"primary"`
}

type ManagedPostgresOrganization struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ManagedPostgresAttachedApp struct {
	Name string `json:"name"`
}

// Statuses a Managed Postgres cluster reports while provisioning.
const (
	ManagedPostgresStatusReady = "ready"
	ManagedPostgresStatusError = "error"
)

// ManagedPostgresCluster is the public projection of a Managed Postgres
// cluster returned by create/show.
type ManagedPostgresCluster struct {
	ID             string                       `json:"id"`
	Name           string                       `json:"name"`
	Status         string                       `json:"status"`
	Region         string                       `json:"region"`
	Plan           string                       `json:"plan"`
	DiskSizeGB     int                          `json:"disk_size_gb"`
	CPUs           int                          `json:"cpus"`
	CPUKind        string                       `json:"cpu_kind"`
	MemoryMB       int                          `json:"memory_mb"`
	Replicas       int                          `json:"replicas"`
	PGMajorVersion string                       `json:"pg_major_version"`
	PostGISEnabled bool                         `json:"postgis_enabled"`
	Endpoints      ManagedPostgresEndpoints     `json:"endpoints"`
	Organization   ManagedPostgresOrganization  `json:"organization"`
	CreatedAt      string                       `json:"created_at"`
	AttachedApps   []ManagedPostgresAttachedApp `json:"attached_apps"`
}

// CreateManagedPostgresClusterRequest is the body for POST /v1/postgres.
type CreateManagedPostgresClusterRequest struct {
	OrgSlug        string `json:"org_slug"`
	Name           string `json:"name"`
	Region         string `json:"region"`
	Plan           string `json:"plan"`
	DiskSizeGB     int    `json:"disk_size_gb"`
	PGMajorVersion string `json:"pg_major_version"`
	PostGISEnabled bool   `json:"postgis_enabled"`
}

// ManagedPostgresUserCredentials is the response from
// GET /v1/postgres/:id/users/:username/credentials.
type ManagedPostgresUserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type managedPostgresClusterEnvelope struct {
	Data ManagedPostgresCluster `json:"data"`
}

type managedPostgresUserCredentialsEnvelope struct {
	Data ManagedPostgresUserCredentials `json:"data"`
}

func (f *Client) CreateManagedPostgresCluster(ctx context.Context, req CreateManagedPostgresClusterRequest) (ManagedPostgresCluster, error) {
	ctx = contextWithAction(ctx, managedPostgresCreate)

	var env managedPostgresClusterEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, "/postgres", req, &env, nil); err != nil {
		return ManagedPostgresCluster{}, fmt.Errorf("failed to create Managed Postgres cluster: %w", err)
	}

	return env.Data, nil
}

func (f *Client) GetManagedPostgresCluster(ctx context.Context, id string) (ManagedPostgresCluster, error) {
	ctx = contextWithAction(ctx, managedPostgresGet)

	endpoint := fmt.Sprintf("/postgres/%s", url.PathEscape(id))

	var env managedPostgresClusterEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return ManagedPostgresCluster{}, fmt.Errorf("failed to get Managed Postgres cluster: %w", err)
	}

	return env.Data, nil
}

func (f *Client) GetManagedPostgresUserCredentials(ctx context.Context, id, username string) (ManagedPostgresUserCredentials, error) {
	ctx = contextWithAction(ctx, managedPostgresUserCredentialsGet)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s/credentials", url.PathEscape(id), url.PathEscape(username))

	var env managedPostgresUserCredentialsEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return ManagedPostgresUserCredentials{}, fmt.Errorf("failed to get Managed Postgres user credentials: %w", err)
	}

	return env.Data, nil
}
