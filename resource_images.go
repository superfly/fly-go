package fly

import (
	"context"
	"fmt"
)

func (client *Client) GetLatestImageTag(ctx context.Context, repository string, snapshotId *string) (string, error) {
	query := `
		query($repository: String!, $snapshotId: ID) {
			latestImageTag(repository: $repository, snapshotId: $snapshotId)
		}
	`
	req := client.NewRequest(query)
	req.Var("repository", repository)
	req.Var("snapshotId", snapshotId)
	ctx = ctxWithAction(ctx, "get_latest_image_tag")

	data, err := client.RunWithContext(ctx, req)
	if err != nil {
		return "", err
	}

	return data.LatestImageTag, nil
}

func (client *Client) GetLatestImageDetails(ctx context.Context, image string, flyVersion string) (*ImageVersion, error) {
	query := `
		query($image: String!, $flyVersion: String) {
			latestImageDetails(image: $image, flyVersion: $flyVersion) {
			  registry
			  repository
			  tag
			  version
			  digest
			}
		}
	`

	req := client.NewRequest(query)
	ctx = ctxWithAction(ctx, "get_latest_image_details")
	req.Var("image", image)
	req.Var("flyVersion", flyVersion)

	data, err := client.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}
	return &data.LatestImageDetails, nil
}

func (c *Client) LatestImage(ctx context.Context, appName string) (string, error) {
	_ = `# @genqlient
	       query LatestImage($appName:String!) {
	               app(name:$appName) {
	                       currentReleaseUnprocessed {
	                               id
	                               version
	                               imageRef
	                       }
	               }
	       }
	      `
	resp, err := LatestImage(ctx, c.genqClient, appName)
	if err != nil {
		return "", err
	}
	if resp.App.CurrentReleaseUnprocessed.ImageRef == "" {
		return "", fmt.Errorf("current release not found for app %s", appName)
	}
	return resp.App.CurrentReleaseUnprocessed.ImageRef, nil
}
