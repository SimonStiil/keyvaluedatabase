package rest

type KVPairV1 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVUpdateV1 struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

type GreetingV1 struct {
	Id      uint32 `json:"id"`
	Content string `json:"content"`
}

type HealthV1 struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
