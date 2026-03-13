package kernel

import (
	"context"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// ForkOOMSimulator exhausts PIDs to cause kubelet fork failures
// NMA watches kubelet journal for: fork/exec.*resource temporarily unavailable
type ForkOOMSimulator struct{}

// NewForkOOMSimulatorV2 creates a simulator that causes real fork failures
func NewForkOOMSimulatorV2() *ForkOOMSimulator {
	return &ForkOOMSimulator{}
}

func (f *ForkOOMSimulator) Name() string {
	return "fork-oom"
}

func (f *ForkOOMSimulator) Description() string {
	return "Exhaust PIDs to cause kubelet fork failures - WARNING: May require node replacement to recover"
}

func (f *ForkOOMSimulator) Category() simulator.Category {
	return simulator.CategoryKernel
}

func (f *ForkOOMSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	return &simulator.Result{
		Success: true,
		Message: "Fork OOM simulation requires shell execution on node. Use --node flag.",
	}, nil
}

func (f *ForkOOMSimulator) Cleanup(ctx context.Context) error {
	return nil
}

func (f *ForkOOMSimulator) DryRun(opts simulator.Options) string {
	return `Would:
1. Lower pid_max and threads-max to just above current PID count
2. Create sleep processes to fill remaining PID slots
3. This causes kubelet to fail when forking (exec, health probes)
4. Kubelet logs 'fork/exec.*resource temporarily unavailable'
5. NMA detects this pattern in kubelet journal`
}

func (f *ForkOOMSimulator) IsReversible() bool {
	return true
}

func (f *ForkOOMSimulator) ShellCommand(opts simulator.Options) []string {
	// This script:
	// 1. Lowers pid_max/threads-max to just above current PID count
	// 2. Creates sleep processes to fill remaining slots
	// 3. Kubelet will fail to fork when trying exec/health probes
	// 4. NMA detects "fork/exec.*resource temporarily unavailable" in kubelet journal
	script := `#!/bin/sh
# Note: No 'set -e' because we expect fork failures

echo "=== Fork OOM Simulation ==="
echo ""
echo "########################################################"
echo "# WARNING: This simulation may make the node           #"
echo "#          UNRECOVERABLE! The node may need to be      #"
echo "#          DELETED and REPLACED after running this.    #"
echo "#                                                      #"
echo "# Only run this on nodes you can afford to lose.       #"
echo "########################################################"
echo ""
echo "This exhausts PIDs to cause kubelet fork failures"
echo "NMA watches kubelet journal for: fork/exec.*resource temporarily unavailable"
echo ""

# Get current state
CURRENT_PIDS=$(ls -d /proc/[0-9]* 2>/dev/null | wc -l)
ORIG_PID_MAX=$(cat /proc/sys/kernel/pid_max)
ORIG_THREADS_MAX=$(cat /proc/sys/kernel/threads-max)

echo "Current PIDs: $CURRENT_PIDS"
echo "Original pid_max: $ORIG_PID_MAX"
echo "Original threads-max: $ORIG_THREADS_MAX"

# Save original values for cleanup
echo "$ORIG_PID_MAX" > /tmp/orig_pid_max
echo "$ORIG_THREADS_MAX" > /tmp/orig_threads_max

# Set new limit to current + 50 (very tight)
# But ensure we don't go below minimum (301 for pid_max)
NEW_LIMIT=$((CURRENT_PIDS + 50))
MIN_PID_MAX=301
if [ "$NEW_LIMIT" -lt "$MIN_PID_MAX" ]; then
  NEW_LIMIT=$MIN_PID_MAX
fi

echo ""
echo "Setting pid_max and threads-max to: $NEW_LIMIT (spare PIDs: $((NEW_LIMIT - CURRENT_PIDS)))"
echo "$NEW_LIMIT" > /proc/sys/kernel/pid_max
echo "$NEW_LIMIT" > /proc/sys/kernel/threads-max

# Create sleep processes to fill up remaining slots (leave ~5 for system)
TO_CREATE=$((NEW_LIMIT - CURRENT_PIDS - 5))
echo ""
echo "Creating $TO_CREATE sleep processes to exhaust PIDs..."

# Create helper script for spawning
cat > /tmp/spawn_forkoom.sh << 'SPAWNER'
#!/bin/sh
COUNT=$1
CREATED=0
while [ "$CREATED" -lt "$COUNT" ]; do
  nohup sleep 3600 >/dev/null 2>&1 &
  CREATED=$((CREATED + 1))
  if [ $((CREATED % 10)) -eq 0 ]; then
    echo "  Created $CREATED processes..."
  fi
done
echo "  Created $CREATED total processes"
SPAWNER
chmod +x /tmp/spawn_forkoom.sh

# Run the spawner
/tmp/spawn_forkoom.sh "$TO_CREATE" 2>/dev/null || true

# Check final state
sleep 1
FINAL_PIDS=$(ls -d /proc/[0-9]* 2>/dev/null | wc -l)
FINAL_MAX=$(cat /proc/sys/kernel/pid_max)
AVAILABLE=$((FINAL_MAX - FINAL_PIDS))

echo ""
echo "Final state:"
echo "  PIDs in use: $FINAL_PIDS"
echo "  pid_max: $FINAL_MAX"
echo "  Available PIDs: $AVAILABLE"
echo ""

if [ "$AVAILABLE" -lt 10 ]; then
  echo "SUCCESS: PID space nearly exhausted"
  echo ""
  echo "Kubelet will now fail to fork when:"
  echo "  - Running health probes"
  echo "  - Handling kubectl exec requests"
  echo "  - Starting new containers"
  echo ""
  echo "To trigger immediately, run from another terminal:"
  echo "  kubectl exec -it <any-pod-on-this-node> -- echo test"
  echo ""
  echo "Check kubelet journal for fork failures:"
  echo "  journalctl -u kubelet --since '1 minute ago' | grep -i 'fork\\|resource temporarily'"
else
  echo "WARNING: Still $AVAILABLE PIDs available - may need more processes"
fi`

	return []string{script}
}

func (f *ForkOOMSimulator) CleanupCommand() []string {
	return []string{
		`#!/bin/sh
echo "Cleaning up fork-oom simulation..."

# Kill all our sleep processes
pkill -9 -f "sleep 3600" 2>/dev/null || true

# Restore original values
if [ -f /tmp/orig_pid_max ]; then
  ORIG_PID_MAX=$(cat /tmp/orig_pid_max)
  echo "Restoring pid_max to: $ORIG_PID_MAX"
  echo "$ORIG_PID_MAX" > /proc/sys/kernel/pid_max
  rm -f /tmp/orig_pid_max
fi

if [ -f /tmp/orig_threads_max ]; then
  ORIG_THREADS_MAX=$(cat /tmp/orig_threads_max)
  echo "Restoring threads-max to: $ORIG_THREADS_MAX"
  echo "$ORIG_THREADS_MAX" > /proc/sys/kernel/threads-max
  rm -f /tmp/orig_threads_max
fi

rm -f /tmp/spawn_forkoom.sh

# Verify
echo ""
echo "Current state:"
echo "  PIDs: $(ls -d /proc/[0-9]* 2>/dev/null | wc -l)"
echo "  pid_max: $(cat /proc/sys/kernel/pid_max)"
echo "  threads-max: $(cat /proc/sys/kernel/threads-max)"
echo ""
echo "Cleanup complete"`,
	}
}

func init() {
	simulator.Register(NewForkOOMSimulatorV2())
}
