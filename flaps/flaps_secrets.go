package flaps

import (
	"context"
	"fmt"
	"net/http"

	fly "github.com/superfly/fly-go"
)

func (f *Client) sendRequestSecrets(ctx context.Context, method, endpoint string, in, out interface{}, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/secrets%s", f.appName, endpoint)
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) ListSecrets(ctx context.Context) ([]fly.ListSecret, error) {
	ctx = contextWithAction(ctx, secretsList)

	out := make([]fly.ListSecret, 0)
	if err := f.sendRequestSecrets(ctx, http.MethodGet, "", nil, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return out, nil
}

func (f *Client) CreateSecret(ctx context.Context, sLabel, sType string, in fly.CreateSecretRequest) error {
	ctx = contextWithAction(ctx, secretCreate)

	path := fmt.Sprintf("/%s/type/%s", sLabel, sType)
	if err := f.sendRequestSecrets(ctx, http.MethodPost, path, in, nil, nil); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

func (f *Client) GenerateSecret(ctx context.Context, sLabel, sType string) error {
	ctx = contextWithAction(ctx, secretCreate)

	path := fmt.Sprintf("/%s/type/%s/generate", sLabel, sType)
	if err := f.sendRequestSecrets(ctx, http.MethodPost, path, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

func (f *Client) DeleteSecret(ctx context.Context, label string) error {
	ctx = contextWithAction(ctx, secretDelete)

	endpoint := fmt.Sprintf("/%s", label)
	if err := f.sendRequestSecrets(ctx, http.MethodDelete, endpoint, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
