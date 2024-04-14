package rest

type KVPairV2 struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}
type NamespaceV2 struct {
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Size   int    `json:"size"`
}

type KVUpdateV2 struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Type      Type   `json:"type"`
}

type Type string

const (
	TypeRoll     Type = "roll"
	TypeGenerate Type = "generate"
)

const (
	KeyMaxLength   uint16 = 64
	ValueMaxLength uint16 = 16000
)

type HealthV1 struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
