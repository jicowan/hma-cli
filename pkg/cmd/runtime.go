package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/simulator"
	_ "github.com/jicowan/hma-cli/pkg/simulator/runtime" // Register runtime simulators
	"github.com/jicowan/hma-cli/pkg/util"
)

var (
	runtimeTarget string
	runtimeCount  int
)

var runtimeCmd = &cobra.Command{
	Use:   "runtime <failure-type>",
	Short: "Simulate container runtime health failures",
	Long: `Simulate container runtime health failures to test the NMA ContainerRuntimeReady condition.

Available failure types:
  systemd-restarts    Restart kubelet/containerd repeatedly (threshold: >3 restarts in 5 min)`,
	Args: cobra.ExactArgs(1),
	RunE: runRuntimeSimulator,
}

func init() {
	runtimeCmd.Flags().StringVar(&runtimeTarget, "service", "kubelet", "Service to restart (kubelet or containerd)")
	runtimeCmd.Flags().IntVar(&runtimeCount, "count", 4, "Number of restarts")
	rootCmd.AddCommand(runtimeCmd)
}

func runRuntimeSimulator(cmd *cobra.Command, args []string) error {
	simName := args[0]

	sim, ok := simulator.Get(simName)
	if !ok {
		available := simulator.ListByCategory(simulator.CategoryRuntime)
		fmt.Println("Unknown failure type:", simName)
		fmt.Println("\nAvailable runtime failure types:")
		for _, s := range available {
			fmt.Printf("  %-18s %s\n", s.Name(), s.Description())
		}
		return fmt.Errorf("unknown failure type: %s", simName)
	}

	if sim.Category() != simulator.CategoryRuntime {
		return fmt.Errorf("%s is not a runtime simulator", simName)
	}

	opts := simulator.Options{
		Force:  force,
		DryRun: dryRun,
		Target: runtimeTarget,
		Count:  runtimeCount,
	}

	if duration != "" {
		d, err := time.ParseDuration(duration)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		opts.Duration = d
	}

	ctx := context.Background()

	// If --node is specified, run remotely
	if nodeName != "" {
		if dryRun {
			runRemoteDryRun(nodeName, sim, opts)
			return nil
		}

		if !force {
			if !util.ConfirmDangerous(fmt.Sprintf("Remote: %s on %s", sim.Description(), nodeName), sim.DryRun(opts)) {
				fmt.Println("Aborted")
				return nil
			}
		}

		var keepAliveDur time.Duration
		if keepAlive != "" {
			if d, err := time.ParseDuration(keepAlive); err == nil {
				keepAliveDur = d
			}
		}

		return runRemoteSimulation(ctx, nodeName, kubeconfig, sim, opts, cleanup, keepAliveDur)
	}

	// Local execution
	if cleanup {
		fmt.Printf("Cleaning up %s simulation...\n", simName)
		if err := sim.Cleanup(ctx); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
		fmt.Println("Cleanup complete")
		return nil
	}

	if dryRun {
		util.PrintDryRun(sim.Description(), []string{sim.DryRun(opts)})
		return nil
	}

	if !force {
		if !util.ConfirmDangerous(sim.Description(), sim.DryRun(opts)) {
			fmt.Println("Aborted")
			return nil
		}
	}

	result, err := sim.Simulate(ctx, opts)
	if err != nil {
		return err
	}

	util.PrintResult(result.Success, result.Message, result.CleanupCmd)
	return nil
}
