package flaps

import (
	"context"
	"fmt"
	"github.com/superfly/fly-go"
	"net/http"
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
