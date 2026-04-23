package fly

import (
	"context"
)

func (c *Client) GetOrgLimitedAccessTokens(ctx context.Context, orgSlug string) ([]LimitedAccessToken, error) {
	query := `
		query ($slug: String!) {
			orgLimitedAccessTokens: organization(slug: $slug) {
				limitedAccessTokens {
					nodes {
						id
						name
						expiresAt
						revokedAt
						user {
							email
						}
					}
				}
			}
		}
	`

	req := c.NewRequest(query)
	req.Var("slug", orgSlug)
	ctx = ctxWithAction(ctx, "get_org_limited_access_tokens")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	if data.OrgLimitedAccessTokens == nil {
		return nil, nil
	}

	return data.OrgLimitedAccessTokens.LimitedAccessTokens.Nodes, nil
}

func (c *Client) GetAppLimitedAccessTokens(ctx context.Context, appName string) ([]LimitedAccessToken, error) {
	query := `
		query ($appName: String!) {
			appLimitedAccessTokens: app(name: $appName) {
				limitedAccessTokens {
					nodes {
						id
						name
						token
						expiresAt
						revokedAt
						user {
							email
						}
					}
				}
			}
		}
	`

	req := c.NewRequest(query)
	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "get_app_limited_access_tokens")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	if data.AppLimitedAccessTokens == nil {
		return nil, nil
	}

	return data.AppLimitedAccessTokens.LimitedAccessTokens.Nodes, nil
}

func (c *Client) RevokeLimitedAccessToken(ctx context.Context, id string) error {
	query := `
		mutation($input:DeleteLimitedAccessTokenInput!) {
			deleteLimitedAccessToken(input: $input) {
				token
			}
		}
	`
	req := c.NewRequest(query)

	req.Var("input", map[string]any{
		"id": id,
	})
	ctx = ctxWithAction(ctx, "revoke_limited_access_token")

	_, err := c.RunWithContext(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
