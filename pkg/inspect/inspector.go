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

package inspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/k8s"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DatasetInspector inspects Dataset and related resources
type DatasetInspector struct {
	client *k8s.Client
}

// NewDatasetInspector creates a new DatasetInspector
func NewDatasetInspector(client *k8s.Client) *DatasetInspector {
	return &DatasetInspector{
		client: client,
	}
}

// Inspect inspects a Dataset and returns the complete result
func (d *DatasetInspector) Inspect(namespace, name string) (*types.InspectionResult, error) {
	ctx := context.Background()

	// Get Dataset
	dataset, err := d.client.GetDataset(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	result := &types.InspectionResult{}

	// Parse Dataset info
	result.Dataset = d.parseDatasetInfo(dataset)

	// Try to find and parse Runtime
	runtime, runtimeType, err := d.client.TryFindRuntime(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find runtime: %w", err)
	}

	if runtime != nil {
		runtimeInfo := d.parseRuntimeInfo(runtime, runtimeType)
		result.Runtime = runtimeInfo
	}

	// Get Kubernetes resources
	result.Resources = d.getResourceStatus(ctx, namespace, name)

	return result, nil
}

// parseDatasetInfo extracts information from a Dataset unstructured object
func (d *DatasetInspector) parseDatasetInfo(dataset *unstructured.Unstructured) types.DatasetInfo {
	info := types.DatasetInfo{
		Name:      dataset.GetName(),
		Namespace: dataset.GetNamespace(),
	}

	// Get status fields
	status, found, _ := unstructured.NestedMap(dataset.Object, "status")
	if found {
		if phase, ok := status["phase"].(string); ok {
			info.Phase = phase
		}
		if ufsTotal, ok := status["ufsTotal"].(string); ok {
			info.UfsTotal = ufsTotal
		}
		if fileNum, ok := status["fileNum"].(string); ok {
			info.FileNum = fileNum
		}

		// Parse conditions
		if conditions, found, _ := unstructured.NestedSlice(dataset.Object, "status", "conditions"); found {
			for _, c := range conditions {
				if cond, ok := c.(map[string]interface{}); ok {
					condInfo := types.ConditionInfo{}
					if t, ok := cond["type"].(string); ok {
						condInfo.Type = t
					}
					if s, ok := cond["status"].(string); ok {
						condInfo.Status = s
					}
					if r, ok := cond["reason"].(string); ok {
						condInfo.Reason = r
					}
					if m, ok := cond["message"].(string); ok {
						condInfo.Message = m
					}
					info.Conditions = append(info.Conditions, condInfo)
				}
			}
		}

		// Parse runtimes
		if runtimes, found, _ := unstructured.NestedSlice(dataset.Object, "status", "runtimes"); found {
			for _, r := range runtimes {
				if rt, ok := r.(map[string]interface{}); ok {
					ref := types.RuntimeRef{}
					if n, ok := rt["name"].(string); ok {
						ref.Name = n
					}
					if ns, ok := rt["namespace"].(string); ok {
						ref.Namespace = ns
					}
					if t, ok := rt["type"].(string); ok {
						ref.Type = t
					}
					info.Runtimes = append(info.Runtimes, ref)
				}
			}
		}
	}

	// Get mount points from spec
	if mounts, found, _ := unstructured.NestedSlice(dataset.Object, "spec", "mounts"); found {
		for _, m := range mounts {
			if mount, ok := m.(map[string]interface{}); ok {
				if mp, ok := mount["mountPoint"].(string); ok {
					info.MountPoints = append(info.MountPoints, mp)
				}
			}
		}
	}

	return info
}

// parseRuntimeInfo extracts information from a Runtime unstructured object
func (d *DatasetInspector) parseRuntimeInfo(runtime *unstructured.Unstructured, resourceType string) *types.RuntimeInfo {
	info := &types.RuntimeInfo{
		Name:      runtime.GetName(),
		Namespace: runtime.GetNamespace(),
		Type:      d.resourceTypeToRuntimeType(resourceType),
	}

	status, found, _ := unstructured.NestedMap(runtime.Object, "status")
	if !found {
		return info
	}

	// Parse Master status
	info.Master = types.ComponentStatus{}
	if phase, ok := status["masterPhase"].(string); ok {
		info.Master.Phase = phase
	}
	if reason, ok := status["masterReason"].(string); ok {
		info.Master.Reason = reason
	}
	if desired, ok := status["desiredMasterNumberScheduled"].(int64); ok {
		info.Master.DesiredScheduled = int32(desired)
	}
	if current, ok := status["currentMasterNumberScheduled"].(int64); ok {
		info.Master.CurrentScheduled = int32(current)
	}
	if ready, ok := status["masterNumberReady"].(int64); ok {
		info.Master.Ready = int32(ready)
	}

	// Parse Worker status
	info.Worker = types.ComponentStatus{}
	if phase, ok := status["workerPhase"].(string); ok {
		info.Worker.Phase = phase
	}
	if reason, ok := status["workerReason"].(string); ok {
		info.Worker.Reason = reason
	}
	if desired, ok := status["desiredWorkerNumberScheduled"].(int64); ok {
		info.Worker.DesiredScheduled = int32(desired)
	}
	if current, ok := status["currentWorkerNumberScheduled"].(int64); ok {
		info.Worker.CurrentScheduled = int32(current)
	}
	if ready, ok := status["workerNumberReady"].(int64); ok {
		info.Worker.Ready = int32(ready)
	}
	if available, ok := status["workerNumberAvailable"].(int64); ok {
		info.Worker.Available = int32(available)
	}
	if unavailable, ok := status["workerNumberUnavailable"].(int64); ok {
		info.Worker.Unavailable = int32(unavailable)
	}

	// Parse Fuse status
	info.Fuse = types.ComponentStatus{}
	if phase, ok := status["fusePhase"].(string); ok {
		info.Fuse.Phase = phase
	}
	if reason, ok := status["fuseReason"].(string); ok {
		info.Fuse.Reason = reason
	}
	if desired, ok := status["desiredFuseNumberScheduled"].(int64); ok {
		info.Fuse.DesiredScheduled = int32(desired)
	}
	if current, ok := status["currentFuseNumberScheduled"].(int64); ok {
		info.Fuse.CurrentScheduled = int32(current)
	}
	if ready, ok := status["fuseNumberReady"].(int64); ok {
		info.Fuse.Ready = int32(ready)
	}
	if available, ok := status["fuseNumberAvailable"].(int64); ok {
		info.Fuse.Available = int32(available)
	}
	if unavailable, ok := status["fuseNumberUnavailable"].(int64); ok {
		info.Fuse.Unavailable = int32(unavailable)
	}

	return info
}

// getResourceStatus fetches Kubernetes resource status
func (d *DatasetInspector) getResourceStatus(ctx context.Context, namespace, name string) types.ResourceStatus {
	result := types.ResourceStatus{}

	// Get Master StatefulSet
	masterSts, _ := d.client.GetStatefulSet(ctx, namespace, name+"-master")
	if masterSts != nil {
		result.MasterStatefulSet = &types.StatefulSetStatus{
			Name:            masterSts.Name,
			Replicas:        *masterSts.Spec.Replicas,
			ReadyReplicas:   masterSts.Status.ReadyReplicas,
			CurrentReplicas: masterSts.Status.CurrentReplicas,
			Healthy:         masterSts.Status.ReadyReplicas == *masterSts.Spec.Replicas,
		}
	}

	// Get Worker StatefulSet
	workerSts, _ := d.client.GetStatefulSet(ctx, namespace, name+"-worker")
	if workerSts != nil {
		result.WorkerStatefulSet = &types.StatefulSetStatus{
			Name:            workerSts.Name,
			Replicas:        *workerSts.Spec.Replicas,
			ReadyReplicas:   workerSts.Status.ReadyReplicas,
			CurrentReplicas: workerSts.Status.CurrentReplicas,
			Healthy:         workerSts.Status.ReadyReplicas == *workerSts.Spec.Replicas,
		}
	}

	// Get Fuse DaemonSet
	fuseDaemonSet, _ := d.client.GetDaemonSet(ctx, namespace, name+"-fuse")
	if fuseDaemonSet != nil {
		result.FuseDaemonSet = &types.DaemonSetStatus{
			Name:             fuseDaemonSet.Name,
			DesiredScheduled: fuseDaemonSet.Status.DesiredNumberScheduled,
			CurrentScheduled: fuseDaemonSet.Status.CurrentNumberScheduled,
			Ready:            fuseDaemonSet.Status.NumberReady,
			Available:        fuseDaemonSet.Status.NumberAvailable,
			Unavailable:      fuseDaemonSet.Status.NumberUnavailable,
			Healthy:          fuseDaemonSet.Status.NumberReady == fuseDaemonSet.Status.DesiredNumberScheduled,
		}
	}

	// Get PVC
	pvc, _ := d.client.GetPVC(ctx, namespace, name)
	if pvc != nil {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity["storage"]; ok {
				capacity = storage.String()
			}
		}
		result.PVC = &types.PVCStatus{
			Name:       pvc.Name,
			Phase:      string(pvc.Status.Phase),
			VolumeName: pvc.Spec.VolumeName,
			Capacity:   capacity,
		}
	}

	return result
}

// resourceTypeToRuntimeType converts resource type to display name
func (d *DatasetInspector) resourceTypeToRuntimeType(resourceType string) string {
	mapping := map[string]string{
		"alluxioruntimes":  "alluxio",
		"jindoruntimes":    "jindo",
		"juicefsruntimes":  "juicefs",
		"efcruntimes":      "efc",
		"thinruntimes":     "thin",
		"vineyardruntimes": "vineyard",
		"goosefsruntimes":  "goosefs",
	}

	lower := strings.ToLower(resourceType)
	if name, ok := mapping[lower]; ok {
		return name
	}
	return resourceType
}
