# Phase 2 & 3: Diagnose Design & AI-Ready Framework

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Diagnose Pipeline](#diagnose-pipeline)
4. [AI-Ready Framework](#ai-ready-framework)
5. [Output Formats](#output-formats)
6. [Usage Guide](#usage-guide)

---

## Executive Summary

This document describes the design and implementation of the `kubectl fluid diagnose` command and the AI-ready diagnostic framework. These features extend kubectl-fluid-inspect to provide comprehensive debugging capabilities while maintaining safety (read-only operations) and preparing for optional AI integration.

### Key Features

| Feature | Description |
|---------|-------------|
| **CR Snapshots** | Clean YAML exports of Dataset and Runtime CRs |
| **Event Collection** | Chronologically sorted Kubernetes events |
| **Resource Status** | Detailed pod-level status for all components |
| **Log Collection** | Tail logs from master, worker, and failing fuse pods |
| **Failure Analysis** | Automatic detection with severity classification |
| **Archive Export** | Shareable .tar.gz diagnostic bundles |
| **AI-Ready Context** | Structured JSON for LLM consumption |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         kubectl-fluid-inspect                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌───────────────┐     ┌───────────────┐     ┌───────────────────────────┐  │
│  │    INSPECT    │     │   DIAGNOSE    │     │      AI LAYER (Future)   │  │
│  │               │     │               │     │                           │  │
│  │ Quick Status  │     │ Deep Analysis │────▶│ DiagnosticContext        │  │
│  │ Overview      │     │ + Logs        │     │        │                  │  │
│  │               │     │ + Events      │     │        ▼                  │  │
│  └───────────────┘     │ + Hints       │     │ LLM Analysis (optional)  │  │
│         │              └───────────────┘     └───────────────────────────┘  │
│         │                     │                         │                    │
│         │                     │                         │                    │
│         └──────────┬──────────┘                         │                    │
│                    │                                    │                    │
│                    ▼                                    │                    │
│  ┌─────────────────────────────────────────────────────┴──────────────────┐ │
│  │                         K8s Client Layer                                │ │
│  │                                                                         │ │
│  │   GET Dataset    GET Runtime    GET Events    GET Pods    GET Logs     │ │
│  │                                                                         │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ READ-ONLY
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Kubernetes API Server                                │
│                                                                              │
│   Datasets    Runtimes    StatefulSets    DaemonSets    Pods    Events     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layer Separation

| Layer | Responsibility | AI Integration |
|-------|----------------|----------------|
| **Inspect** | Quick status overview | Not needed |
| **Diagnose** | Deep analysis + data collection | Prepares DiagnosticContext |
| **AI Layer** | LLM-based analysis (future) | Consumes DiagnosticContext |

---

## Diagnose Pipeline

The `diagnose` command implements a deterministic, ordered data collection pipeline:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DIAGNOSE PIPELINE                                    │
└─────────────────────────────────────────────────────────────────────────────┘

Step 1: CR Snapshots                    Step 2: Events
┌──────────────────────┐               ┌──────────────────────┐
│ GET Dataset CR       │               │ GET Events           │
│ GET Runtime CR       │               │ - Dataset UID        │
│ Clean metadata:      │               │ - Runtime UID        │
│ - managedFields      │      ───▶     │ - Related pods       │
│ - resourceVersion    │               │                      │
│ - uid                │               │ Sort chronologically │
└──────────────────────┘               └──────────────────────┘
                                                 │
                           ┌─────────────────────┘
                           │
                           ▼
Step 3: Resource Status                 Step 4: Logs
┌──────────────────────┐               ┌──────────────────────┐
│ GET Master STS       │               │ Master Container     │
│   └─ List Pods       │               │   └─ Last 100 lines  │
│ GET Worker STS       │      ───▶     │ Worker Container     │
│   └─ List Pods       │               │   └─ 1 healthy pod   │
│ GET Fuse DaemonSet   │               │   └─ 1 failing pod   │
│   └─ List Pods       │               │ Fuse Container       │
│ GET PVC/PV           │               │   └─ Failing pods    │
└──────────────────────┘               └──────────────────────┘
                                                 │
                           ┌─────────────────────┘
                           │
                           ▼
Step 5: Analysis                        Step 6: Output
┌──────────────────────┐               ┌──────────────────────┐
│ Pattern Detection:   │               │ Text: Human-readable │
│ - ImagePullBackOff   │               │ JSON: AI-ready       │
│ - Insufficient       │      ───▶     │ Archive: Shareable   │
│ - FailedMount        │               │         .tar.gz      │
│ - High restarts      │               │                      │
│ Generate hints       │               │                      │
└──────────────────────┘               └──────────────────────┘
```

### Pipeline Stages Detail

| Stage | Data Collected | Error Handling |
|-------|----------------|----------------|
| **1. CR Snapshots** | Dataset YAML, Runtime YAML | Fatal if Dataset not found |
| **2. Events** | All related K8s events | Non-fatal, continue with warnings |
| **3. Resources** | StatefulSet, DaemonSet, PVC status | Non-fatal, partial results |
| **4. Logs** | Container logs (tail 100 lines) | Non-fatal, mark as error |
| **5. Analysis** | Failure hints with severity | Always runs |
| **6. Output** | Formatted result | Always succeeds |

---

## AI-Ready Framework

### DiagnosticContext Structure

The `DiagnosticContext` struct is designed for LLM consumption:

```go
type DiagnosticContext struct {
    // Structured summary for quick understanding
    Summary        ContextSummary    `json:"summary"`
    
    // Raw data for detailed analysis
    DatasetYAML    string            `json:"datasetYaml"`
    RuntimeYAML    string            `json:"runtimeYaml,omitempty"`
    Events         []EventInfo       `json:"events"`
    Logs           map[string]string `json:"logs"`
    
    // Pre-analyzed hints (optional assistance)
    FailureHints   []FailureHint     `json:"failureHints"`
    
    // Metadata
    CollectedAt    time.Time         `json:"collectedAt"`
    Version        string            `json:"version"`
}

type ContextSummary struct {
    DatasetName    string       `json:"datasetName"`
    Namespace      string       `json:"namespace"`
    DatasetPhase   string       `json:"datasetPhase"`
    RuntimeType    string       `json:"runtimeType"`
    HealthStatus   HealthStatus `json:"healthStatus"`
    MasterReady    string       `json:"masterReady"`   // "1/1"
    WorkersReady   string       `json:"workersReady"`  // "3/3"
    FuseReady      string       `json:"fuseReady"`     // "3/4"
    PVCStatus      string       `json:"pvcStatus"`
    ErrorCount     int          `json:"errorCount"`
    WarningCount   int          `json:"warningCount"`
}
```

### Data Flow: Diagnose → Context → AI

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DATA FLOW                                            │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────┐     ┌──────────────────┐     ┌─────────────────────────────┐
│             │     │                  │     │                             │
│  Diagnose   │────▶│  DiagnosticResult│────▶│   DiagnosticContext (JSON) │
│  Command    │     │                  │     │                             │
└─────────────┘     └──────────────────┘     └──────────────┬──────────────┘
                                                            │
                                             ┌──────────────┴──────────────┐
                                             │                             │
                                             ▼                             ▼
                                    ┌────────────────┐           ┌────────────────┐
                                    │ Text Output    │           │ Archive (.tar) │
                                    │ (Terminal)     │           │ context.json   │
                                    └────────────────┘           └────────────────┘
                                                                         │
                                                            ┌────────────┴────────────┐
                                                            │                         │
                                                            ▼                         ▼
                                                   ┌────────────────┐       ┌────────────────┐
                                                   │ Share with     │       │ Feed to LLM    │
                                                   │ Maintainers    │       │ (Optional)     │
                                                   └────────────────┘       └────────────────┘
```

### Why AI is Optional

The design ensures the tool works completely offline:

1. **Pre-computed hints**: The `analyzeAndGenerateHints()` function provides value without any LLM
2. **Structured data**: All information is organized for both human and machine consumption
3. **No network calls**: AI integration is a future layer, not a dependency
4. **Privacy-first**: Secrets are redacted, logs are normalized

### How AI Can Be Plugged Later

```go
// Future AI integration example
type AIAnalyzer interface {
    Analyze(ctx *types.DiagnosticContext) (*AIAnalysisResult, error)
}

// Example implementation (future)
type OpenAIAnalyzer struct {
    client *openai.Client
}

func (a *OpenAIAnalyzer) Analyze(ctx *types.DiagnosticContext) (*AIAnalysisResult, error) {
    prompt := BuildAnalysisPrompt(ctx)
    response, err := a.client.CreateChatCompletion(ctx, prompt)
    // ...
}
```

---

## Output Formats

### Text Output (Default)

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                      FLUID DATASET DIAGNOSTIC REPORT                        ║
╚══════════════════════════════════════════════════════════════════════════════╝

  Dataset: demo-data
  Namespace: default
  Collected At: 2026-02-08 00:30:45
  Health Status: ⚠️  Degraded

=== RESOURCE HIERARCHY ===

  📦 Dataset: demo-data [Bound ✓]
  │
  └── ⚙️ Runtime: alluxio
      ├── 📊 Master: 1/1 ✓
      ├── 📊 Workers: 3/3 ✓
      └── 📊 Fuse: 3/4 ✗
          └── ⚠️ demo-data-fuse-abc: CrashLoopBackOff

=== DETECTED ISSUES ===

  WARNINGS:
  ⚠️ Fuse not healthy: 3/4 ready [fuse]
     → Check fuse pod logs and node selectors/tolerations

=== RECENT EVENTS ===

  TYPE         OBJECT               REASON          MESSAGE
  ----------------------------------------------------------------------------
  Warning      demo-data-fuse-abc   BackOff         Back-off restarting failed...
  Normal       demo-data-master-0   Pulled          Successfully pulled image...

=== LOGS (TAIL) ===

  ┌─ FUSE-0 (FAILING) [demo-data-fuse-abc/alluxio-fuse] (100 lines)
  │
  │ ... 85 lines truncated ...
  │ 2026-02-08 00:30:00 ERROR - Failed to connect to master
  │ 2026-02-08 00:30:01 ERROR - Mount failed: connection refused
  │
  └─

═══════════════════════════════════════════════════════════════════════════════
```

### JSON Output (AI-Ready)

```json
{
  "summary": {
    "datasetName": "demo-data",
    "namespace": "default",
    "datasetPhase": "Bound",
    "runtimeType": "alluxioruntimes",
    "healthStatus": "Degraded",
    "masterReady": "1/1",
    "workersReady": "3/3",
    "fuseReady": "3/4",
    "pvcStatus": "Bound",
    "errorCount": 1,
    "warningCount": 3
  },
  "datasetYaml": "...",
  "runtimeYaml": "...",
  "events": [...],
  "logs": {
    "master": "...",
    "worker-0": "...",
    "fuse-0": "..."
  },
  "failureHints": [
    {
      "severity": "warning",
      "component": "fuse",
      "issue": "Fuse not healthy: 3/4 ready",
      "suggestion": "Check fuse pod logs and node selectors/tolerations"
    }
  ],
  "collectedAt": "2026-02-08T00:30:45Z",
  "version": "1.0"
}
```

### Archive Contents

```
fluid-diagnose-demo-data-20260208-003045.tar.gz
├── dataset.yaml        # Clean Dataset CR
├── runtime.yaml        # Clean Runtime CR
├── events.log          # Formatted events
├── resources.json      # Resource status
├── failure_hints.json  # Detected issues
├── summary.txt         # Human-readable summary
├── context.json        # AI-ready context
└── pods/
    ├── master.log      # Master container logs
    ├── worker-0.log    # Worker container logs
    └── fuse-0.log      # Fuse container logs
```

---

## Usage Guide

### Quick Diagnosis

```bash
# Basic diagnosis
kubectl fluid diagnose dataset demo-data

# With namespace
kubectl fluid diagnose dataset demo-data -n fluid-system
```

### AI Integration Export

```bash
# Export JSON for AI tools
kubectl fluid diagnose dataset demo-data --output json > diagnosis.json

# Pipe to LLM CLI (future)
kubectl fluid diagnose dataset demo-data --output json | llm analyze
```

### Sharing with Maintainers

```bash
# Create shareable archive
kubectl fluid diagnose dataset demo-data --archive

# Output: fluid-diagnose-demo-data-20260208-003045.tar.gz
```

---

## Implementation Summary

### Files Added

| File | Purpose |
|------|---------|
| `pkg/cmd/diagnose.go` | Diagnose parent command |
| `pkg/cmd/diagnose_dataset.go` | Dataset diagnosis command |
| `pkg/types/diagnostic.go` | Diagnostic type definitions |
| `pkg/diagnose/diagnoser.go` | Core diagnostic engine |
| `pkg/output/diagnostic_printer.go` | Color-coded text output |
| `pkg/output/archiver.go` | Archive generation |
| `pkg/k8s/events_logs.go` | Events and logs fetching |

### Key Design Decisions

1. **Non-fatal errors**: Pipeline continues even if some data collection fails
2. **Deterministic ordering**: Events sorted chronologically, hints grouped by severity
3. **Secret redaction**: Logs are scanned and sensitive lines replaced with `[REDACTED]`
4. **Minimal logs**: Only failing pods' logs collected to reduce noise
5. **AI-ready structure**: `DiagnosticContext` designed for LLM token efficiency

---

**End of Phase 2 & 3 Design Document**
