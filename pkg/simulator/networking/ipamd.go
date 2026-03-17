package networking

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// IPAMDSimulator simulates IPAMD process failures
type IPAMDSimulator struct{}

// NewIPAMDSimulator creates a new IPAMD simulator
func NewIPAMDSimulator() *IPAMDSimulator {
	return &IPAMDSimulator{}
}

func (i *IPAMDSimulator) Name() string {
	return "ipamd-down"
}

func (i *IPAMDSimulator) Description() string {
	return "Kill IPAMD (aws-k8s-agent) repeatedly to trigger NetworkingReady condition"
}

func (i *IPAMDSimulator) Category() simulator.Category {
	return simulator.CategoryNetworking
}

func (i *IPAMDSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	// Find IPAMD process
	cmd := exec.CommandContext(ctx, "pkill", "-9", "-f", "aws-k8s-agent")
	if err := cmd.Run(); err != nil {
		// Check if process exists
		checkCmd := exec.CommandContext(ctx, "pgrep", "-f", "aws-k8s-agent")
		if checkCmd.Run() != nil {
			return &simulator.Result{
				Success: false,
				Message: "IPAMD (aws-k8s-agent) process not found. Is this an EKS node with VPC CNI?",
			}, nil
		}
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to kill IPAMD: %v", err),
		}, err
	}

	return &simulator.Result{
		Success: true,
		Message: "IPAMD (aws-k8s-agent) process killed. NMA should detect NetworkingReady=False. Note: The process will be restarted by systemd.",
	}, nil
}

func (i *IPAMDSimulator) Cleanup(ctx context.Context) error {
	// IPAMD is managed by systemd and will restart automatically
	// If needed, we can explicitly restart it
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "aws-k8s-agent")
	return cmd.Run()
}

func (i *IPAMDSimulator) DryRun(opts simulator.Options) string {
	return "Would kill the IPAMD (aws-k8s-agent) process repeatedly every 60s to trigger NMA detection"
}

func (i *IPAMDSimulator) IsReversible() bool {
	return true // systemd will restart it
}

func (i *IPAMDSimulator) ShellCommand(opts simulator.Options) []string {
	// Use [a]ws pattern to avoid matching the pgrep/pkill command itself
	// Kill repeatedly to trigger NMA's "IPAMDRepeatedlyRestart" detection (needs 5 occurrences)
	// Runs in FOREGROUND - requires --keep-alive to keep pod running
	return []string{
		`echo "Starting IPAMD killer loop (runs in foreground, use --keep-alive)"
echo "NMA requires 5 restarts to trigger condition change"
echo ""

count=0
while [ $count -lt 10 ]; do
  pid=$(pgrep -f '[a]ws-k8s-agent' 2>/dev/null | head -1)
  if [ -n "$pid" ]; then
    kill -9 $pid 2>/dev/null
    count=$((count + 1))
    echo "$(date): Killed IPAMD process $pid (kill #$count)"
  else
    echo "$(date): IPAMD not running, waiting..."
  fi

  if [ $count -lt 10 ]; then
    echo "  Sleeping 60s before next kill..."
    sleep 60
  fi
done

echo ""
echo "Completed $count IPAMD kills. NMA should have detected IPAMDRepeatedlyRestart."`,
	}
}

func (i *IPAMDSimulator) CleanupCommand() []string {
	return []string{
		// No cleanup needed - IPAMD auto-restarts via systemd
		// Just confirm it's running
		`pid=$(pgrep -f '[a]ws-k8s-agent' 2>/dev/null | head -1)
if [ -n "$pid" ]; then
  echo "IPAMD is running (PID: $pid)"
else
  echo "IPAMD not running - waiting for systemd to restart it..."
  sleep 5
  pid=$(pgrep -f '[a]ws-k8s-agent' 2>/dev/null | head -1)
  if [ -n "$pid" ]; then
    echo "IPAMD restarted (PID: $pid)"
  else
    echo "WARNING: IPAMD still not running"
  fi
fi`,
	}
}

func init() {
	simulator.Register(NewIPAMDSimulator())
}
