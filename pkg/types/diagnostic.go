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

import (
	"time"
)

// DiagnosticResult contains the complete diagnostic data for a Dataset
type DiagnosticResult struct {
	// Metadata
	CollectedAt time.Time `json:"collectedAt"`
	DatasetName string    `json:"datasetName"`
	Namespace   string    `json:"namespace"`

	// CR Snapshots (cleaned YAML)
	DatasetYAML string `json:"datasetYaml"`
	RuntimeYAML string `json:"runtimeYaml,omitempty"`
	RuntimeType string `json:"runtimeType,omitempty"`

	// Events
	Events []EventInfo `json:"events"`

	// Resource Status
	Resources DiagnosticResources `json:"resources"`

	// Logs
	Logs DiagnosticLogs `json:"logs"`

	// Analysis (for AI integration)
	FailureHints []FailureHint `json:"failureHints,omitempty"`
	HealthStatus HealthStatus  `json:"healthStatus"`
}

// EventInfo contains Kubernetes event information
type EventInfo struct {
	Type           string    `json:"type"` // Normal, Warning
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
	Source         string    `json:"source"`
	ObjectKind     string    `json:"objectKind"`
	ObjectName     string    `json:"objectName"`
}

// DiagnosticResources contains resource status information
type DiagnosticResources struct {
	Master  *PodGroupStatus `json:"master,omitempty"`
	Workers *PodGroupStatus `json:"workers,omitempty"`
	Fuse    *PodGroupStatus `json:"fuse,omitempty"`
	PVC     *PVCDiagnostic  `json:"pvc,omitempty"`
	PV      *PVDiagnostic   `json:"pv,omitempty"`
}

// PodGroupStatus contains status for a group of pods
type PodGroupStatus struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind"` // StatefulSet, DaemonSet
	Desired     int32       `json:"desired"`
	Ready       int32       `json:"ready"`
	Available   int32       `json:"available"`
	Unavailable int32       `json:"unavailable"`
	Healthy     bool        `json:"healthy"`
	Pods        []PodStatus `json:"pods,omitempty"`
	FailingPods []PodStatus `json:"failingPods,omitempty"`
}

// PodStatus contains individual pod status
type PodStatus struct {
	Name           string   `json:"name"`
	Phase          string   `json:"phase"`
	Ready          bool     `json:"ready"`
	RestartCount   int32    `json:"restartCount"`
	Reason         string   `json:"reason,omitempty"`
	Message        string   `json:"message,omitempty"`
	NodeName       string   `json:"nodeName,omitempty"`
	ContainerState string   `json:"containerState,omitempty"`
	Conditions     []string `json:"conditions,omitempty"`
}

// PVCDiagnostic contains PVC diagnostic info
type PVCDiagnostic struct {
	Name         string `json:"name"`
	Phase        string `json:"phase"`
	VolumeName   string `json:"volumeName,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
	Capacity     string `json:"capacity,omitempty"`
}

// PVDiagnostic contains PV diagnostic info
type PVDiagnostic struct {
	Name          string `json:"name"`
	Phase         string `json:"phase"`
	StorageClass  string `json:"storageClass,omitempty"`
	Capacity      string `json:"capacity,omitempty"`
	ReclaimPolicy string `json:"reclaimPolicy,omitempty"`
}

// DiagnosticLogs contains collected logs
type DiagnosticLogs struct {
	Master  *LogEntry  `json:"master,omitempty"`
	Workers []LogEntry `json:"workers,omitempty"`
	Fuse    []LogEntry `json:"fuse,omitempty"`
}

// LogEntry contains log data for a single container
type LogEntry struct {
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	Logs          string `json:"logs"`
	TailLines     int64  `json:"tailLines"`
	Truncated     bool   `json:"truncated"`
	Error         string `json:"error,omitempty"`
}

// FailureHint contains a detected issue with suggested action
type FailureHint struct {
	Severity   string `json:"severity"`  // critical, warning, info
	Component  string `json:"component"` // dataset, runtime, master, worker, fuse, pvc
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion"`
	Evidence   string `json:"evidence,omitempty"`
}

// HealthStatus represents overall health
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "Healthy"
	HealthStatusDegraded  HealthStatus = "Degraded"
	HealthStatusUnhealthy HealthStatus = "Unhealthy"
	HealthStatusUnknown   HealthStatus = "Unknown"
)

// DiagnosticContext is the AI-ready interface for diagnostic data
// This struct is designed for LLM consumption with normalized, deterministic fields
type DiagnosticContext struct {
	// Structured data for AI analysis
	Summary      ContextSummary    `json:"summary"`
	DatasetYAML  string            `json:"datasetYaml"`
	RuntimeYAML  string            `json:"runtimeYaml,omitempty"`
	Events       []EventInfo       `json:"events"`
	Logs         map[string]string `json:"logs"`
	FailureHints []FailureHint     `json:"failureHints"`

	// Metadata
	CollectedAt time.Time `json:"collectedAt"`
	Version     string    `json:"version"`
}

// ContextSummary provides a quick overview for AI
type ContextSummary struct {
	DatasetName  string       `json:"datasetName"`
	Namespace    string       `json:"namespace"`
	DatasetPhase string       `json:"datasetPhase"`
	RuntimeType  string       `json:"runtimeType,omitempty"`
	HealthStatus HealthStatus `json:"healthStatus"`
	MasterReady  string       `json:"masterReady"`
	WorkersReady string       `json:"workersReady"`
	FuseReady    string       `json:"fuseReady"`
	PVCStatus    string       `json:"pvcStatus"`
	ErrorCount   int          `json:"errorCount"`
	WarningCount int          `json:"warningCount"`
}
