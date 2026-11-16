package fly

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	ID       string          `toml:"id,omitempty" json:"id,omitempty"`
	Name     string          `toml:"name,omitempty" json:"name,omitempty"`
	State    string          `toml:"state,omitempty" json:"state,omitempty"`
	Region   string          `toml:"region,omitempty" json:"region,omitempty"`
	ImageRef MachineImageRef `toml:"image_ref,omitempty" json:"image_ref"`
	// InstanceID is unique for each version of the machine
	InstanceID string `toml:"instance_id,omitempty" json:"instance_id,omitempty"`
	Version    string `toml:"version,omitempty" json:"version,omitempty"`
	// PrivateIP is the internal 6PN address of the machine.
	PrivateIP         string                `json:"private_ip,omitempty"`
	CreatedAt         string                `json:"created_at,omitempty"`
	UpdatedAt         string                `json:"updated_at,omitempty"`
	Config            *MachineConfig        `json:"config,omitempty"`
	Events            []*MachineEvent       `json:"events,omitempty"`
	Checks            []*MachineCheckStatus `json:"checks,omitempty"`
	LeaseNonce        string                `json:"nonce,omitempty"`
	HostStatus        HostStatus            `json:"host_status,omitempty" enums:"ok,unknown,unreachable"`
	HostFeatures      []string              `json:"host_features,omitempty"`
	Cordoned          bool                  `json:"cordoned"`
	ContainerStatuses []*ContainerStatus    `json:"containers,omitempty"`

	// When `host_status` isn't "ok", the config can't be fully retrieved and has to be rebuilt from multiple sources
	// to form an partial configuration, not suitable to clone or recreate the original machine
	IncompleteConfig *MachineConfig `toml:"incomplete_config,omitempty" json:"incomplete_config,omitempty"`
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
	m.RemoveCompatChecks()
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
	m.RemoveCompatChecks()
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

// We used to generate `bg_deployments_*` top-level checks during bluegreen deployments.
// This is no longer the case, but flaps still inserts these "fake" checks in its response
// for compatibility with older `flyctl` versions, since the bluegreen strategy only looks
// at those checks.
// So, this function removes these fake checks so that they're not exposed to newer users of
// this library, including new builds of `flyctl`.
func (m *Machine) RemoveCompatChecks() {
	m.Checks = slices.DeleteFunc(m.Checks, func(check *MachineCheckStatus) bool {
		return strings.HasPrefix(check.Name, "bg_deployments_compat")
	})
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
	Registry   string            `toml:"registry,omitempty" json:"registry,omitempty"`
	Repository string            `toml:"repository,omitempty" json:"repository,omitempty"`
	Tag        string            `toml:"tag,omitempty" json:"tag,omitempty"`
	Digest     string            `toml:"digest,omitempty" json:"digest,omitempty"`
	Labels     map[string]string `toml:"labels,omitempty" json:"labels,omitempty"`
}

type MachineEvent struct {
	Type      string          `toml:"type,omitempty" json:"type,omitempty"`
	Status    string          `toml:"status,omitempty" json:"status,omitempty"`
	Request   *MachineRequest `toml:"request,omitempty" json:"request,omitempty"`
	Source    string          `toml:"source,omitempty" json:"source,omitempty"`
	Timestamp int64           `toml:"timestamp,omitempty" json:"timestamp,omitempty"`
}

func (e *MachineEvent) Time() time.Time {
	return time.Unix(e.Timestamp/1000, e.Timestamp%1000*1000000)
}

type MachineRequest struct {
	ExitEvent    *MachineExitEvent    `toml:"exit_event,omitempty" json:"exit_event,omitempty"`
	MonitorEvent *MachineMonitorEvent `toml:"MonitorEvent,omitempty" json:"MonitorEvent,omitempty"`
	RestartCount int                  `toml:"restart_count,omitempty" json:"restart_count,omitempty"`
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
	ExitEvent *MachineExitEvent `toml:"exit_event,omitempty" json:"exit_event,omitempty"`
}

type MachineExitEvent struct {
	ExitCode      int       `toml:"exit_code,omitempty" json:"exit_code,omitempty"`
	GuestExitCode int       `toml:"guest_exit_code,omitempty" json:"guest_exit_code,omitempty"`
	GuestSignal   int       `toml:"guest_signal,omitempty" json:"guest_signal,omitempty"`
	OOMKilled     bool      `toml:"oom_killed,omitempty" json:"oom_killed,omitempty"`
	RequestedStop bool      `toml:"requested_stop,omitempty" json:"requested_stop,omitempty"`
	Restarting    bool      `toml:"restarting,omitempty" json:"restarting,omitempty"`
	Signal        int       `toml:"signal,omitempty" json:"signal,omitempty"`
	ExitedAt      time.Time `toml:"exited_at,omitempty" json:"exited_at"`
}

type StopMachineInput struct {
	ID      string   `toml:"id,omitempty" json:"id,omitempty"`
	Signal  string   `toml:"signal,omitempty" json:"signal,omitempty" enums:"SIGHUP,SIGINT,SIGQUIT,SIGKILL,SIGUSR1,SIGUSR2,SIGTERM"`
	Timeout Duration `toml:"timeout,omitempty" json:"timeout,omitempty" swaggertype:"string" example:"10s"`
}

type RestartMachineInput struct {
	ID               string        `toml:"id,omitempty" json:"id,omitempty"`
	Signal           string        `toml:"signal,omitempty" json:"signal,omitempty" enums:"SIGHUP,SIGINT,SIGQUIT,SIGKILL,SIGUSR1,SIGUSR2,SIGTERM"`
	Timeout          time.Duration `toml:"timeout,omitempty" json:"timeout,omitempty"`
	ForceStop        bool          `toml:"force_stop,omitempty" json:"force_stop,omitempty"`
	SkipHealthChecks bool          `toml:"skip_health_checks,omitempty" json:"skip_health_checks,omitempty"`
}

type MachineIP struct {
	Family   string
	Kind     string
	IP       string
	MaskSize int
}

type RemoveMachineInput struct {
	ID   string `toml:"id,omitempty" json:"id,omitempty"`
	Kill bool   `toml:"kill,omitempty" json:"kill,omitempty"`
}

type MachineRestartPolicy string

var (
	MachineRestartPolicyNo        MachineRestartPolicy = "no"
	MachineRestartPolicyOnFailure MachineRestartPolicy = "on-failure"
	MachineRestartPolicyAlways    MachineRestartPolicy = "always"
)

type MachinePersistRootfs string

var (
	MachinePersistRootfsNone    MachinePersistRootfs = ""
	MachinePersistRootfsNever   MachinePersistRootfs = "never"
	MachinePersistRootfsRestart MachinePersistRootfs = "restart"
	MachinePersistRootfsAlways  MachinePersistRootfs = "always"
)

func (mpr *MachinePersistRootfs) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, []byte("null")) {
		return nil
	}

	var asString string
	err := json.Unmarshal(raw, &asString)
	if err == nil {
		switch asString {
		case "", "none":
			*mpr = MachinePersistRootfsNone
		case "never":
			*mpr = MachinePersistRootfsNever
		case "restart":
			*mpr = MachinePersistRootfsRestart
		case "always":
			*mpr = MachinePersistRootfsAlways
		default:
			return fmt.Errorf("invalid persist_rootfs string value \"%s\"", asString)
		}

		return nil
	}

	parentErr := err

	// Attempt protobuf integer based enums
	var asInt int
	switch err := json.Unmarshal(raw, &asInt); {
	case err != nil:
		return parentErr
	case asInt == 0:
		*mpr = MachinePersistRootfsNone
	case asInt == 1:
		*mpr = MachinePersistRootfsNever
	case asInt == 2:
		*mpr = MachinePersistRootfsRestart
	case asInt == 3:
		*mpr = MachinePersistRootfsAlways
	default:
		return fmt.Errorf("invalid persist_rootfs int value %d", asInt)
	}

	return nil
}

type MachineRootfs struct {
	// Whether changes to the root filesystem are kept. `never` (default): reset on every start.
	// `restart`: kept across stops and restarts, reset when the Machine is updated. `always`: also
	// kept across updates
	Persist MachinePersistRootfs `toml:"persist,omitempty" json:"persist,omitempty" enums:"never,always,restart"`
	// Size of the root filesystem in GB. At most 50; 0 uses the default
	SizeGB uint64 `toml:"size_gb,omitempty" json:"size_gb,omitempty"`
}

type MachineCacheDrive struct {
	// Fixed size of the Machine-owned cache drive in MiB. At most 51200.
	// With a path, at least 16 MiB is required; without a path, 0 disables the device.
	SizeMB uint64 `toml:"size_mb,omitempty" json:"size_mb,omitempty"`
	// Optional absolute guest mount path. When set, the device is formatted as ext4.
	// Without a path, the device retains legacy raw block behavior. Contents survive
	// same-host restarts and path changes, but are discarded on size or mode changes,
	// relocation, recovery on another host, and cloning.
	Path string `toml:"path,omitempty" json:"path,omitempty"`
}

// MachineSpot configures spot pricing behavior for a Machine
type MachineSpot struct {
	// MaxPriceFraction is the maximum fraction of the full Machine price you will pay for this Machine. Range: (0, 1.0]
	MaxPriceFraction float64 `toml:"max_price_fraction,omitempty" json:"max_price_fraction,omitempty"`
}

// @description The Machine restart policy defines whether and how flyd restarts a Machine after its main process exits. See https://fly.io/docs/machines/guides-examples/machine-restart-policy/.
type MachineRestart struct {
	// * no - Never try to restart a Machine automatically when its main process exits, whether that’s on purpose or on a crash.
	// * always - Always restart a Machine automatically and never let it enter a stopped state, even when the main process exits cleanly.
	// * on-failure - Try up to MaxRetries times to automatically restart the Machine if it exits with a non-zero exit code. Default when no explicit policy is set, and for Machines with schedules.
	Policy MachineRestartPolicy `toml:"policy,omitempty" json:"policy,omitempty" enums:"no,always,on-failure"`
	// When policy is on-failure, the maximum number of times to attempt to restart the Machine before letting it stop.
	MaxRetries int `toml:"max_retries,omitempty" json:"max_retries,omitempty"`
}

type MachineMount struct {
	// Whether the volume is encrypted. Set by Fly.io; ignored in requests
	Encrypted bool `toml:"encrypted,omitempty" json:"encrypted,omitempty"`
	// Absolute path in the Machine to mount the volume at. Required
	Path string `toml:"path,omitempty" json:"path,omitempty"`
	// Size of the volume in GB. Set by Fly.io; ignored in requests
	SizeGb int `toml:"size_gb,omitempty" json:"size_gb,omitempty"`
	// ID (`vol_...`) or name of the volume to mount. Required. With a name, any unattached volume
	// of the app with that name in the Machine's region is used
	Volume string `toml:"volume,omitempty" json:"volume,omitempty"`
	// Name of the volume. Set by Fly.io; ignored in requests
	Name string `toml:"name,omitempty" json:"name,omitempty"`
	// Enable automatic volume extension: grow the volume when its usage exceeds this percentage.
	// Between 50 and 99; 0 disables it
	ExtendThresholdPercent int `toml:"extend_threshold_percent,omitempty" json:"extend_threshold_percent,omitempty"`
	// With automatic extension, grow the volume by this many GB at a time. Between 1 and 100.
	// Mutually exclusive with add_size_percent
	AddSizeGb int `toml:"add_size_gb,omitempty" json:"add_size_gb,omitempty"`
	// With automatic extension, never grow the volume beyond this size in GB. At most the
	// organization's maximum volume size
	SizeGbLimit int `toml:"size_gb_limit,omitempty" json:"size_gb_limit,omitempty"`

	// AddSizePercent grows by a percentage of the current size, rounded up to GiB.
	// It is mutually exclusive with AddSizeGb.
	AddSizePercent int `toml:"add_size_percent,omitempty" json:"add_size_percent,omitempty"`
	// MinAddSizeGb is the minimum proportional growth in GiB (default 1).
	MinAddSizeGb int `toml:"min_add_size_gb,omitempty" json:"min_add_size_gb,omitempty"`
}

type MachineGuest struct {
	// Kind of CPU: shared or performance. Performance CPUs must be enabled for the organization.
	// Defaults to shared when `guest` is omitted
	CPUKind string `toml:"cpu_kind,omitempty" json:"cpu_kind,omitempty"`
	// Number of CPUs: 1, 2, 4, 6, 8, 10, 12, 14, 16, 32, 64 or 128, up to 8 for shared and 16 for
	// performance CPUs unless the organization's limit is higher. Defaults to 1 when `guest` is
	// omitted
	CPUs int `toml:"cpus,omitempty" json:"cpus,omitempty"`
	// Memory in MiB. Must be a multiple of 256 for shared and 1024 for performance CPUs, and
	// between 256 and 2048 (shared) or 2048 and 8192 (performance) per CPU. Defaults to 256 when
	// `guest` is omitted
	MemoryMB int `toml:"memory_mb,omitempty" json:"memory_mb,omitempty"`
	// Memory in MiB that the Machine's memory limit can later be raised to without a restart, using
	// the memory endpoints. Must be at least memory_mb and at most the per-CPU maximum. The Machine
	// starts with a limit of memory_mb
	MaxMemoryMB int `toml:"max_memory_mb,omitempty" json:"max_memory_mb,omitempty"`
	// No longer supported. Requests that set it are rejected
	GPUs int `toml:"gpus,omitempty" json:"gpus,omitempty"`
	// No longer supported. Requests that set it are rejected
	GPUKind string `toml:"gpu_kind,omitempty" json:"gpu_kind,omitempty"`
	// Place the Machine on hosts dedicated to the organization under this ID. Only available to
	// organizations with dedicated hosts
	HostDedicationID string `toml:"host_dedication_id,omitempty" json:"host_dedication_id,omitempty"`
	// Host features the Machine's host must have, such as `storage-tier-1`. At most 16, each made
	// of lowercase letters, digits and dashes. Each feature must be enabled for the organization
	RequiredHostFeatures []string `toml:"required_host_features,omitempty" json:"required_host_features,omitempty"`
	// Deprecated: use MachineConfig.Rootfs instead
	PersistRootfs MachinePersistRootfs `toml:"persist_rootfs,omitempty" json:"persist_rootfs,omitempty" enums:"never,always,restart"`

	// Extra arguments appended to the guest kernel command line
	KernelArgs []string `toml:"kernel_args,omitempty" json:"kernel_args,omitempty"`
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
	"shared-cpu-6x": {CPUKind: "shared", CPUs: 6, MemoryMB: 6 * MIN_MEMORY_MB_PER_SHARED_CPU},
	"shared-cpu-8x": {CPUKind: "shared", CPUs: 8, MemoryMB: 8 * MIN_MEMORY_MB_PER_SHARED_CPU},

	"performance-1x":  {CPUKind: "performance", CPUs: 1, MemoryMB: 1 * MIN_MEMORY_MB_PER_CPU},
	"performance-2x":  {CPUKind: "performance", CPUs: 2, MemoryMB: 2 * MIN_MEMORY_MB_PER_CPU},
	"performance-4x":  {CPUKind: "performance", CPUs: 4, MemoryMB: 4 * MIN_MEMORY_MB_PER_CPU},
	"performance-6x":  {CPUKind: "performance", CPUs: 6, MemoryMB: 6 * MIN_MEMORY_MB_PER_CPU},
	"performance-8x":  {CPUKind: "performance", CPUs: 8, MemoryMB: 8 * MIN_MEMORY_MB_PER_CPU},
	"performance-10x": {CPUKind: "performance", CPUs: 10, MemoryMB: 10 * MIN_MEMORY_MB_PER_CPU},
	"performance-12x": {CPUKind: "performance", CPUs: 12, MemoryMB: 12 * MIN_MEMORY_MB_PER_CPU},
	"performance-14x": {CPUKind: "performance", CPUs: 14, MemoryMB: 14 * MIN_MEMORY_MB_PER_CPU},
	"performance-16x": {CPUKind: "performance", CPUs: 16, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},

	"a100-40gb": {GPUKind: "a100-pcie-40gb", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"a100-80gb": {GPUKind: "a100-sxm4-80gb", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"l40s":      {GPUKind: "l40s", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
	"a10":       {GPUKind: "a10", GPUs: 1, CPUKind: "performance", CPUs: 8, MemoryMB: 16 * MIN_MEMORY_MB_PER_CPU},
}

type MachineMetrics struct {
	// Port in the Machine to scrape metrics from
	Port int `toml:"port,omitempty" json:"port,omitempty"`
	// HTTP path to scrape. Defaults to `/`
	Path string `toml:"path,omitempty" json:"path,omitempty"`
	// Scrape over HTTPS instead of HTTP
	Https bool `toml:"https,omitempty" json:"https,omitempty"`
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

type MachineCheck struct {
	// The port to connect to. Required for top-level checks
	Port *int `toml:"port,omitempty" json:"port,omitempty"`
	// tcp or http
	Type *string `toml:"type,omitempty" json:"type,omitempty"`
	// Kind of the check (informational, readiness)
	Kind *MachineCheckKind `toml:"kind,omitempty" json:"kind,omitempty" enums:"informational,readiness"`
	// The time between connectivity checks
	Interval *Duration `toml:"interval,omitempty" json:"interval,omitempty" swaggertype:"string" example:"15s"`
	// The maximum time a connection can take before being reported as failing its health check
	Timeout *Duration `toml:"timeout,omitempty" json:"timeout,omitempty" swaggertype:"string" example:"2s"`
	// The time to wait after a VM starts before checking its health
	GracePeriod *Duration `toml:"grace_period,omitempty" json:"grace_period,omitempty" swaggertype:"string" example:"1s"`
	// For http checks, the HTTP method to use to when making the request
	HTTPMethod *string `toml:"method,omitempty" json:"method,omitempty"`
	// For http checks, the path to send the request to
	HTTPPath *string `toml:"path,omitempty" json:"path,omitempty"`
	// For http checks, whether to use http or https
	HTTPProtocol *string `toml:"protocol,omitempty" json:"protocol,omitempty"`
	// For http checks with https protocol, whether or not to verify the TLS certificate
	HTTPSkipTLSVerify *bool `toml:"tls_skip_verify,omitempty" json:"tls_skip_verify,omitempty"`
	// If the protocol is https, the hostname to use for TLS certificate validation
	HTTPTLSServerName *string `toml:"tls_server_name,omitempty" json:"tls_server_name,omitempty"`
	// For http checks, headers to send with the request
	HTTPHeaders []MachineHTTPHeader `toml:"headers,omitempty" json:"headers,omitempty"`
}

type MachineServiceCheck struct {
	// The port to connect to. Defaults to the service's internal_port
	Port *int `toml:"port,omitempty" json:"port,omitempty"`
	// tcp or http
	Type *string `toml:"type,omitempty" json:"type,omitempty"`
	// The time between connectivity checks
	Interval *Duration `toml:"interval,omitempty" json:"interval,omitempty" swaggertype:"string" example:"15s"`
	// The maximum time a connection can take before being reported as failing its health check
	Timeout *Duration `toml:"timeout,omitempty" json:"timeout,omitempty" swaggertype:"string" example:"2s"`
	// The time to wait after a VM starts before checking its health
	GracePeriod *Duration `toml:"grace_period,omitempty" json:"grace_period,omitempty" swaggertype:"string" example:"1s"`
	// For http checks, the HTTP method to use to when making the request
	HTTPMethod *string `toml:"method,omitempty" json:"method,omitempty"`
	// For http checks, the path to send the request to
	HTTPPath *string `toml:"path,omitempty" json:"path,omitempty"`
	// For http checks, whether to use http or https
	HTTPProtocol *string `toml:"protocol,omitempty" json:"protocol,omitempty"`
	// For http checks with https protocol, whether or not to verify the TLS certificate
	HTTPSkipTLSVerify *bool `toml:"tls_skip_verify,omitempty" json:"tls_skip_verify,omitempty"`
	// If the protocol is https, the hostname to use for TLS certificate validation
	HTTPTLSServerName *string `toml:"tls_server_name,omitempty" json:"tls_server_name,omitempty"`
	// For http checks, headers to send with the request
	HTTPHeaders []MachineHTTPHeader `toml:"headers,omitempty" json:"headers,omitempty"`
}

// @description For http checks, an array of objects with string field Name and array of strings field Values. The key/value pairs specify header and header values that will get passed with the check call.
type MachineHTTPHeader struct {
	// The header name
	Name string `toml:"name,omitempty" json:"name,omitempty"`
	// The header value
	Values []string `toml:"values,omitempty" json:"values,omitempty"`
}

type ConsulCheckStatus string

const (
	Critical ConsulCheckStatus = "critical"
	Warning  ConsulCheckStatus = "warning"
	Passing  ConsulCheckStatus = "passing"
)

type MachineCheckStatus struct {
	Name      string            `toml:"name,omitempty" json:"name,omitempty"`
	Status    ConsulCheckStatus `toml:"status,omitempty" json:"status,omitempty"`
	Output    string            `toml:"output,omitempty" json:"output,omitempty"`
	UpdatedAt *time.Time        `toml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type MachinePort struct {
	// Public port. Use either port, or start_port and end_port
	Port *int `toml:"port,omitempty" json:"port,omitempty"`
	// First public port of a range
	StartPort *int `toml:"start_port,omitempty" json:"start_port,omitempty"`
	// Last public port of a range
	EndPort *int `toml:"end_port,omitempty" json:"end_port,omitempty"`
	// How Fly Proxy handles connections before forwarding them: `tls` terminates TLS, `pg_tls`
	// terminates PostgreSQL TLS, `http` handles HTTP, `proxy_proto` adds a PROXY protocol header.
	// Without handlers, raw TCP is forwarded
	Handlers []string `toml:"handlers,omitempty" json:"handlers,omitempty"`
	// Redirect plain HTTP requests to HTTPS
	ForceHTTPS bool `toml:"force_https,omitempty" json:"force_https,omitempty"`
	// Options for the `tls` handler
	TLSOptions *TLSOptions `toml:"tls_options,omitempty" json:"tls_options,omitempty"`
	// Options for the `http` handler
	HTTPOptions *HTTPOptions `toml:"http_options,omitempty" json:"http_options,omitempty"`
	// Options for the `proxy_proto` handler
	ProxyProtoOptions *ProxyProtoOptions `toml:"proxy_proto_options,omitempty" json:"proxy_proto_options,omitempty"`
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

	switch portRangeCount {
	case 2:
		return !httpInRange || !httpsInRange
	case 1:
		return !httpInRange && !httpsInRange
	}

	return false
}

type ProxyProtoOptions struct {
	// PROXY protocol version: v1 (default) or v2
	Version string `toml:"version,omitempty" json:"version,omitempty"`
	// With version v2, add the Fly-Src header and its signature to the PROXY protocol header
	FlySrc bool `toml:"fly_src,omitempty" json:"fly_src,omitempty"`
}

type TLSOptions struct {
	// ALPN protocols to offer, e.g. `h2`, `http/1.1`. Defaults to `h2` and `http/1.1`
	ALPN []string `toml:"alpn,omitempty" json:"alpn,omitempty"`
	// TLS versions to accept: `TLSv1.2`, `TLSv1.3`. Defaults to both
	Versions []string `toml:"versions,omitempty" json:"versions,omitempty"`
	// Use a self-signed certificate when there is no certificate for the requested hostname
	DefaultSelfSigned *bool `toml:"default_self_signed,omitempty" json:"default_self_signed,omitempty"`
}

type ReplayCache struct {
	// Requests whose path starts with this prefix use the rule. May start with a hostname, e.g.
	// `example.com/api/*`
	PathPrefix string `toml:"path_prefix" json:"path_prefix"`
	// How long a cached replay is used, in seconds
	TTLSeconds int `toml:"ttl_seconds" json:"ttl_seconds"`
	// Currently either "cookie" or "header"
	Type string `toml:"type" json:"type" enums:"cookie,header"`
	// Name of the cookie or header to key the cache on
	Name string `toml:"name" json:"name"`
	// Let clients skip the cache by sending the `fly-replay-cache-control: skip` header
	AllowBypass bool `toml:"allow_bypass,omitempty" json:"allow_bypass,omitempty"`
}

type HTTPOptions struct {
	// Compress response bodies. Defaults to true
	Compress *bool `toml:"compress,omitempty" json:"compress,omitempty"`
	// Changes Fly Proxy makes to responses
	Response *HTTPResponseOptions `toml:"response,omitempty" json:"response,omitempty"`
	// Use HTTP/2 to talk to the Machine
	H2Backend *bool `toml:"h2_backend,omitempty" json:"h2_backend,omitempty"`
	// Seconds a client connection can be idle between requests before it's closed. Defaults to 300
	IdleTimeout *uint32 `toml:"idle_timeout,omitempty" json:"idle_timeout,omitempty"`
	// Seconds Fly Proxy waits for a client to send request headers. Defaults to 5
	HeadersReadTimeout *uint32 `toml:"headers_read_timeout,omitempty" json:"headers_read_timeout,omitempty"`
	// Cache `fly-replay` responses from the Machine: once a client is replayed, its further
	// requests are replayed directly, keyed on a cookie or header, without asking the Machine again
	ReplayCache []ReplayCache `toml:"replay_cache,omitempty" json:"replay_cache,omitempty"`
}

type HTTPResponseOptions struct {
	// Response headers to set. A string sets the header, a list of strings sets it to multiple
	// values, and false removes it
	Headers map[string]any `toml:"headers,omitempty" json:"headers,omitempty"`
	// Don't let Fly Proxy add or change any response headers
	Pristine *bool `toml:"pristine,omitempty" json:"pristine,omitempty"`
}

type MachineService struct {
	// Protocol: tcp or udp. Defaults to tcp
	Protocol string `toml:"protocol,omitempty" json:"protocol,omitempty"`
	// Port in the Machine that Fly Proxy forwards traffic to. Defaults to 8080
	InternalPort int `toml:"internal_port,omitempty" json:"internal_port,omitempty"`
	// Accepts a string (new format) or a boolean (old format). For backward compatibility with older clients, the API continues to use booleans for "off" and "stop" in responses.
	// * "off" or false - Do not autostop the Machine.
	// * "stop" or true - Automatically stop the Machine.
	// * "suspend" - Automatically suspend the Machine, falling back to a full stop if this is not possible.
	Autostop *MachineAutostop `toml:"autostop,omitempty" json:"autostop,omitempty" swaggertype:"string" enums:"off,stop,suspend"`
	// Whether Fly Proxy starts stopped or suspended Machines when requests arrive. Defaults to true
	Autostart *bool `toml:"autostart,omitempty" json:"autostart,omitempty"`
	// When autostop is on, the number of Machines Fly Proxy keeps running in the app's primary
	// region (`PRIMARY_REGION` env var). Machines in other regions can all be stopped
	MinMachinesRunning *int `toml:"min_machines_running,omitempty" json:"min_machines_running,omitempty"`
	// Public ports that Fly Proxy accepts traffic on
	Ports []MachinePort `toml:"ports,omitempty" json:"ports,omitempty"`
	// An optional list of service checks
	Checks []MachineServiceCheck `toml:"checks,omitempty" json:"checks,omitempty"`
	// Load limits Fly Proxy uses to balance traffic across Machines
	Concurrency *MachineServiceConcurrency `toml:"concurrency,omitempty" json:"concurrency,omitempty"`
	// Currently has no effect
	ForceInstanceKey *string `toml:"force_instance_key" json:"force_instance_key"`
	// Currently has no effect
	ForceInstanceDescription *string `toml:"force_instance_description,omitempty" json:"force_instance_description,omitempty"`
}

type MachineServiceConcurrency struct {
	// What to count: `connections` (default) or `requests`
	Type string `toml:"type,omitempty" json:"type,omitempty"`
	// Fly Proxy sends no more than this many connections or requests to a Machine at once. 0 means
	// no hard limit
	HardLimit int `toml:"hard_limit,omitempty" json:"hard_limit,omitempty"`
	// Above this many connections or requests, Fly Proxy prefers other Machines. Defaults to 20,
	// and is lowered to hard_limit if higher
	SoftLimit int `toml:"soft_limit,omitempty" json:"soft_limit,omitempty"`
}

type ContainerStatus struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type MachineConfig struct {
	// Fields managed from fly.toml
	// If you add anything here, ensure appconfig.Config.ToMachine() is updated

	// An object filled with key/value pairs to be set as environment variables
	Env map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	// Arguments for init, Fly.io's footprint inside the Machine, which controls how your own code is run
	Init MachineInit `toml:"init,omitempty" json:"init"`
	// The resources (CPUs, memory, etc.) allocated to the Machine
	Guest *MachineGuest `toml:"guest,omitempty" json:"guest,omitempty"`
	// Key/value pairs of Machine metadata. Fly.io uses metadata internally for routing, process groups, and clusters
	Metadata map[string]string `toml:"metadata,omitempty" json:"metadata,omitempty"`
	// Persistent volumes to mount in the Machine. A Machine can mount at most one volume, which must be on
	// the same host. The mounted volume cannot be changed by updating the Machine
	Mounts []MachineMount `toml:"mounts,omitempty" json:"mounts,omitempty"`
	// Network services that Fly Proxy routes traffic to
	Services []MachineService `toml:"services,omitempty" json:"services,omitempty"`
	// An optional metrics endpoint for Prometheus on Fly.io to scrape
	Metrics *MachineMetrics `toml:"metrics,omitempty" json:"metrics,omitempty"`
	// An optional object that defines one or more named top-level checks. The key for each check is the check name.
	Checks map[string]MachineCheck `toml:"checks,omitempty" json:"checks,omitempty"`
	// Static files to serve under a URL prefix. Fly Proxy routes matching requests to a file server
	// inside the Machine, bypassing your application's own process; the Machine must be running. The
	// files are read either from a path inside the Machine or from a Tigris bucket that belongs to the
	// app; no bucket is created for you
	Statics []*Static `toml:"statics,omitempty" json:"statics,omitempty"`

	// Set by fly deploy or fly machines commands

	// The docker image to run
	Image string `toml:"image,omitempty" json:"image,omitempty"`
	// Files to write into the Machine, from either a raw value or an app secret
	Files []*File `toml:"files,omitempty" json:"files,omitempty"`

	// The following fields can only be set or updated by `fly machines run|update` commands
	// "fly deploy" must preserve them, if you add anything here, ensure it is propagated on deploys

	// Run the Machine at an interval: one of hourly, daily, weekly or monthly. The interval starts when
	// the Machine is created
	Schedule string `toml:"schedule,omitempty" json:"schedule,omitempty"`
	// Optional boolean telling the Machine to destroy itself once it’s complete (default false)
	AutoDestroy bool `toml:"auto_destroy,omitempty" json:"auto_destroy,omitempty"`
	// Whether and how the Machine is restarted after its main process exits
	Restart *MachineRestart `toml:"restart,omitempty" json:"restart,omitempty"`
	// DNS settings for the Machine
	DNS *DNSConfig `toml:"dns,omitempty" json:"dns,omitempty"`
	// Multiple processes to run within this one Machine, all from the Machine's image. Unrelated to the
	// process groups in fly.toml, which run on separate Machines. To run processes from different
	// images, use containers instead
	Processes []MachineProcess `toml:"processes,omitempty" json:"processes,omitempty"`

	// Standbys enable a machine to be a standby for another. In the event of a hardware failure,
	// the standby machine will be started.
	Standbys []string `toml:"standbys,omitempty" json:"standbys,omitempty"`

	// How the Machine is shut down gracefully: the signal sent to it and how long to wait before a hard shutdown
	StopConfig *StopConfig `toml:"stop_config,omitempty" json:"stop_config,omitempty"`

	// Containers are a list of containers that will run in the machine, each from its own image.
	Containers []*ContainerConfig `toml:"containers,omitempty" json:"containers,omitempty"`

	// Volumes describe the set of volumes that can be attached to the machine. Used in conjuction
	// with containers
	Volumes []*VolumeConfig `toml:"volumes,omitempty" json:"volumes,omitempty"`

	// Beta: the size of the Machine's root filesystem and whether it persists. This field may be
	// removed in the future
	Rootfs *MachineRootfs `toml:"rootfs,omitempty" json:"rootfs,omitempty"`
	// Beta: a cache drive attached to the Machine. This field may be removed in the future
	CacheDrive *MachineCacheDrive `toml:"cache_drive,omitempty" json:"cache_drive,omitempty"`

	// Spot pricing settings for the Machine
	Spot *MachineSpot `toml:"spot,omitempty" json:"spot,omitempty"`

	// Deprecated: use Guest instead
	VMSize string `toml:"size,omitempty" json:"size,omitempty"`
	// Deprecated: use Service.Autostart instead
	DisableMachineAutostart *bool `toml:"disable_machine_autostart,omitempty" json:"disable_machine_autostart,omitempty"`
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
	// The path inside the Machine, or inside tigris_bucket when set, where the files to serve are located
	GuestPath string `toml:"guest_path" json:"guest_path" validate:"required"`
	// The URL prefix under which the files are served
	UrlPrefix string `toml:"url_prefix" json:"url_prefix" validate:"required"`
	// An existing Tigris bucket belonging to the app to read the files from, instead of the Machine's
	// filesystem. The files are still served through the Machine
	TigrisBucket string `toml:"tigris_bucket" json:"tigris_bucket"`
	// The document served for requests to the URL prefix or any of its subfolders. Only works with tigris_bucket
	IndexDocument string `toml:"index_document" json:"index_document"`
}

type MachineInit struct {
	// Command to run instead of the image's ENTRYPOINT and CMD. Takes precedence over entrypoint
	// and cmd
	Exec []string `toml:"exec,omitempty" json:"exec,omitempty"`
	// Replaces the image's ENTRYPOINT
	Entrypoint []string `toml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	// Replaces the image's CMD. It's appended to the entrypoint
	Cmd []string `toml:"cmd,omitempty" json:"cmd,omitempty"`
	// Currently has no effect
	Tty bool `toml:"tty,omitempty" json:"tty,omitempty"`
	// Size in MiB of a swap device to give the Machine. Machines with swap cannot be suspended
	SwapSizeMB *int `toml:"swap_size_mb,omitempty" json:"swap_size_mb,omitempty"`
	// Currently has no effect. Use `guest.kernel_args` instead
	KernelArgs []string `toml:"kernel_args,omitempty" json:"kernel_args,omitempty"`
}

type DNSConfig struct {
	// Currently has no effect
	SkipRegistration bool `toml:"skip_registration,omitempty" json:"skip_registration,omitempty"`
	// Nameservers to use instead of the default, Fly.io's private DNS server `fdaa::3`
	Nameservers []string `toml:"nameservers,omitempty" json:"nameservers,omitempty"`
	// Search domains for /etc/resolv.conf
	Searches []string `toml:"searches,omitempty" json:"searches,omitempty"`
	// Options for /etc/resolv.conf
	Options []dnsOption `toml:"options,omitempty" json:"options,omitempty"`
	// Forward DNS queries for some domains to other DNS servers
	DNSForwardRules []dnsForwardRule `toml:"dns_forward_rules,omitempty" json:"dns_forward_rules,omitempty"`
	// Hostname of the Machine. Defaults to the Machine ID. Also added to /etc/hosts with the
	// Machine's 6PN address
	Hostname string `toml:"hostname,omitempty" json:"hostname,omitempty"`
	// Fully qualified domain name added to /etc/hosts for `hostname`
	HostnameFqdn string `toml:"hostname_fqdn,omitempty" json:"hostname_fqdn,omitempty"`
}

type dnsForwardRule struct {
	// Domain to forward queries for, including subdomains. Must end in a dot, e.g. `example.com.`
	Basename string `toml:"basename,omitempty" json:"basename,omitempty"`
	// Address and port of the DNS server to forward queries to over TCP, e.g. `10.0.0.1:53`
	Addr string `toml:"addr,omitempty" json:"addr,omitempty"`
}

type dnsOption struct {
	// Option name, e.g. `ndots`
	Name string `toml:"name,omitempty" json:"name,omitempty"`
	// Option value, e.g. `5`
	Value string `toml:"value,omitempty" json:"value,omitempty"`
}

type StopConfig struct {
	// How long to wait for the Machine to exit after sending the signal, before killing it. At most
	// 5 minutes (1 minute for spot Machines). Defaults to 5s (60s for spot Machines)
	Timeout *Duration `toml:"timeout,omitempty" json:"timeout,omitempty" swaggertype:"string" example:"10s"`
	// Signal to send to the Machine's processes. Defaults to SIGINT
	Signal *string `toml:"signal,omitempty" json:"signal,omitempty" enums:"SIGHUP,SIGINT,SIGQUIT,SIGKILL,SIGUSR1,SIGUSR2,SIGTERM"`
}

// @description A file that will be written to the Machine. One of RawValue or SecretName must be set.
type File struct {
	// GuestPath is the path on the machine where the file will be written and must be an absolute path.
	// For example: /full/path/to/file.json
	GuestPath string `toml:"guest_path,omitempty" json:"guest_path,omitempty"`

	// The base64 encoded string of the file contents.
	RawValue *string `toml:"raw_value,omitempty" json:"raw_value,omitempty"`

	// The name of the secret that contains the base64 encoded file contents.
	SecretName *string `toml:"secret_name,omitempty" json:"secret_name,omitempty"`

	// The name of an image to use the OCI image config as the file contents.
	ImageConfig *string `json:"image_config,omitempty"`

	// Mode bits used to set permissions on this file as accepted by chmod(2).
	Mode uint32 `toml:"mode,omitempty" json:"mode,omitempty"`
}

type ContainerConfig struct {
	// Name is used to identify the container in the machine.
	Name string `toml:"name" json:"name"`

	// Image is the docker image to run.
	Image string `toml:"image" json:"image"`

	// Image Config overrides - these fields are used to override the image configuration.
	// If not provided, the image configuration will be used.
	// ExecOverride is used to override the default command of the image.
	ExecOverride []string `toml:"exec,omitempty" json:"exec,omitempty"`
	// EntrypointOverride is used to override the default entrypoint of the image.
	EntrypointOverride []string `toml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	// CmdOverride is used to override the default command of the image.
	CmdOverride []string `toml:"cmd,omitempty" json:"cmd,omitempty"`
	// UserOverride is used to override the default user of the image.
	UserOverride string `toml:"user,omitempty" json:"user,omitempty"`
	// ExtraEnv is used to add additional environment variables to the container.
	ExtraEnv map[string]string `toml:"env,omitempty" json:"env,omitempty"`

	// Secrets can be provided at the process level to explicitly indicate which secrets should be
	// used for the process. If not provided, the secrets provided at the machine level will be used.
	Secrets []MachineSecret `toml:"secrets,omitempty" json:"secrets,omitempty"`

	// EnvFrom can be provided to set environment variables from machine fields.
	EnvFrom []EnvFrom `toml:"env_from,omitempty" json:"env_from,omitempty"`

	// Files are files that will be written to the container file system.
	Files []*File `toml:"files,omitempty" json:"files,omitempty"`

	// Restart is used to define the restart policy for the container.
	Restart *MachineRestart `toml:"restart,omitempty" json:"restart,omitempty"`

	// Stop is used to define the signal and timeout for stopping the container.
	Stop *StopConfig `toml:"stop,omitempty" json:"stop,omitempty"`

	// DependsOn can be used to define dependencies between containers. The container will only be
	// started after all of its dependent conditions have been satisfied.
	DependsOn []ContainerDependency `toml:"depends_on,omitempty" json:"depends_on,omitempty"`

	// Healthchecks determine the health of your containers. Healthchecks can use HTTP, TCP or an Exec command.
	Healthchecks []ContainerHealthcheck `toml:"healthchecks,omitempty" json:"healthchecks,omitempty"`

	// Set of mounts added to the container. These must reference a volume in the machine config via its name.
	Mounts []ContainerMount `toml:"mounts,omitempty" json:"mounts,omitempty"`
}

type ContainerMount struct {
	// The name of the volume. Must exist in the volumes field in the machine configuration
	Name string `toml:"name" json:"name"`
	// The path to mount the volume within the container
	Path string `toml:"path" json:"path"`
}

type ContainerDependency struct {
	// Name of the container this one depends on
	Name string `toml:"name" json:"name"`
	// When the container can start: once the dependency has `started`, is `healthy`, or has
	// `exited_successfully`
	Condition ContainerDependencyCondition `toml:"condition" json:"condition" enums:"exited_successfully,healthy,started"`
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
	// Check that an HTTP request succeeds
	HTTP *HTTPHealthcheck `toml:"http,omitempty" json:"http,omitempty"`
	// Check that a TCP connection can be opened
	TCP *TCPHealthcheck `toml:"tcp,omitempty" json:"tcp,omitempty"`
	// Check that a command run in the container exits with status 0
	Exec *ExecHealthcheck `toml:"exec,omitempty" json:"exec,omitempty"`
}

type ContainerHealthcheck struct {
	// The name of the check. Must be unique within the container.
	Name string `toml:"name" json:"name"`
	// The time in seconds between executing the defined check.
	Interval int64 `toml:"interval,omitempty" json:"interval,omitempty"`
	// The time in seconds to wait after a container starts before checking its health.
	GracePeriod int64 `toml:"grace_period,omitempty" json:"grace_period,omitempty"`
	// The number of times the check must succeeed before considering the container healthy.
	SuccessThreshold int32 `toml:"success_threshold,omitempty" json:"success_threshold,omitempty"`
	// The number of times the check must fail before considering the container unhealthy.
	FailureThreshold int32 `toml:"failure_threshold,omitempty" json:"failure_threshold,omitempty"`
	// The time in seconds to wait for the check to complete.
	Timeout int64 `toml:"timeout,omitempty" json:"timeout,omitempty"`
	// Kind of healthcheck (readiness, liveness)
	Kind ContainerHealthcheckKind `toml:"kind,omitempty" json:"kind,omitempty"`
	// Unhealthy policy that determines what action to take if a container is deemed unhealthy
	Unhealthy UnhealthyPolicy `toml:"unhealthy,omitempty" json:"unhealthy,omitempty"`
	// The type of healthcheck
	ContainerHealthcheckType
}

type HTTPHealthcheck struct {
	// The port to connect to, often the same as internal_port
	Port int32 `toml:"port" json:"port"`
	// The HTTP method to use to when making the request
	Method string `toml:"method,omitempty" json:"method,omitempty"`
	// The path to send the request to
	Path string `toml:"path,omitempty" json:"path,omitempty"`
	// Additional headers to send with the request
	Headers []MachineHTTPHeader `toml:"headers,omitempty" json:"headers,omitempty"`
	// Whether to use http or https
	Scheme ContainerHealthcheckScheme `toml:"scheme,omitempty" json:"scheme,omitempty"`
	// If the protocol is https, whether or not to verify the TLS certificate
	TLSSkipVerify *bool `toml:"tls_skip_verify,omitempty" json:"tls_skip_verify,omitempty"`
	// If the protocol is https, the hostname to use for TLS certificate validation
	TLSServerName string `toml:"tls_server_name,omitempty" json:"tls_server_name,omitempty"`
}

type TCPHealthcheck struct {
	// The port to connect to, often the same as internal_port
	Port int32 `toml:"port" json:"port"`
}

type ExecHealthcheck struct {
	// The command to run to check the health of the container (e.g. ["cat", "/tmp/healthy"])
	Command []string `toml:"command" json:"command"`
}

type VolumeConfig struct {
	// The name of the volume. A volume must have a unique name within an app
	Name string `toml:"name" json:"name"`
	// The volume resource, provides configuration for the volume
	VolumeResource
}

type VolumeResource struct {
	TempDir *TempDirVolume `toml:"temp_dir,omitempty" json:"temp_dir,omitempty"`
	Image   string         `toml:"image,omitempty" json:"image,omitempty"`
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
	StorageType StorageType `toml:"storage_type" json:"storage_type"`
	// The size limit of the temp dir, only applicable when using disk backed storage.
	SizeMB uint64 `toml:"size_mb,omitempty" json:"size_mb,omitempty"`
}

type MachineLease struct {
	Status  string            `toml:"status,omitempty" json:"status,omitempty"`
	Data    *MachineLeaseData `toml:"data,omitempty" json:"data,omitempty"`
	Message string            `toml:"message,omitempty" json:"message,omitempty"`
	Code    string            `toml:"code,omitempty" json:"code,omitempty"`
}

type MachineLeaseData struct {
	Nonce     string `toml:"nonce,omitempty" json:"nonce,omitempty"`
	ExpiresAt int64  `toml:"expires_at,omitempty" json:"expires_at,omitempty"`
	Owner     string `toml:"owner,omitempty" json:"owner,omitempty"`
	Version   string `toml:"version,omitempty" json:"version,omitempty"`
}

type MachineStartResponse struct {
	Message       string `toml:"message,omitempty" json:"message,omitempty"`
	Status        string `toml:"status,omitempty" json:"status,omitempty"`
	PreviousState string `toml:"previous_state,omitempty" json:"previous_state,omitempty"`
}

type LaunchMachineInput struct {
	Config                  *MachineConfig `toml:"config,omitempty" json:"config,omitempty"`
	Region                  string         `toml:"region,omitempty" json:"region,omitempty"`
	Name                    string         `toml:"name,omitempty" json:"name,omitempty"`
	SkipLaunch              bool           `toml:"skip_launch,omitempty" json:"skip_launch,omitempty"`
	SkipServiceRegistration bool           `toml:"skip_service_registration,omitempty" json:"skip_service_registration,omitempty"`
	SkipSecrets             bool           `json:"skip_secrets"`
	MinSecretsVersion       *uint64        `json:"min_secrets_version,omitempty"`

	LeaseTTL int `toml:"lease_ttl,omitempty" json:"lease_ttl,omitempty"`

	// Client side only
	ID                  string `toml:"-" json:"-"`
	SkipHealthChecks    bool   `toml:"-" json:"-"`
	RequiresReplacement bool   `toml:"-" json:"-"`
	Timeout             int    `toml:"-" json:"-"`
}

type MachineProcess struct {
	// Command to run instead of the image's ENTRYPOINT and CMD. Takes precedence over entrypoint
	// and cmd
	ExecOverride []string `toml:"exec,omitempty" json:"exec,omitempty"`
	// Replaces the image's ENTRYPOINT for this process
	EntrypointOverride []string `toml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	// Replaces the image's CMD for this process. It's appended to the entrypoint
	CmdOverride []string `toml:"cmd,omitempty" json:"cmd,omitempty"`
	// User to run the process as, instead of the image's USER
	UserOverride string `toml:"user,omitempty" json:"user,omitempty"`
	// Environment variables for this process, in addition to the Machine's `env`
	ExtraEnv map[string]string `toml:"env,omitempty" json:"env,omitempty"`
	// Secrets can be provided at the process level to explicitly indicate which secrets should be
	// used for the process. If not provided, the secrets provided at the machine level will be used.
	Secrets []MachineSecret `toml:"secrets,omitempty" json:"secrets,omitempty"`
	// IgnoreAppSecrets can be set to true to ignore the secrets for the App the Machine belongs to
	// and only use the secrets provided at the process level. The default/legacy behavior is to use
	// the secrets provided at the App level.
	IgnoreAppSecrets bool `toml:"ignore_app_secrets,omitempty" json:"ignore_app_secrets,omitempty"`

	// EnvFrom can be provided to set environment variables from machine fields.
	EnvFrom []EnvFrom `toml:"env_from,omitempty" json:"env_from,omitempty"`
}

// @description A Secret needing to be set in the environment of the Machine. env_var is required
// and name can be used to reference a secret name where the environment variable is different
// from what was set originally using the API. NOTE: When secrets are provided on any process, it
// will override the secrets provided at the machine level.
type MachineSecret struct {
	// EnvVar is required and is the name of the environment variable that will be set from the
	// secret. It must be a valid environment variable name.
	EnvVar string `toml:"env_var" json:"env_var"`

	// Name is optional and when provided is used to reference a secret name where the EnvVar is
	// different from what was set as the secret name.
	Name string `toml:"name" json:"name"`
}

// @description EnvVar defines an environment variable to be populated from a machine field, env_var
// and field_ref are required.
type EnvFrom struct {
	// EnvVar is required and is the name of the environment variable that will be set from the
	// secret. It must be a valid environment variable name.
	EnvVar string `toml:"env_var" json:"env_var"`

	// FieldRef selects a field of the Machine: supports id, version, app_name, private_ip, region, image.
	FieldRef string `toml:"field_ref" json:"field_ref" enums:"id,version,app_name,private_ip,region,image"`
}

type MachineExecRequest struct {
	Container string `toml:"container,omitempty" json:"container,omitempty"`
	// Machine runs the command in the machine's own namespace instead of in a
	// container. It is mutually exclusive with Container.
	Machine bool   `toml:"machine,omitempty" json:"machine,omitempty"`
	Cmd     string `toml:"cmd,omitempty" json:"cmd,omitempty"`
	Stdin   string `toml:"stdin,omitempty" json:"stdin,omitempty"`
	Timeout int    `toml:"timeout,omitempty" json:"timeout,omitempty"`
}

type MachineExecResponse struct {
	ExitCode int32  `toml:"exit_code,omitempty" json:"exit_code,omitempty"`
	StdOut   string `toml:"stdout,omitempty" json:"stdout,omitempty"`
	StdErr   string `toml:"stderr,omitempty" json:"stderr,omitempty"`
}

type MachinePsResponse []ProcessStat

type ProcessStat struct {
	Pid           int32          `toml:"pid" json:"pid"`
	Stime         uint64         `toml:"stime" json:"stime"`
	Rtime         uint64         `toml:"rtime" json:"rtime"`
	Command       string         `toml:"command" json:"command"`
	Directory     string         `toml:"directory" json:"directory"`
	Cpu           uint64         `toml:"cpu" json:"cpu"`
	Rss           uint64         `toml:"rss" json:"rss"`
	ListenSockets []ListenSocket `toml:"listen_sockets" json:"listen_sockets"`
}

type ListenSocket struct {
	Proto   string `toml:"proto" json:"proto"`
	Address string `toml:"address" json:"address"`
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
