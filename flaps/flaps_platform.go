package flaps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/superfly/fly-go"
)

type RegionData struct {
	Regions []fly.Region `json:"Regions"`
	Nearest string       `json:"Nearest"`
}

func (f *Client) GetRegions(ctx context.Context) (*RegionData, error) {
	data := &RegionData{}
	ctx = contextWithAction(ctx, regionsGet)
	endpoint := "/platform/regions"
	if err := f._sendRequest(ctx, http.MethodGet, endpoint, nil, data, nil); err != nil {
		return nil, fmt.Errorf("failed to get regions: %w", err)
	}
	return data, nil
}

// Weights override default placement preferences.
// `region` (default 50) prefers regions closer to the target region.
// `spread` (default 1) prefers spreading placements across different hosts instead of packed on the same host.
type Weights map[string]int64

type GetPlacementsRequest struct {
	// Resource requirements for the Machine to simulate. Defaults to a performance-1x machine
	ComputeRequirements *fly.MachineGuest `json:"compute"`

	// Region expression for placement as a comma-delimited set of regions or aliases.
	// Defaults to "[region],any", to prefer the API endpoint's local region with any other region as fallback.
	Region string `json:"region"`

	// Number of machines to simulate placement.
	// Defaults to 0, which returns the org-specific limit for each region.
	Count uint64 `json:"count"`

	VolumeName      string `json:"volume_name"`
	VolumeSizeBytes uint64 `json:"volume_size_bytes"`

	// Optional weights to override default placement preferences.
	Weights *Weights `json:"weights"`

	Org string `json:"org_slug"`
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
	regions := &GetPlacementsResponse{}
	if err := f._sendRequest(ctx, http.MethodPost, endpoint, request, regions, nil); err != nil {
		return nil, fmt.Errorf("failed to get placements: %w", err)
	}
	return regions.Regions, nil
}
