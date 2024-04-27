package fly

import (
	"context"
)

func (c *Client) GetAppNameFromVolume(ctx context.Context, volID string) (*string, error) {
	query := `
query($id: ID!) {
	volume: node(id: $id) {
		... on Volume {
			app {
				name
			}
		}
	}
}
	`

	req := c.NewRequest(query)

	req.Var("id", volID)
	ctx = ctxWithAction(ctx, "get_app_name_from_volume")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.Volume.App.Name, nil
}

func (c *Client) GetAppNameStateFromVolume(ctx context.Context, volID string) (*string, *string, error) {
	query := `
query GetAppNameStateFromVolume($id: ID!) {
	volume: node(id: $id) {
		... on Volume {
			app {
				name
			}
			state
		}
	}
}
	`

	req := c.NewRequest(query)

	req.Var("id", volID)
	ctx = ctxWithAction(ctx, "get_app_name_state_from_volume")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return &data.Volume.App.Name, &data.Volume.State, nil
}

func (c *Client) GetSnapshotsFromVolume(ctx context.Context, volID string) ([]VolumeSnapshot, error) {
	query := `
query GetSnapshotsFromVolume($id: ID!) {
	volume: node(id: $id) {
		... on Volume {
			snapshots {
				nodes {
					id
					size
					digest
					createdAt
				}
			}
		}
	}
}
	`

	req := c.NewRequest(query)

	req.Var("id", volID)
	ctx = ctxWithAction(ctx, "get_snapshots_from_volume")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}
	var snapshots []VolumeSnapshot
	for _, snapshot := range data.Volume.Snapshots.Nodes {
		snapshots = append(snapshots, NewVolumeSnapshotFrom(snapshot))
	}
	return snapshots, nil
}
