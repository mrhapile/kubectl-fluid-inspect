# Phase 0: Architecture & Design Analysis
## kubectl-fluid-inspect

**Version:** 1.0  
**Date:** February 8, 2026  
**Author:** kubectl-fluid-inspect Development Team

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Fluid Architecture Overview](#fluid-architecture-overview)
3. [CRD Analysis](#crd-analysis)
   - [Dataset CRD](#dataset-crd)
   - [Runtime CRDs](#runtime-crds)
4. [Controller Flow Analysis](#controller-flow-analysis)
5. [Resource Mapping](#resource-mapping)
6. [Failure Modes](#failure-modes)
7. [Why kubectl-fluid-inspect is Needed](#why-kubectl-fluid-inspect-is-needed)
8. [CLI Design Specification](#cli-design-specification)

---

## Executive Summary

This document provides a comprehensive architectural analysis of the CNCF Fluid project, focusing on understanding the relationship between Dataset and Runtime CRDs, controller flows, Kubernetes resource mappings, and common failure modes. This analysis forms the foundation for building `kubectl-fluid-inspect`, a read-only CLI tool for inspecting Fluid Dataset status and underlying resources.

### Key Findings

- **Dataset** is the logical data abstraction; **Runtime** is the execution engine
- A Dataset binds to exactly one Runtime (though the relationship is flexible by design)
- Each Runtime type (Alluxio, Jindo, JuiceFS, EFC, Thin) creates specific Kubernetes resources
- Status propagation flows: Runtime → Dataset Controller → Dataset.Status
- Multiple failure points exist that require cross-resource inspection for diagnosis

---

## Fluid Architecture Overview

Fluid is a CNCF incubating project providing a Kubernetes-native distributed dataset orchestrator and accelerator. It abstracts data access for AI/ML and Big Data workloads.

### Core Concepts

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Creates                             │
├─────────────────────────────────────────────────────────────────┤
│    Dataset CR                          Runtime CR                │
│    (data.fluid.io/v1alpha1)           (data.fluid.io/v1alpha1)  │
│                                                                  │
│    - Defines mount points              - Defines cache engine    │
│    - Specifies access modes            - Specifies replicas      │
│    - References Runtime type           - Configures Master/      │
│                                           Worker/Fuse            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Fluid Controllers                             │
├─────────────────────────────────────────────────────────────────┤
│  Dataset Controller        Runtime Controller (per type)        │
│  - Binds Dataset to        - Creates StatefulSets (Master)      │
│    Runtime                 - Creates StatefulSets (Worker)      │
│  - Creates PVC             - Creates DaemonSets (Fuse)          │
│  - Updates Dataset.Status  - Updates Runtime.Status             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Kubernetes Resources                           │
├─────────────────────────────────────────────────────────────────┤
│  StatefulSet (Master)     StatefulSet (Worker)    DaemonSet     │
│  ConfigMaps               Services                (Fuse)        │
│  PersistentVolumeClaim    PersistentVolume        Pods          │
└─────────────────────────────────────────────────────────────────┘
```

---

## CRD Analysis

### Dataset CRD

**API Group:** `data.fluid.io/v1alpha1`  
**Kind:** `Dataset`

#### DatasetSpec Fields

| Field | Type | Description |
|-------|------|-------------|
| `mounts` | `[]Mount` | Mount points to be mounted on cache runtime |
| `owner` | `*User` | Owner of the dataset |
| `nodeAffinity` | `*CacheableNodeAffinity` | Constraints limiting which nodes cache data |
| `tolerations` | `[]v1.Toleration` | Pod tolerations |
| `accessModes` | `[]PersistentVolumeAccessMode` | Volume access modes |
| `runtimes` | `[]Runtime` | Runtimes supporting this dataset |
| `placement` | `PlacementMode` | Exclusive or Shared placement |
| `dataRestoreLocation` | `*DataRestoreLocation` | Backup restore location |
| `sharedOptions` | `map[string]string` | Options applied to all mounts |
| `sharedEncryptOptions` | `[]EncryptOption` | Encryption options for all mounts |

#### DatasetStatus Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | `DatasetPhase` | Current phase: Pending, Bound, NotBound, Failed, Updating, DataMigrating |
| `runtimes` | `[]Runtime` | Bound runtime information |
| `conditions` | `[]DatasetCondition` | Array of current observed conditions |
| `ufsTotal` | `string` | Total underlying filesystem size |
| `cacheStates` | `CacheStateList` | Cache statistics |
| `hcfs` | `*HCFSStatus` | HCFS endpoint information |
| `fileNum` | `string` | Number of files in dataset |
| `mounts` | `[]Mount` | Currently mounted mount points |
| `operationRef` | `map[string]string` | Locks for ongoing operations |

#### Dataset Phases

| Phase | Description |
|-------|-------------|
| `Pending` | Dataset created but not yet bound to a Runtime |
| `Bound` | Dataset successfully bound to Runtime, ready for use |
| `NotBound` | Dataset not bound to any Runtime |
| `Failed` | Dataset failed to bind or setup |
| `Updating` | Dataset is being updated |
| `DataMigrating` | Data migration in progress |

#### Dataset Condition Types

| Condition | Description |
|-----------|-------------|
| `RuntimeScheduled` | Runtime CRD accepted, but components not ready |
| `Ready` | Cache system is ready |
| `NotReady` | Dataset not bound due to unexpected error |
| `UpdateReady` | Cache system updated successfully |
| `Updating` | Cache system is being updated |
| `Initialized` | Cache system is initialized |

---

### Runtime CRDs

Fluid supports multiple Runtime types, each implementing the same status interface:

| Runtime Type | Kind | Purpose |
|-------------|------|---------|
| **Alluxio** | `AlluxioRuntime` | General-purpose caching with Alluxio |
| **Jindo** | `JindoRuntime` | Alibaba Cloud optimized caching |
| **JuiceFS** | `JuiceFSRuntime` | JuiceFS-based distributed file system |
| **EFC** | `EFCRuntime` | Elastic File Client for NAS |
| **Thin** | `ThinRuntime` | Lightweight, generic runtime |

#### RuntimeStatus (Common to All Runtimes)

| Field | Type | Description |
|-------|------|-------------|
| `valueFileConfigmap` | `string` | ConfigMap containing configuration values |
| `masterPhase` | `RuntimePhase` | Master component phase |
| `masterReason` | `string` | Reason for master phase transition |
| `workerPhase` | `RuntimePhase` | Worker component phase |
| `workerReason` | `string` | Reason for worker phase transition |
| `fusePhase` | `RuntimePhase` | Fuse component phase |
| `fuseReason` | `string` | Reason for fuse phase transition |
| `desiredWorkerNumberScheduled` | `int32` | Workers that should be running |
| `currentWorkerNumberScheduled` | `int32` | Workers currently running |
| `workerNumberReady` | `int32` | Workers ready |
| `workerNumberAvailable` | `int32` | Workers available |
| `workerNumberUnavailable` | `int32` | Workers unavailable |
| `desiredMasterNumberScheduled` | `int32` | Masters that should be running |
| `currentMasterNumberScheduled` | `int32` | Masters currently running |
| `masterNumberReady` | `int32` | Masters ready |
| `currentFuseNumberScheduled` | `int32` | Fuse pods currently running |
| `desiredFuseNumberScheduled` | `int32` | Fuse pods that should be running |
| `fuseNumberReady` | `int32` | Fuse pods ready |
| `fuseNumberUnavailable` | `int32` | Fuse pods unavailable |
| `fuseNumberAvailable` | `int32` | Fuse pods available |
| `setupDuration` | `string` | Time spent setting up runtime |
| `conditions` | `[]RuntimeCondition` | Array of runtime conditions |
| `cacheStates` | `CacheStateList` | Cache statistics |
| `mountTime` | `*metav1.Time` | Last mount time |
| `mounts` | `[]Mount` | Current mount points |

#### Runtime Phases

| Phase | Description |
|-------|-------------|
| `Pending` | Component created but not scheduled |
| `Ready` | Component running and ready |
| `NotReady` | Component not ready |
| `Partial` | Some replicas ready, others not |

---

## Controller Flow Analysis

### Dataset → Runtime Reconciliation Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. User creates Dataset CR                                       │
│    namespace/name: default/demo-data                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. User creates corresponding Runtime CR                        │
│    namespace/name: default/demo-data (same name as Dataset)     │
│    Kind: AlluxioRuntime / JindoRuntime / etc.                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Dataset Controller detects Runtime                           │
│    - Validates Runtime matches Dataset                           │
│    - Sets Dataset.Status.Phase = "Pending"                       │
│    - Creates PVC for Dataset                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Runtime Controller (e.g., AlluxioController)                  │
│    - Creates Master StatefulSet                                  │
│    - Creates Worker StatefulSet                                  │
│    - Creates Fuse DaemonSet                                      │
│    - Creates ConfigMaps and Services                             │
│    - Updates Runtime.Status with component phases                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Dataset Controller observes Runtime.Status                    │
│    - When all components Ready:                                  │
│      Dataset.Status.Phase = "Bound"                              │
│    - When components fail:                                       │
│      Dataset.Status.Phase = "Failed"                             │
│    - Updates Dataset.Status.Conditions                           │
└─────────────────────────────────────────────────────────────────┘
```

### Controller Ownership

| Controller | Responsibility |
|-----------|----------------|
| `dataset-controller` | Watches Dataset CRs, manages binding to Runtime, creates PVC |
| `alluxioruntime-controller` | Manages AlluxioRuntime lifecycle and Kubernetes resources |
| `jindoruntime-controller` | Manages JindoRuntime lifecycle and Kubernetes resources |
| `juicefsruntime-controller` | Manages JuiceFSRuntime lifecycle and Kubernetes resources |
| `efcruntime-controller` | Manages EFCRuntime lifecycle and Kubernetes resources |
| `thinruntime-controller` | Manages ThinRuntime lifecycle and Kubernetes resources |

---

## Resource Mapping

### For a Given Runtime (e.g., AlluxioRuntime "demo-data" in namespace "default")

| Fluid Component | Kubernetes Resource | Naming Convention | Purpose |
|----------------|---------------------|-------------------|---------|
| Master | StatefulSet | `demo-data-master` | Alluxio master nodes, manages metadata |
| Worker | StatefulSet | `demo-data-worker` | Alluxio worker nodes, provides caching |
| Fuse | DaemonSet | `demo-data-fuse` | FUSE mount client on nodes |
| Master Service | Service | `demo-data-master-0` | Exposes master to workers |
| Config | ConfigMap | `demo-data-alluxio-values` | Runtime configuration |
| PVC | PersistentVolumeClaim | `demo-data` | Volume claim for dataset access |
| PV | PersistentVolume | (auto-generated) | Backing volume |

### Label Conventions

Fluid uses consistent labels across resources:

| Label | Description | Example Value |
|-------|-------------|---------------|
| `app` | Application identifier | `alluxio` |
| `release` | Release name (Dataset name) | `demo-data` |
| `role` | Component role | `alluxio-master`, `alluxio-worker`, `alluxio-fuse` |
| `fluid.io/dataset` | Dataset name | `demo-data` |
| `fluid.io/dataset-namespace` | Dataset namespace | `default` |

### OwnerReferences

All Runtime-created resources include OwnerReferences pointing to the Runtime CR:

```yaml
ownerReferences:
  - apiVersion: data.fluid.io/v1alpha1
    kind: AlluxioRuntime
    name: demo-data
    uid: <runtime-uid>
    controller: true
    blockOwnerDeletion: true
```

---

## Failure Modes

### Common Failure Scenarios

| Failure Mode | Symptoms | Root Causes |
|-------------|----------|-------------|
| **Dataset Stuck Pending** | Dataset.Status.Phase = "Pending" for extended time | 1. No matching Runtime CR<br>2. Runtime controller not running<br>3. Runtime failed to initialize |
| **Runtime NotReady** | Runtime.Status.*Phase = "NotReady" | 1. Image pull failure<br>2. Resource quota exceeded<br>3. Node selector mismatch<br>4. PVC binding failed |
| **Fuse Not Scheduled** | fuseNumberReady < desiredFuseNumberScheduled | 1. Node selector excludes all nodes<br>2. DaemonSet tolerations missing<br>3. Node cordoned/drained |
| **Master Not Ready** | masterPhase = "NotReady" | 1. Master pod crash loop<br>2. UFS mount failure<br>3. Configuration error |
| **Worker Partial** | workerNumberReady < desiredWorkerNumberScheduled | 1. Some workers pending scheduling<br>2. Resource limits exceeded on some nodes<br>3. Storage provisioning failed |
| **PVC Bound but Pod Not Running** | PVC.Status.Phase = "Bound", but application pod pending | 1. Fuse not scheduled on target node<br>2. CSI driver issue<br>3. Volume attachment failure |

### Diagnostic Information Required

To diagnose these failures, the following information is needed:

1. **Dataset CR**: Phase, Conditions, Runtimes
2. **Runtime CR**: All phase fields, reason fields, replica counts
3. **Master StatefulSet**: replicas, readyReplicas, conditions
4. **Worker StatefulSet**: replicas, readyReplicas, conditions
5. **Fuse DaemonSet**: desiredNumberScheduled, numberReady, numberUnavailable
6. **PVC**: Phase, bound volume
7. **Pod Events**: For any non-ready pods

---

## Why kubectl-fluid-inspect is Needed

### The Problem

Currently, debugging Fluid issues requires:

1. **Multiple kubectl commands** to gather information:
   ```bash
   kubectl get dataset demo-data -o yaml
   kubectl get alluxioruntime demo-data -o yaml
   kubectl get sts -l release=demo-data
   kubectl get ds -l release=demo-data
   kubectl get pvc demo-data
   kubectl get events --field-selector involvedObject.name=demo-data
   ```

2. **Understanding of internal architecture** to know which resources to check

3. **Manual correlation** of status across multiple resources

4. **Domain expertise** to interpret the meaning of various status fields

### The Solution: kubectl-fluid-inspect

A single command that:

- **Aggregates** all relevant resource statuses
- **Correlates** Dataset ↔ Runtime ↔ K8s resources
- **Presents** a unified, human-readable view
- **Flags** potential issues with visual indicators

### Expected Output

```
================================================================================
                          FLUID DATASET INSPECTION
================================================================================

DATASET: demo-data
NAMESPACE: default
STATUS: Bound
UFS TOTAL: 100.5 GB
FILE COUNT: 15,432

================================================================================
RUNTIME: alluxio (AlluxioRuntime)
================================================================================

COMPONENT STATUS:
---------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Ready (3/3)
Fuse DaemonSet:     Partial (3/4) ⚠️

CACHE STATUS:
---------------------------------
Cached:     45.2 GB (45%)
Cacheable:  100.5 GB
Low Water:  10%
High Water: 80%

PERSISTENT VOLUME CLAIM:
---------------------------------
Name:   demo-data
Phase:  Bound
Volume: pvc-abc123

================================================================================
CONDITIONS:
================================================================================
Ready: True (DatasetReady) - The cache system is ready
RuntimeScheduled: True (Master is ready) - Master component initialized

================================================================================
```

---

## CLI Design Specification

### Command Structure

```
kubectl fluid inspect <subcommand> <name> [flags]
```

### Phase 1 Commands

| Command | Description |
|---------|-------------|
| `kubectl fluid inspect dataset <name>` | Inspect Dataset and all related resources |

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--namespace` | `-n` | Target namespace | `default` |
| `--output` | `-o` | Output format (text, json, yaml) | `text` |
| `--kubeconfig` | | Path to kubeconfig file | `$KUBECONFIG` or `~/.kube/config` |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - Dataset is healthy (Bound) |
| 1 | Warning - Dataset has issues but recoverable |
| 2 | Error - Dataset is in failed state |
| 3 | Not Found - Dataset or Runtime not found |

---

## Appendix A: Fluid Runtime Types Summary

| Runtime | Use Case | Underlying Technology |
|---------|----------|----------------------|
| AlluxioRuntime | General-purpose, S3/HDFS caching | Alluxio |
| JindoRuntime | Alibaba Cloud optimized | JindoFS |
| JuiceFSRuntime | S3-compatible with metadata separation | JuiceFS |
| EFCRuntime | NAS with distributed cache | Elastic File Client |
| ThinRuntime | Custom/third-party storage | Pluggable |

---

## Appendix B: API References

- **Fluid API v1alpha1**: `github.com/fluid-cloudnative/fluid/api/v1alpha1`
- **Dataset Types**: `dataset_types.go`
- **Runtime Status**: `status.go`
- **Alluxio Runtime**: `alluxioruntime_types.go`

---

## Appendix C: Key Status Transition Constants

```go
// Dataset Condition Reasons
const (
    DatasetReadyReason         = "DatasetReady"
    DatasetUpdatingReason      = "DatasetUpdating"
    DatasetDataSetFailedReason = "DatasetFailed"
    DatasetFailedToSetupReason = "DatasetFailedToSetup"
)

// Runtime Condition Reasons
const (
    RuntimeMasterInitializedReason = "Master is initialized"
    RuntimeMasterReadyReason       = "Master is ready"
    RuntimeWorkersInitializedReason = "Workers are initialized"
    RuntimeWorkersReadyReason      = "Workers are ready"
    RuntimeFusesInitializedReason  = "Fuses are initialized"
    RuntimeFusesReadyReason        = "Fuses are ready"
)
```

---

**End of Phase 0 Design Document**
