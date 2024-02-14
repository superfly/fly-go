package fly

import "context"

func (c *Client) GetDNSRecords(ctx context.Context, domainName string) ([]*DNSRecord, error) {
	query := `
		query($domainName: String!) {
			domain(name: $domainName) {
				dnsRecords {
					nodes {
						id
						fqdn
						name
						type
						ttl
						rdata
						isApex
						isWildcard
						isSystem
						createdAt
						updatedAt
					}
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("domainName", domainName)
	ctx = ctxWithAction(ctx, "get_dns_records")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	if data.Domain == nil {
		return nil, ErrNotFound
	}

	return *data.Domain.DnsRecords.Nodes, nil
}

func (c *Client) ExportDNSRecords(ctx context.Context, domainId string) (string, error) {
	query := `
		mutation($input: ExportDNSZoneInput!) {
			exportDnsZone(input: $input) {
				contents
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("input", map[string]interface{}{
		"domainId": domainId,
	})
	ctx = ctxWithAction(ctx, "export_dns_records")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return "", err
	}

	return data.ExportDnsZone.Contents, nil
}

func (c *Client) ImportDNSRecords(ctx context.Context, domainId string, zonefile string) ([]ImportDnsWarning, []ImportDnsChange, error) {
	query := `
		mutation($input: ImportDNSZoneInput!) {
			importDnsZone(input: $input) {
				changes {
					action
					newText
					oldText
				}
				warnings {
					action
					message
					attributes {
						name
						rdata
						ttl
						type
					}
				}
			}
		}
	`

	req := c.NewRequest(query)

	req.Var("input", map[string]interface{}{
		"domainId": domainId,
		"zonefile": zonefile,
	})
	ctx = ctxWithAction(ctx, "import_dns_records")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	return data.ImportDnsZone.Warnings, data.ImportDnsZone.Changes, nil
}
