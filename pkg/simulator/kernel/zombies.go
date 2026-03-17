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
	// Create zombie processes and keep running in FOREGROUND
	// Requires --keep-alive to maintain the zombies
	// Uses shell script that forks children which exit immediately
	script := fmt.Sprintf(`echo "=== Zombie Process Simulation ==="
echo "NMA threshold: >= 20 zombies"
echo "Runs in FOREGROUND - use --keep-alive to maintain zombies"
echo ""

count_zombies() {
  ps -eo stat 2>/dev/null | grep -c "^Z" || echo 0
}

before=$(count_zombies)
echo "Zombies before: $before"
echo "Creating %d zombie processes..."

# Create a child script that forks zombies and then sleeps forever
# The parent shell will wait on this, keeping the pod alive
cat > /tmp/zombie_parent.sh << 'ZSCRIPT'
#!/bin/sh
# Fork the requested number of children that exit immediately
# Since parent doesn't wait(), they become zombies
COUNT=$1
i=0
while [ $i -lt $COUNT ]; do
  # Fork a child that exits immediately
  ( exit 0 ) &
  i=$((i + 1))
done

# Report status
sleep 2
AFTER=$(ps -eo stat 2>/dev/null | grep -c "^Z" || echo 0)
echo "Zombies after: $AFTER"
if [ "$AFTER" -ge 20 ]; then
  echo "SUCCESS: Zombie count exceeds NMA threshold (>= 20)"
else
  echo "NOTE: Zombie count below threshold"
fi
echo ""
echo "Keeping parent alive to maintain zombies..."
echo "Press Ctrl+C or wait for --keep-alive to expire."

# Loop forever, reporting status
while true; do
  sleep 60
  CURRENT=$(ps -eo stat 2>/dev/null | grep -c "^Z" || echo 0)
  echo "  [$(date '+%%H:%%M:%%S')] Zombie count: $CURRENT"
done
ZSCRIPT
chmod +x /tmp/zombie_parent.sh

# Run the zombie parent script (blocks until killed)
exec /tmp/zombie_parent.sh %d`, count, count)
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
