package fly

import (
	"context"
	"net"
	"time"
)

func (c *Client) GetIPAddresses(ctx context.Context, appName string) ([]IPAddress, error) {
	query := `
		query ($appName: String!) {
			app(name: $appName) {
				ipAddresses {
					nodes {
						id
						address
						type
						region
						createdAt
						network {
							name
							organization {
								slug
							}
						}
						serviceName
					}
				}
				sharedIpAddress
			}
		}
	`

	req := c.NewRequest(query)
	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "get_ip_addresses")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	ips := data.App.IPAddresses.Nodes

	// ugly hack
	if data.App.SharedIPAddress != "" {
		ips = append(ips, IPAddress{
			ID:        "",
			Address:   data.App.SharedIPAddress,
			Type:      "shared_v4",
			Region:    "",
			CreatedAt: time.Time{},
		})
	}

	return ips, nil
}

func (c *Client) AllocateIPAddress(ctx context.Context, appName string, addrType string, region string, orgID string, network string) (*IPAddress, error) {
	query := `
		mutation($input: AllocateIPAddressInput!) {
			allocateIpAddress(input: $input) {
				ipAddress {
					id
					address
					type
					region
					createdAt
				}
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "allocate_ip_address")
	input := AllocateIPAddressInput{AppID: appName, Type: addrType, Region: region}

	if orgID != "" {
		input.OrganizationID = orgID
	}

	if network != "" {
		input.Network = network
	}

	req.Var("input", input)

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.AllocateIPAddress.IPAddress, nil
}

func (c *Client) AllocateSharedIPAddress(ctx context.Context, appName string) (net.IP, error) {
	query := `
		mutation($input: AllocateIPAddressInput!) {
			allocateIpAddress(input: $input) {
				app {
					sharedIpAddress
				}
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "allocate_shared_ip_address")
	req.Var("input", AllocateIPAddressInput{AppID: appName, Type: "shared_v4"})

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return net.ParseIP(data.AllocateIPAddress.App.SharedIPAddress), nil
}

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
								updatedAt
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

func (c *Client) ReleaseIPAddress(ctx context.Context, appName string, ip string) error {
	query := `
		mutation($input: ReleaseIPAddressInput!) {
			releaseIpAddress(input: $input) {
				clientMutationId
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "release_ip_address")
	req.Var("input", ReleaseIPAddressInput{AppID: &appName, IP: &ip})

	_, err := c.RunWithContext(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetAppScopedEgressIPAddresses(ctx context.Context, appName string) (map[string][]EgressIPAddress, error) {
	query := `
		query ($appName: String!) {
			app(name: $appName) {
				egressIpAddresses {
					nodes {
						id
						ip
						version
						region
						updatedAt
					}
				}
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "get_app_scoped_egress_ip_addresses")
	req.Var("appName", appName)

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	ret := make(map[string][]EgressIPAddress)
	for _, ip := range data.App.EgressIpAddresses.Nodes {
		if _, ok := ret[ip.Region]; !ok {
			ret[ip.Region] = make([]EgressIPAddress, 0)
		}

		ret[ip.Region] = append(ret[ip.Region], *ip)
	}

	return ret, nil
}

func (c *Client) AllocateAppScopedEgressIPAddress(ctx context.Context, appName string, region string) (net.IP, net.IP, error) {
	query := `
		mutation($input: AllocateEgressIPAddressInput!) {
			allocateEgressIpAddress(input: $input) {
				v4,
				v6
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "allocate_app_scoped_egress_ip_address")
	req.Var("input", AllocateEgressIPAddressInput{AppID: appName, Region: region})

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return net.ParseIP(data.AllocateEgressIPAddress.V4), net.ParseIP(data.AllocateEgressIPAddress.V6), nil
}

func (c *Client) ReleaseAppScopedEgressIPAddress(ctx context.Context, appName, ip string) error {
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
	ctx = ctxWithAction(ctx, "release_app_scoped_egress_ip_address")
	req.Var("input", ReleaseEgressIPAddressInput{AppID: appName, IP: ip})

	_, err := c.RunWithContext(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
