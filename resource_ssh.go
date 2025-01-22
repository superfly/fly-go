package fly

import (
	"context"
	"crypto/ed25519"
	"strings"

	"golang.org/x/crypto/ssh"
)

func (c *Client) GetLoggedCertificates(ctx context.Context, slug string) ([]LoggedCertificate, error) {
	req := c.NewRequest(`
query($slug: String!) {
  organization(slug: $slug) {
    loggedCertificates {
      nodes {
        root
        cert
      }
    }
  }
}
`)
	req.Var("slug", slug)
	ctx = ctxWithAction(ctx, "get_logged_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return data.Organization.LoggedCertificates.Nodes, nil
}

func (c *Client) IssueSSHCertificate(ctx context.Context, org OrganizationImpl, principals []string, appNames []string, valid_hours *int, publicKey ed25519.PublicKey) (*IssuedCertificate, error) {
	req := c.NewRequest(`
mutation($input: IssueCertificateInput!) {
  issueCertificate(input: $input) {
    certificate, key
  }
}
`)
	var pubStr string
	if len(publicKey) > 0 {
		sshPub, err := ssh.NewPublicKey(publicKey)
		if err != nil {
			return nil, err
		}

		pubStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	}

	inputs := map[string]interface{}{
		"organizationId": org.GetID(),
		"principals":     principals,
		"appNames":       appNames,
		"publicKey":      pubStr,
	}

	if valid_hours != nil {
		inputs["validHours"] = *valid_hours
	}

	req.Var("input", inputs)
	ctx = ctxWithAction(ctx, "issue_ssh_certificates")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return &data.IssueCertificate, nil
}
