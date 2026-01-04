package kubernetes

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterStatus struct {
	NodesReady     int
	NodesTotal     int
	PodsRunning    int
	PodsPending    int
	PodsFailed     int
	ContainerCount int
	Healthy        bool
	ErrorMsg       string
}

type Client struct {
	clientset *kubernetes.Clientset
}

func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

func (c *Client) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	status := &ClusterStatus{}

	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		status.Healthy = false
		status.ErrorMsg = "API unreachable"
		return status, err
	}
	status.Healthy = true

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		status.ErrorMsg = "failed to list nodes"
		return status, err
	}

	status.NodesTotal = len(nodes.Items)
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				status.NodesReady++
				break
			}
		}
	}

	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		status.ErrorMsg = "failed to list pods"
		return status, err
	}

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			status.PodsRunning++
		case corev1.PodPending:
			status.PodsPending++
		case corev1.PodFailed:
			status.PodsFailed++
		}
		status.ContainerCount += len(pod.Spec.Containers)
	}

	return status, nil
}

func (c *Client) HealthCheck(ctx context.Context) bool {
	_, err := c.clientset.Discovery().ServerVersion()
	return err == nil
}
