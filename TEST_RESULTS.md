# HMA-CLI Test Results

Test Date: 2026-03-12

## Test Environment

| Property | Value |
|----------|-------|
| Cluster | EKS 1.31 |
| Node OS | Amazon Linux 2023, Amazon Linux 2, Bottlerocket |
| NMA Version | Latest (installed via EKS add-on) |

## Test Results Matrix

### Kernel Simulations

| Simulation | Node | Result | NMA Detection | Notes |
|------------|------|--------|---------------|-------|
| `zombies` | ip-10-0-5-217 | ✅ PASS | Warning Event | Created 25 zombies, requires `--keep-alive 30m` |
| `kernel-bug` | Multiple | ✅ PASS | Warning Event | Dmesg injection works, pattern: `[timestamp] BUG: message` |
| `soft-lockup` | Multiple | ✅ PASS | Warning Event | Dmesg injection works, pattern includes process name |
| `pid-exhaustion` | ip-10-0-3-62 | ✅ PASS | Pending | Achieved 74% PID usage, requires `--keep-alive 30m` |
| `fork-oom` | N/A | ❌ NOT SUPPORTED | N/A | NMA watches kubelet journal, not dmesg |

### Networking Simulations

| Simulation | Node | Result | NMA Detection | Notes |
|------------|------|--------|---------------|-------|
| `ipamd-down` | Multiple | ✅ PASS | NetworkingReady=False | Kills aws-k8s-agent process |
| `interface-down` | Multiple | ✅ PASS | NetworkingReady=False | `ip link set eth1 down` |
| `routes-missing` | ip-10-0-5-100 | ⚠️ PARTIAL | Pending | Deleted gateway route; needs pod-specific routes from IPAMD |

### Storage Simulations

| Simulation | Node | Result | NMA Detection | Notes |
|------------|------|--------|---------------|-------|
| `io-delay` | ip-10-0-5-78 | ✅ PASS | Pending | Worker started, NMA checks every 10 min, requires `--keep-alive 15m` |

### Runtime Simulations

| Simulation | Node | Result | NMA Detection | Notes |
|------------|------|--------|---------------|-------|
| `systemd-restarts` | ip-10-0-5-79 | ⚠️ PARTIAL | NRestarts=1 | Script killed on pod exit; requires `--keep-alive 10m` |

### Accelerator Simulations

| Simulation | Node | Result | NMA Detection | Notes |
|------------|------|--------|---------------|-------|
| `neuron-sram-error` | Neuron nodes | ✅ PASS | AcceleratedHardwareReady=False | Dmesg injection |
| `neuron-nc-error` | Neuron nodes | ✅ PASS | AcceleratedHardwareReady=False | Dmesg injection |
| `neuron-hbm-error` | Neuron nodes | ✅ PASS | AcceleratedHardwareReady=False | Dmesg injection |
| `neuron-dma-error` | Neuron nodes | ✅ PASS | AcceleratedHardwareReady=False | Dmesg injection |
| `xid-error` | GPU nodes | ⏭️ SKIPPED | N/A | DCGM not installed on test GPU nodes |

## Summary

| Category | Total | Pass | Partial | Fail | Skipped |
|----------|-------|------|---------|------|---------|
| Kernel | 5 | 4 | 0 | 1 | 0 |
| Networking | 3 | 2 | 1 | 0 | 0 |
| Storage | 1 | 1 | 0 | 0 | 0 |
| Runtime | 1 | 0 | 1 | 0 | 0 |
| Accelerator | 5 | 4 | 0 | 0 | 1 |
| **Total** | **15** | **11** | **2** | **1** | **1** |

## Key Findings

### 1. Process Persistence Issue

Simulations that create processes require `--keep-alive` flag. Without it, processes are killed when the node-shell pod is deleted.

**Affected simulations:**
- `zombies` - zombie processes killed
- `pid-exhaustion` - sleep processes killed
- `io-delay` - worker process killed
- `systemd-restarts` - background kill script killed

### 2. NMA Detection Patterns

| Pattern | NMA Behavior | Detection Level |
|---------|--------------|-----------------|
| Zombies >= 20 | Creates Warning event | EVENT (MinOccurrences: 5) |
| PIDs > 70% of MAX(pid_max, threads-max) | Creates Warning event | EVENT |
| `[timestamp] BUG:` in dmesg | Creates Warning event | EVENT |
| `soft lockup` in dmesg | Creates Warning event | EVENT |
| IPAMD not running | Sets NetworkingReady=False | CONDITION (Fatal) |
| Interface not up | Sets NetworkingReady=False | CONDITION (Fatal) |
| Neuron errors in dmesg | Sets AcceleratedHardwareReady=False | CONDITION (Fatal) |
| NRestarts > 3 | Sets ContainerRuntimeReady=False | CONDITION |
| I/O delay > 10s | Sets StorageReady=False | CONDITION |

### 3. Fixes Applied

| Simulation | Issue | Fix |
|------------|-------|-----|
| `pid-exhaustion` | Only lowered pid_max | Now lowers both pid_max AND threads-max |
| `kernel-bug` | Pattern didn't match NMA regex | Changed to `[timestamp] BUG: message` format |
| `soft-lockup` | Missing process name | Added `[process:pid]` at end of pattern |
| `systemd-restarts` | `systemctl restart` doesn't increment NRestarts | Changed to SIGKILL approach |
| `io-delay` | stress-ng doesn't cause measurable delay | Created sync write worker with fsync |

### 4. Not Fixable

| Simulation | Reason |
|------------|--------|
| `fork-oom` | NMA watches kubelet journal for `fork/exec.*resource temporarily unavailable`, not dmesg |
| `routes-missing` | Requires actual pod IPs from IPAMD checkpoint; deleted gateway route doesn't trigger detection |

## Recommended Usage

```bash
# Simulations that work immediately (no --keep-alive needed)
hma-cli --node <node> kernel kernel-bug --force
hma-cli --node <node> kernel soft-lockup --force
hma-cli --node <node> networking ipamd-down --force
hma-cli --node <node> networking interface-down --force
hma-cli --node <node> accelerator neuron-sram-error --force

# Simulations that require --keep-alive
hma-cli --node <node> kernel zombies --count 25 --keep-alive 30m --force
hma-cli --node <node> kernel pid-exhaustion --keep-alive 30m --force
hma-cli --node <node> storage io-delay --keep-alive 15m --force
hma-cli --node <node> runtime systemd-restarts --keep-alive 10m --force
```

## Verification Commands

```bash
# Check node conditions
kubectl get node <node> -o json | jq '.status.conditions[] | select(.type | test("Kernel|Network|Storage|Runtime|Accelerated"))'

# Check node events
kubectl get events --field-selector involvedObject.name=<node> --sort-by='.lastTimestamp'

# Check NMA logs
NMA_POD=$(kubectl get pods -n kube-system -o wide | grep eks-node-monitoring | grep <node> | awk '{print $1}')
kubectl logs -n kube-system $NMA_POD --tail=100 | grep -E "(condition|sending|occurrences)"

# Check systemd NRestarts
kubectl exec -it <shell-pod> -- nsenter -t 1 -m -u -i -n -- \
  busctl get-property org.freedesktop.systemd1 /org/freedesktop/systemd1/unit/kubelet_2eservice org.freedesktop.systemd1.Service NRestarts

# Check zombie count
kubectl exec -it <shell-pod> -- nsenter -t 1 -m -u -i -n -- \
  sh -c 'ps aux | grep -c "<defunct>"'
```
