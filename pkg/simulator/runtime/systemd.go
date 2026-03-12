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
	// NMA threshold: currentRestarts > 3 && currentRestarts > previousRestarts
	script := fmt.Sprintf(`#!/bin/sh
echo "=== SystemD NRestarts Simulation ==="
echo "NMA checks dbus NRestarts property every 5 minutes"
echo "Threshold: NRestarts > 3 AND increasing"
echo ""
echo "IMPORTANT: 'systemctl restart' does NOT increment NRestarts!"
echo "We must KILL the process to trigger systemd's auto-restart."
echo ""

SERVICE="%s"
COUNT=%d
LOG=/tmp/systemd-restarts.log

# Check current NRestarts
get_nrestarts() {
  busctl get-property org.freedesktop.systemd1 \
    "/org/freedesktop/systemd1/unit/${SERVICE}_2eservice" \
    org.freedesktop.systemd1.Service NRestarts 2>/dev/null | awk '{print $2}'
}

INITIAL_RESTARTS=$(get_nrestarts)
echo "Current NRestarts for $SERVICE: ${INITIAL_RESTARTS:-0}"
echo ""

# Create kill script that runs in background
cat > /tmp/do-kills.sh << 'KILLSCRIPT'
#!/bin/sh
SERVICE=$1
COUNT=$2
LOG=$3
echo "$(date): Starting $COUNT process kills for $SERVICE" >> "$LOG"
i=1
while [ "$i" -le "$COUNT" ]; do
  # Get the main PID of the service
  MAIN_PID=$(systemctl show -p MainPID "$SERVICE" --value 2>/dev/null)
  if [ -z "$MAIN_PID" ] || [ "$MAIN_PID" = "0" ]; then
    echo "$(date): Waiting for $SERVICE to start..." >> "$LOG"
    sleep 5
    continue
  fi

  echo "$(date): Killing $SERVICE (PID: $MAIN_PID) - attempt $i/$COUNT" >> "$LOG"

  # Kill the process - this causes systemd to auto-restart and increment NRestarts
  kill -9 "$MAIN_PID" 2>/dev/null

  # Wait for service to restart
  sleep 15

  # Check NRestarts
  NRESTARTS=$(busctl get-property org.freedesktop.systemd1 \
    "/org/freedesktop/systemd1/unit/${SERVICE}_2eservice" \
    org.freedesktop.systemd1.Service NRestarts 2>/dev/null | awk '{print $2}')
  echo "$(date): NRestarts now: ${NRESTARTS:-unknown}" >> "$LOG"

  i=$((i + 1))
done
echo "$(date): Completed $COUNT kills" >> "$LOG"

# Final status
FINAL_RESTARTS=$(busctl get-property org.freedesktop.systemd1 \
  "/org/freedesktop/systemd1/unit/${SERVICE}_2eservice" \
  org.freedesktop.systemd1.Service NRestarts 2>/dev/null | awk '{print $2}')
echo "$(date): Final NRestarts: ${FINAL_RESTARTS:-unknown}" >> "$LOG"
if [ "${FINAL_RESTARTS:-0}" -gt 3 ]; then
  echo "$(date): SUCCESS - NRestarts > 3, NMA should detect RepeatedRestart" >> "$LOG"
else
  echo "$(date): WARNING - NRestarts (${FINAL_RESTARTS:-0}) <= 3, may need more kills" >> "$LOG"
fi
KILLSCRIPT
chmod +x /tmp/do-kills.sh

# Run in background
echo "Starting kill process in background..."
rm -f "$LOG"
setsid /tmp/do-kills.sh "$SERVICE" "$COUNT" "$LOG" </dev/null >/dev/null 2>&1 &
BGPID=$!
echo "Kill script started (PID: $BGPID)"
echo "Monitor: tail -f $LOG"
echo ""
sleep 3
cat "$LOG" 2>/dev/null || echo "Waiting for first kill..."`, service, restartCount)
	return []string{script}
}

func (s *SystemdSimulator) CleanupCommand() []string {
	// Services stabilize on their own
	return []string{"echo 'Services will stabilize on their own'"}
}

func init() {
	simulator.Register(NewSystemdSimulator())
}
