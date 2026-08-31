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

// Statuses a Managed Postgres cluster reports while provisioning. The API
// documents "failed" as the terminal failure status; "error" is declared too
// because the service behind it reports that spelling for the same state.
const (
	ManagedPostgresStatusReady  = "ready"
	ManagedPostgresStatusFailed = "failed"
	ManagedPostgresStatusError  = "error"
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

// ManagedPostgresClusterSummary is the public projection returned by
// GET /v1/postgres.
type ManagedPostgresClusterSummary struct {
	ID           string                       `json:"id"`
	Name         string                       `json:"name"`
	Status       string                       `json:"status"`
	Region       string                       `json:"region"`
	Plan         string                       `json:"plan"`
	CreatedAt    string                       `json:"created_at"`
	DeletedAt    string                       `json:"deleted_at"`
	AttachedApps []ManagedPostgresAttachedApp `json:"attached_apps"`
}

// CreateManagedPostgresClusterRequest is the body for POST /v1/postgres.
type CreateManagedPostgresClusterRequest struct {
	OrgSlug        string `json:"org_slug"`
	Name           string `json:"name,omitempty"`
	Region         string `json:"region"`
	Plan           string `json:"plan"`
	DiskSizeGB     int    `json:"disk_size_gb,omitempty"`
	PGMajorVersion string `json:"pg_major_version,omitempty"`
	PostGISEnabled bool   `json:"postgis_enabled"`
}

// ListManagedPostgresClustersRequest contains the query parameters for
// GET /v1/postgres.
type ListManagedPostgresClustersRequest struct {
	OrgSlug        string
	IncludeDeleted bool
}

// ManagedPostgresUserCredentials is the response from
// GET /v1/postgres/:id/users/:username/credentials.
type ManagedPostgresUserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ManagedPostgresUserRole is a role assigned to a Managed Postgres user.
type ManagedPostgresUserRole string

// Roles the API accepts when creating or updating a Managed Postgres user.
const (
	ManagedPostgresUserRoleSchemaAdmin ManagedPostgresUserRole = "schema_admin"
	ManagedPostgresUserRoleWriter      ManagedPostgresUserRole = "writer"
	ManagedPostgresUserRoleReader      ManagedPostgresUserRole = "reader"
)

// ManagedPostgresUser is the public projection of a user within a Managed
// Postgres cluster.
type ManagedPostgresUser struct {
	Username string                  `json:"username"`
	Role     ManagedPostgresUserRole `json:"role"`
}

// CreateManagedPostgresUserRequest is the body for
// POST /v1/postgres/:id/users.
type CreateManagedPostgresUserRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// UpdateManagedPostgresUserRoleRequest is the body for
// PATCH /v1/postgres/:id/users/:username.
type UpdateManagedPostgresUserRoleRequest struct {
	Role string `json:"role"`
}

// ManagedPostgresDatabase is the public projection of a database within a
// Managed Postgres cluster returned by list/create.
type ManagedPostgresDatabase struct {
	Name string `json:"name"`
}

// CreateManagedPostgresDatabaseRequest is the body for
// POST /v1/postgres/:id/databases.
type CreateManagedPostgresDatabaseRequest struct {
	Name string `json:"name"`
}

// ManagedPostgresBackup is the public projection of a backup belonging to a
// Managed Postgres cluster.
type ManagedPostgresBackup struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	SizeBytes  int64  `json:"size_bytes"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
}

// CreateManagedPostgresBackupRequest is the body for
// POST /v1/postgres/:id/backups.
type CreateManagedPostgresBackupRequest struct {
	Type string `json:"type"`
}

// RestoreManagedPostgresClusterRequest is the body for
// POST /v1/postgres/:id/restore. Exactly one of BackupID or PITRTime is
// required by the API; Name is optional.
type RestoreManagedPostgresClusterRequest struct {
	BackupID string `json:"backup_id,omitempty"`
	PITRTime string `json:"pitr_time,omitempty"`
	Name     string `json:"name,omitempty"`
}

// ManagedPostgresInstalledExtension describes an installed extension version
// and schema.
type ManagedPostgresInstalledExtension struct {
	Version string `json:"version"`
	Schema  string `json:"schema"`
}

// ManagedPostgresExtension is the public projection of an extension available
// to a database.
type ManagedPostgresExtension struct {
	Name           string                             `json:"name"`
	Description    *string                            `json:"description"`
	DefaultVersion *string                            `json:"default_version"`
	System         bool                               `json:"system"`
	Installed      *ManagedPostgresInstalledExtension `json:"installed"`
}

// EnableManagedPostgresExtensionRequest is the body for
// POST /v1/postgres/:id/databases/:database/extensions.
type EnableManagedPostgresExtensionRequest struct {
	Name         string `json:"name"`
	Schema       string `json:"schema,omitempty"`
	CreateSchema bool   `json:"create_schema"`
}

type managedPostgresClusterEnvelope struct {
	Data ManagedPostgresCluster `json:"data"`
}

type managedPostgresClusterListEnvelope struct {
	Data []ManagedPostgresClusterSummary `json:"data"`
}

type managedPostgresUserCredentialsEnvelope struct {
	Data ManagedPostgresUserCredentials `json:"data"`
}

type managedPostgresUserEnvelope struct {
	Data ManagedPostgresUser `json:"data"`
}

type managedPostgresUserListEnvelope struct {
	Data []ManagedPostgresUser `json:"data"`
}

type managedPostgresDatabaseEnvelope struct {
	Data ManagedPostgresDatabase `json:"data"`
}

type managedPostgresDatabaseListEnvelope struct {
	Data []ManagedPostgresDatabase `json:"data"`
}

type managedPostgresBackupListEnvelope struct {
	Data []ManagedPostgresBackup `json:"data"`
}

type managedPostgresExtensionListEnvelope struct {
	Data []ManagedPostgresExtension `json:"data"`
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

func (f *Client) ListManagedPostgresClusters(ctx context.Context, req ListManagedPostgresClustersRequest) ([]ManagedPostgresClusterSummary, error) {
	ctx = contextWithAction(ctx, managedPostgresList)

	query := url.Values{}
	query.Set("org_slug", req.OrgSlug)
	if req.IncludeDeleted {
		query.Set("include_deleted", "true")
	}

	var env managedPostgresClusterListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, "/postgres?"+query.Encode(), nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres clusters: %w", err)
	}

	return env.Data, nil
}

func (f *Client) DeleteManagedPostgresCluster(ctx context.Context, id string) error {
	ctx = contextWithAction(ctx, managedPostgresDelete)

	endpoint := fmt.Sprintf("/postgres/%s", url.PathEscape(id))
	if err := f._sendRequest(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete Managed Postgres cluster: %w", err)
	}

	return nil
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

func (f *Client) ListManagedPostgresUsers(ctx context.Context, id string) ([]ManagedPostgresUser, error) {
	ctx = contextWithAction(ctx, managedPostgresUserList)

	endpoint := fmt.Sprintf("/postgres/%s/users", url.PathEscape(id))

	var env managedPostgresUserListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres users: %w", err)
	}

	return env.Data, nil
}

func (f *Client) CreateManagedPostgresUser(ctx context.Context, id string, req CreateManagedPostgresUserRequest) (ManagedPostgresUser, error) {
	ctx = contextWithAction(ctx, managedPostgresUserCreate)

	endpoint := fmt.Sprintf("/postgres/%s/users", url.PathEscape(id))

	var env managedPostgresUserEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresUser{}, fmt.Errorf("failed to create Managed Postgres user: %w", err)
	}

	return env.Data, nil
}

func (f *Client) UpdateManagedPostgresUserRole(ctx context.Context, id, username string, req UpdateManagedPostgresUserRoleRequest) error {
	ctx = contextWithAction(ctx, managedPostgresUserUpdate)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s", url.PathEscape(id), url.PathEscape(username))
	if err := f._sendRequest(ctx, http.MethodPatch, endpoint, req, nil, nil); err != nil {
		return fmt.Errorf("failed to update Managed Postgres user role: %w", err)
	}

	return nil
}

func (f *Client) DeleteManagedPostgresUser(ctx context.Context, id, username string) error {
	ctx = contextWithAction(ctx, managedPostgresUserDelete)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s", url.PathEscape(id), url.PathEscape(username))
	if err := f._sendRequest(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete Managed Postgres user: %w", err)
	}

	return nil
}

func (f *Client) ListManagedPostgresDatabases(ctx context.Context, id string) ([]ManagedPostgresDatabase, error) {
	ctx = contextWithAction(ctx, managedPostgresDatabaseList)

	endpoint := fmt.Sprintf("/postgres/%s/databases", url.PathEscape(id))

	var env managedPostgresDatabaseListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres databases: %w", err)
	}

	return env.Data, nil
}

func (f *Client) CreateManagedPostgresDatabase(ctx context.Context, id string, req CreateManagedPostgresDatabaseRequest) (ManagedPostgresDatabase, error) {
	ctx = contextWithAction(ctx, managedPostgresDatabaseCreate)

	endpoint := fmt.Sprintf("/postgres/%s/databases", url.PathEscape(id))

	var env managedPostgresDatabaseEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresDatabase{}, fmt.Errorf("failed to create Managed Postgres database: %w", err)
	}

	return env.Data, nil
}

func (f *Client) ListManagedPostgresBackups(ctx context.Context, id string) ([]ManagedPostgresBackup, error) {
	ctx = contextWithAction(ctx, managedPostgresBackupList)

	endpoint := fmt.Sprintf("/postgres/%s/backups", url.PathEscape(id))

	var env managedPostgresBackupListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres backups: %w", err)
	}

	return env.Data, nil
}

func (f *Client) CreateManagedPostgresBackup(ctx context.Context, id string, req CreateManagedPostgresBackupRequest) error {
	ctx = contextWithAction(ctx, managedPostgresBackupCreate)

	endpoint := fmt.Sprintf("/postgres/%s/backups", url.PathEscape(id))
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, nil, nil); err != nil {
		return fmt.Errorf("failed to create Managed Postgres backup: %w", err)
	}

	return nil
}

func (f *Client) RestoreManagedPostgresCluster(ctx context.Context, id string, req RestoreManagedPostgresClusterRequest) (ManagedPostgresCluster, error) {
	ctx = contextWithAction(ctx, managedPostgresRestore)

	endpoint := fmt.Sprintf("/postgres/%s/restore", url.PathEscape(id))

	var env managedPostgresClusterEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresCluster{}, fmt.Errorf("failed to restore Managed Postgres cluster: %w", err)
	}

	return env.Data, nil
}

func (f *Client) ListManagedPostgresExtensions(ctx context.Context, id, database string) ([]ManagedPostgresExtension, error) {
	ctx = contextWithAction(ctx, managedPostgresExtensionList)

	endpoint := fmt.Sprintf("/postgres/%s/databases/%s/extensions", url.PathEscape(id), url.PathEscape(database))

	var env managedPostgresExtensionListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres extensions: %w", err)
	}

	return env.Data, nil
}

func (f *Client) EnableManagedPostgresExtension(ctx context.Context, id, database string, req EnableManagedPostgresExtensionRequest) error {
	ctx = contextWithAction(ctx, managedPostgresExtensionEnable)

	endpoint := fmt.Sprintf("/postgres/%s/databases/%s/extensions", url.PathEscape(id), url.PathEscape(database))
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, nil, nil); err != nil {
		return fmt.Errorf("failed to enable Managed Postgres extension: %w", err)
	}

	return nil
}

func (f *Client) DisableManagedPostgresExtension(ctx context.Context, id, database, name string, force bool) error {
	ctx = contextWithAction(ctx, managedPostgresExtensionDisable)

	endpoint := fmt.Sprintf("/postgres/%s/databases/%s/extensions/%s", url.PathEscape(id), url.PathEscape(database), url.PathEscape(name))
	if force {
		endpoint += "?force=true"
	}
	if err := f._sendRequest(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to disable Managed Postgres extension: %w", err)
	}

	return nil
}
