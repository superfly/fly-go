package fly

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	MachineConfigMetadataKeyFlyManagedPostgres  = "fly-managed-postgres"
	MachineConfigMetadataKeyFlyPlatformVersion  = "fly_platform_version"
	MachineConfigMetadataKeyFlyReleaseId        = "fly_release_id"
	MachineConfigMetadataKeyFlyReleaseVersion   = "fly_release_version"
	MachineConfigMetadataKeyFlyProcessGroup     = "fly_process_group"
	MachineConfigMetadataKeyFlyPreviousAlloc    = "fly_previous_alloc"
	MachineConfigMetadataKeyFlyctlVersion       = "fly_flyctl_version"
	MachineConfigMetadataKeyFlyctlBGTag         = "fly_bluegreen_deployment_tag"
	MachineFlyPlatformVersion2                  = "v2"
	MachineProcessGroupApp                      = "app"
	MachineProcessGroupFlyAppReleaseCommand     = "fly_app_release_command"
	MachineProcessGroupFlyAppTestMachineCommand = "fly_app_test_machine_command"
	MachineProcessGroupFlyAppConsole            = "fly_app_console"
	MachineStateDestroyed                       = "destroyed"
	MachineStateDestroying                      = "destroying"
	MachineStateStarted                         = "started"
	MachineStateStopped                         = "stopped"
	MachineStateSuspended                       = "suspended"
	MachineStateCreated                         = "created"
	DefaultVMSize                               = "shared-cpu-1x"
	DefaultGPUVMSize                            = "performance-8x"
)

type HostStatus string

var (
	HostStatusOk          HostStatus = "ok"
	HostStatusUnknown     HostStatus = "unknown"
	HostStatusUnreachable HostStatus = "unreachable"
)

type Machine struct {
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	State    string          `json:"state,omitempty"`
	Region   string          `json:"region,omitempty"`
	ImageRef MachineImageRef `json:"image_ref,omitempty"`
	// InstanceID is unique for each version of the machine
	InstanceID string `json:"instance_id,omitempty"`
	Version    string `json:"version,omitempty"`
	// PrivateIP is the internal 6PN address of the machine.
	PrivateIP  string                `json:"private_ip,omitempty"`
	CreatedAt  string                `json:"created_at,omitempty"`
	UpdatedAt  string                `json:"updated_at,omitempty"`
	Config     *MachineConfig        `json:"config,omitempty"`
	Events     []*MachineEvent       `json:"events,omitempty"`
	Checks     []*MachineCheckStatus `json:"checks,omitempty"`
	LeaseNonce string                `json:"nonce,omitempty"`
	HostStatus HostStatus            `json:"host_status,omitempty" enums:"ok,unknown,unreachable"`

	// When `host_status` isn't "ok", the config can't be fully retrieved and has to be rebuilt from multiple sources
	// to form an partial configuration, not suitable to clone or recreate the original machine
	IncompleteConfig *MachineConfig `json:"incomplete_config,omitempty"`
}

func (m *Machine) FullImageRef() string {
	imgStr := fmt.Sprintf("%s/%s", m.ImageRef.Registry, m.ImageRef.Repository)
	tag := m.ImageRef.Tag
	digest := m.ImageRef.Digest

	if tag != "" && digest != "" {
		imgStr = fmt.Sprintf("%s:%s@%s", imgStr, tag, digest)
	} else if digest != "" {
		imgStr = fmt.Sprintf("%s@%s", imgStr, digest)
	} else if tag != "" {
		imgStr = fmt.Sprintf("%s:%s", imgStr, tag)
	}

	return imgStr
}

func (m *Machine) ImageRefWithVersion() string {
	ref := fmt.Sprintf("%s:%s", m.ImageRef.Repository, m.ImageRef.Tag)
	version := m.ImageRef.Labels["fly.version"]
	if version != "" {
		ref = fmt.Sprintf("%s (%s)", ref, version)
	}

	return ref
}

// GetConfig returns `IncompleteConfig` if `Config` is unset which happens when
// `HostStatus` isn't "ok"
func (m *Machine) GetConfig() *MachineConfig {
	if m.Config != nil {
		return m.Config
	}
	return m.IncompleteConfig
}

func (m *Machine) GetMetadataByKey(key string) string {
	c := m.GetConfig()
	if c == nil || c.Metadata == nil {
		return ""
	}
	return c.Metadata[key]
}

func (m *Machine) IsAppsV2() bool {
	return m.GetMetadataByKey(MachineConfigMetadataKeyFlyPlatformVersion) == MachineFlyPlatformVersion2
}

func (m *Machine) IsFlyAppsPlatform() bool {
	return m.IsAppsV2() && m.IsActive()
}

func (m *Machine) IsFlyAppsReleaseCommand() bool {
	return m.IsFlyAppsPlatform() && m.IsReleaseCommandMachine()
}

func (m *Machine) IsFlyAppsConsole() bool {
	return m.IsFlyAppsPlatform() && m.HasProcessGroup(MachineProcessGroupFlyAppConsole)
}

func (m *Machine) IsActive() bool {
	return m.State != MachineStateDestroyed && m.State != MachineStateDestroying
}

func (m *Machine) ProcessGroup() string {
	return m.GetConfig().ProcessGroup()
}

func (m *Machine) HasProcessGroup(desired string) bool {
	return m.ProcessGroup() == desired
}

func (m *Machine) ImageVersion() string {
	if m.ImageRef.Labels == nil {
		return ""
	}
	return m.ImageRef.Labels["fly.version"]
}

func (m *Machine) ImageRepository() string {
	return m.ImageRef.Repository
}

func (m *Machine) TopLevelChecks() *HealthCheckStatus {
	res := &HealthCheckStatus{}
	total := 0

	for _, check := range m.Checks {
		if !strings.HasPrefix(check.Name, "servicecheck-") {
			total++
			switch check.Status {
			case Passing:
				res.Passing += 1
			case Warning:
				res.Warn += 1
			case Critical:
				res.Critical += 1
			}
		}
	}

	res.Total = total
	return res
}

type HealthCheckStatus struct {
	Total, Passing, Warn, Critical int
}

func (hcs *HealthCheckStatus) AllPassing() bool {
	return hcs.Passing == hcs.Total
}

func (m *Machine) AllHealthChecks() *HealthCheckStatus {
	res := &HealthCheckStatus{}
	res.Total = len(m.Checks)
	for _, check := range m.Checks {
		switch check.Status {
		case Passing:
			res.Passing += 1
		case Warning:
			res.Warn += 1
		case Critical:
			res.Critical += 1
		}
	}
	return res
}

func (m *Machine) GetLatestEventOfType(eventType string) *MachineEvent {
	for _, event := range m.Events {
		if event.Type == eventType {
			return event
		}
	}
	return nil
}

// Finds the latest event of type latestEventType, which happened after the most recent event of type firstEventType
func (m *Machine) GetLatestEventOfTypeAfterType(latestEventType, firstEventType string) *MachineEvent {
	firstIndex := 0
	for i, e := range m.Events {
		if e.Type == firstEventType {
			firstIndex = i
			break
		}
	}
	for _, e := range m.Events[0:firstIndex] {
		if e.Type == latestEventType {
			return e
		}
	}
	return nil
}

func (m *Machine) MostRecentStartTimeAfterLaunch() (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("machine is nil")
	}
	var (
		firstStart  = -1
		firstLaunch = -1
		firstExit   = -1
	)
	for i, e := range m.Events {
		switch e.Type {
		case "start":
			firstStart = i
		case "launch":
			firstLaunch = i
		case "exit":
			firstExit = i
		}
		if firstStart != -1 && firstLaunch != -1 {
			break
		}
	}
	switch {
	case firstStart == -1:
		return time.Time{}, fmt.Errorf("no start event found")
	case firstStart >= firstLaunch:
		return time.Time{}, fmt.Errorf("no start event found after launch")
	case firstExit != -1 && firstExit <= firstStart:
		return time.Time{}, fmt.Errorf("no start event found after most recent exit")
	default:
		return m.Events[firstStart].Time(), nil
	}
}

func (m *Machine) IsReleaseCommandMachine() bool {
	return m.HasProcessGroup(MachineProcessGroupFlyAppReleaseCommand) || m.GetMetadataByKey("process_group") == "release_command"
}

type MachineImageRef struct {
	Registry   string            `json:"registry,omitempty"`
	Repository string            `json:"repository,omitempty"`
	Tag        string            `json:"tag,omitempty"`
	Digest     string            `json:"digest,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type MachineEvent struct {
	Type      string          `json:"type,omitempty"`
	Status    string          `json:"status,omitempty"`
	Request   *MachineRequest `json:"request,omitempty"`
	Source    string          `json:"source,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

func (e *MachineEvent) Time() time.Time {
	return time.Unix(e.Timestamp/1000, e.Timestamp%1000*1000000)
}

type MachineRequest struct {
	ExitEvent    *MachineExitEvent    `json:"exit_event,omitempty"`
	MonitorEvent *MachineMonitorEvent `json:"MonitorEvent,omitempty"`
	RestartCount int                  `json:"restart_count,omitempty"`
}

// returns the ExitCode from MonitorEvent if it exists, otherwise ExitEvent
// error when MonitorEvent and ExitEvent are both nil
func (mr *MachineRequest) GetExitCode() (int, error) {
	if mr.MonitorEvent != nil && mr.MonitorEvent.ExitEvent != nil {
		return mr.MonitorEvent.ExitEvent.ExitCode, nil
	} else if mr.ExitEvent != nil {
		return mr.ExitEvent.ExitCode, nil
	} else {
		return -1, fmt.Errorf("error no exit code in this MachineRequest")
	}
}

type MachineMonitorEvent struct {
	ExitEvent *MachineExitEvent `json:"exit_event,omitempty"`
}

type MachineExitEvent struct {
	ExitCode      int       `json:"exit_code,omitempty"`
	GuestExitCode int       `json:"guest_exit_code,omitempty"`
	GuestSignal   int       `json:"guest_signal,omitempty"`
	OOMKilled     bool      `json:"oom_killed,omitempty"`
	RequestedStop bool      `json:"requested_stop,omitempty"`
	Restarting    bool      `json:"restarting,omitempty"`
	Signal        int       `json:"signal,omitempty"`
	ExitedAt      time.Time `json:"exited_at,omitempty"`
}

type StopMachineInput struct {
	ID      string   `json:"id,omitempty"`
	Signal  string   `json:"signal,omitempty"`
	Timeout Duration `json:"timeout,omitempty"`
}

type RestartMachineInput struct {
	ID               string        `json:"id,omitempty"`
	Signal           string        `json:"signal,omitempty"`
	Timeout          time.Duration `json:"timeout,omitempty"`
	ForceStop        bool          `json:"force_stop,omitempty"`
	SkipHealthChecks bool          `json:"skip_health_checks,omitempty"`
}

type MachineIP struct {
	Family   string
	Kind     string
	IP       string
	MaskSize int
}

type RemoveMachineInput struct {
	ID   string `json:"id,omitempty"`
	Kill bool   `json:"kill,omitempty"`
}

type MachineRestartPolicy string

var (
	MachineRestartPolicyNo        MachineRestartPolicy = "no"
	MachineRestartPolicyOnFailure MachineRestartPolicy = "on-failure"
	MachineRestartPolicyAlways    MachineRestartPolicy = "always"
	MachineRestartPolicySpotPrice MachineRestartPolicy = "spot-price"
)

// @description The Machine restart policy defines whether and how flyd restarts a Machine after its main process exits. See https://fly.io/docs/machines/guides-examples/machine-restart-policy/.
type MachineRestart struct {
	// * no - Never try to restart a Machine automatically when its main process exits, whether that’s on purpose or on a crash.
	// * always - Always restart a Machine automatically and never let it enter a stopped state, even when the main process exits cleanly.
	// * on-failure - Try up to MaxRetries times to automatically restart the Machine if it exits with a non-zero exit code. Default when no explicit policy is set, and for Machines with schedules.
	// * spot-price - Starts the Machine only when there is capacity and the spot price is less than or equal to the bid price.
	Policy MachineRestartPolicy `json:"policy,omitempty" enums:"no,always,on-failure,spot-price"`
	// When policy is on-failure, the maximum number of times to attempt to restart the Machine before letting it stop.
	MaxRetries int `json:"max_retries,omitempty"`
	// GPU bid price for spot Machines.
	GPUBidPrice float32 `json:"gpu_bid_price,omitempty"`
}

type MachineMount struct {
	Encrypted              bool   `json:"encrypted,omitempty"`
	Path                   string `json:"path,omitempty"`
	SizeGb                 int    `json:"size_gb,omitempty"`
	Volume                 string `json:"volume,omitempty"`
	Name                   string `json:"name,omitempty"`
	ExtendThresholdPercent int    `json:"extend_threshold_percent,omitempty"`
	AddSizeGb              int    `json:"add_size_gb,omitempty"`
	SizeGbLimit            int    `json:"size_gb_limit,omitempty"`
}

type MachineGuest struct {
	CPUKind          string `json:"cpu_kind,omitempty" toml:"cpu_kind,omitempty"`
	CPUs             int    `json:"cpus,omitempty" toml:"cpus,omitempty"`
	MemoryMB         int    `json:"memory_mb,omitempty" toml:"memory_mb,omitempty"`
	GPUs             int    `json:"gpus,omitempty" toml:"gpus,omitempty"`
	GPUKind          string `json:"gpu_kind,omitempty" toml:"gpu_kind,omitempty"`
	HostDedicationID string `json:"host_dedication_id,omitempty" toml:"host_dedication_id,omitempty"`

	KernelArgs []string `json:"kernel_args,omitempty" toml:"kernel_args,omitempty"`
}

func (mg *MachineGuest) SetSize(size string) error {
	guest, ok := MachinePresets[size]
	if !ok {
		var machine_type string

		if strings.HasPrefix(size, "shared") {
			machine_type = "shared"
		} else if strings.HasPrefix(size, "performance") {
			machine_type = "performance"
		} else {
			return fmt.Errorf("invalid machine preset requested, '%s', expected to start with 'shared' or 'performance'", size)
		}

		validSizes := []string{}
		for size := range MachinePresets {
			if strings.HasPrefix(size, machine_type) {
				validSizes = append(validSizes, size)
			}
		}
		sort.Strings(validSizes)
		return fmt.Errorf("'%s' is an invalid machine size, choose one of: %v", size, validSizes)
	}

	mg.CPUs = guest.CPUs
	mg.CPUKind = guest.CPUKind
	mg.MemoryMB = guest.MemoryMB
	mg.GPUKind = guest.GPUKind
	mg.GPUs = guest.GPUs
	return nil
}

// ToSize converts Guest into VMSize on a best effort way
func (mg *MachineGuest) ToSize() string {
	if mg == nil {
		return ""
	}
	switch {
	case mg.GPUKind == "a100-pcie-40gb":
		return "a100-40gb"
	case mg.GPUKind == "a100-sxm4-80gb":
		return "a100-80gb"
	case mg.GPUKind == "l40s":
		return "l40s"
	case mg.GPUKind == "a10":
		return "a10"
	case mg.CPUKind == "shared":
		return fmt.Sprintf("shared-cpu-%dx", mg.CPUs)
	case mg.CPUKind == "performance":
		return fmt.Sprintf("performance-%dx", mg.CPUs)
	default:
		return "unknown"
	}
}

// String returns a string representation of the guest
// Formatted as "[cpu_kind], XGB RAM"
// Returns "" if nil
func (mg *MachineGuest) String() string {
	if mg == nil {
		return ""
	}
	size := mg.ToSize()
	gbRam := mg.MemoryMB / 1024
	if gbRam == 0 {
		return fmt.Sprintf("%s, %dMB RAM", size, mg.MemoryMB)
	}
	return fmt.Sprintf("%s, %dGB RAM", size, gbRam)
}

const (
	MIN_MEMORY_MB_PER_SHARED_CPU = 256
	MIN_MEMORY_MB_PER_CPU        = 2048

	MAX_MEMORY_MB_PER_SHARED_CPU = 2048
	MAX_MEMORY_MB_PER_CPU        = 8192
)

// TODO - Determine if we want allocate max memory allocation, or minimum per # cpus.
var MachinePresets map[string]*MachineGuest = map[string]*MachineGuest{
	"shared-cpu-1x": {CPUKind: "shared", CPUs: 1, MemoryMB: 1 * MIN_MEMORY_MB_PER_SHARED_CPU},
	"shared-cpu-2x": {CPUKind: "shared", CPUs: 2, MemoryMB: 2 * MIN_MEMORY_MB_PER_SHARED_CPU},
	"shared-cpu-4x": {CPUKind: "shared", CPUs: 4, MemoryMB: 4 * MIN_MEMORY_MB_PER_SHARED_CPU},
	"shared-cpu-8x": {CPUKind: "shared", CPUs: 8, MemoryMB: 8 * MIN_MEMORY_MB_PER_SHARED_CPU},

	"performance-1x":  {CPUKind: "performance", CPUs: 1, MemoryMB: 1 * MIN_MEMORY_MB_PER_CPU},
	"performance-2x":  {CPUKind: "performance", CPUs: 2, MemoryMB: 2 * MIN_MEMORY_MB_PER_CPU},
	"performance-4x":  {CPUKind: "performance", CPUs: 4, MemoryMB: 4 * MIN_MEMORY_MB_PER_CPU},
	"performance-8x":  {CPUKind: "performance", CPUs: 8, MemoryMB: 8 * MIN_MEMORY_MB_PER_CPU},
	"performance-16x": {CPUKind: "performance", CPUs: 16, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},

	"a100-40gb": {GPUKind: "a100-pcie-40gb", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"a100-80gb": {GPUKind: "a100-sxm4-80gb", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"l40s":      {GPUKind: "l40s", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"a10":       {GPUKind: "a10", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
}

type MachineMetrics struct {
	Port  int    `toml:"port" json:"port,omitempty"`
	Path  string `toml:"path" json:"path,omitempty"`
	Https bool   `toml:"https" json:"https,omitempty"`
}

type MachineCheckKind string

var (
	// Informational check. Showed in status, but doesn't cause any actions.
	// Default value for top-level check if kind is not specified.
	MachineCheckKindInformational MachineCheckKind = "informational"
	// Readiness check. Failed check causes the machine to be taken out of LB pool
	MachineCheckKindReadiness MachineCheckKind = "readiness"
	// TODO: liveness, startup
)

// @description An optional object that defines one or more named checks. The key for each check is the check name.
type MachineCheck struct {
	// The port to connect to, often the same as internal_port
	Port *int `json:"port,omitempty"`
	// tcp or http
	Type *string `json:"type,omitempty"`
	// Kind of the check (informational, readiness)
	Kind *MachineCheckKind `json:"kind,omitempty" enums:"informational,readiness"`
	// The time between connectivity checks
	Interval *Duration `json:"interval,omitempty"`
	// The maximum time a connection can take before being reported as failing its health check
	Timeout *Duration `json:"timeout,omitempty"`
	// The time to wait after a VM starts before checking its health
	GracePeriod *Duration `json:"grace_period,omitempty"`
	// For http checks, the HTTP method to use to when making the request
	HTTPMethod *string `json:"method,omitempty"`
	// For http checks, the path to send the request to
	HTTPPath *string `json:"path,omitempty"`
	// For http checks, whether to use http or https
	HTTPProtocol *string `json:"protocol,omitempty"`
	// For http checks with https protocol, whether or not to verify the TLS certificate
	HTTPSkipTLSVerify *bool `json:"tls_skip_verify,omitempty"`
	// If the protocol is https, the hostname to use for TLS certificate validation
	HTTPTLSServerName *string             `json:"tls_server_name,omitempty"`
	HTTPHeaders       []MachineHTTPHeader `json:"headers,omitempty"`
}

// @description For http checks, an array of objects with string field Name and array of strings field Values. The key/value pairs specify header and header values that will get passed with the check call.
type MachineHTTPHeader struct {
	// The header name
	Name string `json:"name,omitempty"`
	// The header value
	Values []string `json:"values,omitempty"`
}

type ConsulCheckStatus string

const (
	Critical ConsulCheckStatus = "critical"
	Warning  ConsulCheckStatus = "warning"
	Passing  ConsulCheckStatus = "passing"
)

type MachineCheckStatus struct {
	Name      string            `json:"name,omitempty"`
	Status    ConsulCheckStatus `json:"status,omitempty"`
	Output    string            `json:"output,omitempty"`
	UpdatedAt *time.Time        `json:"updated_at,omitempty"`
}

type MachinePort struct {
	Port              *int               `json:"port,omitempty" toml:"port,omitempty"`
	StartPort         *int               `json:"start_port,omitempty" toml:"start_port,omitempty"`
	EndPort           *int               `json:"end_port,omitempty" toml:"end_port,omitempty"`
	Handlers          []string           `json:"handlers,omitempty" toml:"handlers,omitempty"`
	ForceHTTPS        bool               `json:"force_https,omitempty" toml:"force_https,omitempty"`
	TLSOptions        *TLSOptions        `json:"tls_options,omitempty" toml:"tls_options,omitempty"`
	HTTPOptions       *HTTPOptions       `json:"http_options,omitempty" toml:"http_options,omitempty"`
	ProxyProtoOptions *ProxyProtoOptions `json:"proxy_proto_options,omitempty" toml:"proxy_proto_options,omitempty"`
}

func (mp *MachinePort) ContainsPort(port int) bool {
	if mp.Port != nil && port == *mp.Port {
		return true
	}
	if mp.StartPort == nil && mp.EndPort == nil {
		return false
	}
	startPort := 0
	endPort := 65535
	if mp.StartPort != nil {
		startPort = *mp.StartPort
	}
	if mp.EndPort != nil {
		endPort = *mp.EndPort
	}
	return startPort <= port && port <= endPort
}

func (mp *MachinePort) HasNonHttpPorts() bool {
	if mp.Port != nil && *mp.Port != 443 && *mp.Port != 80 {
		return true
	}
	if mp.StartPort == nil && mp.EndPort == nil {
		return false
	}
	startPort := 0
	endPort := 65535
	if mp.StartPort != nil {
		startPort = *mp.StartPort
	}
	if mp.EndPort != nil {
		endPort = *mp.EndPort
	}
	portRangeCount := endPort - startPort + 1
	if portRangeCount > 2 {
		return true
	}
	httpInRange := startPort <= 80 && 80 <= endPort
	httpsInRange := startPort <= 443 && 443 <= endPort
	switch {
	case portRangeCount == 2:
		return !httpInRange || !httpsInRange
	case portRangeCount == 1:
		return !httpInRange && !httpsInRange
	}
	return false
}

type ProxyProtoOptions struct {
	Version string `json:"version,omitempty" toml:"version,omitempty"`
}

type TLSOptions struct {
	ALPN              []string `json:"alpn,omitempty" toml:"alpn,omitempty"`
	Versions          []string `json:"versions,omitempty" toml:"versions,omitempty"`
	DefaultSelfSigned *bool    `json:"default_self_signed,omitempty" toml:"default_self_signed,omitempty"`
}

type HTTPOptions struct {
	Compress           *bool                `json:"compress,omitempty" toml:"compress,omitempty"`
	Response           *HTTPResponseOptions `json:"response,omitempty" toml:"response,omitempty"`
	H2Backend          *bool                `json:"h2_backend,omitempty" toml:"h2_backend,omitempty"`
	IdleTimeout        *uint32              `json:"idle_timeout,omitempty" toml:"idle_timeout,omitempty"`
	HeadersReadTimeout *uint32              `json:"headers_read_timeout,omitempty" toml:"headers_read_timeout,omitempty"`
}

type HTTPResponseOptions struct {
	Headers  map[string]any `json:"headers,omitempty" toml:"headers,omitempty"`
	Pristine *bool          `json:"pristine,omitempty" toml:"pristine,omitempty"`
}

type MachineService struct {
	Protocol     string `json:"protocol,omitempty" toml:"protocol,omitempty"`
	InternalPort int    `json:"internal_port,omitempty" toml:"internal_port,omitempty"`
	// Accepts a string (new format) or a boolean (old format). For backward compatibility with older clients, the API continues to use booleans for "off" and "stop" in responses.
	// * "off" or false - Do not autostop the Machine.
	// * "stop" or true - Automatically stop the Machine.
	// * "suspend" - Automatically suspend the Machine, falling back to a full stop if this is not possible.
	Autostop                 *MachineAutostop           `json:"autostop,omitempty" swaggertype:"string" enums:"off,stop,suspend"`
	Autostart                *bool                      `json:"autostart,omitempty"`
	MinMachinesRunning       *int                       `json:"min_machines_running,omitempty"`
	Ports                    []MachinePort              `json:"ports,omitempty" toml:"ports,omitempty"`
	Checks                   []MachineCheck             `json:"checks,omitempty" toml:"checks,omitempty"`
	Concurrency              *MachineServiceConcurrency `json:"concurrency,omitempty" toml:"concurrency"`
	ForceInstanceKey         *string                    `json:"force_instance_key" toml:"force_instance_key"`
	ForceInstanceDescription *string                    `json:"force_instance_description,omitempty" toml:"force_instance_description"`
}

type MachineServiceConcurrency struct {
	Type      string `json:"type,omitempty" toml:"type,omitempty"`
	HardLimit int    `json:"hard_limit,omitempty" toml:"hard_limit,omitempty"`
	SoftLimit int    `json:"soft_limit,omitempty" toml:"soft_limit,omitempty"`
}

type MachineConfig struct {
	// Fields managed from fly.toml
	// If you add anything here, ensure appconfig.Config.ToMachine() is updated

	// An object filled with key/value pairs to be set as environment variables
	Env      map[string]string       `json:"env,omitempty"`
	Init     MachineInit             `json:"init,omitempty"`
	Guest    *MachineGuest           `json:"guest,omitempty"`
	Metadata map[string]string       `json:"metadata,omitempty"`
	Mounts   []MachineMount          `json:"mounts,omitempty"`
	Services []MachineService        `json:"services,omitempty"`
	Metrics  *MachineMetrics         `json:"metrics,omitempty"`
	Checks   map[string]MachineCheck `json:"checks,omitempty"`
	Statics  []*Static               `json:"statics,omitempty"`

	// Set by fly deploy or fly machines commands

	// The docker image to run
	Image string  `json:"image,omitempty"`
	Files []*File `json:"files,omitempty"`

	// The following fields can only be set or updated by `fly machines run|update` commands
	// "fly deploy" must preserve them, if you add anything here, ensure it is propagated on deploys

	Schedule string `json:"schedule,omitempty"`
	// Optional boolean telling the Machine to destroy itself once it’s complete (default false)
	AutoDestroy bool             `json:"auto_destroy,omitempty"`
	Restart     *MachineRestart  `json:"restart,omitempty"`
	DNS         *DNSConfig       `json:"dns,omitempty"`
	Processes   []MachineProcess `json:"processes,omitempty"`

	// Standbys enable a machine to be a standby for another. In the event of a hardware failure,
	// the standby machine will be started.
	Standbys []string `json:"standbys,omitempty"`

	StopConfig *StopConfig `json:"stop_config,omitempty"`

	// Containers are a list of containers that will run in the machine. Currently restricted to
	// only specific organizations.
	Containers []*ContainerConfig `json:"containers,omitempty"`

	// Volumes describe the set of volumes that can be attached to the machine. Used in conjuction
	// with containers
	Volumes []*VolumeConfig `json:"volumes,omitempty"`

	// Deprecated: use Guest instead
	VMSize string `json:"size,omitempty"`
	// Deprecated: use Service.Autostart instead
	DisableMachineAutostart *bool `json:"disable_machine_autostart,omitempty"`
}

func (c *MachineConfig) ProcessGroup() string {
	// backwards compatible process_group getter.
	// from poking around, "fly_process_group" used to be called "process_group"
	// and since it's a metadata value, it's like a screenshot.
	// so we have 3 scenarios
	// - machines with only 'process_group'
	// - machines with both 'process_group' and 'fly_process_group'
	// - machines with only 'fly_process_group'
	if c == nil || c.Metadata == nil {
		return ""
	}

	flyProcessGroup := c.Metadata[MachineConfigMetadataKeyFlyProcessGroup]
	if flyProcessGroup != "" {
		return flyProcessGroup
	}

	return c.Metadata["process_group"]
}

type Static struct {
	GuestPath     string `toml:"guest_path" json:"guest_path" validate:"required"`
	UrlPrefix     string `toml:"url_prefix" json:"url_prefix" validate:"required"`
	TigrisBucket  string `toml:"tigris_bucket" json:"tigris_bucket"`
	IndexDocument string `toml:"index_document" json:"index_document"`
}

type MachineInit struct {
	Exec       []string `json:"exec,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty"`
	Tty        bool     `json:"tty,omitempty"`
	SwapSizeMB *int     `json:"swap_size_mb,omitempty"`
	KernelArgs []string `json:"kernel_args,omitempty"`
}

type DNSConfig struct {
	SkipRegistration bool             `json:"skip_registration,omitempty"`
	Nameservers      []string         `json:"nameservers,omitempty"`
	Searches         []string         `json:"searches,omitempty"`
	Options          []dnsOption      `json:"options,omitempty"`
	DNSForwardRules  []dnsForwardRule `json:"dns_forward_rules,omitempty"`
	Hostname         string           `json:"hostname,omitempty"`
	HostnameFqdn     string           `json:"hostname_fqdn,omitempty"`
}

type dnsForwardRule struct {
	Basename string `json:"basename,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

type dnsOption struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type StopConfig struct {
	Timeout *Duration `json:"timeout,omitempty"`
	Signal  *string   `json:"signal,omitempty"`
}

// @description A file that will be written to the Machine. One of RawValue or SecretName must be set.
type File struct {
	// GuestPath is the path on the machine where the file will be written and must be an absolute path.
	// For example: /full/path/to/file.json
	GuestPath string `json:"guest_path,omitempty"`

	// The base64 encoded string of the file contents.
	RawValue *string `json:"raw_value,omitempty"`

	// The name of the secret that contains the base64 encoded file contents.
	SecretName *string `json:"secret_name,omitempty"`

	// Mode bits used to set permissions on this file as accepted by chmod(2).
	Mode uint32 `json:"mode,omitempty"`
}

type ContainerConfig struct {
	// Name is used to identify the container in the machine.
	Name string `json:"name"`

	// Image is the docker image to run.
	Image string `json:"image"`

	// Image Config overrides - these fields are used to override the image configuration.
	// If not provided, the image configuration will be used.
	// ExecOverride is used to override the default command of the image.
	ExecOverride []string `json:"exec,omitempty"`
	// EntrypointOverride is used to override the default entrypoint of the image.
	EntrypointOverride []string `json:"entrypoint,omitempty"`
	// CmdOverride is used to override the default command of the image.
	CmdOverride []string `json:"cmd,omitempty"`
	// UserOverride is used to override the default user of the image.
	UserOverride string `json:"user,omitempty"`
	// ExtraEnv is used to add additional environment variables to the container.
	ExtraEnv map[string]string `json:"env,omitempty"`

	// Secrets can be provided at the process level to explicitly indicate which secrets should be
	// used for the process. If not provided, the secrets provided at the machine level will be used.
	Secrets []MachineSecret `json:"secrets,omitempty"`

	// EnvFrom can be provided to set environment variables from machine fields.
	EnvFrom []EnvFrom `json:"env_from,omitempty"`

	// Files are files that will be written to the container file system.
	Files []*File `json:"files,omitempty"`

	// Restart is used to define the restart policy for the container. NOTE: spot-price is not
	// supported for containers.
	Restart *MachineRestart `json:"restart,omitempty"`

	// Stop is used to define the signal and timeout for stopping the container.
	Stop *StopConfig `json:"stop,omitempty"`

	// DependsOn can be used to define dependencies between containers. The container will only be
	// started after all of its dependent conditions have been satisfied.
	DependsOn []ContainerDependency `json:"depends_on,omitempty"`

	// Healthchecks determine the health of your containers. Healthchecks can use HTTP, TCP or an Exec command.
	Healthchecks []ContainerHealthcheck `json:"healthchecks,omitempty"`

	// Set of mounts added to the container. These must reference a volume in the machine config via its name.
	Mounts []ContainerMount `json:"mounts,omitempty"`
}

type ContainerMount struct {
	// The name of the volume. Must exist in the volumes field in the machine configuration
	Name string `json:"name"`
	// The path to mount the volume within the container
	Path string `json:"path"`
}

type ContainerDependency struct {
	Name      string                       `json:"name"`
	Condition ContainerDependencyCondition `json:"condition" enums:"exited_successfully,healthy,started"`
}

type ContainerDependencyCondition string

const (
	ExitedSuccessfully ContainerDependencyCondition = "exited_successfully"
	Healthy            ContainerDependencyCondition = "healthy"
	Started            ContainerDependencyCondition = "started"
)

type ContainerHealthcheckKind string

const (
	// Readiness checks ensure your container is ready to receive traffic.
	Readiness ContainerHealthcheckKind = "readiness"
	// Liveness checks ensure your container is reachable and functional. When a liveness check
	// fails, a policy is used to determine what action to perform such as restarting the container.
	Liveness ContainerHealthcheckKind = "liveness"
)

type ContainerHealthcheckScheme string

const (
	HTTP  ContainerHealthcheckScheme = "http"
	HTTPS ContainerHealthcheckScheme = "https"
)

type UnhealthyPolicy string

const (
	// When a container becomes unhealthy, stop it. If there is a restart policy set on
	// the container, it will be applied
	UnhealthyPolicyStop UnhealthyPolicy = "stop"
)

type ContainerHealthcheckType struct {
	HTTP *HTTPHealthcheck `json:"http,omitempty"`
	TCP  *TCPHealthcheck  `json:"tcp,omitempty"`
	Exec *ExecHealthcheck `json:"exec,omitempty"`
}

type ContainerHealthcheck struct {
	// The name of the check. Must be unique within the container.
	Name string `json:"name"`
	// The time in seconds between executing the defined check.
	Interval int64 `json:"interval,omitempty"`
	// The time in seconds to wait after a container starts before checking its health.
	GracePeriod int64 `json:"grace_period,omitempty"`
	// The number of times the check must succeeed before considering the container healthy.
	SuccessThreshold int32 `json:"success_threshold,omitempty"`
	// The number of times the check must fail before considering the container unhealthy.
	FailureThreshold int32 `json:"failure_threshold,omitempty"`
	// The time in seconds to wait for the check to complete.
	Timeout int64 `json:"timeout,omitempty"`
	// Kind of healthcheck (readiness, liveness)
	Kind ContainerHealthcheckKind `json:"kind,omitempty"`
	// Unhealthy policy that determines what action to take if a container is deemed unhealthy
	Unhealthy UnhealthyPolicy `json:"unhealthy,omitempty"`
	// The type of healthcheck
	ContainerHealthcheckType
}

type HTTPHealthcheck struct {
	// The port to connect to, often the same as internal_port
	Port int32 `json:"port"`
	// The HTTP method to use to when making the request
	Method string `json:"method,omitempty"`
	// The path to send the request to
	Path string `json:"path,omitempty"`
	// Additional headers to send with the request
	Headers []MachineHTTPHeader `json:"headers,omitempty"`
	// Whether to use http or https
	Scheme ContainerHealthcheckScheme `json:"scheme,omitempty"`
	// If the protocol is https, whether or not to verify the TLS certificate
	TLSSkipVerify *bool `json:"tls_skip_verify,omitempty"`
	// If the protocol is https, the hostname to use for TLS certificate validation
	TLSServerName string `json:"tls_server_name,omitempty"`
}

type TCPHealthcheck struct {
	// The port to connect to, often the same as internal_port
	Port int32 `json:"port"`
}

type ExecHealthcheck struct {
	// The command to run to check the health of the container (e.g. ["cat", "/tmp/healthy"])
	Command []string `json:"command"`
}

type VolumeConfig struct {
	// The name of the volume. A volume must have a unique name within an app
	Name string `json:"name"`
	// The volume resource, provides configuration for the volume
	VolumeResource
}

type VolumeResource struct {
	TempDir *TempDirVolume `json:"temp_dir,omitempty"`
}

type StorageType string

const (
	StorageTypeDisk   = "disk"
	StorageTypeMemory = "memory"
)

// A TempDir is an ephemeral directory tied to the lifecycle of a Machine. It
// is often used as scratch space, to communicate between containers and so on.
type TempDirVolume struct {
	// The type of storage used to back the temp dir. Either disk or memory.
	StorageType StorageType `json:"storage_type"`
	// The size limit of the temp dir, only applicable when using disk backed storage.
	SizeMB uint64 `json:"size_mb,omitempty"`
}

type MachineLease struct {
	Status  string            `json:"status,omitempty"`
	Data    *MachineLeaseData `json:"data,omitempty"`
	Message string            `json:"message,omitempty"`
	Code    string            `json:"code,omitempty"`
}

type MachineLeaseData struct {
	Nonce     string `json:"nonce,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Version   string `json:"version,omitempty"`
}

type MachineStartResponse struct {
	Message       string `json:"message,omitempty"`
	Status        string `json:"status,omitempty"`
	PreviousState string `json:"previous_state,omitempty"`
}

type LaunchMachineInput struct {
	Config                  *MachineConfig `json:"config,omitempty"`
	Region                  string         `json:"region,omitempty"`
	Name                    string         `json:"name,omitempty"`
	SkipLaunch              bool           `json:"skip_launch,omitempty"`
	SkipServiceRegistration bool           `json:"skip_service_registration,omitempty"`
	LSVD                    bool           `json:"lsvd,omitempty"`

	LeaseTTL int `json:"lease_ttl,omitempty"`

	// Client side only
	ID                  string `json:"-"`
	SkipHealthChecks    bool   `json:"-"`
	RequiresReplacement bool   `json:"-"`
	Timeout             int    `json:"-"`
}

type MachineProcess struct {
	ExecOverride       []string          `json:"exec,omitempty"`
	EntrypointOverride []string          `json:"entrypoint,omitempty"`
	CmdOverride        []string          `json:"cmd,omitempty"`
	UserOverride       string            `json:"user,omitempty"`
	ExtraEnv           map[string]string `json:"env,omitempty"`
	// Secrets can be provided at the process level to explicitly indicate which secrets should be
	// used for the process. If not provided, the secrets provided at the machine level will be used.
	Secrets []MachineSecret `json:"secrets,omitempty"`
	// IgnoreAppSecrets can be set to true to ignore the secrets for the App the Machine belongs to
	// and only use the secrets provided at the process level. The default/legacy behavior is to use
	// the secrets provided at the App level.
	IgnoreAppSecrets bool `json:"ignore_app_secrets,omitempty"`

	// EnvFrom can be provided to set environment variables from machine fields.
	EnvFrom []EnvFrom `json:"env_from,omitempty"`
}

// @description A Secret needing to be set in the environment of the Machine. env_var is required
// and name can be used to reference a secret name where the environment variable is different
// from what was set originally using the API. NOTE: When secrets are provided on any process, it
// will override the secrets provided at the machine level.
type MachineSecret struct {
	// EnvVar is required and is the name of the environment variable that will be set from the
	// secret. It must be a valid environment variable name.
	EnvVar string `json:"env_var"`

	// Name is optional and when provided is used to reference a secret name where the EnvVar is
	// different from what was set as the secret name.
	Name string `json:"name"`
}

// @description EnvVar defines an environment variable to be populated from a machine field, env_var
// and field_ref are required.
type EnvFrom struct {
	// EnvVar is required and is the name of the environment variable that will be set from the
	// secret. It must be a valid environment variable name.
	EnvVar string `json:"env_var"`

	// FieldRef selects a field of the Machine: supports id, version, app_name, private_ip, region, image.
	FieldRef string `json:"field_ref" enums:"id,version,app_name,private_ip,region,image"`
}

type MachineExecRequest struct {
	Container string `json:"container,omitempty"`
	Cmd       string `json:"cmd,omitempty"`
	Stdin     string `json:"stdin,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

type MachineExecResponse struct {
	ExitCode int32  `json:"exit_code,omitempty"`
	StdOut   string `json:"stdout,omitempty"`
	StdErr   string `json:"stderr,omitempty"`
}

type MachinePsResponse []ProcessStat

type ProcessStat struct {
	Pid           int32          `json:"pid"`
	Stime         uint64         `json:"stime"`
	Rtime         uint64         `json:"rtime"`
	Command       string         `json:"command"`
	Directory     string         `json:"directory"`
	Cpu           uint64         `json:"cpu"`
	Rss           uint64         `json:"rss"`
	ListenSockets []ListenSocket `json:"listen_sockets"`
}

type ListenSocket struct {
	Proto   string `json:"proto"`
	Address string `json:"address"`
}

type MachineAutostop int

const (
	MachineAutostopOff MachineAutostop = iota
	MachineAutostopStop
	MachineAutostopSuspend
)

func (s MachineAutostop) String() string {
	switch s {
	case MachineAutostopOff:
		return "off"
	case MachineAutostopStop:
		return "stop"
	case MachineAutostopSuspend:
		return "suspend"
	default:
		return "off"
	}
}

func (s *MachineAutostop) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, []byte("null")) {
		return nil
	}

	var asBool bool
	if err := json.Unmarshal(raw, &asBool); err == nil {
		if asBool {
			*s = MachineAutostopStop
		} else {
			*s = MachineAutostopOff
		}
		return nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		switch asString {
		case "off":
			*s = MachineAutostopOff
			return nil
		case "stop":
			*s = MachineAutostopStop
			return nil
		case "suspend":
			*s = MachineAutostopSuspend
			return nil
		default:
			return fmt.Errorf("invalid autostop string value \"%s\"", asString)
		}
	}

	return errors.New("autostop value is not a valid bool or string")
}

func (s MachineAutostop) MarshalJSON() ([]byte, error) {
	// For backward compatibility, we continue to serialize "off" and
	// "stop" as booleans.
	switch s {
	case MachineAutostopOff:
		return []byte("false"), nil
	case MachineAutostopStop:
		return []byte("true"), nil
	case MachineAutostopSuspend:
		return []byte(`"suspend"`), nil
	default:
		return []byte("false"), nil
	}
}

func (s *MachineAutostop) UnmarshalText(raw []byte) error {
	switch string(raw) {
	case "false", "off":
		*s = MachineAutostopOff
		return nil
	case "true", "stop":
		*s = MachineAutostopStop
		return nil
	case "suspend":
		*s = MachineAutostopSuspend
		return nil
	default:
		return fmt.Errorf("invalid autostop value \"%s\"", string(raw))
	}
}

func (s MachineAutostop) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}
