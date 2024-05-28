package fly

import (
	"context"
)

func (c *Client) CreateBuild(ctx context.Context, input CreateBuildInput) (*CreateBuildResponse, error) {
	_ = `# @genqlient
	mutation CreateBuild($input:CreateBuildInput!) {
		createBuild(input:$input) {
			id
			status
		}
	}
	`
	return CreateBuild(ctx, c.genqClient, input)
}

func (c *Client) FinishBuild(ctx context.Context, input FinishBuildInput) (*FinishBuildResponse, error) {
	_ = `# @genqlient
	mutation FinishBuild($input:FinishBuildInput!) {
		finishBuild(input:$input) {
			id
			status
			wallclockTimeMs
		}
	}
	`
	return FinishBuild(ctx, c.genqClient, input)
}
