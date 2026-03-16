package fly

import (
	"context"
)

func (c *Client) AttachPostgresCluster(ctx context.Context, input AttachPostgresClusterInput) (*AttachPostgresClusterPayload, error) {
	query := `
		mutation($input: AttachPostgresClusterInput!) {
			attachPostgresCluster(input: $input) {
				app {
					name
				}
				postgresClusterApp {
					name
				}
				environmentVariableName
				connectionString
				environmentVariableName
			}
		}
		`

	req := c.NewRequest(query)
	req.Var("input", input)
	ctx = ctxWithAction(ctx, "attach_postgres_cluster")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.AttachPostgresCluster, nil
}

func (c *Client) DetachPostgresCluster(ctx context.Context, input DetachPostgresClusterInput) error {
	query := `
		mutation($input: DetachPostgresClusterInput!) {
			detachPostgresCluster(input: $input) {
				clientMutationId
			}
		}
		`

	req := c.NewRequest(query)
	req.Var("input", input)
	ctx = ctxWithAction(ctx, "detach_postgres_cluster")

	_, err := c.RunWithContext(ctx, req)

	return err
}

func (c *Client) ListPostgresClusterAttachments(ctx context.Context, appName, postgresAppName string) ([]*PostgresClusterAttachment, error) {
	query := `
		query($appName: String!, $postgresAppName: String!) {
			postgresAttachments(appName: $appName, postgresAppName: $postgresAppName) {
				nodes {
					id
					databaseName
					databaseUser
					environmentVariableName
				}
		  }
		}
		`

	req := c.NewRequest(query)
	req.Var("appName", appName)
	req.Var("postgresAppName", postgresAppName)
	ctx = ctxWithAction(ctx, "list_postgres_cluster_attachments")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.PostgresAttachments.Nodes, nil
}

func (c *Client) EnablePostgresConsul(ctx context.Context, appName string) (*PostgresEnableConsulPayload, error) {
	const query = `
		mutation($appName: ID!) {
			enablePostgresConsul(input: {appId: $appName}) {
				consulUrl
			}
		}
	`
	req := c.NewRequest(query)
	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "enable_postgres_consul")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.EnablePostgresConsul, nil
}
