package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	fly "github.com/superfly/fly-go"
)

func (f *Client) sendRequestSecretkeys(ctx context.Context, method, endpoint string, in, out any, qs url.Values, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/secretkeys%s", url.PathEscape(f.appName), endpoint)
	if qs != nil {
		endpoint += "?" + qs.Encode()
	}
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) ListSecretkeys(ctx context.Context, version *uint64) ([]fly.SecretKey, error) {
	ctx = contextWithAction(ctx, secretkeysList)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	out := fly.ListSecretkeysResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodGet, "", nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to list secret keys: %w", err)
	}

	return out.Secrets, nil
}

func (f *Client) GetSecretkey(ctx context.Context, name string, version *uint64) (*fly.SecretKey, error) {
	ctx = contextWithAction(ctx, secretkeyGet)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	out := fly.SecretKey{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodGet, path, nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) SetSecretkey(ctx context.Context, name string, typ string, value []byte) (*fly.SetSecretkeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySet)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	in := fly.SetSecretkeyRequest{Type: typ, Value: value}
	out := fly.SetSecretkeyResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, &out, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to set secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) GenerateSecretkey(ctx context.Context, name string, typ string) (*fly.SetSecretkeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySet)

	path := fmt.Sprintf("/%s/generate", url.PathEscape(name))
	in := fly.SetSecretkeyRequest{Type: typ}
	out := fly.SetSecretkeyResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, &out, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to set secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) DeleteSecretkey(ctx context.Context, name string) error {
	ctx = contextWithAction(ctx, secretkeyDelete)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	if err := f.sendRequestSecretkeys(ctx, http.MethodDelete, path, nil, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete secret key: %w", err)
	}

	return nil
}

func (f *Client) EncryptSecretkey(ctx context.Context, name string, plaintext, assoc []byte, version *uint64) (*fly.EncryptSecretkeyResp, error) {
	ctx = contextWithAction(ctx, secretkeyEncrypt)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/encrypt", url.PathEscape(name))
	in := fly.EncryptSecretkeyRequest{Plaintext: plaintext, AssocData: assoc}
	out := fly.EncryptSecretkeyResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to encrypt with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) DecryptSecretkey(ctx context.Context, name string, ciphertext, assoc []byte, version *uint64) (*fly.DecryptSecretkeyResp, error) {
	ctx = contextWithAction(ctx, secretkeyDecrypt)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/decrypt", url.PathEscape(name))
	in := fly.DecryptSecretkeyRequest{Ciphertext: ciphertext, AssocData: assoc}
	out := fly.DecryptSecretkeyResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to decrypt with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) SignSecretkey(ctx context.Context, name string, plaintext []byte, version *uint64) (*fly.SignSecretkeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySign)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/sign", url.PathEscape(name))
	in := fly.SignSecretkeyRequest{Plaintext: plaintext}
	out := fly.SignSecretkeyResp{}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to sign with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) VerifySecretkey(ctx context.Context, name string, plaintext, sig []byte, version *uint64) error {
	ctx = contextWithAction(ctx, secretkeyVerify)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/verify", url.PathEscape(name))
	in := fly.VerifySecretkeyRequest{Plaintext: plaintext, Signature: sig}
	if err := f.sendRequestSecretkeys(ctx, http.MethodPost, path, in, nil, qs, nil); err != nil {
		return fmt.Errorf("failed to verify with secret key: %w", err)
	}

	return nil
}
