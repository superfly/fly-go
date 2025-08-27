package fly

import (
	"context"
	"net"
)

func (c *Client) AllocateEgressIPAddress(ctx context.Context, appName string, machineId string) (net.IP, net.IP, error) {
	query := `
		mutation($input: AllocateEgressIPAddressInput!) {
			allocateEgressIpAddress(input: $input) {
				v4,
				v6
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "allocate_egress_ip_address")
	req.Var("input", AllocateEgressIPAddressInput{AppID: appName, MachineID: machineId})

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return net.ParseIP(data.AllocateEgressIPAddress.V4), net.ParseIP(data.AllocateEgressIPAddress.V6), nil
}

func (c *Client) GetEgressIPAddresses(ctx context.Context, appName string) (map[string][]EgressIPAddress, error) {
	query := `
		query ($appName: String!) {
			app(name: $appName) {
				machines {
					nodes {
						id
						egressIpAddresses {
							nodes {
								id
								ip
								version
								region
							}
						}
					}
				}
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "get_egress_ip_addresses")
	req.Var("appName", appName)

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	ret := make(map[string][]EgressIPAddress)
	for _, m := range data.App.Machines.Nodes {
		if len(m.EgressIpAddresses.Nodes) == 0 {
			continue
		}

		ret[m.ID] = make([]EgressIPAddress, len(m.EgressIpAddresses.Nodes))

		for i, ip := range m.EgressIpAddresses.Nodes {
			ret[m.ID][i] = *ip
		}
	}

	return ret, nil
}

func (c *Client) ReleaseEgressIPAddress(ctx context.Context, appName, machineID string) (net.IP, net.IP, error) {
	query := `
		mutation($input: ReleaseEgressIPAddressInput!) {
			releaseEgressIpAddress(input: $input) {
				v4
				v6
				clientMutationId
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "release_egress_ip_address")
	req.Var("input", ReleaseEgressIPAddressInput{AppID: appName, MachineID: machineID})

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return net.ParseIP(data.ReleaseEgressIPAddress.V4), net.ParseIP(data.ReleaseEgressIPAddress.V6), nil
}
