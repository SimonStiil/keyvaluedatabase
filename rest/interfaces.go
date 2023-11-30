package rest

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVUpdate struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

type Greeting struct {
	Id      uint32 `json:"id"`
	Content string `json:"content"`
}

type Health struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}
