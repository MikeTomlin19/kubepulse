package types

type ResourceMetrics struct {
	Usage    int64 `json:"usage"`
	Requests int64 `json:"requests"`
	Limits   int64 `json:"limits"`
	Capacity int64 `json:"capacity"`
}

type PodMetrics struct {
	CPU    ResourceMetrics `json:"cpu"`
	Memory ResourceMetrics `json:"memory"`
}

type Pod struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Status    string     `json:"status"`
	Node      string     `json:"node"`
	Metrics   PodMetrics `json:"metrics"`
}

type Node struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Status  string          `json:"status"`
	Metrics ResourceMetrics `json:"metrics"`
	Pods    []Pod           `json:"pods"`
}

type ClusterState struct {
	Nodes []Node `json:"nodes"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
