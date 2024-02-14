package fly

import "context"

func (c *Client) CreateDoctorUrl(ctx context.Context) (putUrl string, err error) {
	query := `
		mutation {
			createDoctorUrl {
				putUrl
			}
		}
	`

	req := c.NewRequest(query)
	ctx = ctxWithAction(ctx, "create_doctor_url")

	data, err := c.RunWithContext(ctx, req)
	if err != nil {
		return "", err
	}

	return data.CreateDoctorUrl.PutUrl, nil
}
