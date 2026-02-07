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

// Package mock provides mock diagnostic data for demos and testing.
// This package enables the CLI to run without a Kubernetes cluster,
// producing realistic output suitable for documentation, proposals,
// and development workflows.
package mock

import (
	"fmt"
	"time"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
)

// MockDiagnosticResult generates a realistic DiagnosticResult for demos.
// The mock data simulates a partially failing Fluid deployment with:
// - Dataset: Bound
// - Runtime: AlluxioRuntime with degraded workers and fuse issues
// - Realistic Kubernetes events and failure hints
func MockDiagnosticResult(datasetName, namespace string) *types.DiagnosticResult {
	now := time.Now()

	return &types.DiagnosticResult{
		CollectedAt: now,
		DatasetName: datasetName,
		Namespace:   namespace,

		// Clean Dataset YAML
		DatasetYAML: generateMockDatasetYAML(datasetName, namespace),

		// Clean Runtime YAML
		RuntimeYAML: generateMockRuntimeYAML(datasetName, namespace),
		RuntimeType: "alluxioruntimes",

		// Simulated events
		Events: generateMockEvents(datasetName, now),

		// Resource status with realistic failures
		Resources: generateMockResources(datasetName),

		// Collected logs
		Logs: generateMockLogs(datasetName),

		// Pre-computed failure hints
		FailureHints: generateMockFailureHints(),

		// Overall health
		HealthStatus: types.HealthStatusDegraded,
	}
}

// MockDiagnosticContext generates an AI-ready DiagnosticContext for demos.
func MockDiagnosticContext(datasetName, namespace string) *types.DiagnosticContext {
	result := MockDiagnosticResult(datasetName, namespace)

	return &types.DiagnosticContext{
		Summary: types.ContextSummary{
			DatasetName:  datasetName,
			Namespace:    namespace,
			DatasetPhase: "Bound",
			RuntimeType:  "alluxioruntimes",
			HealthStatus: types.HealthStatusDegraded,
			MasterReady:  "1/1",
			WorkersReady: "1/2",
			FuseReady:    "2/3",
			PVCStatus:    "Bound",
			ErrorCount:   3,
			WarningCount: 5,
		},
		DatasetYAML:  result.DatasetYAML,
		RuntimeYAML:  result.RuntimeYAML,
		Events:       result.Events,
		Logs:         convertLogsToMap(result.Logs),
		FailureHints: result.FailureHints,
		CollectedAt:  result.CollectedAt,
		Version:      "1.0",
	}
}

func generateMockDatasetYAML(name, namespace string) string {
	return fmt.Sprintf(`apiVersion: data.fluid.io/v1alpha1
kind: Dataset
metadata:
  name: %s
  namespace: %s
spec:
  mounts:
  - mountPoint: cos://my-bucket.cos.ap-guangzhou.myqcloud.com/data
    name: data
    options:
      fs.cosn.userinfo.secretId: "<redacted>"
      fs.cosn.userinfo.secretKey: "<redacted>"
    path: /
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: fluid.io/dataset-placement
          operator: In
          values:
          - speed
  tolerations:
  - effect: NoSchedule
    key: fluid.io/cache
    operator: Exists
status:
  conditions:
  - lastTransitionTime: "2026-02-07T18:00:00Z"
    lastUpdateTime: "2026-02-07T18:00:00Z"
    message: The ddc runtime is ready.
    reason: DatasetReady
    status: "True"
    type: Ready
  datasetRef:
  - name: %s
    namespace: %s
  mounts:
  - mountPoint: cos://my-bucket.cos.ap-guangzhou.myqcloud.com/data
    name: data
  phase: Bound
  runtimes:
  - name: %s
    namespace: %s
    type: alluxio
  ufsTotal: 128.5GiB
  fileNum: "54321"
`, name, namespace, name, namespace, name, namespace)
}

func generateMockRuntimeYAML(name, namespace string) string {
	return fmt.Sprintf(`apiVersion: data.fluid.io/v1alpha1
kind: AlluxioRuntime
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 2
  tieredstore:
    levels:
    - high: "0.95"
      low: "0.7"
      mediumtype: MEM
      path: /dev/shm
      quota: 4Gi
  master:
    replicas: 1
    resources:
      limits:
        cpu: "2"
        memory: 4Gi
      requests:
        cpu: "1"
        memory: 2Gi
  worker:
    replicas: 2
    resources:
      limits:
        cpu: "4"
        memory: 8Gi
      requests:
        cpu: "2"
        memory: 4Gi
  fuse:
    resources:
      limits:
        cpu: "2"
        memory: 4Gi
      requests:
        cpu: "1"
        memory: 2Gi
status:
  conditions:
  - lastProbeTime: "2026-02-07T18:30:00Z"
    lastTransitionTime: "2026-02-07T18:00:00Z"
    message: The master is initialized.
    reason: MasterInitialized
    status: "True"
    type: MasterInitialized
  - lastProbeTime: "2026-02-07T18:30:00Z"
    lastTransitionTime: "2026-02-07T18:05:00Z"
    message: The workers are partially ready.
    reason: WorkerPartiallyReady
    status: "False"
    type: WorkersReady
  currentFuseNumberScheduled: 2
  currentMasterNumberScheduled: 1
  currentWorkerNumberScheduled: 1
  desiredFuseNumberScheduled: 3
  desiredMasterNumberScheduled: 1
  desiredWorkerNumberScheduled: 2
  fuseNumberAvailable: 2
  fuseNumberReady: 2
  fuseNumberUnavailable: 1
  fusePhase: PartialReady
  fuseReason: "Fuse pod on node worker-3 is unschedulable: node has taint"
  masterNumberReady: 1
  masterPhase: Ready
  valueFile: %s-alluxio-values
  workerNumberAvailable: 1
  workerNumberReady: 1
  workerNumberUnavailable: 1
  workerPhase: PartialReady
  workerReason: "Worker pod %s-worker-1 pending: Insufficient memory"
`, name, namespace, name, name)
}

func generateMockEvents(datasetName string, now time.Time) []types.EventInfo {
	return []types.EventInfo{
		{
			Type:           "Warning",
			Reason:         "FailedScheduling",
			Message:        "0/5 nodes are available: 1 node(s) had taint {fluid.io/cache: }, that the pod didn't tolerate, 2 node(s) had untolerated taint {node.kubernetes.io/disk-pressure: }, 2 Insufficient memory.",
			Count:          3,
			FirstTimestamp: now.Add(-30 * time.Minute),
			LastTimestamp:  now.Add(-5 * time.Minute),
			Source:         "default-scheduler",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-fuse-x7k2p",
		},
		{
			Type:           "Warning",
			Reason:         "Unhealthy",
			Message:        "Readiness probe failed: alluxio worker not ready - master connection timeout",
			Count:          8,
			FirstTimestamp: now.Add(-25 * time.Minute),
			LastTimestamp:  now.Add(-3 * time.Minute),
			Source:         "kubelet",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-worker-1",
		},
		{
			Type:           "Warning",
			Reason:         "FailedMount",
			Message:        "MountVolume.SetUp failed for volume \"alluxio-fuse-mount\" : rpc error: code = DeadlineExceeded",
			Count:          2,
			FirstTimestamp: now.Add(-20 * time.Minute),
			LastTimestamp:  now.Add(-10 * time.Minute),
			Source:         "kubelet",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-fuse-abc12",
		},
		{
			Type:           "Normal",
			Reason:         "Pulled",
			Message:        "Successfully pulled image \"fluidcloudnative/alluxio:v2.9.0\" in 12.5s",
			Count:          1,
			FirstTimestamp: now.Add(-45 * time.Minute),
			LastTimestamp:  now.Add(-45 * time.Minute),
			Source:         "kubelet",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-master-0",
		},
		{
			Type:           "Normal",
			Reason:         "Started",
			Message:        "Started container alluxio-master",
			Count:          1,
			FirstTimestamp: now.Add(-44 * time.Minute),
			LastTimestamp:  now.Add(-44 * time.Minute),
			Source:         "kubelet",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-master-0",
		},
		{
			Type:           "Normal",
			Reason:         "Created",
			Message:        "Created pod: " + datasetName + "-worker-0",
			Count:          1,
			FirstTimestamp: now.Add(-43 * time.Minute),
			LastTimestamp:  now.Add(-43 * time.Minute),
			Source:         "statefulset-controller",
			ObjectKind:     "StatefulSet",
			ObjectName:     datasetName + "-worker",
		},
		{
			Type:           "Warning",
			Reason:         "Evicted",
			Message:        "Pod was evicted due to node memory pressure",
			Count:          1,
			FirstTimestamp: now.Add(-15 * time.Minute),
			LastTimestamp:  now.Add(-15 * time.Minute),
			Source:         "kubelet",
			ObjectKind:     "Pod",
			ObjectName:     datasetName + "-worker-1",
		},
	}
}

func generateMockResources(datasetName string) types.DiagnosticResources {
	return types.DiagnosticResources{
		Master: &types.PodGroupStatus{
			Name:        datasetName + "-master",
			Kind:        "StatefulSet",
			Desired:     1,
			Ready:       1,
			Available:   1,
			Unavailable: 0,
			Healthy:     true,
			Pods: []types.PodStatus{
				{
					Name:           datasetName + "-master-0",
					Phase:          "Running",
					Ready:          true,
					RestartCount:   0,
					NodeName:       "node-1",
					ContainerState: "Running",
				},
			},
		},
		Workers: &types.PodGroupStatus{
			Name:        datasetName + "-worker",
			Kind:        "StatefulSet",
			Desired:     2,
			Ready:       1,
			Available:   1,
			Unavailable: 1,
			Healthy:     false,
			Pods: []types.PodStatus{
				{
					Name:           datasetName + "-worker-0",
					Phase:          "Running",
					Ready:          true,
					RestartCount:   1,
					NodeName:       "node-1",
					ContainerState: "Running",
				},
			},
			FailingPods: []types.PodStatus{
				{
					Name:           datasetName + "-worker-1",
					Phase:          "Pending",
					Ready:          false,
					RestartCount:   3,
					Reason:         "Unschedulable",
					Message:        "0/5 nodes are available: 2 Insufficient memory.",
					ContainerState: "Waiting",
					Conditions:     []string{"PodScheduled: Unschedulable (0/5 nodes are available: 2 Insufficient memory)"},
				},
			},
		},
		Fuse: &types.PodGroupStatus{
			Name:        datasetName + "-fuse",
			Kind:        "DaemonSet",
			Desired:     3,
			Ready:       2,
			Available:   2,
			Unavailable: 1,
			Healthy:     false,
			Pods: []types.PodStatus{
				{
					Name:           datasetName + "-fuse-abc12",
					Phase:          "Running",
					Ready:          true,
					RestartCount:   0,
					NodeName:       "node-1",
					ContainerState: "Running",
				},
				{
					Name:           datasetName + "-fuse-def34",
					Phase:          "Running",
					Ready:          true,
					RestartCount:   0,
					NodeName:       "node-2",
					ContainerState: "Running",
				},
			},
			FailingPods: []types.PodStatus{
				{
					Name:           datasetName + "-fuse-x7k2p",
					Phase:          "Pending",
					Ready:          false,
					RestartCount:   0,
					Reason:         "Unschedulable",
					Message:        "0/5 nodes are available: node has taint {fluid.io/cache: } that pod didn't tolerate",
					ContainerState: "Waiting",
					Conditions:     []string{"PodScheduled: Unschedulable (node taint not tolerated)"},
				},
			},
		},
		PVC: &types.PVCDiagnostic{
			Name:         datasetName,
			Phase:        "Bound",
			VolumeName:   datasetName,
			StorageClass: "fluid",
			Capacity:     "100Gi",
		},
		PV: &types.PVDiagnostic{
			Name:          datasetName,
			Phase:         "Bound",
			StorageClass:  "fluid",
			Capacity:      "100Gi",
			ReclaimPolicy: "Delete",
		},
	}
}

func generateMockLogs(datasetName string) types.DiagnosticLogs {
	return types.DiagnosticLogs{
		Master: &types.LogEntry{
			PodName:       datasetName + "-master-0",
			ContainerName: "alluxio-master",
			TailLines:     100,
			Truncated:     true,
			Logs: `2026-02-07 18:00:01 INFO  [main] AlluxioMaster - Starting Alluxio master @ node-1
2026-02-07 18:00:02 INFO  [main] AlluxioMasterProcess - Alluxio master version: 2.9.0
2026-02-07 18:00:03 INFO  [main] RaftJournalSystem - Initializing Raft journal system
2026-02-07 18:00:05 INFO  [main] MetaMaster - Starting meta master
2026-02-07 18:00:06 INFO  [main] BlockMaster - Starting block master
2026-02-07 18:00:08 INFO  [main] FileSystemMaster - Starting file system master
2026-02-07 18:00:10 INFO  [grpc-default-executor-0] DefaultBlockMaster - Registering worker: WorkerId{id=1, address=WorkerNetAddress{host=` + datasetName + `-worker-0}}
2026-02-07 18:00:15 INFO  [HeartbeatThread] DefaultBlockMaster - Worker heartbeat received from ` + datasetName + `-worker-0
2026-02-07 18:00:45 WARN  [HeartbeatThread] DefaultBlockMaster - No heartbeat from ` + datasetName + `-worker-1 for 30s
2026-02-07 18:01:15 WARN  [HeartbeatThread] DefaultBlockMaster - Worker ` + datasetName + `-worker-1 timed out, removing from registry
2026-02-07 18:02:00 INFO  [grpc-default-executor-1] FileSystemMaster - UFS path mounted: cos://my-bucket.cos.ap-guangzhou.myqcloud.com/data
2026-02-07 18:05:00 INFO  [HeartbeatThread] DefaultBlockMaster - Cluster capacity: 4GB, used: 1.2GB (30%)
2026-02-07 18:10:00 INFO  [HeartbeatThread] DefaultBlockMaster - Cluster capacity: 4GB, used: 2.8GB (70%)
2026-02-07 18:15:00 WARN  [AsyncCacheRequestManager] FileSystemMaster - Cache request queue is 80% full
2026-02-07 18:20:00 INFO  [HeartbeatThread] DefaultBlockMaster - Cluster status: 1 active workers, 1 lost workers`,
		},
		Workers: []types.LogEntry{
			{
				PodName:       datasetName + "-worker-0",
				ContainerName: "alluxio-worker",
				TailLines:     100,
				Truncated:     true,
				Logs: `2026-02-07 18:00:15 INFO  [main] AlluxioWorker - Starting Alluxio worker @ node-1
2026-02-07 18:00:16 INFO  [main] AlluxioWorkerProcess - Alluxio worker version: 2.9.0
2026-02-07 18:00:17 INFO  [main] BlockWorker - Initializing block worker with 4GB tiered storage
2026-02-07 18:00:18 INFO  [main] TieredBlockStore - Tier 0 (MEM): /dev/shm, capacity: 4GB
2026-02-07 18:00:20 INFO  [main] WorkerNetAddress - Worker registered at ` + datasetName + `-worker-0:29999
2026-02-07 18:00:25 INFO  [grpc-default-executor-0] BlockWorker - Connected to master @ ` + datasetName + `-master-0:19998
2026-02-07 18:05:00 INFO  [CacheManager] BlockWorker - Caching block 123456 from UFS (256MB)
2026-02-07 18:05:30 INFO  [CacheManager] BlockWorker - Block 123456 cached successfully
2026-02-07 18:10:00 INFO  [CacheManager] BlockWorker - Caching block 234567 from UFS (512MB)
2026-02-07 18:11:00 INFO  [CacheManager] BlockWorker - Block 234567 cached successfully
2026-02-07 18:15:00 WARN  [CacheManager] BlockWorker - High memory pressure, eviction triggered
2026-02-07 18:15:05 INFO  [Evictor] TieredBlockStore - Evicting LRU blocks to free 1GB
2026-02-07 18:20:00 INFO  [HeartbeatThread] BlockWorker - Storage usage: 2.8GB / 4GB (70%)`,
			},
		},
		Fuse: []types.LogEntry{
			{
				PodName:       datasetName + "-fuse-x7k2p",
				ContainerName: "alluxio-fuse",
				TailLines:     100,
				Truncated:     false,
				Logs: `2026-02-07 18:30:00 INFO  [main] AlluxioFuse - Starting Alluxio FUSE @ node-3
2026-02-07 18:30:01 INFO  [main] AlluxioFuseProcess - Alluxio FUSE version: 2.9.0
2026-02-07 18:30:02 INFO  [main] AlluxioFuseFileSystem - Mounting /runtime-mnt/` + datasetName + `/fuse
2026-02-07 18:30:05 ERROR [main] AlluxioFuseFileSystem - Failed to connect to master: Connection refused
2026-02-07 18:30:06 ERROR [main] AlluxioFuseFileSystem - Master address: ` + datasetName + `-master-0:19998
2026-02-07 18:30:10 WARN  [main] RetryUtils - Retrying connection to master (attempt 1/5)
2026-02-07 18:30:15 ERROR [main] AlluxioFuseFileSystem - Connection attempt 1 failed: Connection timed out
2026-02-07 18:30:20 WARN  [main] RetryUtils - Retrying connection to master (attempt 2/5)
2026-02-07 18:30:25 ERROR [main] AlluxioFuseFileSystem - Connection attempt 2 failed: Connection timed out
2026-02-07 18:30:30 WARN  [main] RetryUtils - Retrying connection to master (attempt 3/5)
2026-02-07 18:30:35 ERROR [main] AlluxioFuseFileSystem - Connection attempt 3 failed: Connection timed out
2026-02-07 18:30:40 ERROR [main] AlluxioFuse - FUSE mount failed after 3 retries
2026-02-07 18:30:41 ERROR [main] AlluxioFuse - Cause: Unable to reach master, check network connectivity and master status`,
			},
		},
	}
}

func generateMockFailureHints() []types.FailureHint {
	return []types.FailureHint{
		{
			Severity:   "critical",
			Component:  "fuse",
			Issue:      "Fuse pod unschedulable on node-3",
			Suggestion: "Add toleration for taint {fluid.io/cache: } or remove taint from node",
			Evidence:   "0/5 nodes are available: node has taint {fluid.io/cache: } that pod didn't tolerate",
		},
		{
			Severity:   "critical",
			Component:  "worker",
			Issue:      "Worker pod pending due to insufficient memory",
			Suggestion: "Reduce worker memory request or add nodes with more memory",
			Evidence:   "0/5 nodes are available: 2 Insufficient memory",
		},
		{
			Severity:   "warning",
			Component:  "worker",
			Issue:      "Workers not healthy: 1/2 ready",
			Suggestion: "Check worker pod logs and node resources",
		},
		{
			Severity:   "warning",
			Component:  "fuse",
			Issue:      "Fuse not healthy: 2/3 ready",
			Suggestion: "Check fuse pod logs and node selectors/tolerations",
		},
		{
			Severity:   "warning",
			Component:  "fuse",
			Issue:      "FUSE mount failure detected",
			Suggestion: "Check master connectivity and fuse container logs",
			Evidence:   "MountVolume.SetUp failed for volume: rpc error: code = DeadlineExceeded",
		},
		{
			Severity:   "info",
			Component:  "worker",
			Issue:      "High restart count (3) for pod " + "demo-data-worker-1",
			Suggestion: "Check pod logs for crash reasons",
		},
	}
}

func convertLogsToMap(logs types.DiagnosticLogs) map[string]string {
	result := make(map[string]string)

	if logs.Master != nil && logs.Master.Logs != "" {
		result["master"] = logs.Master.Logs
	}

	for i, entry := range logs.Workers {
		if entry.Logs != "" {
			result[fmt.Sprintf("worker-%d", i)] = entry.Logs
		}
	}

	for i, entry := range logs.Fuse {
		if entry.Logs != "" {
			result[fmt.Sprintf("fuse-%d", i)] = entry.Logs
		}
	}

	return result
}
