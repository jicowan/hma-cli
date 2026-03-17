package storage

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// IOSimulator simulates I/O delays using stress-ng
type IOSimulator struct {
	stressCmd *exec.Cmd
}

// NewIOSimulator creates a new I/O delay simulator
func NewIOSimulator() *IOSimulator {
	return &IOSimulator{}
}

func (i *IOSimulator) Name() string {
	return "io-delay"
}

func (i *IOSimulator) Description() string {
	return "Create process with >10s I/O delay to trigger StorageReady event (EVENT-level, checks /proc/PID/stat)"
}

func (i *IOSimulator) Category() simulator.Category {
	return simulator.CategoryStorage
}

func (i *IOSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	// Check if stress-ng is available
	if _, err := exec.LookPath("stress-ng"); err != nil {
		return &simulator.Result{
			Success: false,
			Message: "stress-ng not found. Install it with: yum install stress-ng (or apt install stress-ng)",
		}, nil
	}

	// Start stress-ng in background
	// This creates heavy I/O load that can cause I/O delays > 10 seconds
	i.stressCmd = exec.CommandContext(ctx, "stress-ng",
		"--iomix", "4",       // 4 I/O stress workers
		"--iomix-bytes", "1G", // Work with 1GB data
		"--timeout", "60s",   // Run for 60 seconds
	)

	if err := i.stressCmd.Start(); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to start stress-ng: %v", err),
		}, err
	}

	return &simulator.Result{
		Success:    true,
		Message:    "Started I/O stress test. NMA should detect StorageReady=False if I/O delays exceed 10 seconds.",
		CleanupCmd: "hma-cli storage io-delay --cleanup",
	}, nil
}

func (i *IOSimulator) Cleanup(ctx context.Context) error {
	if i.stressCmd != nil && i.stressCmd.Process != nil {
		return i.stressCmd.Process.Kill()
	}
	// Also kill any remaining stress-ng processes
	exec.Command("pkill", "-9", "stress-ng").Run()
	return nil
}

func (i *IOSimulator) DryRun(opts simulator.Options) string {
	return "Would run 'stress-ng --iomix 4 --iomix-bytes 1G --timeout 60s' to create I/O pressure"
}

func (i *IOSimulator) IsReversible() bool {
	return true
}

func (i *IOSimulator) ShellCommand(opts simulator.Options) []string {
	// NMA reads /proc/PID/stat column 42 (delayacct_blkio_ticks) which measures
	// time spent waiting for block I/O in centiseconds. It compares the delta
	// between checks (every 10 min) and triggers if delta >= 10 seconds.
	//
	// Runs in FOREGROUND - requires --keep-alive (recommend 15m+)
	return []string{
		`echo "=== I/O Delay Simulation ==="
echo "NMA checks /proc/PID/stat column 42 every 10 minutes"
echo "Threshold: >10 seconds of I/O wait time between checks"
echo "Runs in FOREGROUND - use --keep-alive 15m or longer"
echo ""

TESTDIR=/tmp/io-delay-test
mkdir -p "$TESTDIR"

MYPID=$$
echo "Worker PID: $MYPID"
echo "Starting continuous synchronous I/O writes..."
echo ""

COUNTER=0
while true; do
  # Write 64MB with fdatasync to force synchronous I/O
  dd if=/dev/zero of="$TESTDIR/iotest_$COUNTER" bs=1M count=64 conv=fdatasync 2>/dev/null
  sync
  rm -f "$TESTDIR/iotest_$COUNTER"

  COUNTER=$((COUNTER + 1))
  if [ $((COUNTER % 10)) -eq 0 ]; then
    if [ -f "/proc/$MYPID/stat" ]; then
      DELAY=$(awk '{print $42}' "/proc/$MYPID/stat")
      DELAY_SEC=$((DELAY / 100))
      echo "  [$(date '+%H:%M:%S')] I/O delay: ${DELAY_SEC}s (${DELAY} centiseconds) - iteration $COUNTER"
    fi
  fi
done`,
	}
}

func (i *IOSimulator) CleanupCommand() []string {
	return []string{
		`#!/bin/sh
# Kill I/O delay worker
if [ -f /tmp/io-delay-test.pid ]; then
  PID=$(cat /tmp/io-delay-test.pid)
  kill -9 "$PID" 2>/dev/null && echo "Killed I/O worker (PID: $PID)" || echo "Worker already stopped"
  rm -f /tmp/io-delay-test.pid
fi

# Also kill by script name
pkill -9 -f "io_delay_worker" 2>/dev/null && echo "Killed io_delay_worker processes" || true
pkill -9 stress-ng 2>/dev/null && echo "Killed stress-ng processes" || true
pkill -9 -f "dd if=/dev/zero" 2>/dev/null && echo "Killed dd processes" || true

# Cleanup test files
rm -rf /tmp/io-delay-test 2>/dev/null || true
rm -rf /tmp/io-stress-test 2>/dev/null || true
rm -f /tmp/io_delay_worker.sh 2>/dev/null || true
rm -f /tmp/io-delay-test.log 2>/dev/null || true

echo "I/O delay cleanup complete"`,
	}
}

func init() {
	simulator.Register(NewIOSimulator())
}
