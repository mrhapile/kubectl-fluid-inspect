/*
Copyright 2026 kubectl-fluid-inspect Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client functionality
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*Client, error) {
	config, err := getConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}, nil
}

// getConfig returns the Kubernetes config
func getConfig(kubeconfigPath string) (*rest.Config, error) {
	// If kubeconfig is explicitly provided, use it
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// Check KUBECONFIG environment variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Fall back to default kubeconfig location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Try in-cluster config as last resort
		return rest.InClusterConfig()
	}

	kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		// Try in-cluster config as last resort
		return rest.InClusterConfig()
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetDataset fetches a Dataset CR
func (c *Client) GetDataset(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    "data.fluid.io",
		Version:  "v1alpha1",
		Resource: "datasets",
	}

	dataset, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("dataset %s/%s not found", namespace, name)
		}
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}

	return dataset, nil
}

// GetRuntime fetches a Runtime CR by type
func (c *Client) GetRuntime(ctx context.Context, namespace, name, runtimeType string) (*unstructured.Unstructured, error) {
	resourceName := runtimeTypeToResourceName(runtimeType)
	gvr := schema.GroupVersionResource{
		Group:    "data.fluid.io",
		Version:  "v1alpha1",
		Resource: resourceName,
	}

	runtime, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // Runtime not found is not an error
		}
		return nil, fmt.Errorf("failed to get runtime: %w", err)
	}

	return runtime, nil
}

// TryFindRuntime attempts to find a Runtime CR for a Dataset by trying different types
func (c *Client) TryFindRuntime(ctx context.Context, namespace, name string) (*unstructured.Unstructured, string, error) {
	runtimeTypes := []string{
		"alluxioruntimes",
		"jindoruntimes",
		"juicefsruntimes",
		"efcruntimes",
		"thinruntimes",
		"vineyardruntimes",
		"goosefsruntimes",
	}

	for _, resourceName := range runtimeTypes {
		gvr := schema.GroupVersionResource{
			Group:    "data.fluid.io",
			Version:  "v1alpha1",
			Resource: resourceName,
		}

		runtime, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			// Ignore other errors (e.g., CRD not installed)
			continue
		}

		return runtime, resourceName, nil
	}

	return nil, "", nil
}

// GetStatefulSet fetches a StatefulSet
func (c *Client) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	sts, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get statefulset: %w", err)
	}
	return sts, nil
}

// GetDaemonSet fetches a DaemonSet
func (c *Client) GetDaemonSet(ctx context.Context, namespace, name string) (*appsv1.DaemonSet, error) {
	ds, err := c.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get daemonset: %w", err)
	}
	return ds, nil
}

// GetPVC fetches a PersistentVolumeClaim
func (c *Client) GetPVC(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pvc: %w", err)
	}
	return pvc, nil
}

// ListStatefulSetsByLabel lists StatefulSets by label selector
func (c *Client) ListStatefulSetsByLabel(ctx context.Context, namespace, labelSelector string) (*appsv1.StatefulSetList, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// ListDaemonSetsByLabel lists DaemonSets by label selector
func (c *Client) ListDaemonSetsByLabel(ctx context.Context, namespace, labelSelector string) (*appsv1.DaemonSetList, error) {
	return c.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// runtimeTypeToResourceName converts runtime type to resource name
func runtimeTypeToResourceName(runtimeType string) string {
	mapping := map[string]string{
		"AlluxioRuntime":  "alluxioruntimes",
		"JindoRuntime":    "jindoruntimes",
		"JuiceFSRuntime":  "juicefsruntimes",
		"EFCRuntime":      "efcruntimes",
		"ThinRuntime":     "thinruntimes",
		"VineyardRuntime": "vineyardruntimes",
		"GooseFSRuntime":  "goosefsruntimes",
	}

	if name, ok := mapping[runtimeType]; ok {
		return name
	}
	return runtimeType
}
