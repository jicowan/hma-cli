package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// SystemdSimulator simulates systemd service restart issues
type SystemdSimulator struct{}

// NewSystemdSimulator creates a new systemd restart simulator
func NewSystemdSimulator() *SystemdSimulator {
	return &SystemdSimulator{}
}

func (s *SystemdSimulator) Name() string {
	return "systemd-restarts"
}

func (s *SystemdSimulator) Description() string {
	return "Kill kubelet process to trigger NRestarts increment (threshold: >3, NMA checks dbus NRestarts property)"
}

func (s *SystemdSimulator) Category() simulator.Category {
	return simulator.CategoryRuntime
}

func (s *SystemdSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	service := "kubelet"
	if opts.Target != "" {
		service = opts.Target
	}

	// NMA triggers at > 3 restarts in 5 minutes
	restartCount := 4
	if opts.Count > 0 {
		restartCount = opts.Count
	}

	fmt.Printf("Restarting %s %d times...\n", service, restartCount)

	for i := 0; i < restartCount; i++ {
		select {
		case <-ctx.Done():
			return &simulator.Result{
				Success: false,
				Message: "Interrupted",
			}, ctx.Err()
		default:
		}

		cmd := exec.CommandContext(ctx, "systemctl", "restart", service)
		if err := cmd.Run(); err != nil {
			return &simulator.Result{
				Success: false,
				Message: fmt.Sprintf("Failed to restart %s on attempt %d: %v", service, i+1, err),
			}, err
		}
		fmt.Printf("  Restart %d/%d complete\n", i+1, restartCount)

		// Small delay between restarts
		if i < restartCount-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return &simulator.Result{
		Success: true,
		Message: fmt.Sprintf("Restarted %s %d times. NMA should detect ContainerRuntimeReady=False", service, restartCount),
	}, nil
}

func (s *SystemdSimulator) Cleanup(ctx context.Context) error {
	// Services will stabilize on their own
	return nil
}

func (s *SystemdSimulator) DryRun(opts simulator.Options) string {
	service := "kubelet"
	if opts.Target != "" {
		service = opts.Target
	}
	restartCount := 4
	if opts.Count > 0 {
		restartCount = opts.Count
	}
	return fmt.Sprintf("Would restart %s %d times using 'systemctl restart %s'", service, restartCount, service)
}

func (s *SystemdSimulator) IsReversible() bool {
	return true // Services stabilize on their own
}

func (s *SystemdSimulator) ShellCommand(opts simulator.Options) []string {
	service := "kubelet"
	if opts.Target != "" {
		service = opts.Target
	}
	restartCount := 4
	if opts.Count > 0 {
		restartCount = opts.Count
	}
	// NMA checks the dbus NRestarts property, which ONLY increments when systemd
	// auto-restarts a service after it FAILS. Using 'systemctl restart' does NOT
	// increment NRestarts - we must KILL the process to cause a failure.
	//
	// Runs in FOREGROUND - requires --keep-alive
	script := fmt.Sprintf(`echo "=== SystemD NRestarts Simulation ==="
echo "NMA checks dbus NRestarts property every 5 minutes"
echo "Threshold: NRestarts > 3 AND increasing"
echo "Runs in FOREGROUND - use --keep-alive 10m or longer"
echo ""
echo "IMPORTANT: 'systemctl restart' does NOT increment NRestarts!"
echo "We must KILL the process to trigger systemd's auto-restart."
echo ""

SERVICE="%s"
COUNT=%d

# Check current NRestarts
get_nrestarts() {
  busctl get-property org.freedesktop.systemd1 \
    "/org/freedesktop/systemd1/unit/${SERVICE}_2eservice" \
    org.freedesktop.systemd1.Service NRestarts 2>/dev/null | awk '{print $2}'
}

INITIAL_RESTARTS=$(get_nrestarts)
echo "Current NRestarts for $SERVICE: ${INITIAL_RESTARTS:-0}"
echo ""

i=1
while [ "$i" -le "$COUNT" ]; do
  # Get the main PID of the service
  MAIN_PID=$(systemctl show -p MainPID "$SERVICE" --value 2>/dev/null)
  if [ -z "$MAIN_PID" ] || [ "$MAIN_PID" = "0" ]; then
    echo "  Waiting for $SERVICE to start..."
    sleep 5
    continue
  fi

  echo "  [$(date '+%%H:%%M:%%S')] Killing $SERVICE (PID: $MAIN_PID) - attempt $i/$COUNT"
  kill -9 "$MAIN_PID" 2>/dev/null

  # Wait for service to restart
  sleep 15

  NRESTARTS=$(get_nrestarts)
  echo "  NRestarts now: ${NRESTARTS:-unknown}"

  i=$((i + 1))
done

echo ""
FINAL_RESTARTS=$(get_nrestarts)
echo "Final NRestarts: ${FINAL_RESTARTS:-unknown}"
if [ "${FINAL_RESTARTS:-0}" -gt 3 ]; then
  echo "SUCCESS: NRestarts > 3, NMA should detect RepeatedRestart"
else
  echo "WARNING: NRestarts (${FINAL_RESTARTS:-0}) <= 3, may need more kills"
fi

echo ""
echo "Keeping pod alive for NMA to detect..."
echo "Press Ctrl+C or wait for --keep-alive to expire."

while true; do
  sleep 60
  CURRENT=$(get_nrestarts)
  echo "  [$(date '+%%H:%%M:%%S')] NRestarts: ${CURRENT:-unknown}"
done`, service, restartCount)
	return []string{script}
}

func (s *SystemdSimulator) CleanupCommand() []string {
	// Services stabilize on their own
	return []string{"echo 'Services will stabilize on their own'"}
}

func init() {
	simulator.Register(NewSystemdSimulator())
}
