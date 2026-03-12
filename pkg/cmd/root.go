package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	nodeName   string
	kubeconfig string
	dryRun     bool
	duration   string
	force      bool
	cleanup    bool
	keepAlive  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "hma-cli",
	Short: "EKS Node Health Monitoring Agent CLI",
	Long: `hma-cli is a CLI tool to simulate various failure conditions on EKS worker nodes
for testing the EKS Node Health Monitoring Agent (NMA).

It can simulate failures across 5 categories:
  - kernel: PID exhaustion, zombies, kernel bugs
  - networking: IPAMD down, missing routes, interface issues
  - storage: I/O delays, EBS throttling
  - runtime: systemd restarts, stuck pods
  - accelerator: NVIDIA XID errors, Neuron errors

The tool can also create NodeDiagnostic CRs to collect node logs.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&nodeName, "node", "", "Target node name (creates privileged pod automatically)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: ~/.kube/config)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")
	rootCmd.PersistentFlags().StringVar(&duration, "duration", "", "Auto-cleanup after duration (e.g., 5m)")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&cleanup, "cleanup", false, "Revert a previous simulation")
	rootCmd.PersistentFlags().StringVar(&keepAlive, "keep-alive", "", "Keep simulation running for duration (e.g., 30m for NMA to detect zombies)")
}
