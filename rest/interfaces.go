package rest

type KVPairV1 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	ValueMaxLength uint16 = 21800
)

type GreetingV1 struct {
	Id      uint32 `json:"id"`
	Content string `json:"content"`
}

type HealthV1 struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
