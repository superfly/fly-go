package fly

import "context"

func (c *Client) GetAppCertificates(ctx context.Context, appName string) ([]AppCertificateCompact, error) {
	query := `
		query($appName: String!) {
			appcertscompact:app(name: $appName) {
				certificates {
					nodes {
						createdAt
						hostname
						clientStatus
					}
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("appName", appName)
	ctx = ctxWithAction(ctx, "get_app_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.AppCertsCompact.Certificates.Nodes, nil
}

func (c *Client) CheckAppCertificate(ctx context.Context, appName, hostname string) (*AppCertificate, *HostnameCheck, error) {
	query := `
		mutation($input: CheckCertificateInput!) {
			checkCertificate(input: $input) {
				certificate {
					acmeDnsConfigured
					acmeAlpnConfigured
					configured
					certificateAuthority
					createdAt
					dnsProvider
					dnsValidationInstructions
					dnsValidationHostname
					dnsValidationTarget
					hostname
					id
					source
					clientStatus
					isApex
					isWildcard
					issued {
						nodes {
							type
							expiresAt
						}
					}
				}
				check {
					aRecords
				   	aaaaRecords
				   	cnameRecords
				   	soa
			   		dnsProvider
			   		dnsVerificationRecord
				 	resolvedAddresses
			   }
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("input", map[string]string{
		"appId":    appName,
		"hostname": hostname,
	})
	ctx = ctxWithAction(ctx, "check_app_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return data.CheckCertificate.Certificate, data.CheckCertificate.Check, nil
}

func (c *Client) AddCertificate(ctx context.Context, appName, hostname string) (*AppCertificate, *HostnameCheck, error) {
	query := `
		mutation($appId: ID!, $hostname: String!) {
			addCertificate(appId: $appId, hostname: $hostname) {
				certificate {
					acmeDnsConfigured
					acmeAlpnConfigured
					configured
					certificateAuthority
					createdAt
					dnsProvider
					dnsValidationInstructions
					dnsValidationHostname
					dnsValidationTarget
					hostname
					id
					source
					clientStatus
					isApex
					isWildcard
					issued {
						nodes {
							type
							expiresAt
						}
					}
				}
				check {
					aRecords
					aaaaRecords
					cnameRecords
					soa
					dnsProvider
					dnsVerificationRecord
				  	resolvedAddresses
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("appId", appName)
	req.Var("hostname", hostname)
	ctx = ctxWithAction(ctx, "add_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return data.AddCertificate.Certificate, data.AddCertificate.Check, nil
}

func (c *Client) DeleteCertificate(ctx context.Context, appName, hostname string) (*DeleteCertificatePayload, error) {
	query := `
		mutation($appId: ID!, $hostname: String!) {
			deleteCertificate(appId: $appId, hostname: $hostname) {
				app {
					name
				}
				certificate {
					hostname
					id
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("appId", appName)
	req.Var("hostname", hostname)
	ctx = ctxWithAction(ctx, "delete_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.DeleteCertificate, nil
}
