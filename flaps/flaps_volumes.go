package flaps

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	fly "github.com/superfly/fly-go"
)

var destroyedVolumeStates = []string{"scheduling_destroy", "fork_cleanup", "waiting_for_detach", "pending_destroy", "destroying"}

func (f *Client) sendRequestVolumes(ctx context.Context, method, endpoint string, in, out interface{}, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/volumes%s", f.appName, endpoint)
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) GetAllVolumes(ctx context.Context) ([]fly.Volume, error) {
	listVolumesEndpoint := ""

	out := make([]fly.Volume, 0)
	ctx = contextWithAction(ctx, volumeList)

	err := f.sendRequestVolumes(ctx, http.MethodGet, listVolumesEndpoint, nil, &out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	return out, nil
}

func (f *Client) GetVolumes(ctx context.Context) ([]fly.Volume, error) {
	volumes, err := f.GetAllVolumes(ctx)
	if err != nil {
		return nil, err
	}

	volumes = slices.DeleteFunc(volumes, func(v fly.Volume) bool {
		return slices.Contains(destroyedVolumeStates, v.State)
	})

	return volumes, nil
}

func (f *Client) CreateVolume(ctx context.Context, req fly.CreateVolumeRequest) (*fly.Volume, error) {
	createVolumeEndpoint := ""

	out := new(fly.Volume)
	ctx = contextWithAction(ctx, volumeCreate)

	err := f.sendRequestVolumes(ctx, http.MethodPost, createVolumeEndpoint, req, out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}
	return out, nil
}

func (f *Client) UpdateVolume(ctx context.Context, volumeId string, req fly.UpdateVolumeRequest) (*fly.Volume, error) {
	updateVolumeEndpoint := fmt.Sprintf("/%s", volumeId)

	out := new(fly.Volume)
	ctx = contextWithAction(ctx, volumetUpdate)

	err := f.sendRequestVolumes(ctx, http.MethodPut, updateVolumeEndpoint, req, out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update volume: %w", err)
	}
	return out, nil
}

func (f *Client) GetVolume(ctx context.Context, volumeId string) (*fly.Volume, error) {
	getVolumeEndpoint := fmt.Sprintf("/%s", volumeId)

	out := new(fly.Volume)
	ctx = contextWithAction(ctx, volumeGet)

	err := f.sendRequestVolumes(ctx, http.MethodGet, getVolumeEndpoint, nil, out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume %s: %w", volumeId, err)
	}
	return out, nil
}

func (f *Client) GetVolumeSnapshots(ctx context.Context, volumeId string) ([]fly.VolumeSnapshot, error) {
	getVolumeSnapshotsEndpoint := fmt.Sprintf("/%s/snapshots", volumeId)

	out := make([]fly.VolumeSnapshot, 0)
	ctx = contextWithAction(ctx, volumeSnapshotList)

	err := f.sendRequestVolumes(ctx, http.MethodGet, getVolumeSnapshotsEndpoint, nil, &out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume %s snapshots: %w", volumeId, err)
	}
	return out, nil
}

func (f *Client) CreateVolumeSnapshot(ctx context.Context, volumeId string) error {
	ctx = contextWithAction(ctx, volumeSnapshotCreate)

	err := f.sendRequestVolumes(
		ctx, http.MethodPost, fmt.Sprintf("/%s/snapshots", volumeId),
		nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to snapshot %s: %w", volumeId, err)
	}
	return nil
}

type ExtendVolumeRequest struct {
	SizeGB int `json:"size_gb"`
}

type ExtendVolumeResponse struct {
	Volume       *fly.Volume `json:"volume"`
	NeedsRestart bool        `json:"needs_restart"`
}

func (f *Client) ExtendVolume(ctx context.Context, volumeId string, size_gb int) (*fly.Volume, bool, error) {
	extendVolumeEndpoint := fmt.Sprintf("/%s/extend", volumeId)

	req := ExtendVolumeRequest{
		SizeGB: size_gb,
	}

	out := new(ExtendVolumeResponse)
	ctx = contextWithAction(ctx, volumeExtend)

	err := f.sendRequestVolumes(ctx, http.MethodPut, extendVolumeEndpoint, req, out, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to extend volume %s: %w", volumeId, err)
	}
	return out.Volume, out.NeedsRestart, nil
}

func (f *Client) DeleteVolume(ctx context.Context, volumeId string) (*fly.Volume, error) {
	destroyVolumeEndpoint := fmt.Sprintf("/%s", volumeId)

	out := new(fly.Volume)
	ctx = contextWithAction(ctx, volumeDelete)

	err := f.sendRequestVolumes(ctx, http.MethodDelete, destroyVolumeEndpoint, nil, out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to destroy volume %s: %w", volumeId, err)
	}
	return out, nil
}
