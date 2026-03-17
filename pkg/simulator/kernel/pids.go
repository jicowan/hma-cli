package kernel

import (
	"context"
	"fmt"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// PIDExhaustionSimulator creates processes to exhaust PIDs
type PIDExhaustionSimulator struct {
	pm *util.ProcessManager
}

// NewPIDExhaustionSimulator creates a new PID exhaustion simulator
func NewPIDExhaustionSimulator() *PIDExhaustionSimulator {
	return &PIDExhaustionSimulator{
		pm: util.NewProcessManager(),
	}
}

func (p *PIDExhaustionSimulator) Name() string {
	return "pid-exhaustion"
}

func (p *PIDExhaustionSimulator) Description() string {
	return "Create sleeping processes to exhaust PIDs (threshold: > 70% of pid_max)"
}

func (p *PIDExhaustionSimulator) Category() simulator.Category {
	return simulator.CategoryKernel
}

func (p *PIDExhaustionSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	// Get current PID stats
	pidMax, err := util.GetPIDMax()
	if err != nil {
		return nil, fmt.Errorf("failed to get pid_max: %w", err)
	}

	currentPIDs, err := util.CountCurrentPIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to count current PIDs: %w", err)
	}

	// Calculate how many processes to create to reach 75% threshold
	// NMA triggers at > 70%
	targetPIDs := int(float64(pidMax) * 0.75)
	toCreate := targetPIDs - currentPIDs

	if toCreate <= 0 {
		return &simulator.Result{
			Success: true,
			Message: fmt.Sprintf("PID usage already at %d/%d (%.1f%%). No processes created.",
				currentPIDs, pidMax, float64(currentPIDs)/float64(pidMax)*100),
		}, nil
	}

	// Limit to a reasonable number if count is specified
	if opts.Count > 0 && opts.Count < toCreate {
		toCreate = opts.Count
	}

	// Safety limit
	if toCreate > 50000 {
		toCreate = 50000
	}

	pids, err := p.pm.CreateSleepingProcesses(toCreate)
	if err != nil {
		return &simulator.Result{
			Success:    false,
			Message:    fmt.Sprintf("Created %d processes before error: %v", len(pids), err),
			CleanupCmd: "hma-cli kernel pid-exhaustion --cleanup",
		}, err
	}

	newTotal := currentPIDs + len(pids)
	return &simulator.Result{
		Success: true,
		Message: fmt.Sprintf("Created %d sleeping processes. PID usage now: %d/%d (%.1f%%). NMA should detect KernelReady=False",
			len(pids), newTotal, pidMax, float64(newTotal)/float64(pidMax)*100),
		CleanupCmd: "hma-cli kernel pid-exhaustion --cleanup",
	}, nil
}

func (p *PIDExhaustionSimulator) Cleanup(ctx context.Context) error {
	return p.pm.KillAll()
}

func (p *PIDExhaustionSimulator) DryRun(opts simulator.Options) string {
	pidMax, _ := util.GetPIDMax()
	currentPIDs, _ := util.CountCurrentPIDs()
	targetPIDs := int(float64(pidMax) * 0.75)
	toCreate := targetPIDs - currentPIDs

	if toCreate <= 0 {
		return fmt.Sprintf("PID usage already at %d/%d (%.1f%%). No processes would be created.",
			currentPIDs, pidMax, float64(currentPIDs)/float64(pidMax)*100)
	}

	return fmt.Sprintf("Would create ~%d sleeping processes to reach 75%% PID usage (%d/%d)",
		toCreate, targetPIDs, pidMax)
}

func (p *PIDExhaustionSimulator) IsReversible() bool {
	return true
}

func (p *PIDExhaustionSimulator) ShellCommand(opts simulator.Options) []string {
	// NMA uses MAX(kernel.pid_max, kernel.threads-max) as the denominator
	// We need to lower BOTH to make the 70% threshold achievable
	// Runs in FOREGROUND - keeps pod alive to maintain processes
	script := `echo "=== PID Exhaustion Simulation ==="
echo "NMA threshold: > 70% of MAX(pid_max, threads-max)"
echo "Runs in FOREGROUND - use --keep-alive to maintain processes"
echo ""

ORIG_PID_MAX=$(cat /proc/sys/kernel/pid_max)
ORIG_THREADS_MAX=$(cat /proc/sys/kernel/threads-max)
CURRENT_PIDS=$(ls -d /proc/[0-9]* 2>/dev/null | wc -l)

# NMA uses MAX(pid_max, threads-max)
if [ "$ORIG_PID_MAX" -gt "$ORIG_THREADS_MAX" ]; then
  EFFECTIVE_MAX=$ORIG_PID_MAX
else
  EFFECTIVE_MAX=$ORIG_THREADS_MAX
fi

echo "Original pid_max: $ORIG_PID_MAX, threads-max: $ORIG_THREADS_MAX"
echo "NMA effective max: $EFFECTIVE_MAX, Current PIDs: $CURRENT_PIDS"

# Calculate a new limit that puts us close to but below 70%
NEW_LIMIT=$(( CURRENT_PIDS * 100 / 60 ))

# Ensure new limit is reasonable (at least current + 1000)
MIN_LIMIT=$(( CURRENT_PIDS + 1000 ))
if [ "$NEW_LIMIT" -lt "$MIN_LIMIT" ]; then
  NEW_LIMIT=$MIN_LIMIT
fi

echo "Setting both pid_max and threads-max to $NEW_LIMIT temporarily..."
echo "$NEW_LIMIT" > /proc/sys/kernel/pid_max
echo "$NEW_LIMIT" > /proc/sys/kernel/threads-max

# Save original values for cleanup
echo "$ORIG_PID_MAX" > /tmp/orig_pid_max
echo "$ORIG_THREADS_MAX" > /tmp/orig_threads_max

# Calculate how many to create to reach 75%
TARGET=$(( NEW_LIMIT * 75 / 100 ))
TO_CREATE=$(( TARGET - CURRENT_PIDS ))

if [ "$TO_CREATE" -le 0 ]; then
  echo "Already at target. Creating 100 processes anyway."
  TO_CREATE=100
fi

echo "Creating $TO_CREATE sleeping processes..."
CREATED=0
# Store child PIDs so we can wait on them
PIDS=""
while [ "$CREATED" -lt "$TO_CREATE" ]; do
  sleep 86400 &
  PIDS="$PIDS $!"
  CREATED=$((CREATED + 1))
  if [ $((CREATED % 500)) -eq 0 ]; then
    echo "  Created $CREATED processes..."
  fi
done
echo "  Created $CREATED total processes"

sleep 2
NEW_COUNT=$(ls -d /proc/[0-9]* 2>/dev/null | wc -l)
NEW_PID_MAX_NOW=$(cat /proc/sys/kernel/pid_max)
NEW_THREADS_MAX_NOW=$(cat /proc/sys/kernel/threads-max)
if [ "$NEW_PID_MAX_NOW" -gt "$NEW_THREADS_MAX_NOW" ]; then
  NMA_EFFECTIVE=$NEW_PID_MAX_NOW
else
  NMA_EFFECTIVE=$NEW_THREADS_MAX_NOW
fi
USAGE_PCT=$(( NEW_COUNT * 100 / NMA_EFFECTIVE ))

echo ""
echo "PID usage now: $NEW_COUNT/$NMA_EFFECTIVE (${USAGE_PCT}%)"
echo "(pid_max: $NEW_PID_MAX_NOW, threads-max: $NEW_THREADS_MAX_NOW)"
if [ "$USAGE_PCT" -ge 70 ]; then
  echo "SUCCESS: PID usage exceeds NMA threshold (>= 70%)"
else
  echo "WARNING: PID usage ${USAGE_PCT}% is below 70% threshold"
fi

echo ""
echo "Keeping processes alive..."
echo "Press Ctrl+C or wait for --keep-alive to expire."
echo "IMPORTANT: Run --cleanup to restore pid_max/threads-max"

# Keep running and report status periodically
while true; do
  sleep 60
  CURRENT=$(ls -d /proc/[0-9]* 2>/dev/null | wc -l)
  PCT=$(( CURRENT * 100 / NMA_EFFECTIVE ))
  echo "  [$(date '+%H:%M:%S')] PIDs: $CURRENT/$NMA_EFFECTIVE (${PCT}%)"
done`
	return []string{script}
}

func (p *PIDExhaustionSimulator) CleanupCommand() []string {
	return []string{
		`#!/bin/sh
echo "PIDs before cleanup: $(ls -d /proc/[0-9]* 2>/dev/null | wc -l)"

# Kill sleeping processes created by simulation
pkill -9 -f "sleep 86400" 2>/dev/null || true

# Restore original pid_max if saved
if [ -f /tmp/orig_pid_max ]; then
  ORIG=$(cat /tmp/orig_pid_max)
  echo "Restoring pid_max to $ORIG"
  echo $ORIG > /proc/sys/kernel/pid_max
  rm -f /tmp/orig_pid_max
fi

# Restore original threads-max if saved
if [ -f /tmp/orig_threads_max ]; then
  ORIG=$(cat /tmp/orig_threads_max)
  echo "Restoring threads-max to $ORIG"
  echo $ORIG > /proc/sys/kernel/threads-max
  rm -f /tmp/orig_threads_max
fi

sleep 2
echo "PIDs after cleanup: $(ls -d /proc/[0-9]* 2>/dev/null | wc -l)"
echo "Current pid_max: $(cat /proc/sys/kernel/pid_max)"
echo "Current threads-max: $(cat /proc/sys/kernel/threads-max)"
echo "Cleanup complete"`,
	}
}

func init() {
	simulator.Register(NewPIDExhaustionSimulator())
}
