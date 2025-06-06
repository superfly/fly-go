package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	fly "github.com/superfly/fly-go"
)

func (f *Client) sendRequestSecrets(ctx context.Context, method, endpoint string, in, out any, qs url.Values, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/secrets%s", url.PathEscape(f.appName), endpoint)
	if qs != nil {
		endpoint += "?" + qs.Encode()
	}
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) ListAppSecrets(ctx context.Context, version *uint64, showSecrets bool) ([]fly.AppSecret, error) {
	ctx = contextWithAction(ctx, appSecretsList)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}
	if showSecrets {
		qs.Set("show_secrets", "true")
	}

	out := fly.ListAppSecretsResp{}
	if err := f.sendRequestSecrets(ctx, http.MethodGet, "", nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to list app secrets: %w", err)
	}

	return out.Secrets, nil
}

func (f *Client) GetAppSecrets(ctx context.Context, name string, version *uint64, showSecrets bool) (*fly.AppSecret, error) {
	ctx = contextWithAction(ctx, appSecretGet)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}
	if showSecrets {
		qs.Set("show_secrets", "true")
	}

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	out := fly.AppSecret{}
	if err := f.sendRequestSecrets(ctx, http.MethodGet, path, nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to get app secret: %w", err)
	}

	return &out, nil
}

func (f *Client) SetAppSecret(ctx context.Context, name string, value string) (*fly.SetAppSecretResp, error) {
	ctx = contextWithAction(ctx, appSecretSet)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	in := fly.SetAppSecretRequest{Value: value}
	out := fly.SetAppSecretResp{}
	if err := f.sendRequestSecrets(ctx, http.MethodPost, path, in, &out, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to set app secret: %w", err)
	}

	return &out, nil
}

func (f *Client) DeleteAppSecret(ctx context.Context, name string) error {
	ctx = contextWithAction(ctx, appSecretDelete)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	if err := f.sendRequestSecrets(ctx, http.MethodDelete, path, nil, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete app secret: %w", err)
	}

	return nil
}
