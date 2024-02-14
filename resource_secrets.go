package fly

import "context"

func (c *Client) SetSecrets(ctx context.Context, appName string, secrets map[string]string) (*Release, error) {
	query := `
		mutation($input: SetSecretsInput!) {
			setSecrets(input: $input) {
				release {
					id
					version
					reason
					description
					user {
						id
						email
						name
					}
					evaluationId
					createdAt
				}
			}
		}
	`

	input := SetSecretsInput{AppID: appName}
	for k, v := range secrets {
		input.Secrets = append(input.Secrets, SetSecretsInputSecret{Key: k, Value: v})
	}

	req := c.NewRequest(query)

	req.Var("input", input)
	ctx = ctxWithAction(ctx, "set_secrets")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.SetSecrets.Release, nil
}

func (c *Client) UnsetSecrets(ctx context.Context, appName string, keys []string) (*Release, error) {
	query := `
		mutation($input: UnsetSecretsInput!) {
			unsetSecrets(input: $input) {
				release {
					id
					version
					reason
					description
					user {
						id
						email
						name
					}
					evaluationId
					createdAt
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("input", UnsetSecretsInput{AppID: appName, Keys: keys})
	ctx = ctxWithAction(ctx, "unset_secrets")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.UnsetSecrets.Release, nil
}

func (c *Client) GetAppSecrets(ctx context.Context, appName string) ([]Secret, error) {
	query := `
		query ($appName: String!) {
			app(name: $appName) {
				secrets {
					name
					digest
					createdAt
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "get_app_secrets")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.App.Secrets, nil
}
