package fly

import "context"

func (c *Client) GetMachine(ctx context.Context, machineId string) (*GqlMachine, error) {
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

	req := c.NewRequest(query)
	req.Var("machineId", machineId)
	ctx = ctxWithAction(ctx, "get_machine")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.GqlMachine, nil
}
