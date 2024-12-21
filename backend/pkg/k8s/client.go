package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"kubepulse/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Client struct {
	clientset     *kubernetes.Clientset
	metricsClient *metrics.Clientset
	config        *rest.Config
	subscribers   map[chan types.ClusterState]bool
	mutex         sync.RWMutex
}

func NewClient() (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %v", err)
	}

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %v", err)
	}

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		config:        config,
		subscribers:   make(map[chan types.ClusterState]bool),
	}, nil
}

func (c *Client) Subscribe() chan types.ClusterState {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ch := make(chan types.ClusterState, 1)
	c.subscribers[ch] = true
	return ch
}

func (c *Client) Unsubscribe(ch chan types.ClusterState) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.subscribers, ch)
	close(ch)
}

func (c *Client) StartWatching(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state, err := c.GetClusterState()
				if err != nil {
					continue
				}
				c.mutex.RLock()
				for ch := range c.subscribers {
					select {
					case ch <- state:
					default:
					}
				}
				c.mutex.RUnlock()
			}
		}
	}()
}

func (c *Client) GetClusterState() (types.ClusterState, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return types.ClusterState{}, fmt.Errorf("failed to list nodes: %v", err)
	}

	var clusterState types.ClusterState
	for _, node := range nodes.Items {
		nodeState := types.Node{
			ID:     string(node.UID),
			Name:   node.Name,
			Status: getNodeStatus(node),
		}

		// Get node metrics
		nodeMetrics, err := c.metricsClient.MetricsV1beta1().NodeMetricses().Get(context.Background(), node.Name, metav1.GetOptions{})
		if err == nil {
			nodeState.Metrics = types.ResourceMetrics{
				Usage:    nodeMetrics.Usage.Cpu().MilliValue(),
				Capacity: node.Status.Capacity.Cpu().MilliValue(),
				Requests: 0, // Will be calculated from pods
				Limits:   0, // Will be calculated from pods
			}
		}

		// Get pods for this node
		pods, err := c.clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
		})
		if err == nil {
			for _, pod := range pods.Items {
				podState := types.Pod{
					ID:        string(pod.UID),
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Status:    getPodStatus(pod),
					Node:      node.Name,
					Metrics: types.PodMetrics{
						CPU: types.ResourceMetrics{
							Requests: getPodCPURequests(pod),
							Limits:   getPodCPULimits(pod),
						},
						Memory: types.ResourceMetrics{
							Requests: getPodMemoryRequests(pod),
							Limits:   getPodMemoryLimits(pod),
						},
					},
				}

				// Get pod metrics
				podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				if err == nil {
					for _, container := range podMetrics.Containers {
						podState.Metrics.CPU.Usage += container.Usage.Cpu().MilliValue()
						podState.Metrics.Memory.Usage += container.Usage.Memory().Value()
					}
				}

				nodeState.Pods = append(nodeState.Pods, podState)
				nodeState.Metrics.Requests += podState.Metrics.CPU.Requests
				nodeState.Metrics.Limits += podState.Metrics.CPU.Limits
			}
		}

		clusterState.Nodes = append(clusterState.Nodes, nodeState)
	}

	return clusterState, nil
}

func getNodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getPodStatus(pod corev1.Pod) string {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		return "running"
	case corev1.PodPending:
		return "pending"
	default:
		return "error"
	}
}

func getPodCPURequests(pod corev1.Pod) int64 {
	var total int64
	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				total += cpu.MilliValue()
			}
		}
	}
	return total
}

func getPodCPULimits(pod corev1.Pod) int64 {
	var total int64
	for _, container := range pod.Spec.Containers {
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				total += cpu.MilliValue()
			}
		}
	}
	return total
}

func getPodMemoryRequests(pod corev1.Pod) int64 {
	var total int64
	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if mem := container.Resources.Requests.Memory(); mem != nil {
				total += mem.Value()
			}
		}
	}
	return total
}

func getPodMemoryLimits(pod corev1.Pod) int64 {
	var total int64
	for _, container := range pod.Spec.Containers {
		if container.Resources.Limits != nil {
			if mem := container.Resources.Limits.Memory(); mem != nil {
				total += mem.Value()
			}
		}
	}
	return total
}
