package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/simulator"
	_ "github.com/jicowan/hma-cli/pkg/simulator/networking" // Register networking simulators
	"github.com/jicowan/hma-cli/pkg/util"
)

var (
	networkTarget string
)

var networkingCmd = &cobra.Command{
	Use:   "networking <failure-type>",
	Short: "Simulate networking health failures",
	Long: `Simulate networking health failures to test the NMA NetworkingReady condition.

Available failure types:
  ipamd-down      Kill the IPAMD (aws-k8s-agent) process
  routes-missing  Delete VPC routes
  interface-down  Bring down secondary ENI (eth1)`,
	Args: cobra.ExactArgs(1),
	RunE: runNetworkingSimulator,
}

func init() {
	networkingCmd.Flags().StringVar(&networkTarget, "target", "", "Target interface or route (e.g., eth1 or 10.0.0.0/16)")
	rootCmd.AddCommand(networkingCmd)
}

func runNetworkingSimulator(cmd *cobra.Command, args []string) error {
	simName := args[0]

	sim, ok := simulator.Get(simName)
	if !ok {
		available := simulator.ListByCategory(simulator.CategoryNetworking)
		fmt.Println("Unknown failure type:", simName)
		fmt.Println("\nAvailable networking failure types:")
		for _, s := range available {
			fmt.Printf("  %-15s %s\n", s.Name(), s.Description())
		}
		return fmt.Errorf("unknown failure type: %s", simName)
	}

	if sim.Category() != simulator.CategoryNetworking {
		return fmt.Errorf("%s is not a networking simulator", simName)
	}

	opts := simulator.Options{
		Force:  force,
		DryRun: dryRun,
		Target: networkTarget,
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
