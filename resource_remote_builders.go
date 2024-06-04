package fly

import "context"

func (client *Client) EnsureRemoteBuilder(ctx context.Context, orgID, appName, region string) (*GqlMachine, *App, error) {
	query := `
		mutation($input: EnsureMachineRemoteBuilderInput!) {
			ensureMachineRemoteBuilder(input: $input) {
				machine {
					id
					state
					ips {
						nodes {
							family
							kind
							ip
						}
					}
				},
				app {
					name
					organization {
						id
						slug
					}
				}
			}
		}
	`

	req := client.NewRequest(query)
	ctx = ctxWithAction(ctx, "ensure_remote_builder")

	input := EnsureRemoteBuilderInput{}
	if region != "" {
		input.Region = StringPointer(region)
	}
	if orgID != "" {
		input.OrganizationID = StringPointer(orgID)
	} else {
		input.AppName = StringPointer(appName)
	}
	req.Var("input", input)

	data, err := client.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return data.EnsureMachineRemoteBuilder.Machine, data.EnsureMachineRemoteBuilder.App, nil
}

func (client *Client) SetRemoteBuilder(ctx context.Context, appName string) error {
	query := `
		mutation($input: SetRemoteBuilderInput!) {
			setRemoteBuilder(input: $input) {
				app {
					name
					organization {
						id
						slug
					}
				}
			}
		}
	`

	req := client.NewRequest(query)
	ctx = ctxWithAction(ctx, "set_remote_builder")

	input := SetRemoteBuilderInput{
		AppName: StringPointer(appName),
	}
	req.Var("input", input)

	_, err := client.RunWithContext(ctx, req)
	return err
}
