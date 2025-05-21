package fly

const (
	SECRETKEY_TYPE_HS256          = "hs256"
	SECRETKEY_TYPE_HS384          = "hs384"
	SECRETKEY_TYPE_HS512          = "hs512"
	SECRETKEY_TYPE_XAES256GCM     = "xaes256gcm"
	SECRETKEY_TYPE_NACL_AUTH      = "nacl_auth"
	SECRETKEY_TYPE_NACL_BOX       = "nacl_box"
	SECRETKEY_TYPE_NACL_SECRETBOX = "nacl_secretbox"
	SECRETKEY_TYPE_NACL_SIGN      = "nacl_sign"
)

type AppSecret struct {
	Name   string  `json:"name"`
	Value  *string `json:"value,omitempty"`
	Digest string  `json:"digest"`
}

type ListAppSecretsResp struct {
	Secrets []AppSecret `json:"secrets"`
}

type SetAppSecretRequest struct {
	Value string `json:"value"`
}

type SetAppSecretResp struct {
	AppSecret
	Version uint64
}

type SecretKey struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Publickey []byte `json:"public_key,omitempty"`
}

type ListSecretkeysResp struct {
	Secrets []SecretKey `json:"secret_keys"`
}

type SetSecretkeyRequest struct {
	Type  string `json:"type"`
	Value []byte `json:"value"`
}

type SetSecretkeyResp struct {
	SecretKey
	Version uint64
}

type EncryptSecretkeyRequest struct {
	Plaintext []byte `json:"plaintext"`
	AssocData []byte `json:"associated_data,omitempty"`
}

type EncryptSecretkeyResp struct {
	Ciphertext []byte `json:"ciphertext"`
}

type DecryptSecretkeyRequest struct {
	Ciphertext []byte `json:"ciphertext"`
	AssocData  []byte `json:"associated_data,omitempty"`
}

type DecryptSecretkeyResp struct {
	Plaintext []byte `json:"plaintext"`
}

type SignSecretkeyRequest struct {
	Plaintext []byte `json:"plaintext"`
}

type SignSecretkeyResp struct {
	Signature []byte `json:"signature"`
}

type VerifySecretkeyRequest struct {
	Plaintext []byte `json:"plaintext"`
	Signature []byte `json:"signature"`
}
