package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	nodeName   string
	kubeconfig string
	dryRun     bool
	force      bool
	cleanup    bool
	keepAlive  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "hma-cli --node <node-name> <command> [flags]",
	Short: "EKS Node Health Monitoring Agent CLI",
	Long: `hma-cli is a CLI tool to simulate various failure conditions on EKS worker nodes
for testing the EKS Node Health Monitoring Agent (NMA).

IMPORTANT: The --node flag is REQUIRED for all simulations. The CLI creates a
privileged pod on the target node to execute commands.

For process-based simulations (zombies, pid-exhaustion, io-delay, systemd-restarts, ipamd-down),
use --keep-alive to prevent processes from being killed when the pod exits.

Available categories:
  - kernel:      PID exhaustion, zombies, kernel bugs, soft lockups
  - networking:  IPAMD down, interface down
  - storage:     I/O delays
  - runtime:     systemd restarts
  - accelerator: NVIDIA XID errors, Neuron errors

The tool can also create NodeDiagnostic CRs to collect node logs (diagnose command).`,
	Example: `  # Simulate zombie processes (requires --keep-alive)
  hma-cli --node ip-10-0-1-123.ec2.internal kernel zombies --keep-alive 30m --force

  # Inject kernel bug message to dmesg (instant, no --keep-alive needed)
  hma-cli --node ip-10-0-1-123.ec2.internal kernel kernel-bug --force

  # Bring down secondary ENI (auto-detects eth1/ens6)
  hma-cli --node ip-10-0-1-123.ec2.internal networking interface-down --force

  # Kill IPAMD repeatedly (NMA requires 5 restarts)
  hma-cli --node ip-10-0-1-123.ec2.internal networking ipamd-down --keep-alive 10m --force

  # Dry run to see what would happen
  hma-cli --node ip-10-0-1-123.ec2.internal kernel zombies --dry-run

  # Cleanup (needed for: pid-exhaustion, interface-down)
  hma-cli --node ip-10-0-1-123.ec2.internal networking interface-down --cleanup --force

  # Collect node logs (auto-generates presigned S3 URL)
  hma-cli diagnose --node ip-10-0-1-123.ec2.internal --bucket my-logs-bucket --wait

  # List all available simulations
  hma-cli list`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&nodeName, "node", "", "[REQUIRED] Target node name (creates privileged pod automatically)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: ~/.kube/config)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&cleanup, "cleanup", false, "Revert simulation (needed for: pid-exhaustion, interface-down)")
	rootCmd.PersistentFlags().StringVar(&keepAlive, "keep-alive", "", "Keep pod alive for duration (REQUIRED for: zombies, pid-exhaustion, io-delay, systemd-restarts, ipamd-down)")
}
