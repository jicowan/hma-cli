# hma-cli

A CLI tool to simulate failure conditions on EKS worker nodes for testing the [EKS Node Health Monitoring Agent (NMA)](https://docs.aws.amazon.com/eks/latest/userguide/node-health-nma.html).

## Overview

The EKS Node Monitoring Agent continuously monitors node health across 5 categories and can trigger automatic node repairs. This CLI helps you test the NMA's detection capabilities by simulating various failure conditions.

## Installation

```bash
# Clone and build
git clone https://github.com/jicowan/hma-cli.git
cd hma-cli
make build

# Or install directly
go install github.com/jicowan/hma-cli/cmd/hma-cli@latest
```

### Build for Linux (EKS nodes)

```bash
make build-linux
# Creates: hma-cli-linux-amd64, hma-cli-linux-arm64
```

## Usage

```bash
hma-cli [--node <node-name>] <category> <failure-type> [flags]
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--node` | Target node name (creates privileged pod automatically) |
| `--kubeconfig` | Path to kubeconfig (default: ~/.kube/config) |
| `--dry-run` | Show what would happen without executing |
| `--duration` | Auto-cleanup after duration (e.g., 5m) |
| `--force` | Skip confirmation prompts |
| `--cleanup` | Revert a previous simulation |

## Available Simulations

### Kernel (`KernelReady` condition)

```bash
# Create zombie processes (threshold: >= 20)
hma-cli kernel zombies --count 25

# Exhaust PIDs (threshold: > 70%)
hma-cli kernel pid-exhaustion

# Inject kernel log patterns
hma-cli kernel fork-oom
hma-cli kernel kernel-bug
hma-cli kernel soft-lockup
```

### Networking (`NetworkingReady` condition)

```bash
# Kill IPAMD process
hma-cli networking ipamd-down

# Delete VPC routes
hma-cli networking routes-missing

# Bring down secondary ENI
hma-cli networking interface-down --target eth1
```

### Storage (`StorageReady` condition)

```bash
# Create I/O pressure (requires stress-ng)
hma-cli storage io-delay
```

### Runtime (`ContainerRuntimeReady` condition)

```bash
# Restart kubelet repeatedly (threshold: > 3 in 5 min)
hma-cli runtime systemd-restarts --service kubelet --count 4
```

### Accelerator (`AcceleratedHardwareReady` condition)

```bash
# Inject NVIDIA XID error (fatal codes: 13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140)
hma-cli accelerator xid-error --code 79

# Inject AWS Neuron errors
hma-cli accelerator neuron-sram-error
hma-cli accelerator neuron-hbm-error
hma-cli accelerator neuron-nc-error
hma-cli accelerator neuron-dma-error
```

### NodeDiagnostic (Log Collection)

Create a `NodeDiagnostic` CR to collect logs from a node:

```bash
# Create NodeDiagnostic CR
hma-cli diagnose --node ip-10-0-1-123.ec2.internal \
  --destination "https://mybucket.s3.amazonaws.com/logs.tar.gz?X-Amz-..."

# Wait for completion
hma-cli diagnose --node my-node --destination "https://..." --wait

# Check status
hma-cli diagnose --node my-node --status

# Create, wait, then delete
hma-cli diagnose --node my-node --destination "https://..." --wait --delete
```

## Examples

### Dry Run

See what a simulation would do without executing:

```bash
hma-cli kernel zombies --dry-run
```

### Auto-Cleanup

Automatically revert after a duration:

```bash
hma-cli kernel zombies --duration 2m
```

### Manual Cleanup

Revert a simulation manually:

```bash
hma-cli kernel zombies --cleanup
```

### List All Simulations

```bash
hma-cli list
```

## Verification

After running a simulation, verify NMA detection:

```bash
# Check NMA logs
kubectl logs -n kube-system -l app=eks-node-monitoring-agent

# Check node conditions
kubectl get node <node-name> -o jsonpath='{.status.conditions}' | jq
```

## Requirements

- Go 1.21+ (for building)
- Root/sudo access on target node
- `stress-ng` for storage simulations (optional)
- kubectl access to EKS cluster (for `--node` flag and `diagnose` command)

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint
```

## How It Works

### Node Access

When using `--node`, the CLI:
1. Creates a privileged pod on the target node
2. Uses `nsenter` to enter the node's namespaces
3. Executes simulation commands
4. Cleans up the pod when done

### Failure Detection

The NMA monitors:
- **Kernel**: `/proc` filesystem, dmesg patterns
- **Networking**: IPAMD process, routes, interfaces
- **Storage**: I/O latency, EBS metrics
- **Runtime**: systemd service restarts
- **Accelerator**: NVIDIA DCGM, Neuron dmesg patterns

## License

Apache 2.0
