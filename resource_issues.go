package fly

import "context"

func (c *Client) GetAppHostIssues(ctx context.Context, appName string) ([]HostIssue, error) {
	query := `
		query($appName: String!) {
			apphostissues:app(name: $appName) {
				hostIssues {
					nodes {
						internalId
						message
						createdAt
						updatedAt
					}
				}
			}
		}
	`

	req := c.NewRequest(query)
	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "get_app_host_issues")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.AppHostIssues.HostIssues.Nodes, nil
}
