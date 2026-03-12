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
	return []string{
		`pid=$(pgrep -f '[a]ws-k8s-agent' 2>/dev/null | head -1)
if [ -n "$pid" ]; then
  kill -9 $pid 2>/dev/null
  echo "IPAMD (aws-k8s-agent) process $pid killed initially."

  # Start background loop to keep killing IPAMD every 60s
  # NMA needs 5 restarts to trigger condition change
  nohup sh -c '
    count=0
    while true; do
      sleep 60
      pid=$(pgrep -f "[a]ws-k8s-agent" 2>/dev/null | head -1)
      if [ -n "$pid" ]; then
        kill -9 $pid 2>/dev/null
        count=$((count + 1))
        echo "$(date): Killed IPAMD (restart #$count)" >> /tmp/ipamd-kills.log
      fi
    done
  ' > /dev/null 2>&1 &
  echo "Started background loop to kill IPAMD every 60s (PID: $!)"
  echo "Kill log: /tmp/ipamd-kills.log"
else
  echo "WARNING: IPAMD (aws-k8s-agent) process not found. Is VPC CNI running on this node?"
fi`,
	}
}

func (i *IPAMDSimulator) CleanupCommand() []string {
	return []string{
		// Kill the background loop first, then restart IPAMD
		"pkill -f 'while true.*aws-k8s-agent' 2>/dev/null || true",
		"rm -f /tmp/ipamd-kills.log 2>/dev/null || true",
		"systemctl restart aws-k8s-agent || echo 'Could not restart aws-k8s-agent service'",
	}
}

func init() {
	simulator.Register(NewIPAMDSimulator())
}
