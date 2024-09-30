package fly

import (
	"strconv"
	"time"
)

type Volume struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	State              string    `json:"state"`
	SizeGb             int       `json:"size_gb"`
	Region             string    `json:"region"`
	Zone               string    `json:"zone"`
	Encrypted          bool      `json:"encrypted"`
	AttachedMachine    *string   `json:"attached_machine_id"`
	AttachedAllocation *string   `json:"attached_alloc_id"`
	CreatedAt          time.Time `json:"created_at"`
	HostDedicationID   string    `json:"host_dedication_id"`
	SnapshotRetention  int       `json:"snapshot_retention"`
	AutoBackupEnabled  bool      `json:"auto_backup_enabled"`
	HostStatus         string    `json:"host_status,omitempty"`
}

func (v Volume) IsAttached() bool {
	return v.AttachedMachine != nil || v.AttachedAllocation != nil
}

type CreateVolumeRequest struct {
	Name              string `json:"name"`
	Region            string `json:"region"`
	SizeGb            *int   `json:"size_gb"`
	Encrypted         *bool  `json:"encrypted"`
	RequireUniqueZone *bool  `json:"require_unique_zone"`
	SnapshotRetention *int   `json:"snapshot_retention"`
	AutoBackupEnabled *bool  `json:"auto_backup_enabled"`

	// FSType sets the filesystem of this volume. The valid values are "ext4" and "raw".
	// Not setting the value results "ext4".
	FSType *string `json:"fstype"`

	// restore from snapshot
	SnapshotID *string `json:"snapshot_id"`
	// fork from remote volume
	SourceVolumeID *string `json:"source_volume_id"`

	// If the volume is going to be attached to a new machine, make the placement logic aware of it
	ComputeRequirements *MachineGuest `json:"compute"`
	ComputeImage        string        `json:"compute_image,omitempty"`
}

type UpdateVolumeRequest struct {
	SnapshotRetention *int  `json:"snapshot_retention"`
	AutoBackupEnabled *bool `json:"auto_backup_enabled"`
}

type VolumeSnapshot struct {
	ID            string    `json:"id"`
	Size          int       `json:"size"`
	Digest        string    `json:"digest"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
	RetentionDays *int      `json:"retention_days"`
}

type VolumeSnapshotGql struct {
	ID            string    `json:"id"`
	Size          string    `json:"size"`
	Digest        string    `json:"digest"`
	CreatedAt     time.Time `json:"createdAt"`
	Status        string    `json:"status"`
	RetentionDays *int      `json:"retentionDays"`
}

func NewVolumeSnapshotFrom(v VolumeSnapshotGql) VolumeSnapshot {
	size, _ := strconv.Atoi(v.Size)
	return VolumeSnapshot{
		ID:        v.ID,
		Size:      size,
		Digest:    v.Digest,
		CreatedAt: v.CreatedAt,
		Status:    v.Status,
	}
}
