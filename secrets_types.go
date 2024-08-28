package fly

const (
	// Secret types
	AppSecret                      = "AppSecret"
	VolumeEncryptionKey            = "VolumeEncryptionKey"
	SECRET_TYPE_KMS_HS256          = "SECRET_TYPE_KMS_HS256"
	SECRET_TYPE_KMS_HS384          = "SECRET_TYPE_KMS_HS384"
	SECRET_TYPE_KMS_HS512          = "SECRET_TYPE_KMS_HS512"
	SECRET_TYPE_KMS_XAES256GCM     = "SECRET_TYPE_KMS_XAES256GCM"
	SECRET_TYPE_KMS_NACL_AUTH      = "SECRET_TYPE_KMS_NACL_AUTH"
	SECRET_TYPE_KMS_NACL_BOX       = "SECRET_TYPE_KMS_NACL_BOX"
	SECRET_TYPE_KMS_NACL_SECRETBOX = "SECRET_TYPE_KMS_NACL_SECRETBOX"
	SECRET_TYPE_KMS_NACL_SIGN      = "SECRET_TYPE_KMS_NACL_SIGN"
)

type ListSecret struct {
	Label string `json:"label"`
	Type  string `json:"type"`
}

type CreateSecretRequest struct {
	Label string `json:"label"`
	Type  string `json:"type"`
	Value []byte `json:"value,omitempty"`
}
