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
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEventsForObject fetches events related to an object
func (c *Client) GetEventsForObject(ctx context.Context, namespace, name, uid string) ([]types.EventInfo, error) {
	// Get events involving this object
	eventList, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []types.EventInfo
	for _, e := range eventList.Items {
		events = append(events, types.EventInfo{
			Type:           e.Type,
			Reason:         e.Reason,
			Message:        e.Message,
			Count:          e.Count,
			FirstTimestamp: e.FirstTimestamp.Time,
			LastTimestamp:  e.LastTimestamp.Time,
			Source:         e.Source.Component,
			ObjectKind:     e.InvolvedObject.Kind,
			ObjectName:     e.InvolvedObject.Name,
		})
	}

	// Sort by last timestamp (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp)
	})

	return events, nil
}

// GetAllRelatedEvents fetches events for Dataset, Runtime, and related resources
func (c *Client) GetAllRelatedEvents(ctx context.Context, namespace, datasetName string) ([]types.EventInfo, error) {
	var allEvents []types.EventInfo

	// Events for Dataset
	datasetEvents, _ := c.GetEventsForObject(ctx, namespace, datasetName, "")
	allEvents = append(allEvents, datasetEvents...)

	// Events for Runtime resources (using naming convention)
	resourceNames := []string{
		datasetName + "-master",
		datasetName + "-worker",
		datasetName + "-fuse",
	}

	for _, name := range resourceNames {
		events, _ := c.GetEventsForObject(ctx, namespace, name, "")
		allEvents = append(allEvents, events...)
	}

	// Events for pods matching the release label
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("release=%s", datasetName),
	})
	if err == nil {
		for _, pod := range podList.Items {
			podEvents, _ := c.GetEventsForObject(ctx, namespace, pod.Name, "")
			allEvents = append(allEvents, podEvents...)
		}
	}

	// Sort all events chronologically
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].LastTimestamp.After(allEvents[j].LastTimestamp)
	})

	return allEvents, nil
}

// GetPodLogs fetches logs from a pod container
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, stream)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// GetPodsByLabel fetches pods by label selector
func (c *Client) GetPodsByLabel(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// GetPod fetches a single pod
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetPV fetches a PersistentVolume
func (c *Client) GetPV(ctx context.Context, name string) (*corev1.PersistentVolume, error) {
	return c.clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
}

// ListPodsByOwner lists pods owned by a specific controller
func (c *Client) ListPodsByOwner(ctx context.Context, namespace, ownerName, ownerKind string) ([]corev1.Pod, error) {
	// Use label selector based on Fluid conventions
	labelSelector := fmt.Sprintf("release=%s", ownerName)

	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var matchingPods []corev1.Pod
	for _, pod := range podList.Items {
		// Filter by owner reference
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == ownerKind {
				matchingPods = append(matchingPods, pod)
				break
			}
		}
	}

	return matchingPods, nil
}
