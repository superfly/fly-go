package flaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	fly "github.com/superfly/fly-go"
)

func (f *Client) sendRequestSecretKeys(ctx context.Context, method, endpoint string, in, out any, qs url.Values, headers map[string][]string) error {
	endpoint = fmt.Sprintf("/apps/%s/secretkeys%s", url.PathEscape(f.appName), endpoint)
	if qs != nil {
		endpoint += "?" + qs.Encode()
	}
	return f._sendRequest(ctx, method, endpoint, in, out, headers)
}

func (f *Client) ListSecretKeys(ctx context.Context, version *uint64) ([]fly.SecretKey, error) {
	ctx = contextWithAction(ctx, secretkeysList)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	out := fly.ListSecretKeysResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodGet, "", nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to list secret keys: %w", err)
	}

	return out.Secrets, nil
}

func (f *Client) GetSecretKey(ctx context.Context, name string, version *uint64) (*fly.SecretKey, error) {
	ctx = contextWithAction(ctx, secretkeyGet)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	out := fly.SecretKey{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodGet, path, nil, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) SetSecretKey(ctx context.Context, name string, typ string, value []byte) (*fly.SetSecretKeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySet)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	in := fly.SetSecretKeyRequest{Type: typ, Value: value}
	out := fly.SetSecretKeyResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, &out, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to set secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) GenerateSecretKey(ctx context.Context, name string, typ string) (*fly.SetSecretKeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySet)

	path := fmt.Sprintf("/%s/generate", url.PathEscape(name))
	in := fly.SetSecretKeyRequest{Type: typ}
	out := fly.SetSecretKeyResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, &out, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to set secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) DeleteSecretKey(ctx context.Context, name string) error {
	ctx = contextWithAction(ctx, secretkeyDelete)

	path := fmt.Sprintf("/%s", url.PathEscape(name))
	if err := f.sendRequestSecretKeys(ctx, http.MethodDelete, path, nil, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete secret key: %w", err)
	}

	return nil
}

func (f *Client) EncryptSecretKey(ctx context.Context, name string, plaintext, assoc []byte, version *uint64) (*fly.EncryptSecretKeyResp, error) {
	ctx = contextWithAction(ctx, secretkeyEncrypt)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/encrypt", url.PathEscape(name))
	in := fly.EncryptSecretKeyRequest{Plaintext: plaintext, AssocData: assoc}
	out := fly.EncryptSecretKeyResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to encrypt with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) DecryptSecretKey(ctx context.Context, name string, ciphertext, assoc []byte, version *uint64) (*fly.DecryptSecretKeyResp, error) {
	ctx = contextWithAction(ctx, secretkeyDecrypt)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/decrypt", url.PathEscape(name))
	in := fly.DecryptSecretKeyRequest{Ciphertext: ciphertext, AssocData: assoc}
	out := fly.DecryptSecretKeyResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to decrypt with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) SignSecretKey(ctx context.Context, name string, plaintext []byte, version *uint64) (*fly.SignSecretKeyResp, error) {
	ctx = contextWithAction(ctx, secretkeySign)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/sign", url.PathEscape(name))
	in := fly.SignSecretKeyRequest{Plaintext: plaintext}
	out := fly.SignSecretKeyResp{}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, &out, qs, nil); err != nil {
		return nil, fmt.Errorf("failed to sign with secret key: %w", err)
	}

	return &out, nil
}

func (f *Client) VerifySecretKey(ctx context.Context, name string, plaintext, sig []byte, version *uint64) error {
	ctx = contextWithAction(ctx, secretkeyVerify)

	var qs url.Values
	if version != nil {
		qs.Set("version", fmt.Sprintf("%d", *version))
	}

	path := fmt.Sprintf("/%s/verify", url.PathEscape(name))
	in := fly.VerifySecretKeyRequest{Plaintext: plaintext, Signature: sig}
	if err := f.sendRequestSecretKeys(ctx, http.MethodPost, path, in, nil, qs, nil); err != nil {
		return fmt.Errorf("failed to verify with secret key: %w", err)
	}

	return nil
}
