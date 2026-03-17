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
hma-cli --node <node-name> <category> <failure-type> [flags]
```

**Important:** The `--node` flag is required for all simulations. The CLI creates a privileged pod on the target node to execute commands.

### Global Flags

| Flag | Description |
|------|-------------|
| `--node` | **Required.** Target node name (creates privileged pod automatically) |
| `--keep-alive` | Keep pod alive for duration (e.g., `30m`). **Required for process-based simulations.** |
| `--kubeconfig` | Path to kubeconfig (default: ~/.kube/config) |
| `--dry-run` | Show what would happen without executing |
| `--force` | Skip confirmation prompts |
| `--cleanup` | Revert simulation. **Needed for: `pid-exhaustion`, `interface-down`** |

## Available Simulations

### Kernel (`KernelReady` condition)

```bash
# Create zombie processes (threshold: >= 20)
# Requires --keep-alive to prevent processes from being killed on pod exit
hma-cli --node <node-name> kernel zombies --count 25 --keep-alive 30m --force

# Exhaust PIDs (threshold: > 70% of MAX(pid_max, threads-max))
# Requires --keep-alive to keep sleep processes alive
hma-cli --node <node-name> kernel pid-exhaustion --keep-alive 30m --force

# Inject kernel log patterns (dmesg injection, no --keep-alive needed)
hma-cli --node <node-name> kernel kernel-bug --force    # Creates Warning event
hma-cli --node <node-name> kernel soft-lockup --force   # Creates Warning event

# Exhaust PIDs to cause kubelet fork failures - triggers KernelReady=False
# WARNING: This may make the node unrecoverable and require node replacement!
hma-cli --node <node-name> kernel fork-oom --force
```

> **Warning:** The `fork-oom` simulation exhausts node PIDs and may make the node unrecoverable. The node may need to be deleted and replaced after running this simulation. Only use on nodes you can afford to lose.

### Networking (`NetworkingReady` condition)

```bash
# Kill IPAMD process - triggers NetworkingReady=False
hma-cli --node <node-name> networking ipamd-down --force

# Bring down secondary ENI (eth1) - triggers NetworkingReady=False
hma-cli --node <node-name> networking interface-down --force
```

### Storage (`StorageReady` condition)

```bash
# Create I/O delay process (NMA checks every 10 minutes)
# Requires --keep-alive for at least 15 minutes
hma-cli --node <node-name> storage io-delay --keep-alive 15m --force
```

### Runtime (`ContainerRuntimeReady` condition)

```bash
# Kill kubelet process repeatedly to increment NRestarts counter
# NMA threshold: NRestarts > 3 AND increasing
# Requires --keep-alive to complete all kills
hma-cli --node <node-name> runtime systemd-restarts --keep-alive 10m --force
```

### Accelerator (`AcceleratedHardwareReady` condition)

```bash
# Inject NVIDIA XID error (requires DCGM installed)
# Fatal codes: 13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140
hma-cli --node <node-name> accelerator xid-error --code 79 --force

# Inject AWS Neuron errors (dmesg injection)
hma-cli --node <node-name> accelerator neuron-sram-error --force
hma-cli --node <node-name> accelerator neuron-hbm-error --force
hma-cli --node <node-name> accelerator neuron-nc-error --force
hma-cli --node <node-name> accelerator neuron-dma-error --force
```

### NodeDiagnostic (Log Collection)

Create a `NodeDiagnostic` CR to collect logs from a node:

```bash
# Create NodeDiagnostic CR
hma-cli diagnose --node <node-name> \
  --destination "https://mybucket.s3.amazonaws.com/logs.tar.gz?X-Amz-..."

# Wait for completion
hma-cli diagnose --node <node-name> --destination "https://..." --wait

# Check status
hma-cli diagnose --node <node-name> --status

# Create, wait, then delete
hma-cli diagnose --node <node-name> --destination "https://..." --wait --delete
```

## Understanding `--keep-alive` and `--cleanup`

Some simulations create processes that must remain running for NMA to detect them. Without `--keep-alive`, the node-shell pod is deleted immediately after running the command, which kills all child processes.

Some simulations modify persistent system state that survives pod deletion. Use `--cleanup` to revert these changes.

| Simulation | Needs `--keep-alive` | Needs `--cleanup` | Notes |
|------------|---------------------|-------------------|-------|
| `zombies` | Yes (30m) | No | Processes die with pod |
| `pid-exhaustion` | Yes (30m) | **Yes** | Lowered pid_max persists |
| `io-delay` | Yes (15m) | No | Worker dies with pod |
| `systemd-restarts` | Yes (10m) | No | Kubelet auto-restarts |
| `fork-oom` | No | No | **Node may be unrecoverable** |
| `kernel-bug` | No | No | Dmesg injection is instant |
| `soft-lockup` | No | No | Dmesg injection is instant |
| `ipamd-down` | No | No | Systemd auto-restarts |
| `interface-down` | No | **Yes** | Interface stays down |
| `neuron-*` | No | No | Dmesg can't be cleaned |

## Examples

### List All Simulations

```bash
hma-cli list
```

### Dry Run

See what a simulation would do without executing:

```bash
hma-cli --node <node-name> kernel zombies --dry-run
```

### Manual Cleanup

Cleanup is only needed for simulations that modify persistent system state:

```bash
# Restore pid_max and threads-max after pid-exhaustion
hma-cli --node <node-name> kernel pid-exhaustion --cleanup --force

# Bring interface back up after interface-down
hma-cli --node <node-name> networking interface-down --cleanup --force
```

For other simulations, the pod exit handles cleanup automatically.

## Verification

After running a simulation, verify NMA detection:

```bash
# Check node conditions
kubectl get node <node-name> -o json | jq '.status.conditions[] | select(.type | test("Kernel|Network|Storage|Runtime|Accelerated"))'

# Check node events (for Warning-level detections)
kubectl get events --field-selector involvedObject.name=<node-name> --sort-by='.lastTimestamp'

# Check NMA logs on the node
NMA_POD=$(kubectl get pods -n kube-system -o wide | grep eks-node-monitoring | grep <node-name> | awk '{print $1}')
kubectl logs -n kube-system $NMA_POD --tail=100
```

## NMA Detection Levels

The NMA has two detection levels:

| Level | Effect | Examples |
|-------|--------|----------|
| **CONDITION** | Changes node condition to `False` | `ipamd-down`, `interface-down`, `neuron-*` |
| **EVENT** | Creates Warning event only (condition stays `True`) | `zombies`, `kernel-bug`, `soft-lockup` |

## Requirements

- Go 1.21+ (for building)
- kubectl access to EKS cluster
- Cluster must have NMA installed
- For GPU simulations: DCGM must be installed on GPU nodes

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
4. Keeps the pod alive if `--keep-alive` is specified
5. Cleans up the pod when done

### NMA Detection Patterns

The NMA monitors:
- **Kernel**: Zombie count (>=20), PID usage (>70%), dmesg patterns (`BUG:`, `soft lockup`)
- **Networking**: IPAMD process, interface state
- **Storage**: Per-process I/O delay from `/proc/[PID]/stat` (>10s)
- **Runtime**: systemd NRestarts counter via dbus (>3 and increasing)
- **Accelerator**: NVIDIA XID errors via DCGM, Neuron errors via dmesg

## License

Apache 2.0
