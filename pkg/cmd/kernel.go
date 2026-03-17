package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/simulator"
	_ "github.com/jicowan/hma-cli/pkg/simulator/kernel" // Register kernel simulators
	"github.com/jicowan/hma-cli/pkg/util"
)

var (
	kernelCount int
)

var kernelCmd = &cobra.Command{
	Use:   "kernel <failure-type>",
	Short: "Simulate kernel health failures",
	Long: `Simulate kernel health failures to test the NMA KernelReady condition.

Available failure types:
  zombies        Create zombie processes (threshold: >= 20)
  pid-exhaustion Create sleeping processes to exhaust PIDs (threshold: > 70%)
  fork-oom       Inject fork_oom message to dmesg
  kernel-bug     Inject 'kernel BUG at' message to dmesg
  soft-lockup    Inject soft lockup message to dmesg`,
	Args: cobra.ExactArgs(1),
	RunE: runKernelSimulator,
}

func init() {
	kernelCmd.Flags().IntVar(&kernelCount, "count", 0, "Number of processes to create (for zombies/pid-exhaustion)")
	rootCmd.AddCommand(kernelCmd)
}

func runKernelSimulator(cmd *cobra.Command, args []string) error {
	simName := args[0]

	sim, ok := simulator.Get(simName)
	if !ok {
		// List available simulators
		available := simulator.ListByCategory(simulator.CategoryKernel)
		fmt.Println("Unknown failure type:", simName)
		fmt.Println("\nAvailable kernel failure types:")
		for _, s := range available {
			fmt.Printf("  %-15s %s\n", s.Name(), s.Description())
		}
		return fmt.Errorf("unknown failure type: %s", simName)
	}

	// Verify it's a kernel simulator
	if sim.Category() != simulator.CategoryKernel {
		return fmt.Errorf("%s is not a kernel simulator", simName)
	}

	opts := simulator.Options{
		Force:  force,
		DryRun: dryRun,
		Count:  kernelCount,
	}

	ctx := context.Background()

	// If --node is specified, run remotely
	if nodeName != "" {
		// Handle dry-run for remote
		if dryRun {
			runRemoteDryRun(nodeName, sim, opts)
			return nil
		}

		// Confirm if not forced
		if !force {
			if !util.ConfirmDangerous(fmt.Sprintf("Remote: %s on %s", sim.Description(), nodeName), sim.DryRun(opts)) {
				fmt.Println("Aborted")
				return nil
			}
		}

		// Parse keep-alive duration
		var keepAliveDur time.Duration
		if keepAlive != "" {
			if d, err := time.ParseDuration(keepAlive); err == nil {
				keepAliveDur = d
			}
		}

		return runRemoteSimulation(ctx, nodeName, kubeconfig, sim, opts, cleanup, keepAliveDur)
	}

	// Handle cleanup (local)
	if cleanup {
		fmt.Printf("Cleaning up %s simulation...\n", simName)
		if err := sim.Cleanup(ctx); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
		fmt.Println("Cleanup complete")
		return nil
	}

	// Handle dry-run (local)
	if dryRun {
		util.PrintDryRun(sim.Description(), []string{sim.DryRun(opts)})
		return nil
	}

	// Confirm if not forced
	if !force {
		if !util.ConfirmDangerous(sim.Description(), sim.DryRun(opts)) {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Run simulation locally
	result, err := sim.Simulate(ctx, opts)
	if err != nil {
		return err
	}

	util.PrintResult(result.Success, result.Message, result.CleanupCmd)

	return nil
}
