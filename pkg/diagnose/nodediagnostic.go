package diagnose

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// NodeDiagnostic API info
	apiGroup   = "eks.amazonaws.com"
	apiVersion = "v1alpha1"
	resource   = "nodediagnostics"
)

var gvr = schema.GroupVersionResource{
	Group:    apiGroup,
	Version:  apiVersion,
	Resource: resource,
}

// NodeDiagnosticClient handles NodeDiagnostic CR operations
type NodeDiagnosticClient struct {
	client dynamic.Interface
}

// Status represents the status of a NodeDiagnostic
type Status struct {
	Phase   string
	Message string
}

// NewNodeDiagnosticClient creates a new client for NodeDiagnostic operations
func NewNodeDiagnosticClient(kubeconfig string) (*NodeDiagnosticClient, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try in-cluster config first, then default kubeconfig
		config, err = rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &NodeDiagnosticClient{client: client}, nil
}

// Create creates a NodeDiagnostic CR for the specified node
func (c *NodeDiagnosticClient) Create(ctx context.Context, nodeName, destination string) error {
	nodeDiagnostic := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", apiGroup, apiVersion),
			"kind":       "NodeDiagnostic",
			"metadata": map[string]interface{}{
				"name": nodeName,
			},
			"spec": map[string]interface{}{
				"logCapture": map[string]interface{}{
					"destination": destination,
				},
			},
		},
	}

	_, err := c.client.Resource(gvr).Create(ctx, nodeDiagnostic, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create NodeDiagnostic: %w", err)
	}

	return nil
}

// GetStatus retrieves the status of a NodeDiagnostic
func (c *NodeDiagnosticClient) GetStatus(ctx context.Context, nodeName string) (*Status, error) {
	nd, err := c.client.Resource(gvr).Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get NodeDiagnostic: %w", err)
	}

	status := &Status{}

	// Extract status from the unstructured object
	statusObj, found, err := unstructured.NestedMap(nd.Object, "status")
	if err != nil || !found {
		status.Phase = "Unknown"
		status.Message = "Status not available"
		return status, nil
	}

	if phase, ok := statusObj["phase"].(string); ok {
		status.Phase = phase
	}
	if message, ok := statusObj["message"].(string); ok {
		status.Message = message
	}

	return status, nil
}

// Delete deletes a NodeDiagnostic CR
func (c *NodeDiagnosticClient) Delete(ctx context.Context, nodeName string) error {
	err := c.client.Resource(gvr).Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete NodeDiagnostic: %w", err)
	}
	return nil
}

// WaitForCompletion waits for the NodeDiagnostic to complete
func (c *NodeDiagnosticClient) WaitForCompletion(ctx context.Context, nodeName string, timeout time.Duration) (*Status, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutCh:
			return nil, fmt.Errorf("timeout waiting for NodeDiagnostic to complete")
		case <-ticker.C:
			status, err := c.GetStatus(ctx, nodeName)
			if err != nil {
				return nil, err
			}

			switch status.Phase {
			case "Success", "SuccessWithErrors":
				return status, nil
			case "Failure":
				return status, fmt.Errorf("NodeDiagnostic failed: %s", status.Message)
			}
			// Continue waiting for Pending/InProgress
		}
	}
}

// Exists checks if a NodeDiagnostic already exists for the node
func (c *NodeDiagnosticClient) Exists(ctx context.Context, nodeName string) (bool, error) {
	_, err := c.client.Resource(gvr).Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}
