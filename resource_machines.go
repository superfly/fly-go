package fly

import "context"

func (client *Client) GetMachine(ctx context.Context, machineId string) (*GqlMachine, error) {
	query := `
		query ($machineId: String!) {
			gqlmachine:machine(machineId: $machineId) {
				id
				name
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
	req.Var("machineId", machineId)
	ctx = ctxWithAction(ctx, "get_machine")

	data, err := client.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.GqlMachine, nil
}

func (client *Client) GetMachineWithEvents(ctx context.Context, machineId string) (*GqlMachine, error) {
	query := `
		query ($machineId: String!) {
			gqlmachine:machine(machineId: $machineId) {
				instanceId
				events {
					nodes {
					    body
						kind
						machineVersion {
							id
						}
						source
						status
						timestamp
					}
				}
			}
		}
	`

	req := client.NewRequest(query)
	req.Var("machineId", machineId)
	ctx = ctxWithAction(ctx, "get_machine_with_events")

	data, err := client.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.GqlMachine, nil
}
