package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jicowan/hma-cli/pkg/nodeshell"
	"github.com/jicowan/hma-cli/pkg/simulator"
)

// runRemoteSimulation creates a node-shell pod and executes the simulation remotely
func runRemoteSimulation(ctx context.Context, nodeName, kubeconfig string, sim simulator.Simulator, opts simulator.Options, isCleanup bool, keepAliveDuration time.Duration) error {
	action := "simulation"
	if isCleanup {
		action = "cleanup"
	}

	fmt.Println("========================================")
	fmt.Printf("Remote %s: %s\n", action, sim.Name())
	fmt.Printf("Target node: %s\n", nodeName)
	fmt.Println("========================================")

	// Create NodeShell
	fmt.Print("\n[1/4] Creating node-shell pod... ")
	shell, err := nodeshell.NewNodeShell(kubeconfig, nodeName)
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("failed to create node shell: %w", err)
	}

	// Create pod
	if err := shell.CreatePod(ctx); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("failed to create node-shell pod: %w", err)
	}
	fmt.Println("OK")
	fmt.Printf("       Pod: %s/%s\n", shell.GetNamespace(), shell.GetPodName())

	// Ensure cleanup
	defer func() {
		fmt.Print("\n[4/4] Deleting node-shell pod... ")
		if cleanupErr := shell.Cleanup(ctx); cleanupErr != nil {
			fmt.Printf("FAILED (%v)\n", cleanupErr)
		} else {
			fmt.Println("OK")
		}
	}()

	// Wait for ready
	fmt.Print("\n[2/4] Waiting for pod to be ready... ")
	if err := shell.WaitForReady(ctx, 60*time.Second); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("node-shell pod failed to become ready: %w", err)
	}
	fmt.Println("OK")

	// Create Executor
	exec, err := nodeshell.NewExecutor(kubeconfig, shell.GetNamespace(), shell.GetPodName(), "shell")
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Get commands to execute
	var commands []string
	if isCleanup {
		commands = sim.CleanupCommand()
	} else {
		commands = sim.ShellCommand(opts)
	}

	if len(commands) == 0 {
		return fmt.Errorf("no commands to execute for %s", sim.Name())
	}

	// Execute commands
	fmt.Printf("\n[3/4] Executing %d command(s) on node...\n", len(commands))
	fmt.Println("----------------------------------------")
	for i, cmd := range commands {
		fmt.Printf("\n> Command %d/%d:\n", i+1, len(commands))
		// Print abbreviated command (first line or first 80 chars)
		cmdPreview := cmd
		if len(cmdPreview) > 80 {
			cmdPreview = cmdPreview[:77] + "..."
		}
		fmt.Printf("  $ %s\n", cmdPreview)

		result, err := exec.Exec(ctx, []string{"sh", "-c", cmd})
		if err != nil {
			fmt.Printf("  Status: FAILED (exec error: %v)\n", err)
			return fmt.Errorf("failed to execute command: %w", err)
		}

		if result.ExitCode != 0 {
			fmt.Printf("  Status: FAILED (exit code %d)\n", result.ExitCode)
			if result.Stdout != "" {
				fmt.Printf("  Output: %s\n", result.Stdout)
			}
			if result.Stderr != "" {
				fmt.Printf("  Stderr: %s\n", result.Stderr)
			}
			return fmt.Errorf("command failed with exit code %d", result.ExitCode)
		}

		fmt.Printf("  Status: OK\n")
		if result.Stdout != "" {
			fmt.Printf("  Output:\n")
			// Indent output
			for _, line := range splitLines(result.Stdout) {
				if line != "" {
					fmt.Printf("    %s\n", line)
				}
			}
		}
	}
	fmt.Println("----------------------------------------")

	// If keep-alive is specified, keep the pod running so the simulation persists
	if keepAliveDuration > 0 && !isCleanup {
		fmt.Printf("\nKeeping simulation alive for %s...\n", keepAliveDuration)
		fmt.Println("Press Ctrl+C to stop early and cleanup.")
		fmt.Printf("\nTo check node condition, run:\n")
		fmt.Printf("  kubectl get node %s -o jsonpath='{.status.conditions}' | jq\n\n", nodeName)

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		deadline := time.Now().Add(keepAliveDuration)

		for {
			select {
			case <-ctx.Done():
				fmt.Println("\nInterrupted - cleaning up...")
				return nil
			case <-ticker.C:
				remaining := time.Until(deadline)
				if remaining <= 0 {
					fmt.Printf("\nKeep-alive duration completed.\n")
					goto keepAliveDone
				}
				fmt.Printf("  [%s remaining]\n", remaining.Round(time.Second))
			}
		}
	keepAliveDone:
	}

	// Summary
	fmt.Println("\n========================================")
	if isCleanup {
		fmt.Printf("SUCCESS: Cleanup of '%s' completed on %s\n", sim.Name(), nodeName)
	} else {
		fmt.Printf("SUCCESS: Simulation '%s' executed on %s\n", sim.Name(), nodeName)
		fmt.Printf("\nDescription: %s\n", sim.Description())
		if sim.IsReversible() {
			fmt.Printf("\nTo cleanup, run:\n")
			fmt.Printf("  hma-cli --node %s %s %s --cleanup\n", nodeName, sim.Category(), sim.Name())
		}
	}
	fmt.Println("========================================")

	return nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// runRemoteDryRun shows what remote commands would be executed
func runRemoteDryRun(nodeName string, sim simulator.Simulator, opts simulator.Options) {
	fmt.Printf("=== DRY RUN (remote on %s) ===\n", nodeName)
	fmt.Printf("Would create a privileged node-shell pod on node %s\n\n", nodeName)
	fmt.Println("Commands that would be executed:")
	commands := sim.ShellCommand(opts)
	for i, cmd := range commands {
		fmt.Printf("  %d. %s\n", i+1, cmd)
	}
	fmt.Println("\nThen the node-shell pod would be deleted.")
}
