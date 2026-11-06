package flaps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	fly "github.com/superfly/fly-go"
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
	ID                      string                       `json:"id"`
	Name                    string                       `json:"name"`
	Status                  string                       `json:"status"`
	Region                  string                       `json:"region"`
	Plan                    string                       `json:"plan"`
	DiskSizeGB              int                          `json:"disk_size_gb"`
	StorageUsedBytes        *int64                       `json:"storage_used_bytes"`
	StorageProvisionedBytes *int64                       `json:"storage_provisioned_bytes"`
	CPUs                    int                          `json:"cpus"`
	CPUKind                 string                       `json:"cpu_kind"`
	MemoryMB                int                          `json:"memory_mb"`
	Replicas                int                          `json:"replicas"`
	PGMajorVersion          string                       `json:"pg_major_version"`
	PostGISEnabled          bool                         `json:"postgis_enabled"`
	Endpoints               ManagedPostgresEndpoints     `json:"endpoints"`
	Organization            ManagedPostgresOrganization  `json:"organization"`
	CreatedAt               string                       `json:"created_at"`
	AttachedApps            []ManagedPostgresAttachedApp `json:"attached_apps"`
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
	// ManagedPostgresUserRoleCustom grants nothing until an admin grants it
	// with SQL. It is the only role that can be limited to databases.
	ManagedPostgresUserRoleCustom ManagedPostgresUserRole = "custom"
)

// ManagedPostgresUser is the public projection of a user within a Managed
// Postgres cluster.
type ManagedPostgresUser struct {
	Username string                  `json:"username"`
	Role     ManagedPostgresUserRole `json:"role"`
	// Roles lists the customer roles the user is a direct member of, sorted
	// by name in byte order. It is nil when the cluster doesn't report them
	// yet, and empty when the user is in none.
	Roles []string `json:"roles"`
	// DatabasesLimited is false when the user can reach every database,
	// including ones created later. Clusters that can't limit users report
	// false.
	DatabasesLimited bool `json:"databases_limited"`
	// Databases lists the databases a limited user can reach. It is empty
	// when DatabasesLimited is false.
	Databases []string `json:"databases"`
}

// CreateManagedPostgresUserRequest is the body for
// POST /v1/postgres/:id/users.
type CreateManagedPostgresUserRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	// Databases limits a custom user to these databases. Leave it empty to
	// give the user every database, including ones created later. An empty
	// list is never sent.
	Databases []string `json:"databases,omitempty"`
}

// UpdateManagedPostgresUserRoleRequest is the body for
// PATCH /v1/postgres/:id/users/:username.
type UpdateManagedPostgresUserRoleRequest struct {
	Role string `json:"role"`
}

// RotateManagedPostgresUserPasswordRequest is the body for
// POST /v1/postgres/:id/users/:username/rotate_password.
type RotateManagedPostgresUserPasswordRequest struct {
	KillSessions bool `json:"kill_sessions,omitempty"`
}

// SetManagedPostgresUserDatabasesRequest is the body for
// PUT /v1/postgres/:id/users/:username/databases. It replaces the user's
// whole database list.
type SetManagedPostgresUserDatabasesRequest struct {
	// DatabasesLimited false gives the user every database, including ones
	// created later. Databases must then be empty.
	DatabasesLimited bool `json:"databases_limited"`
	// Databases is the full list a limited user can reach. It is required
	// when DatabasesLimited is true. An empty list is never sent.
	Databases []string `json:"databases,omitempty"`
}

// ManagedPostgresRole is a customer role: one made with SQL, not one of the
// built-in roles.
type ManagedPostgresRole struct {
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
}

// ManagedPostgresRoleMember is a direct member of a customer role: a user or
// another role.
type ManagedPostgresRoleMember struct {
	Name string `json:"name"`
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

// ForkManagedPostgresClusterRequest is the body for
// POST /v1/postgres/:id/fork. Both fields are optional; empty values are
// omitted so the API applies its defaults.
type ForkManagedPostgresClusterRequest struct {
	// Name for the fork. Defaults to the source's name with a -fork suffix.
	Name string `json:"name,omitempty"`
	// Plan for the fork. Defaults to the source cluster's plan.
	Plan string `json:"plan,omitempty"`
}

// ManagedPostgresPlan is the public projection of a Managed Postgres plan
// returned by GET /v1/postgres/plans.
type ManagedPostgresPlan struct {
	// Plan is the slug to pass as plan on create/fork, e.g. "basic". Slugs
	// are not uniformly lowercase ("Performance" is capitalized).
	Plan string `json:"plan"`
	// CPUKind is "shared" or "performance".
	CPUKind         string `json:"cpu_kind"`
	CPUs            int    `json:"cpus"`
	MemoryMB        int    `json:"memory_mb"`
	MonthlyPriceUSD int    `json:"monthly_price_usd"`
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

// ManagedPostgresAttachment is the public projection of an attachment between a
// Managed Postgres cluster and an app.
type ManagedPostgresAttachment struct {
	PostgresClusterID string `json:"postgres_cluster_id"`
	AppName           string `json:"app_name"`
	AttachedAt        string `json:"attached_at"`
}

// CreateManagedPostgresAttachmentRequest is the body for
// POST /v1/postgres/:id/attachments.
type CreateManagedPostgresAttachmentRequest struct {
	AppName string `json:"app_name"`
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

type managedPostgresPlanListEnvelope struct {
	Data []ManagedPostgresPlan `json:"data"`
}

// managedPostgresPageEnvelope is one page of a paged Managed Postgres list.
// NextCursor is missing or empty on the last page.
type managedPostgresPageEnvelope[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor"`
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

// ListManagedPostgresPlans returns the Managed Postgres plans available to an
// organization. orgSlug may be "personal".
func (f *Client) ListManagedPostgresPlans(ctx context.Context, orgSlug string) ([]ManagedPostgresPlan, error) {
	ctx = contextWithAction(ctx, managedPostgresPlanList)

	query := url.Values{}
	query.Set("org_slug", orgSlug)

	var env managedPostgresPlanListEnvelope
	if err := f._sendRequest(ctx, http.MethodGet, "/postgres/plans?"+query.Encode(), nil, &env, nil); err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres plans: %w", err)
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
	ctx = fly.WithSensitiveResponseBody(ctx)

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

func (f *Client) RotateManagedPostgresUserPassword(ctx context.Context, id, username string, req RotateManagedPostgresUserPasswordRequest) (ManagedPostgresUserCredentials, error) {
	ctx = contextWithAction(ctx, managedPostgresUserRotatePassword)
	ctx = fly.WithSensitiveResponseBody(ctx)
	ctx = fly.WithoutHTTPRetries(ctx)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s/rotate_password", url.PathEscape(id), url.PathEscape(username))

	var env managedPostgresUserCredentialsEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresUserCredentials{}, fmt.Errorf("failed to rotate Managed Postgres user password: %w", err)
	}

	return env.Data, nil
}

func (f *Client) DeleteManagedPostgresUser(ctx context.Context, id, username string) error {
	ctx = contextWithAction(ctx, managedPostgresUserDelete)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s", url.PathEscape(id), url.PathEscape(username))
	if err := f._sendRequest(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete Managed Postgres user: %w", err)
	}

	return nil
}

// ListManagedPostgresRoles returns the cluster's customer roles, sorted by
// name in byte order. It reads every page. A cluster too old to report roles
// gives an error that matches ErrFlapsNotImplemented.
func (f *Client) ListManagedPostgresRoles(ctx context.Context, id string) ([]ManagedPostgresRole, error) {
	ctx = contextWithAction(ctx, managedPostgresRoleList)

	endpoint := fmt.Sprintf("/postgres/%s/roles", url.PathEscape(id))

	roles, err := listManagedPostgresPages[ManagedPostgresRole](ctx, f, endpoint, url.Values{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres roles: %w", err)
	}

	return roles, nil
}

// ListManagedPostgresRoleMembers returns the direct members of a customer
// role, sorted by name in byte order. It reads every page. The role goes in
// the query, so any name works. A missing role matches ErrFlapsNotFound, and
// a cluster too old to report roles matches ErrFlapsNotImplemented.
func (f *Client) ListManagedPostgresRoleMembers(ctx context.Context, id, role string) ([]ManagedPostgresRoleMember, error) {
	ctx = contextWithAction(ctx, managedPostgresRoleMemberList)

	endpoint := fmt.Sprintf("/postgres/%s/roles/members", url.PathEscape(id))

	query := url.Values{}
	query.Set("role", role)

	members, err := listManagedPostgresPages[ManagedPostgresRoleMember](ctx, f, endpoint, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list Managed Postgres role members: %w", err)
	}

	return members, nil
}

// SetManagedPostgresUserDatabases replaces the databases a custom user can
// reach. Limiting a user ends its sessions in the databases it loses.
//
// The change is applied before those sessions are ended. If ending them
// fails, the call returns an error, usually with status 500, although the
// new limits are in place. The call is a full replace, so retrying it with
// the same request is safe. A cluster too old for database limits gives an
// error that matches ErrFlapsNotImplemented.
func (f *Client) SetManagedPostgresUserDatabases(ctx context.Context, id, username string, req SetManagedPostgresUserDatabasesRequest) error {
	ctx = contextWithAction(ctx, managedPostgresUserDatabasesSet)

	endpoint := fmt.Sprintf("/postgres/%s/users/%s/databases", url.PathEscape(id), url.PathEscape(username))
	if err := f._sendRequest(ctx, http.MethodPut, endpoint, req, nil, nil); err != nil {
		return fmt.Errorf("failed to set Managed Postgres user databases: %w", err)
	}

	return nil
}

// managedPostgresPageSize is the largest page the API serves.
const managedPostgresPageSize = 1_000

// listManagedPostgresPages reads every page of a paged Managed Postgres list.
// query holds the list's own parameters; limit and cursor are added here.
// Only a failed page is retried, and any page that still fails fails the
// whole call.
func listManagedPostgresPages[T any](ctx context.Context, f *Client, endpoint string, query url.Values) ([]T, error) {
	var items []T
	var cursor string
	for {
		query.Set("limit", strconv.Itoa(managedPostgresPageSize))
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		var page managedPostgresPageEnvelope[T]
		err := retryListAppsPage(ctx, func() error {
			page = managedPostgresPageEnvelope[T]{}
			return f._sendRequest(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil, &page, nil)
		})
		if err != nil {
			return nil, err
		}

		items = append(items, page.Data...)
		if page.NextCursor == "" {
			return items, nil
		}
		if page.NextCursor == cursor {
			return nil, errors.New("server returned the same cursor again")
		}
		cursor = page.NextCursor
	}
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

// ForkManagedPostgresCluster creates a new cluster from the current data of
// the source cluster identified by id. The fork is provisioned asynchronously;
// the returned cluster reflects its initial state.
func (f *Client) ForkManagedPostgresCluster(ctx context.Context, id string, req ForkManagedPostgresClusterRequest) (ManagedPostgresCluster, error) {
	ctx = contextWithAction(ctx, managedPostgresFork)

	endpoint := fmt.Sprintf("/postgres/%s/fork", url.PathEscape(id))

	var env managedPostgresClusterEnvelope
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresCluster{}, fmt.Errorf("failed to fork Managed Postgres cluster: %w", err)
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

type managedPostgresAttachmentEnvelope struct {
	Data ManagedPostgresAttachment `json:"data"`
}

// CreateManagedPostgresAttachment attaches an app to a Managed Postgres cluster.
// The server returns 201 if the attachment was created, or 200 if it already
// existed (idempotent find-or-create semantics). Both are treated as success.
// The cluster ID is escaped as URL path segments.
func (f *Client) CreateManagedPostgresAttachment(ctx context.Context, id string, req CreateManagedPostgresAttachmentRequest) (ManagedPostgresAttachment, error) {
	ctx = contextWithAction(ctx, managedPostgresAttachmentCreate)

	endpoint := fmt.Sprintf("/postgres/%s/attachments", url.PathEscape(id))

	var env managedPostgresAttachmentEnvelope
	// Both 200 and 201 are treated as success; the Machines API proxy preserves
	// ui-ex's find-or-create status distinction.
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, req, &env, nil); err != nil {
		return ManagedPostgresAttachment{}, fmt.Errorf("failed to create Managed Postgres attachment: %w", err)
	}

	return env.Data, nil
}

// DeleteManagedPostgresAttachment removes an app's attachment to a Managed Postgres
// cluster. Returns no content on success. Both the cluster ID and app name are
// escaped as URL path segments.
func (f *Client) DeleteManagedPostgresAttachment(ctx context.Context, id, appName string) error {
	ctx = contextWithAction(ctx, managedPostgresAttachmentDelete)

	endpoint := fmt.Sprintf("/postgres/%s/attachments/%s", url.PathEscape(id), url.PathEscape(appName))
	if err := f._sendRequest(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete Managed Postgres attachment: %w", err)
	}

	return nil
}
