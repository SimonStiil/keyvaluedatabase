package rest

type ObjectType string

const (
	KeyMaxLength   uint16 = 64
	ValueMaxLength uint16 = 16000

	TypeKey       ObjectType = "key"
	TypeNamespace ObjectType = "namespace"
	TypeRoll      ObjectType = "roll"
	TypeGenerate  ObjectType = "generate"
)

type ObjectV1 struct {
	Type  ObjectType `json:"type"`
	Value string     `json:"value"`
}
type KVPairListV1 []KVPairV2

type KVPairV2 struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}
type NamespaceListV1 []NamespaceV2

type NamespaceV2 struct {
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Size   int    `json:"size"`
}

type HealthV1 struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
