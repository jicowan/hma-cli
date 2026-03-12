package kernel

import (
	"context"
	"fmt"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

const (
	// DefaultZombieCount is the default number of zombies to create
	// NMA triggers at >= 20 zombies
	DefaultZombieCount = 25
)

// ZombieSimulator creates zombie processes to trigger KernelReady condition
type ZombieSimulator struct {
	pm *util.ProcessManager
}

// NewZombieSimulator creates a new zombie simulator
func NewZombieSimulator() *ZombieSimulator {
	return &ZombieSimulator{
		pm: util.NewProcessManager(),
	}
}

func (z *ZombieSimulator) Name() string {
	return "zombies"
}

func (z *ZombieSimulator) Description() string {
	return "Create zombie processes to trigger KernelReady=False (threshold: >= 20 zombies)"
}

func (z *ZombieSimulator) Category() simulator.Category {
	return simulator.CategoryKernel
}

func (z *ZombieSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	count := opts.Count
	if count == 0 {
		count = DefaultZombieCount
	}

	// Check current zombie count
	currentZombies, err := util.CountZombies()
	if err != nil {
		return nil, fmt.Errorf("failed to count current zombies: %w", err)
	}

	pids, err := z.pm.CreateZombies(count)
	if err != nil {
		return &simulator.Result{
			Success:    false,
			Message:    fmt.Sprintf("Failed to create zombies: %v", err),
			CleanupCmd: "hma-cli kernel zombies --cleanup",
		}, err
	}

	return &simulator.Result{
		Success: true,
		Message: fmt.Sprintf("Created %d zombie processes (total now: ~%d). NMA should detect KernelReady=False",
			len(pids), currentZombies+len(pids)),
		CleanupCmd: "hma-cli kernel zombies --cleanup",
	}, nil
}

func (z *ZombieSimulator) Cleanup(ctx context.Context) error {
	return z.pm.KillAll()
}

func (z *ZombieSimulator) DryRun(opts simulator.Options) string {
	count := opts.Count
	if count == 0 {
		count = DefaultZombieCount
	}
	return fmt.Sprintf("Would create %d zombie processes by forking children without calling wait()", count)
}

func (z *ZombieSimulator) IsReversible() bool {
	return true
}

func (z *ZombieSimulator) ShellCommand(opts simulator.Options) []string {
	count := opts.Count
	if count == 0 {
		count = DefaultZombieCount
	}
	// Create zombie processes using perl (most reliable and commonly available)
	// Use setsid to create a new session so the process survives pod termination
	script := fmt.Sprintf(`before=$(ps -eo stat 2>/dev/null | grep "^Z" | wc -l)
echo "Zombies before: $before"

if command -v perl >/dev/null 2>&1; then
  echo "Creating %d zombie processes using perl..."
  # Use setsid to detach from the pod's process group (if available)
  if command -v setsid >/dev/null 2>&1; then
    setsid perl -e 'for(1..%d){fork or exit 0}sleep 86400' </dev/null >/dev/null 2>&1 &
  else
    nohup perl -e 'for(1..%d){fork or exit 0}sleep 86400' </dev/null >/dev/null 2>&1 &
  fi
  sleep 3
elif command -v python3 >/dev/null 2>&1; then
  echo "Creating %d zombie processes using python3..."
  python3 -c "
import os, time
for i in range(%d):
    if os.fork() == 0:
        os._exit(0)
time.sleep(86400)
" &
  sleep 3
else
  echo "ERROR: Neither perl nor python3 found. Cannot create zombies reliably."
  exit 1
fi

after=$(ps -eo stat 2>/dev/null | grep "^Z" | wc -l)
created=$((after - before))

echo "Zombies after: $after (created: $created)"
if [ "$after" -ge 20 ]; then
  echo "SUCCESS: Zombie count exceeds NMA threshold (>= 20)"
else
  echo "NOTE: Zombie count is below NMA threshold (>= 20)"
fi`, count, count, count, count, count)
	return []string{script}
}

func (z *ZombieSimulator) CleanupCommand() []string {
	return []string{
		`#!/bin/sh
echo "Zombies before: $(ps -eo stat 2>/dev/null | grep "^Z" | wc -l)"
echo "Killing zombie parent processes..."
# Kill perl-based zombie parents
pkill -9 -f "[p]erl.*fork.*sleep" 2>/dev/null || true
# Kill python-based zombie parents
pkill -9 -f "[p]ython.*os.fork" 2>/dev/null || true
sleep 1
echo "Zombies after: $(ps -eo stat 2>/dev/null | grep "^Z" | wc -l)"
echo "Cleanup complete"`,
	}
}

func init() {
	simulator.Register(NewZombieSimulator())
}
