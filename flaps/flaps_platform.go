package flaps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/superfly/fly-go"
)

func (f *Client) GetRegions(ctx context.Context, size string) ([]fly.Region, error) {
	ctx = contextWithAction(ctx, regionsGet)
	endpoint := "/platform/regions"
	if size != "" {
		endpoint += fmt.Sprintf("?size=%s", size)
	}
	regions := &struct{ Regions []fly.Region }{}
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, regions, nil); err != nil {
		return nil, fmt.Errorf("failed to get regions: %w", err)
	}
	return regions.Regions, nil
}

type Weights map[string]int64

type GetPlacementsRequest struct {
	VM              *fly.MachineGuest `json:"vm"`
	Region          string            `json:"region"`
	Count           int64             `json:"count"`
	VolumeName      string            `json:"volume_name"`
	VolumeSizeBytes uint64            `json:"volume_size_bytes"`
	Weights         *Weights          `json:"weights"`
	Size            string            `json:"size"`
	Org             string            `json:"org_slug"`
}

type RegionPlacement struct {
	Region      string
	Count       int
	Concurrency int
}

type GetPlacementsResponse struct {
	Regions []RegionPlacement
}

func (f *Client) GetPlacements(ctx context.Context, request *GetPlacementsRequest) ([]RegionPlacement, error) {
	ctx = contextWithAction(ctx, placementPost)
	endpoint := "/platform/placements"
	regions := &struct{ Regions []RegionPlacement }{}
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, request, regions, nil); err != nil {
		return nil, fmt.Errorf("failed to get placements: %w", err)
	}
	return regions.Regions, nil
}
