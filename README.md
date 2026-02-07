# kubectl-fluid-inspect

[![Go Version](https://img.shields.io/github/go-mod/go-version/mrhapile/kubectl-fluid-inspect)](go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A kubectl plugin for inspecting and diagnosing [CNCF Fluid](https://github.com/fluid-cloudnative/fluid) Datasets and Runtimes in a unified view.

## Overview

**kubectl-fluid-inspect** provides streamlined tools for understanding and debugging Fluid Datasets. Instead of running multiple `kubectl` commands to diagnose issues, this plugin aggregates all relevant information and presents it in human-friendly and machine-readable formats.

### Commands

| Command | Purpose |
|---------|---------|
| `kubectl fluid inspect` | Quick status overview of Dataset and Runtime |
| `kubectl fluid diagnose` | Comprehensive debugging with logs, events, and failure analysis |

### Key Features

- ✅ **Read-only, safe operations** - Only `GET` API calls, never modifies resources
- ✅ **Unified view** - Aggregates Dataset + Runtime + K8s resources
- ✅ **Multi-runtime support** - Alluxio, Jindo, JuiceFS, EFC, Thin, Vineyard, GooseFS
- ✅ **Visual indicators** - Color-coded ✓ ⚠️ ❌ for status
- ✅ **Failure analysis** - Automatic detection of common issues
- ✅ **AI-ready export** - Structured JSON output for LLM integration
- ✅ **Shareable archives** - Generate `.tar.gz` bundles for maintainers
- ✅ **Mock mode** - Demo without a cluster using `--mock`

---

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mrhapile/kubectl-fluid-inspect.git
cd kubectl-fluid-inspect

# Build the binary
make build
# or: go build -o bin/kubectl-fluid ./cmd/kubectl-fluid/

# Install to PATH (optional)
sudo mv bin/kubectl-fluid /usr/local/bin/
```

### Verify Installation

```bash
kubectl fluid --help
```

---

## Quick Start

### Inspect Command

Quick status overview:

```bash
# Inspect a Dataset
kubectl fluid inspect dataset demo-data

# With namespace
kubectl fluid inspect dataset demo-data -n fluid-system
```

**Example Output:**

```
================================================================================
                          FLUID DATASET INSPECTION
================================================================================

DATASET: demo-data
NAMESPACE: default
STATUS: Bound ✓
UFS TOTAL: 50.2GB

================================================================================
RUNTIME: demo-data (AlluxioRuntime)
================================================================================

COMPONENT STATUS:
----------------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Ready (3/3)
Fuse DaemonSet:     Ready (3/3)

================================================================================
```

### Diagnose Command

Comprehensive debugging:

```bash
# Full diagnosis with events and logs
kubectl fluid diagnose dataset demo-data

# Export AI-ready JSON
kubectl fluid diagnose dataset demo-data --output json

# Generate shareable archive
kubectl fluid diagnose dataset demo-data --archive
```

**Example Output (Text):**

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

=== DETECTED ISSUES ===

  WARNINGS:
  ⚠️ Fuse not healthy: 3/4 ready [fuse]
     → Check fuse pod logs and node selectors/tolerations

=== RECENT EVENTS ===

  TYPE         OBJECT               REASON          MESSAGE
  ----------------------------------------------------------------------------
  Warning      demo-data-fuse-xyz   BackOff         Back-off restarting failed...
  Normal       demo-data-master-0   Pulled          Successfully pulled image...

=== LOGS (TAIL) ===

  ┌─ FUSE-0 (FAILING) [demo-data-fuse-xyz/alluxio-fuse] (100 lines)
  │
  │ 2026-02-08 00:30:00 ERROR - Failed to connect to master
  │ 2026-02-08 00:30:01 ERROR - Mount failed: connection refused
  │
  └─
```

---

## Command Reference

### inspect dataset

Quick status overview of a Dataset and bound Runtime.

```bash
kubectl fluid inspect dataset <name> [flags]
```

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--namespace` | `-n` | Target namespace | `default` |
| `--kubeconfig` | | Path to kubeconfig | `$KUBECONFIG` |

### diagnose dataset

Comprehensive debugging with CR snapshots, events, resource status, and logs.

```bash
kubectl fluid diagnose dataset <name> [flags]
```

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--namespace` | `-n` | Target namespace | `default` |
| `--output` | `-o` | Output format: `text`, `json` | `text` |
| `--archive` | | Generate `.tar.gz` archive | `false` |
| `--mock` | | Use mock data (no cluster required) | `false` |
| `--kubeconfig` | | Path to kubeconfig | `$KUBECONFIG` |

---

## AI-Ready Integration

The diagnose command can export structured JSON suitable for LLM analysis:

```bash
# Export AI-ready context
kubectl fluid diagnose dataset demo-data --output json > diagnosis.json
```

### DiagnosticContext Structure

```json
{
  "summary": {
    "datasetName": "demo-data",
    "namespace": "default",
    "healthStatus": "Degraded",
    "masterReady": "1/1",
    "workersReady": "3/3",
    "fuseReady": "3/4",
    "errorCount": 1,
    "warningCount": 2
  },
  "datasetYaml": "...",
  "events": [...],
  "logs": {
    "master": "...",
    "worker-0": "...",
    "fuse-0": "..."
  },
  "failureHints": [...]
}
```

### Why AI is Optional

- **Works offline**: All analysis happens locally without external API calls
- **Privacy-first**: Secrets are redacted from logs
- **Pre-computed hints**: Failure patterns detected without LLM
- **Future-ready**: Structured for easy LLM integration when needed

---

## Archive Format

When using `--archive`, a `.tar.gz` file is created:

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
    ├── master.log
    ├── worker-0.log
    └── fuse-0.log
```

---

## 🧪 Mock Diagnose Mode (No Cluster Required)

The `--mock` flag enables running the full diagnostic pipeline without a Kubernetes cluster. This is ideal for:

- **Demos and screenshots** - Show realistic output without infrastructure
- **Documentation** - Generate example output for docs and proposals
- **Development** - Test output formatting without cluster access
- **CI/CD** - Validate CLI behavior in environments without K8s

### Why Mock Mode Exists

Real Fluid deployments require:
- A running Kubernetes cluster
- Fluid operator installed
- Dataset and Runtime CRs deployed

Mock mode removes these dependencies while producing **identical output structure** to real diagnoses.

### What Mock Mode Provides

The mock data simulates a realistic **degraded Fluid deployment** with:

| Component | Status | Issue |
|-----------|--------|-------|
| Dataset | Bound | ✓ Healthy |
| Master | 1/1 Ready | ✓ Healthy |
| Workers | 1/2 Ready | ⚠️ Insufficient memory |
| Fuse | 2/3 Ready | ❌ Node taint not tolerated |
| PVC | Bound | ✓ Healthy |

This includes realistic:
- Kubernetes events (FailedScheduling, Unhealthy, FailedMount, Evicted)
- Container logs with error patterns
- Failure hints with actionable suggestions

### Usage Examples

```bash
# Basic mock diagnosis
./bin/kubectl-fluid diagnose dataset demo-data --mock

# Mock with JSON output (AI-ready)
./bin/kubectl-fluid diagnose dataset demo-data --mock -o json

# Generate mock archive for sharing
./bin/kubectl-fluid diagnose dataset demo-data --mock --archive

# Specify custom namespace (reflected in output)
./bin/kubectl-fluid diagnose dataset my-dataset --mock -n production
```

### Pipeline Equivalence

| Real Mode | Mock Mode |
|-----------|-----------|
| Connects to K8s API | No network calls |
| Fetches real CRs | Returns mock CRs |
| Reads pod logs | Returns mock logs |
| Queries events | Returns mock events |
| Same printers | Same printers |
| Same archivers | Same archivers |

Both modes use **100% identical output logic**.

---

## Architecture

```
kubectl-fluid
├── inspect     ─────▶ Quick status check
│                      (Dataset + Runtime + Resources)
│
└── diagnose    ─────▶ Deep analysis
                       │
                       ├── CR Snapshots (clean YAML)
                       ├── Events Collection
                       ├── Resource Status
                       ├── Log Collection
                       └── Failure Analysis
                                │
                                ├── Text Output (terminal)
                                ├── JSON Output (AI-ready)
                                └── Archive (.tar.gz)
```

---

## Supported Runtimes

| Runtime | Resource | Status |
|---------|----------|--------|
| Alluxio | `AlluxioRuntime` | ✅ Supported |
| Jindo | `JindoRuntime` | ✅ Supported |
| JuiceFS | `JuiceFSRuntime` | ✅ Supported |
| EFC | `EFCRuntime` | ✅ Supported |
| Thin | `ThinRuntime` | ✅ Supported |
| Vineyard | `VineyardRuntime` | ✅ Supported |
| GooseFS | `GooseFSRuntime` | ✅ Supported |

---

## Development

### Prerequisites

- Go 1.21+
- Access to a Kubernetes cluster with Fluid installed

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Project Structure

```
kubectl-fluid-inspect/
├── cmd/kubectl-fluid/main.go
├── pkg/
│   ├── cmd/              # CLI commands (Cobra)
│   ├── inspect/          # Inspect logic
│   ├── diagnose/         # Diagnose logic
│   ├── k8s/              # Kubernetes client
│   ├── output/           # Output formatters
│   └── types/            # Type definitions
├── PHASE0_DESIGN.md      # Architecture design
├── PHASE2_3_DESIGN.md    # Diagnose & AI design
├── README.md
├── Makefile
└── go.mod
```

---

## Design Documents

- [PHASE0_DESIGN.md](PHASE0_DESIGN.md) - Architecture analysis, CRD mappings, failure modes
- [PHASE2_3_DESIGN.md](PHASE2_3_DESIGN.md) - Diagnose pipeline, AI-ready framework

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

Apache License 2.0

## Related Projects

- [CNCF Fluid](https://github.com/fluid-cloudnative/fluid) - The data acceleration framework this plugin inspects

## Acknowledgments

This project was created as part of exploring opportunities for CNCF LFX mentorship contributions.
