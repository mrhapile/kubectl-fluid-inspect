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

package types

// InspectionResult contains the complete inspection result for a Dataset
type InspectionResult struct {
	Dataset     DatasetInfo
	Runtime     *RuntimeInfo
	Resources   ResourceStatus
	Conditions  []ConditionInfo
	CacheStatus *CacheStatus
}

// DatasetInfo contains Dataset information
type DatasetInfo struct {
	Name        string
	Namespace   string
	Phase       string
	UfsTotal    string
	FileNum     string
	Runtimes    []RuntimeRef
	Conditions  []ConditionInfo
	MountPoints []string
}

// RuntimeRef contains a reference to a Runtime
type RuntimeRef struct {
	Name      string
	Namespace string
	Type      string
}

// RuntimeInfo contains Runtime status information
type RuntimeInfo struct {
	Name      string
	Namespace string
	Type      string // alluxio, jindo, juicefs, etc.
	Master    ComponentStatus
	Worker    ComponentStatus
	Fuse      ComponentStatus
}

// ComponentStatus contains status for a single component (master/worker/fuse)
type ComponentStatus struct {
	Phase            string
	Reason           string
	DesiredScheduled int32
	CurrentScheduled int32
	Ready            int32
	Available        int32
	Unavailable      int32
}

// ResourceStatus contains Kubernetes resource status
type ResourceStatus struct {
	MasterStatefulSet *StatefulSetStatus
	WorkerStatefulSet *StatefulSetStatus
	FuseDaemonSet     *DaemonSetStatus
	PVC               *PVCStatus
}

// StatefulSetStatus contains StatefulSet status
type StatefulSetStatus struct {
	Name            string
	Replicas        int32
	ReadyReplicas   int32
	CurrentReplicas int32
	Healthy         bool
}

// DaemonSetStatus contains DaemonSet status
type DaemonSetStatus struct {
	Name             string
	DesiredScheduled int32
	CurrentScheduled int32
	Ready            int32
	Available        int32
	Unavailable      int32
	Healthy          bool
}

// PVCStatus contains PersistentVolumeClaim status
type PVCStatus struct {
	Name       string
	Phase      string
	VolumeName string
	Capacity   string
}

// ConditionInfo contains condition information
type ConditionInfo struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// CacheStatus contains cache statistics
type CacheStatus struct {
	CacheCapacity    string
	Cached           string
	CachedPercentage string
	Cacheable        string
	LowWaterMark     string
	HighWaterMark    string
}

// RuntimeTypes defines supported runtime types
var RuntimeTypes = []string{
	"AlluxioRuntime",
	"JindoRuntime",
	"JuiceFSRuntime",
	"EFCRuntime",
	"ThinRuntime",
	"VineyardRuntime",
	"GooseFSRuntime",
}

// RuntimeTypeToAPIVersion maps runtime type to API version
var RuntimeTypeToAPIVersion = map[string]string{
	"AlluxioRuntime":  "data.fluid.io/v1alpha1",
	"JindoRuntime":    "data.fluid.io/v1alpha1",
	"JuiceFSRuntime":  "data.fluid.io/v1alpha1",
	"EFCRuntime":      "data.fluid.io/v1alpha1",
	"ThinRuntime":     "data.fluid.io/v1alpha1",
	"VineyardRuntime": "data.fluid.io/v1alpha1",
	"GooseFSRuntime":  "data.fluid.io/v1alpha1",
}
