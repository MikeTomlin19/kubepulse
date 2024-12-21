export interface ResourceMetrics {
  usage: number;
  requests: number;
  limits: number;
  capacity: number;
}

export interface PodMetrics {
  CPU: ResourceMetrics;
  Memory: ResourceMetrics;
}

export interface Pod {
  id: string;
  name: string;
  namespace: string;
  status: 'running' | 'pending' | 'error';
  node: string;
  metrics: PodMetrics;
}

export interface Node {
  id: string;
  name: string;
  status: 'Ready' | 'NotReady';
  metrics: ResourceMetrics;
  pods: Pod[];
}

export interface ClusterData {
  nodes: Node[];
}

