package nodeshell

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Namespace for node-shell pods
	Namespace = "default"

	// PodNamePrefix is the prefix for node-shell pod names
	PodNamePrefix = "hma-cli-shell"

	// Image to use for node-shell pods
	Image = "alpine:latest"
)

// NodeShell provides node-level access via privileged pods
type NodeShell struct {
	client    kubernetes.Interface
	nodeName  string
	podName   string
	namespace string
}

// NewNodeShell creates a new NodeShell for the specified node
func NewNodeShell(kubeconfig, nodeName string) (*NodeShell, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &NodeShell{
		client:    client,
		nodeName:  nodeName,
		podName:   fmt.Sprintf("%s-%s", PodNamePrefix, nodeName),
		namespace: Namespace,
	}, nil
}

// CreatePod creates a privileged pod on the target node
func (n *NodeShell) CreatePod(ctx context.Context) error {
	privileged := true
	hostPathType := corev1.HostPathDirectory

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.podName,
			Namespace: n.namespace,
			Labels: map[string]string{
				"app":     "hma-cli",
				"purpose": "node-shell",
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": n.nodeName,
			},
			HostPID:     true,
			HostNetwork: true,
			HostIPC:     true,
			Containers: []corev1.Container{
				{
					Name:  "shell",
					Image: Image,
					Command: []string{
						"nsenter",
						"-t", "1",
						"-m", "-u", "-i", "-n",
						"--",
						"sleep", "infinity",
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "host",
							MountPath: "/host",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "host",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
							Type: &hostPathType,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Tolerations: []corev1.Toleration{
				{
					Operator: corev1.TolerationOpExists,
				},
			},
		},
	}

	_, err := n.client.CoreV1().Pods(n.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

// WaitForReady waits for the pod to be ready
func (n *NodeShell) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		pod, err := n.client.CoreV1().Pods(n.namespace).Get(ctx, n.podName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}

		if pod.Status.Phase == corev1.PodFailed {
			return false, fmt.Errorf("pod failed")
		}

		return false, nil
	})
}

// Cleanup deletes the node-shell pod
func (n *NodeShell) Cleanup(ctx context.Context) error {
	err := n.client.CoreV1().Pods(n.namespace).Delete(ctx, n.podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}

// GetPodName returns the name of the node-shell pod
func (n *NodeShell) GetPodName() string {
	return n.podName
}

// GetNamespace returns the namespace of the node-shell pod
func (n *NodeShell) GetNamespace() string {
	return n.namespace
}
