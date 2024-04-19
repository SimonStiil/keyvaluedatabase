package rest

type ObjectType string

const (
	KeyMaxLength   uint16 = 64
	ValueMaxLength uint16 = 16000

	TypeKey       ObjectType = "Key"
	TypeNamespace ObjectType = "Namespace"
	TypeRoll      ObjectType = "Roll"
	TypeGenerate  ObjectType = "Generate"
)

type ObjectV1 struct {
	Type  ObjectType `json:"type"`
	Value string     `json:"value"`
}
type KVPairListV1 []KVPairV1

type KVPairV1 struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}
type NamespaceListV1 []NamespaceListObjV1

type NamespaceListObjV1 struct {
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Size   int    `json:"size"`
}

type HealthV1 struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
