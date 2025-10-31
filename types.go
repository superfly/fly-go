package fly

import (
	"fmt"
	"time"
)

// Query - Master query which encapsulates all possible returned structures
type Query struct {
	Errors Errors

	Apps struct {
		PageInfo struct {
			HasNextPage bool
			EndCursor   string
		}
		Nodes []App
	}
	App             App
	AppCompact      AppCompact
	AppBasic        AppBasic
	AppCertsCompact AppCertsCompact
	Viewer          User
	GqlMachine      GqlMachine
	Organizations   struct {
		Nodes []Organization
	}

	Organization        *Organization
	OrganizationDetails OrganizationDetails
	Volume              struct {
		App struct {
			Name string
		}
		State     string
		Snapshots struct {
			Nodes []VolumeSnapshotGql
		}
	}
	Domain *Domain

	Node  interface{}
	Nodes []interface{}

	Platform struct {
		RequestRegion string
		Regions       []Region
	}

	NearestRegion *Region

	LatestImageTag     string
	LatestImageDetails ImageVersion
	AppHostIssues      AppHostIssues
	// aliases & nodes

	// mutations
	CreateApp struct {
		App App
	}

	SetSecrets struct {
		Release Release
	}

	UnsetSecrets struct {
		Release Release
	}

	EnsureRemoteBuilder *struct {
		App     *App
		URL     string
		Release Release
	}

	EnsureMachineRemoteBuilder *struct {
		App     *App
		Machine *GqlMachine
	}

	CreateDoctorUrl SignedUrl

	AddCertificate struct {
		Certificate *AppCertificate
		Check       *HostnameCheck
	}

	DeleteCertificate DeleteCertificatePayload

	CheckCertificate struct {
		App         *App
		Certificate *AppCertificate
		Check       *HostnameCheck
	}

	AllocateIPAddress struct {
		App       App
		IPAddress IPAddress
	}
	AllocateEgressIPAddress struct {
		V4 string
		V6 string
	}
	ReleaseEgressIPAddress struct {
		V4 string
		V6 string
	}
	ReleaseIPAddress struct {
		App App
	}

	CreateDomain struct {
		Domain *Domain
	}
	CreateAndRegisterDomain struct {
		Domain *Domain
	}

	CheckDomain *CheckDomainResult

	ExportDnsZone struct {
		Contents string
	}

	ImportDnsZone struct {
		Warnings []ImportDnsWarning
		Changes  []ImportDnsChange
	}
	CreateOrganization CreateOrganizationPayload
	DeleteOrganization DeleteOrganizationPayload

	AddWireGuardPeer              CreatedWireGuardPeer
	EstablishSSHKey               SSHCertificate
	IssueCertificate              IssuedCertificate
	CreateDelegatedWireGuardToken DelegatedWireGuardToken
	DeleteDelegatedWireGuardToken DelegatedWireGuardToken

	RemoveWireGuardPeer struct {
		Organization Organization
	}

	AttachPostgresCluster *AttachPostgresClusterPayload
	EnablePostgresConsul  *PostgresEnableConsulPayload

	CreateOrganizationInvitation CreateOrganizationInvitation

	ValidateWireGuardPeers struct {
		InvalidPeerIPs []string
	}

	PostgresAttachments struct {
		Nodes []*PostgresClusterAttachment
	}

	DeleteOrganizationMembership *DeleteOrganizationMembershipPayload

	UpdateRemoteBuilder struct {
		Organization Organization
	}

	CanPerformBluegreenDeployment bool
	AppNameAvailable              bool

	LockApp *LockApp
}

type LockApp struct {
	LockID     string `json:"lockId"`
	Expiration time.Time
}

type LockAppInput struct {
	AppID  string `json:"app_id"`
	LockID string `json:"lock_id"`
}

type UnlockAppInput struct {
	AppID  string `json:"app_id"`
	LockID string `json:"lock_id"`
}

type CreatedWireGuardPeer struct {
	Peerip     string `json:"peerip"`
	Endpointip string `json:"endpointip"`
	Pubkey     string `json:"pubkey"`
}

type DeleteOrganizationMembershipPayload struct {
	Organization *Organization
	User         *User
}

type DelegatedWireGuardToken struct {
	Token string
}

type DelegatedWireGuardTokenHandle /* whatever */ struct {
	Name string
}

type SSHCertificate struct {
	Certificate string
}

type IssuedCertificate struct {
	Certificate string
	Key         string
}

type Definition map[string]interface{}

func DefinitionPtr(in map[string]interface{}) *Definition {
	if len(in) > 0 {
		return Pointer(Definition(in))
	}
	return nil
}

type ImageVersion struct {
	Registry   string
	Repository string
	Tag        string
	Version    string
	Digest     string
}

func (img *ImageVersion) FullImageRef() string {
	imgStr := fmt.Sprintf("%s/%s", img.Registry, img.Repository)
	tag := img.Tag
	digest := img.Digest

	if tag != "" && digest != "" {
		imgStr = fmt.Sprintf("%s:%s@%s", imgStr, tag, digest)
	} else if digest != "" {
		imgStr = fmt.Sprintf("%s@%s", imgStr, digest)
	} else if tag != "" {
		imgStr = fmt.Sprintf("%s:%s", imgStr, tag)
	}

	return imgStr
}

type App struct {
	ID                string
	InternalNumericID int32
	Name              string
	State             string
	Status            string
	Deployed          bool
	Hostname          string
	AppURL            string
	Version           int
	NetworkID         int
	Network           string

	Release        *Release
	Organization   Organization
	Secrets        []Secret
	CurrentRelease *Release
	Releases       struct {
		Nodes []Release
	}
	IPAddresses struct {
		Nodes []IPAddress
	}
	SharedIPAddress string
	CNAMETarget     string
	IPAddress       *IPAddress
	Certificates    struct {
		Nodes []AppCertificate
	}
	Certificate     AppCertificate
	PostgresAppRole *struct {
		Name string
	}
	Image *Image

	ImageUpgradeAvailable       bool
	ImageVersionTrackingEnabled bool
	ImageDetails                ImageVersion
	LatestImageDetails          ImageVersion

	PlatformVersion     string
	LimitedAccessTokens *struct {
		Nodes []LimitedAccessToken
	}

	Machines struct {
		Nodes []GqlMachine
	}

	CurrentLock *struct {
		LockID     string
		Expiration string
	}
}

func (app *App) Compact() *AppCompact {
	return &AppCompact{
		ID:              app.ID,
		Name:            app.Name,
		Status:          app.Status,
		Deployed:        app.Deployed,
		Hostname:        app.Hostname,
		AppURL:          app.AppURL,
		Organization:    app.Organization.Basic(),
		PlatformVersion: app.PlatformVersion,
		PostgresAppRole: app.PostgresAppRole,
	}
}

type LimitedAccessToken struct {
	Id        string
	Name      string
	Token     string
	ExpiresAt time.Time
	RevokedAt *time.Time
	User      User
}

type AppCertsCompact struct {
	Certificates struct {
		Nodes []AppCertificateCompact
	}
}

type AppCertificateCompact struct {
	CreatedAt    time.Time
	Hostname     string
	ClientStatus string
}

type AppCompact struct {
	ID                string
	InternalNumericID int32
	Name              string
	Status            string
	Deployed          bool
	Hostname          string
	CnameTarget       string
	Network           string
	AppURL            string
	Organization      *OrganizationBasic
	PlatformVersion   string
	PostgresAppRole   *struct {
		Name string
	}
}

func (app *AppCompact) IsPostgresApp() bool {
	// check app.PostgresAppRole.Name == "postgres_cluster"
	return app.PostgresAppRole != nil && app.PostgresAppRole.Name == "postgres_cluster"
}

type AppInfo struct {
	ID              string
	Name            string
	Status          string
	Deployed        bool
	Hostname        string
	Version         int
	PlatformVersion string
	Organization    *OrganizationBasic
	IPAddresses     struct {
		Nodes []IPAddress
	}
}

type AppBasic struct {
	ID              string
	Name            string
	PlatformVersion string
	Organization    *OrganizationBasic
}

type Organization struct {
	ID                 string
	InternalNumericID  string
	Name               string
	RemoteBuilderImage string
	RemoteBuilderApp   *App
	Slug               string
	RawSlug            string
	Type               string
	PaidPlan           bool
	Billable           bool
	Settings           map[string]any

	Domains struct {
		Nodes *[]*Domain
		Edges *[]*struct {
			Cursor *string
			Node   *Domain
		}
	}

	WireGuardPeer *WireGuardPeer

	WireGuardPeers struct {
		Nodes *[]*WireGuardPeer
		Edges *[]*struct {
			Cursor *string
			Node   *WireGuardPeer
		}
	}

	DelegatedWireGuardTokens struct {
		Nodes *[]*DelegatedWireGuardTokenHandle
		Edges *[]*struct {
			Cursor *string
			Node   *DelegatedWireGuardTokenHandle
		}
	}

	LoggedCertificates *struct {
		Nodes []LoggedCertificate
	}

	LimitedAccessTokens *struct {
		Nodes []LimitedAccessToken
	}
}

func (o *Organization) Basic() *OrganizationBasic {
	if o == nil {
		return nil
	}

	return &OrganizationBasic{
		ID:       o.ID,
		Name:     o.Name,
		Slug:     o.Slug,
		RawSlug:  o.RawSlug,
		PaidPlan: o.PaidPlan,
	}
}

func (o *Organization) GetID() string {
	return o.ID
}

func (o *Organization) GetSlug() string {
	return o.Slug
}

type OrganizationBasic struct {
	ID                string
	InternalNumericID string
	Name              string
	Slug              string
	RawSlug           string
	PaidPlan          bool
}

func (o *OrganizationBasic) GetID() string {
	return o.ID
}

func (o *OrganizationBasic) GetSlug() string {
	return o.Slug
}

type OrganizationDetails struct {
	ID                 string
	InternalNumericID  string
	Name               string
	RemoteBuilderImage string
	RemoteBuilderApp   *App
	Slug               string
	Type               string
	ViewerRole         string
	Apps               struct {
		Nodes []App
	}
	Members struct {
		Edges []OrganizationMembershipEdge
	}
}

type OrganizationMembershipEdge struct {
	Cursor   string
	Node     User
	Role     string
	JoinedAt time.Time
}

type Billable struct {
	Category string
	Product  string
	Time     time.Time
	Quantity float64
	App      App
}

type DNSRecords struct {
	ID         string
	Name       string
	Ttl        int
	Values     []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Fqdn       string
	IsApex     bool
	IsSystem   bool
	IsWildcard bool
	Domain     *Domain
}

type IPAddress struct {
	ID          string
	Address     string
	Type        string
	Region      string
	CreatedAt   time.Time
	ServiceName string
	Network     *struct {
		Name         string
		Organization *struct {
			Slug string
		}
	}
}

type EgressIPAddress struct {
	ID      string
	IP      string
	Version int
	Region  string
}

type VMSize struct {
	Name        string
	CPUCores    float32
	CPUClass    string
	MemoryGB    float32
	MemoryMB    int
	PriceMonth  float32
	PriceSecond float32
}

type User struct {
	ID              string
	Name            string
	Email           string
	EnablePaidHobby bool
}

type Secret struct {
	Name      string
	Digest    string
	CreatedAt time.Time
}

type SetSecretsInput struct {
	AppID   string                  `json:"appId"`
	Secrets []SetSecretsInputSecret `json:"secrets"`
}

type SetSecretsInputSecret struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type UnsetSecretsInput struct {
	AppID string   `json:"appId"`
	Keys  []string `json:"keys"`
}

type CreateAppInput struct {
	OrganizationID  string  `json:"organizationId"`
	Name            string  `json:"name"`
	PreferredRegion *string `json:"preferredRegion,omitempty"`
	Network         *string `json:"network,omitempty"`
	AppRoleID       string  `json:"appRoleId,omitempty"`
	Machines        bool    `json:"machines"`
}

type LogEntry struct {
	Timestamp string
	Message   string
	Level     string
	Instance  string
	Region    string
	Meta      struct {
		Instance string
		Region   string
		Event    struct {
			Provider string
		}
		HTTP struct {
			Request struct {
				ID      string
				Method  string
				Version string
			}
			Response struct {
				StatusCode int `json:"status_code"`
			}
		}
		Error struct {
			Code    int
			Message string
		}
		URL struct {
			Full string
		}
	}
}

type GeoRegion string

const (
	Africa       GeoRegion = "africa"
	AsiaPacific  GeoRegion = "asia_pacific"
	Europe       GeoRegion = "europe"
	NorthAmerica GeoRegion = "north_america"
	SouthAmerica GeoRegion = "south_america"
)

var geoRegionNames = map[GeoRegion]string{
	Africa:       "Africa",
	AsiaPacific:  "Asia Pacific",
	Europe:       "Europe",
	NorthAmerica: "North America",
	SouthAmerica: "South America",
}

func (p GeoRegion) String() string {
	if name, ok := geoRegionNames[p]; ok {
		return name
	}
	return ""
}

type Region struct {
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Latitude         float32   `json:"latitude"`
	Longitude        float32   `json:"longitude"`
	GatewayAvailable bool      `json:"gateway_available"`
	RequiresPaidPlan bool      `json:"requires_paid_plan"`
	Deprecated       bool      `json:"deprecated"`
	Capacity         int64     `json:"capacity"`
	GeoRegion        GeoRegion `json:"geo_region"`
}

type Release struct {
	ID                 string
	Version            int
	Stable             bool
	InProgress         bool
	Reason             string
	Description        string
	Status             string
	DeploymentStrategy string
	Metadata           *ReleaseMetadata
	User               User
	EvaluationID       string
	CreatedAt          time.Time
	ImageRef           string
}

type PostDeploymentInfo struct {
	FlyctlVersion string `json:"flyctl_version"`
	Error         string `json:"error"`
}
type ReleaseMetadata struct {
	PostDeploymentInfo PostDeploymentInfo `json:"post_deployment_info,omitempty"`
}

type SignedUrl struct {
	PutUrl string
}

type AppCertificate struct {
	ID                        string
	AcmeDNSConfigured         bool
	AcmeALPNConfigured        bool
	Configured                bool
	CertificateAuthority      string
	CreatedAt                 time.Time
	DNSProvider               string
	DNSValidationInstructions string
	DNSValidationHostname     string
	DNSValidationTarget       string
	Hostname                  string
	Source                    string
	ClientStatus              string
	IsApex                    bool
	IsWildcard                bool
	ValidationErrors          []AppCertificateValidationError
	Issued                    struct {
		Nodes []struct {
			ExpiresAt time.Time
			Type      string
		}
	}
}

type AppCertificateValidationError struct {
	ErrorCode   string    `json:"errorCode"`
	Message     string    `json:"message"`
	Remediation string    `json:"remediation"`
	Timestamp   time.Time `json:"timestamp"`
}

type CreateOrganizationPayload struct {
	Organization Organization
}

type DeleteOrganizationPayload struct {
	DeletedOrganizationId string
}

type HostnameCheck struct {
	ARecords              []string `json:"aRecords"`
	AAAARecords           []string `json:"aaaaRecords"`
	CNAMERecords          []string `json:"cnameRecords"`
	SOA                   string   `json:"soa"`
	DNSProvider           string   `json:"dnsProvider"`
	DNSVerificationRecord string   `json:"dnsVerificationRecord"`
	ResolvedAddresses     []string `json:"resolvedAddresses"`
}

type DeleteCertificatePayload struct {
	App         App
	Certificate AppCertificate
}

type AllocateIPAddressInput struct {
	AppID          string `json:"appId"`
	Type           string `json:"type"`
	Region         string `json:"region"`
	OrganizationID string `json:"organizationId,omitempty"`
	Network        string `json:"network,omitempty"`
}

type AllocateEgressIPAddressInput struct {
	AppID     string `json:"appId"`
	MachineID string `json:"machineId"`
}

type ReleaseEgressIPAddressInput struct {
	AppID     string `json:"appId"`
	MachineID string `json:"machineId"`
}

type ReleaseIPAddressInput struct {
	AppID       *string `json:"appId"`
	IPAddressID *string `json:"ipAddressId"`
	IP          *string `json:"ip"`
}

type Errors []Error

type Error struct {
	Message    string
	Path       []string
	Extensions Extensions
}

type Extensions struct {
	Code        string
	ServiceName string
	Query       string
	Variables   map[string]string
}

type Domain struct {
	ID                   string
	Name                 string
	CreatedAt            time.Time
	Organization         *Organization
	AutoRenew            *bool
	DelegatedNameservers *[]string
	ZoneNameservers      *[]string
	DnsStatus            *string
	RegistrationStatus   *string
	ExpiresAt            time.Time
	DnsRecords           *struct {
		Nodes *[]*DNSRecord
	}
}

type CheckDomainResult struct {
	DomainName            string
	TLD                   string
	RegistrationSupported bool
	RegistrationAvailable bool
	RegistrationPrice     int
	RegistrationPeriod    int
	TransferAvailable     bool
	DnsAvailable          bool
}

type DNSRecord struct {
	ID         string
	Name       string
	FQDN       string
	IsApex     bool
	IsWildcard bool
	IsSystem   bool
	TTL        int
	Type       string
	RData      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ImportDnsChange struct {
	Action  string
	OldText string
	NewText string
}

type ImportDnsWarning struct {
	Action     string
	Attributes struct {
		Name  string
		Type  string
		TTL   int
		Rdata string
	}
	Message string
}

type WireGuardPeer struct {
	ID            string
	Pubkey        string
	Region        string
	Name          string
	Peerip        string
	GatewayStatus *WireGuardPeerStatus
}

type WireGuardPeerStatus struct {
	Endpoint       string
	LastHandshake  string
	SinceHandshake string
	Rx             int64
	Tx             int64
	Added          string
	SinceAdded     string
	Live           bool
	WgError        string
}

type LoggedCertificate struct {
	Root bool
	Cert string
}

type AttachPostgresClusterInput struct {
	AppID                string  `json:"appId"`
	PostgresClusterAppID string  `json:"postgresClusterAppId"`
	DatabaseName         *string `json:"databaseName,omitempty"`
	DatabaseUser         *string `json:"databaseUser,omitempty"`
	VariableName         *string `json:"variableName,omitempty"`
	ManualEntry          bool    `json:"manualEntry,omitempty"`
}

type DetachPostgresClusterInput struct {
	AppID                       string `json:"appId"`
	PostgresClusterId           string `json:"postgresClusterAppId"`
	PostgresClusterAttachmentId string `json:"postgresClusterAttachmentId"`
}

type AttachPostgresClusterPayload struct {
	App                     App
	PostgresClusterApp      App
	ConnectionString        string
	EnvironmentVariableName string
}

type PostgresEnableConsulPayload struct {
	ConsulURL string `json:"consulUrl"`
}

type EnsureRemoteBuilderInput struct {
	AppName        *string `json:"appName"`
	OrganizationID *string `json:"organizationId"`
	Region         *string `json:"region"`
}

type PostgresClusterAttachment struct {
	ID                      string
	DatabaseName            string
	DatabaseUser            string
	EnvironmentVariableName string
}

type Image struct {
	ID             string
	Digest         string
	Ref            string
	CompressedSize string
	Manifest       *ImageManifest
}

type ImageManifest struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Invitation struct {
	ID           string
	Email        string
	CreatedAt    time.Time
	Redeemed     bool
	Inviter      *User
	Organization *Organization
}

type CreateOrganizationInvitation struct {
	Invitation Invitation
}

type GqlMachine struct {
	ID     string
	Name   string
	State  string
	Region string
	Config MachineConfig

	App *AppCompact

	IPs struct {
		Nodes []*MachineIP
	}

	EgressIpAddresses struct {
		Nodes []*EgressIPAddress
	}
}

type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
}

type AppHostIssues struct {
	HostIssues struct {
		Nodes []HostIssue
	}
}

type HostIssue struct {
	InternalId string
	Message    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
