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

package diagnose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/k8s"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const (
	defaultTailLines = 100
	maxLogsPerGroup  = 2
)

// DatasetDiagnoser performs comprehensive diagnosis of a Dataset
type DatasetDiagnoser struct {
	client    *k8s.Client
	tailLines int64
}

// NewDatasetDiagnoser creates a new DatasetDiagnoser
func NewDatasetDiagnoser(client *k8s.Client) *DatasetDiagnoser {
	return &DatasetDiagnoser{
		client:    client,
		tailLines: defaultTailLines,
	}
}

// Diagnose performs a complete diagnosis of a Dataset
func (d *DatasetDiagnoser) Diagnose(ctx context.Context, namespace, name string) (*types.DiagnosticResult, error) {
	result := &types.DiagnosticResult{
		CollectedAt: time.Now(),
		DatasetName: name,
		Namespace:   namespace,
	}

	// Step 1: Fetch and clean CR snapshots
	if err := d.collectCRSnapshots(ctx, namespace, name, result); err != nil {
		return nil, fmt.Errorf("failed to collect CR snapshots: %w", err)
	}

	// Step 2: Collect Kubernetes events
	if err := d.collectEvents(ctx, namespace, name, result); err != nil {
		// Non-fatal, continue with diagnosis
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "events",
			Issue:      "Failed to collect events",
			Suggestion: "Check RBAC permissions for event access",
			Evidence:   err.Error(),
		})
	}

	// Step 3: Collect runtime resource status
	if err := d.collectResourceStatus(ctx, namespace, name, result); err != nil {
		// Non-fatal, continue with diagnosis
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "resources",
			Issue:      "Failed to collect resource status",
			Suggestion: "Check RBAC permissions for pod/statefulset/daemonset access",
			Evidence:   err.Error(),
		})
	}

	// Step 4: Collect logs
	if err := d.collectLogs(ctx, namespace, name, result); err != nil {
		// Non-fatal, continue with diagnosis
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "logs",
			Issue:      "Failed to collect some logs",
			Suggestion: "Check RBAC permissions for pod/log access",
			Evidence:   err.Error(),
		})
	}

	// Analyze and generate failure hints
	d.analyzeAndGenerateHints(result)

	// Determine overall health status
	result.HealthStatus = d.determineHealthStatus(result)

	return result, nil
}

// collectCRSnapshots fetches and cleans Dataset and Runtime CRs
func (d *DatasetDiagnoser) collectCRSnapshots(ctx context.Context, namespace, name string, result *types.DiagnosticResult) error {
	// Get Dataset
	dataset, err := d.client.GetDataset(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get dataset: %w", err)
	}

	cleanedDataset := cleanCRForDiagnosis(dataset)
	datasetYAML, err := yaml.Marshal(cleanedDataset)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}
	result.DatasetYAML = string(datasetYAML)

	// Try to find Runtime
	runtime, runtimeType, err := d.client.TryFindRuntime(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to find runtime: %w", err)
	}

	if runtime != nil {
		result.RuntimeType = runtimeType
		cleanedRuntime := cleanCRForDiagnosis(runtime)
		runtimeYAML, err := yaml.Marshal(cleanedRuntime)
		if err != nil {
			return fmt.Errorf("failed to marshal runtime: %w", err)
		}
		result.RuntimeYAML = string(runtimeYAML)
	}

	return nil
}

// collectEvents fetches all related events
func (d *DatasetDiagnoser) collectEvents(ctx context.Context, namespace, name string, result *types.DiagnosticResult) error {
	events, err := d.client.GetAllRelatedEvents(ctx, namespace, name)
	if err != nil {
		return err
	}
	result.Events = events
	return nil
}

// collectResourceStatus fetches status of all runtime resources
func (d *DatasetDiagnoser) collectResourceStatus(ctx context.Context, namespace, name string, result *types.DiagnosticResult) error {
	// Master StatefulSet
	masterSts, _ := d.client.GetStatefulSet(ctx, namespace, name+"-master")
	if masterSts != nil {
		result.Resources.Master = &types.PodGroupStatus{
			Name:        masterSts.Name,
			Kind:        "StatefulSet",
			Desired:     *masterSts.Spec.Replicas,
			Ready:       masterSts.Status.ReadyReplicas,
			Available:   masterSts.Status.AvailableReplicas,
			Unavailable: *masterSts.Spec.Replicas - masterSts.Status.ReadyReplicas,
			Healthy:     masterSts.Status.ReadyReplicas == *masterSts.Spec.Replicas,
		}
		d.collectPodStatus(ctx, namespace, name, "master", result.Resources.Master)
	}

	// Worker StatefulSet
	workerSts, _ := d.client.GetStatefulSet(ctx, namespace, name+"-worker")
	if workerSts != nil {
		result.Resources.Workers = &types.PodGroupStatus{
			Name:        workerSts.Name,
			Kind:        "StatefulSet",
			Desired:     *workerSts.Spec.Replicas,
			Ready:       workerSts.Status.ReadyReplicas,
			Available:   workerSts.Status.AvailableReplicas,
			Unavailable: *workerSts.Spec.Replicas - workerSts.Status.ReadyReplicas,
			Healthy:     workerSts.Status.ReadyReplicas == *workerSts.Spec.Replicas,
		}
		d.collectPodStatus(ctx, namespace, name, "worker", result.Resources.Workers)
	}

	// Fuse DaemonSet
	fuseDaemonSet, _ := d.client.GetDaemonSet(ctx, namespace, name+"-fuse")
	if fuseDaemonSet != nil {
		result.Resources.Fuse = &types.PodGroupStatus{
			Name:        fuseDaemonSet.Name,
			Kind:        "DaemonSet",
			Desired:     fuseDaemonSet.Status.DesiredNumberScheduled,
			Ready:       fuseDaemonSet.Status.NumberReady,
			Available:   fuseDaemonSet.Status.NumberAvailable,
			Unavailable: fuseDaemonSet.Status.NumberUnavailable,
			Healthy:     fuseDaemonSet.Status.NumberReady == fuseDaemonSet.Status.DesiredNumberScheduled,
		}
		d.collectPodStatus(ctx, namespace, name, "fuse", result.Resources.Fuse)
	}

	// PVC
	pvc, _ := d.client.GetPVC(ctx, namespace, name)
	if pvc != nil {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity["storage"]; ok {
				capacity = storage.String()
			}
		}
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}
		result.Resources.PVC = &types.PVCDiagnostic{
			Name:         pvc.Name,
			Phase:        string(pvc.Status.Phase),
			VolumeName:   pvc.Spec.VolumeName,
			StorageClass: storageClass,
			Capacity:     capacity,
		}

		// Get PV if bound
		if pvc.Spec.VolumeName != "" {
			pv, _ := d.client.GetPV(ctx, pvc.Spec.VolumeName)
			if pv != nil {
				pvCapacity := ""
				if pv.Spec.Capacity != nil {
					if storage, ok := pv.Spec.Capacity["storage"]; ok {
						pvCapacity = storage.String()
					}
				}
				result.Resources.PV = &types.PVDiagnostic{
					Name:          pv.Name,
					Phase:         string(pv.Status.Phase),
					StorageClass:  pv.Spec.StorageClassName,
					Capacity:      pvCapacity,
					ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
				}
			}
		}
	}

	return nil
}

// collectPodStatus collects status of pods for a component
func (d *DatasetDiagnoser) collectPodStatus(ctx context.Context, namespace, datasetName, role string, group *types.PodGroupStatus) {
	labelSelector := fmt.Sprintf("release=%s,role=alluxio-%s", datasetName, role)
	pods, err := d.client.GetPodsByLabel(ctx, namespace, labelSelector)
	if err != nil {
		return
	}

	for _, pod := range pods.Items {
		status := d.extractPodStatus(&pod)
		if status.Ready {
			group.Pods = append(group.Pods, status)
		} else {
			group.FailingPods = append(group.FailingPods, status)
		}
	}
}

// extractPodStatus extracts status from a pod
func (d *DatasetDiagnoser) extractPodStatus(pod *corev1.Pod) types.PodStatus {
	status := types.PodStatus{
		Name:     pod.Name,
		Phase:    string(pod.Status.Phase),
		NodeName: pod.Spec.NodeName,
	}

	// Check if ready
	status.Ready = isPodReady(pod)

	// Get restart count and container state
	if len(pod.Status.ContainerStatuses) > 0 {
		cs := pod.Status.ContainerStatuses[0]
		status.RestartCount = cs.RestartCount

		if cs.State.Waiting != nil {
			status.ContainerState = "Waiting"
			status.Reason = cs.State.Waiting.Reason
			status.Message = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			status.ContainerState = "Terminated"
			status.Reason = cs.State.Terminated.Reason
			status.Message = cs.State.Terminated.Message
		} else if cs.State.Running != nil {
			status.ContainerState = "Running"
		}
	}

	// Collect conditions
	for _, cond := range pod.Status.Conditions {
		if cond.Status == corev1.ConditionFalse && cond.Type != corev1.PodReady {
			status.Conditions = append(status.Conditions,
				fmt.Sprintf("%s: %s (%s)", cond.Type, cond.Reason, cond.Message))
		}
	}

	return status
}

// collectLogs fetches logs from relevant pods
func (d *DatasetDiagnoser) collectLogs(ctx context.Context, namespace, name string, result *types.DiagnosticResult) error {
	// Collect master logs
	if result.Resources.Master != nil && len(result.Resources.Master.Pods) > 0 {
		pod := result.Resources.Master.Pods[0]
		logs, err := d.client.GetPodLogs(ctx, namespace, pod.Name, "alluxio-master", d.tailLines)
		if err != nil {
			result.Logs.Master = &types.LogEntry{
				PodName:   pod.Name,
				Error:     err.Error(),
				TailLines: d.tailLines,
			}
		} else {
			result.Logs.Master = &types.LogEntry{
				PodName:       pod.Name,
				ContainerName: "alluxio-master",
				Logs:          logs,
				TailLines:     d.tailLines,
				Truncated:     len(logs) > 0,
			}
		}
	}

	// Collect worker logs (one healthy, one failing if available)
	if result.Resources.Workers != nil {
		// Collect from a healthy pod
		if len(result.Resources.Workers.Pods) > 0 {
			pod := result.Resources.Workers.Pods[0]
			logs, err := d.client.GetPodLogs(ctx, namespace, pod.Name, "alluxio-worker", d.tailLines)
			entry := types.LogEntry{
				PodName:       pod.Name,
				ContainerName: "alluxio-worker",
				TailLines:     d.tailLines,
			}
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Logs = logs
				entry.Truncated = len(logs) > 0
			}
			result.Logs.Workers = append(result.Logs.Workers, entry)
		}

		// Collect from a failing pod
		if len(result.Resources.Workers.FailingPods) > 0 {
			pod := result.Resources.Workers.FailingPods[0]
			logs, err := d.client.GetPodLogs(ctx, namespace, pod.Name, "alluxio-worker", d.tailLines)
			entry := types.LogEntry{
				PodName:       pod.Name,
				ContainerName: "alluxio-worker",
				TailLines:     d.tailLines,
			}
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Logs = logs
				entry.Truncated = len(logs) > 0
			}
			result.Logs.Workers = append(result.Logs.Workers, entry)
		}
	}

	// Collect fuse logs (failing pods only)
	if result.Resources.Fuse != nil && len(result.Resources.Fuse.FailingPods) > 0 {
		for i, pod := range result.Resources.Fuse.FailingPods {
			if i >= maxLogsPerGroup {
				break
			}
			logs, err := d.client.GetPodLogs(ctx, namespace, pod.Name, "alluxio-fuse", d.tailLines)
			entry := types.LogEntry{
				PodName:       pod.Name,
				ContainerName: "alluxio-fuse",
				TailLines:     d.tailLines,
			}
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Logs = logs
				entry.Truncated = len(logs) > 0
			}
			result.Logs.Fuse = append(result.Logs.Fuse, entry)
		}
	}

	return nil
}

// analyzeAndGenerateHints analyzes the diagnostic data and generates failure hints
func (d *DatasetDiagnoser) analyzeAndGenerateHints(result *types.DiagnosticResult) {
	// Check Dataset phase
	datasetPhase := extractPhaseFromYAML(result.DatasetYAML)
	if datasetPhase == "Pending" {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "dataset",
			Issue:      "Dataset is in Pending phase",
			Suggestion: "Check if a matching Runtime CR exists and is healthy",
		})
	} else if datasetPhase == "Failed" {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "critical",
			Component:  "dataset",
			Issue:      "Dataset is in Failed phase",
			Suggestion: "Check events and conditions for failure reason",
		})
	}

	// Check Master status
	if result.Resources.Master != nil && !result.Resources.Master.Healthy {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "critical",
			Component:  "master",
			Issue:      fmt.Sprintf("Master not healthy: %d/%d ready", result.Resources.Master.Ready, result.Resources.Master.Desired),
			Suggestion: "Check master pod logs and events for errors",
		})
	}

	// Check Worker status
	if result.Resources.Workers != nil && !result.Resources.Workers.Healthy {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "worker",
			Issue:      fmt.Sprintf("Workers not healthy: %d/%d ready", result.Resources.Workers.Ready, result.Resources.Workers.Desired),
			Suggestion: "Check worker pod logs and node resources",
		})
	}

	// Check Fuse status
	if result.Resources.Fuse != nil && !result.Resources.Fuse.Healthy {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "warning",
			Component:  "fuse",
			Issue:      fmt.Sprintf("Fuse not healthy: %d/%d ready", result.Resources.Fuse.Ready, result.Resources.Fuse.Desired),
			Suggestion: "Check fuse pod logs and node selectors/tolerations",
		})
	}

	// Check PVC status
	if result.Resources.PVC != nil && result.Resources.PVC.Phase != "Bound" {
		result.FailureHints = append(result.FailureHints, types.FailureHint{
			Severity:   "critical",
			Component:  "pvc",
			Issue:      fmt.Sprintf("PVC is not bound: %s", result.Resources.PVC.Phase),
			Suggestion: "Check if the Dataset is bound to a Runtime",
		})
	}

	// Analyze events for patterns
	warningCount := 0
	for _, event := range result.Events {
		if event.Type == "Warning" {
			warningCount++
			if strings.Contains(event.Message, "ImagePullBackOff") ||
				strings.Contains(event.Message, "ErrImagePull") {
				result.FailureHints = append(result.FailureHints, types.FailureHint{
					Severity:   "critical",
					Component:  event.ObjectKind,
					Issue:      "Image pull failure detected",
					Suggestion: "Check image name, tag, and registry credentials",
					Evidence:   event.Message,
				})
			}
			if strings.Contains(event.Message, "Insufficient") {
				result.FailureHints = append(result.FailureHints, types.FailureHint{
					Severity:   "warning",
					Component:  event.ObjectKind,
					Issue:      "Resource insufficiency detected",
					Suggestion: "Check node resources and pod resource requests",
					Evidence:   event.Message,
				})
			}
			if strings.Contains(event.Message, "FailedMount") ||
				strings.Contains(event.Message, "MountVolume") {
				result.FailureHints = append(result.FailureHints, types.FailureHint{
					Severity:   "critical",
					Component:  event.ObjectKind,
					Issue:      "Volume mount failure detected",
					Suggestion: "Check PVC binding and CSI driver status",
					Evidence:   event.Message,
				})
			}
		}
	}

	// Check for high restart counts
	checkRestartCounts := func(pods []types.PodStatus, component string) {
		for _, pod := range pods {
			if pod.RestartCount > 3 {
				result.FailureHints = append(result.FailureHints, types.FailureHint{
					Severity:   "warning",
					Component:  component,
					Issue:      fmt.Sprintf("High restart count (%d) for pod %s", pod.RestartCount, pod.Name),
					Suggestion: "Check pod logs for crash reasons",
				})
			}
		}
	}

	if result.Resources.Master != nil {
		checkRestartCounts(result.Resources.Master.Pods, "master")
		checkRestartCounts(result.Resources.Master.FailingPods, "master")
	}
	if result.Resources.Workers != nil {
		checkRestartCounts(result.Resources.Workers.Pods, "worker")
		checkRestartCounts(result.Resources.Workers.FailingPods, "worker")
	}
	if result.Resources.Fuse != nil {
		checkRestartCounts(result.Resources.Fuse.Pods, "fuse")
		checkRestartCounts(result.Resources.Fuse.FailingPods, "fuse")
	}
}

// determineHealthStatus determines overall health based on analysis
func (d *DatasetDiagnoser) determineHealthStatus(result *types.DiagnosticResult) types.HealthStatus {
	hasCritical := false
	hasWarning := false

	for _, hint := range result.FailureHints {
		if hint.Severity == "critical" {
			hasCritical = true
		} else if hint.Severity == "warning" {
			hasWarning = true
		}
	}

	if hasCritical {
		return types.HealthStatusUnhealthy
	}
	if hasWarning {
		return types.HealthStatusDegraded
	}

	// Check if all resources are healthy
	allHealthy := true
	if result.Resources.Master != nil && !result.Resources.Master.Healthy {
		allHealthy = false
	}
	if result.Resources.Workers != nil && !result.Resources.Workers.Healthy {
		allHealthy = false
	}
	if result.Resources.Fuse != nil && !result.Resources.Fuse.Healthy {
		allHealthy = false
	}
	if result.Resources.PVC != nil && result.Resources.PVC.Phase != "Bound" {
		allHealthy = false
	}

	if allHealthy {
		return types.HealthStatusHealthy
	}

	return types.HealthStatusDegraded
}

// ToContext converts the diagnostic result to an AI-ready context
func (d *DatasetDiagnoser) ToContext(result *types.DiagnosticResult) *types.DiagnosticContext {
	ctx := &types.DiagnosticContext{
		DatasetYAML:  result.DatasetYAML,
		RuntimeYAML:  result.RuntimeYAML,
		Events:       result.Events,
		FailureHints: result.FailureHints,
		CollectedAt:  result.CollectedAt,
		Version:      "1.0",
		Logs:         make(map[string]string),
	}

	// Build summary
	ctx.Summary = types.ContextSummary{
		DatasetName:  result.DatasetName,
		Namespace:    result.Namespace,
		DatasetPhase: extractPhaseFromYAML(result.DatasetYAML),
		RuntimeType:  result.RuntimeType,
		HealthStatus: result.HealthStatus,
	}

	// Add resource status to summary
	if result.Resources.Master != nil {
		ctx.Summary.MasterReady = fmt.Sprintf("%d/%d", result.Resources.Master.Ready, result.Resources.Master.Desired)
	}
	if result.Resources.Workers != nil {
		ctx.Summary.WorkersReady = fmt.Sprintf("%d/%d", result.Resources.Workers.Ready, result.Resources.Workers.Desired)
	}
	if result.Resources.Fuse != nil {
		ctx.Summary.FuseReady = fmt.Sprintf("%d/%d", result.Resources.Fuse.Ready, result.Resources.Fuse.Desired)
	}
	if result.Resources.PVC != nil {
		ctx.Summary.PVCStatus = result.Resources.PVC.Phase
	}

	// Count events
	for _, event := range result.Events {
		if event.Type == "Warning" {
			ctx.Summary.WarningCount++
		}
	}
	ctx.Summary.ErrorCount = len(result.FailureHints)

	// Normalize and truncate logs
	if result.Logs.Master != nil && result.Logs.Master.Logs != "" {
		ctx.Logs["master"] = normalizeLogs(result.Logs.Master.Logs)
	}
	for i, entry := range result.Logs.Workers {
		if entry.Logs != "" {
			key := fmt.Sprintf("worker-%d", i)
			ctx.Logs[key] = normalizeLogs(entry.Logs)
		}
	}
	for i, entry := range result.Logs.Fuse {
		if entry.Logs != "" {
			key := fmt.Sprintf("fuse-%d", i)
			ctx.Logs[key] = normalizeLogs(entry.Logs)
		}
	}

	return ctx
}

// Helper functions

func cleanCRForDiagnosis(obj *unstructured.Unstructured) map[string]interface{} {
	cleaned := obj.DeepCopy().Object

	// Remove noisy metadata fields
	if metadata, ok := cleaned["metadata"].(map[string]interface{}); ok {
		delete(metadata, "managedFields")
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "generation")
		delete(metadata, "creationTimestamp")
	}

	return cleaned
}

func extractPhaseFromYAML(yamlStr string) string {
	// Simple extraction - look for phase field
	lines := strings.Split(yamlStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "phase:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func normalizeLogs(logs string) string {
	// Remove potential secrets/sensitive data
	lines := strings.Split(logs, "\n")
	var cleaned []string
	for _, line := range lines {
		// Skip lines that might contain secrets
		lower := strings.ToLower(line)
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "apikey") {
			cleaned = append(cleaned, "[REDACTED]")
		} else {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}
