# kubectl-fluid-inspect

[![Go Version](https://img.shields.io/github/go-mod/go-version/mrhapile/kubectl-fluid-inspect)](go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A kubectl plugin for inspecting [CNCF Fluid](https://github.com/fluid-cloudnative/fluid) Datasets and Runtimes in a unified view.

## Overview

**kubectl-fluid-inspect** provides a streamlined way to inspect Fluid Dataset status and all underlying Kubernetes resources in a single command. Instead of running multiple `kubectl` commands to debug Fluid issues, this plugin aggregates all relevant information and presents it in a human-friendly format.

### What It Does

- ✅ Fetches Dataset CR status (phase, conditions)
- ✅ Detects and fetches bound Runtime CR (Alluxio, Jindo, JuiceFS, EFC, Thin, etc.)
- ✅ Retrieves Master/Worker StatefulSet status
- ✅ Retrieves Fuse DaemonSet status
- ✅ Retrieves PersistentVolumeClaim status
- ✅ Visual indicators for healthy/unhealthy components

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/mrhapile/kubectl-fluid-inspect.git
cd kubectl-fluid-inspect

# Build the binary
go build -o bin/kubectl-fluid ./cmd/kubectl-fluid/

# Move to PATH (optional)
sudo mv bin/kubectl-fluid /usr/local/bin/

# Or add to kubectl plugins directory
mkdir -p ~/.kube/plugins/
cp bin/kubectl-fluid ~/.kube/plugins/
```

### Verify Installation

```bash
kubectl fluid --help
```

## Usage

### Basic Usage

```bash
# Inspect a Dataset in the default namespace
kubectl fluid inspect dataset <dataset-name>

# Inspect a Dataset in a specific namespace
kubectl fluid inspect dataset <dataset-name> -n <namespace>

# Use a custom kubeconfig
kubectl fluid inspect dataset <dataset-name> --kubeconfig ~/.kube/custom-config
```

### Example Output

```
================================================================================
                          FLUID DATASET INSPECTION
================================================================================

DATASET: spark-data
NAMESPACE: fluid-system
STATUS: Bound ✓
UFS TOTAL: 50.2GB
FILE COUNT: 5,432

MOUNT POINTS:
  - cos://my-bucket.cos.region.myqcloud.com/spark/

================================================================================
RUNTIME: spark-data (AlluxioRuntime)
================================================================================

COMPONENT STATUS:
----------------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Ready (3/3)
Fuse DaemonSet:     Ready (3/3)

KUBERNETES RESOURCES:
----------------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Ready (3/3)
Fuse DaemonSet:     Ready (3/3)
PVC:                Bound (pvc-abc12345)

================================================================================
CONDITIONS:
================================================================================
✓ Ready: True (DatasetReady)
   The cache system is ready
✓ RuntimeScheduled: True (RuntimeScheduled)
   The runtime is scheduled

================================================================================
```

### When Issues Exist

The output uses visual indicators to highlight problems:

```
================================================================================
COMPONENT STATUS:
----------------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Not Ready (2/3) ⚠️
Fuse DaemonSet:     Ready (3/3)

KUBERNETES RESOURCES:
----------------------------------------
Master StatefulSet: Ready (1/1)
Worker StatefulSet: Not Ready (2/3) ⚠️
Fuse DaemonSet:     Ready (3/3)
PVC:                Pending ⚠️
================================================================================
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--namespace` | `-n` | Target namespace | `default` |
| `--kubeconfig` | | Path to kubeconfig file | `$KUBECONFIG` or `~/.kube/config` |
| `--help` | `-h` | Help for the command | |

## Architecture

This plugin is designed as a **read-only, safe** tool that only performs `GET` operations against the Kubernetes API. It never modifies any resources.

### Key Design Principles

1. **Safety First**: Read-only operations only
2. **Single Source of Truth**: Aggregates from multiple resources
3. **Human-Friendly**: Clear output with visual indicators
4. **Zero Dependencies**: No external tools required beyond kubeconfig

### How It Works

1. Fetches the Dataset CR using the dynamic client
2. Detects the bound Runtime type (Alluxio, Jindo, JuiceFS, etc.)
3. Fetches the corresponding Runtime CR
4. Uses naming conventions to find related StatefulSets (master, worker) and DaemonSet (fuse)
5. Fetches the PVC using the Dataset name
6. Aggregates all status information
7. Formats and outputs the result

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

## Development

### Prerequisites

- Go 1.21+
- Access to a Kubernetes cluster with Fluid installed

### Building

```bash
make build
# or
go build -o bin/kubectl-fluid ./cmd/kubectl-fluid/
```

### Running Tests

```bash
make test
# or
go test ./...
```

### Project Structure

```
kubectl-fluid-inspect/
├── cmd/
│   └── kubectl-fluid/
│       └── main.go           # Entry point
├── pkg/
│   ├── cmd/                   # CLI commands (Cobra)
│   │   ├── root.go
│   │   ├── inspect.go
│   │   └── inspect_dataset.go
│   ├── inspect/               # Inspection logic
│   │   └── inspector.go
│   ├── k8s/                   # Kubernetes client
│   │   └── client.go
│   ├── output/                # Output formatters
│   │   └── printer.go
│   └── types/                 # Type definitions
│       └── types.go
├── PHASE0_DESIGN.md           # Architecture design document
├── go.mod
├── go.sum
└── README.md
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## License

Apache License 2.0

## Related Projects

- [CNCF Fluid](https://github.com/fluid-cloudnative/fluid) - The data acceleration framework this plugin inspects
- [kubectl](https://kubernetes.io/docs/reference/kubectl/) - The Kubernetes CLI this extends

## Acknowledgments

This project was created as part of exploring opportunities for CNCF LFX mentorship contributions.
