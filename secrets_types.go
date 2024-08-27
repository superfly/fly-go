package fly

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=SecretType
type SecretType int32

const (
	AppSecret                      = SecretType(1)
	VolumeEncryptionKey            = SecretType(2)
	SECRET_TYPE_KMS_HS256          = SecretType(3)
	SECRET_TYPE_KMS_HS384          = SecretType(4)
	SECRET_TYPE_KMS_HS512          = SecretType(5)
	SECRET_TYPE_KMS_XAES256GCM     = SecretType(6)
	SECRET_TYPE_KMS_NACL_AUTH      = SecretType(7)
	SECRET_TYPE_KMS_NACL_BOX       = SecretType(8)
	SECRET_TYPE_KMS_NACL_SECRETBOX = SecretType(9)
	SECRET_TYPE_KMS_NACL_SIGN      = SecretType(10)
)

type ListSecret struct {
	Label string     `json:"label"`
	Type  SecretType `json:"type"`
}

type CreateSecretRequest struct {
	Label string     `json:"label"`
	Type  SecretType `json:"type"`
	Value []byte     `json:"value,omitempty"`
}
