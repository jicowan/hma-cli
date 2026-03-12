package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/simulator"
	_ "github.com/jicowan/hma-cli/pkg/simulator/accelerator" // Register accelerator simulators
	"github.com/jicowan/hma-cli/pkg/util"
)

var (
	xidCode int
)

var acceleratorCmd = &cobra.Command{
	Use:   "accelerator <failure-type>",
	Short: "Simulate accelerator (GPU/Neuron) health failures",
	Long: `Simulate accelerator health failures to test the NMA AcceleratedHardwareReady condition.

Available failure types:
  xid-error           Inject NVIDIA XID error (use --code for specific XID)
  neuron-sram-error   Inject Neuron SRAM uncorrectable error
  neuron-nc-error     Inject Neuron NC uncorrectable error
  neuron-hbm-error    Inject Neuron HBM uncorrectable error
  neuron-dma-error    Inject Neuron DMA error

Fatal XID codes: 13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140`,
	Args: cobra.ExactArgs(1),
	RunE: runAcceleratorSimulator,
}

func init() {
	acceleratorCmd.Flags().IntVar(&xidCode, "code", 79, "XID error code for xid-error simulation")
	rootCmd.AddCommand(acceleratorCmd)
}

func runAcceleratorSimulator(cmd *cobra.Command, args []string) error {
	simName := args[0]

	sim, ok := simulator.Get(simName)
	if !ok {
		available := simulator.ListByCategory(simulator.CategoryAccelerator)
		fmt.Println("Unknown failure type:", simName)
		fmt.Println("\nAvailable accelerator failure types:")
		for _, s := range available {
			fmt.Printf("  %-20s %s\n", s.Name(), s.Description())
		}
		return fmt.Errorf("unknown failure type: %s", simName)
	}

	if sim.Category() != simulator.CategoryAccelerator {
		return fmt.Errorf("%s is not an accelerator simulator", simName)
	}

	opts := simulator.Options{
		Force:  force,
		DryRun: dryRun,
		Code:   xidCode,
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
