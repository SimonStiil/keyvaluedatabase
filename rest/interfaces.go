package rest

type ObjectType string

const (
	KvPair    ObjectType = "KvPair"
	Namespace ObjectType = "Namespace"
	Update    ObjectType = "KVUpdate"
)

type ObjectV1 struct {
	Type      ObjectType  `json:"type"`
	Namespace NamespaceV1 `json:"namespace"`
	KVPair    KVPairV1    `json:"kvPair"`
	KVUpdate  KVUpdateV1  `json:"kvUpdate"`
}
type KVPairListV1 []KVPairV1

type KVPairV1 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type KVPairObjV1 struct {
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}
type NamespaceV1 struct {
	Name string `json:"name"`
}
type NamespaceListV1 []NamespaceListObjV1

type NamespaceListObjV1 struct {
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Size   int    `json:"size"`
}

type KVUpdateV1 struct {
	Key  string `json:"key"`
	Type Type   `json:"type"`
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
