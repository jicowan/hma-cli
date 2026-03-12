package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/diagnose"
)

var (
	destination   string
	wait          bool
	deleteAfter   bool
	statusOnly    bool
	waitTimeout   time.Duration
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Create NodeDiagnostic CR to collect node logs",
	Long: `Create a NodeDiagnostic custom resource to trigger log collection from an EKS node.

The NodeDiagnostic CR will instruct the node monitoring agent to collect logs and upload
them to the specified destination URL (typically a pre-signed S3 URL).

Examples:
  # Create NodeDiagnostic to collect logs
  hma-cli diagnose --node ip-10-0-1-123.ec2.internal --destination "https://mybucket.s3.amazonaws.com/logs.tar.gz?..."

  # Create and wait for completion
  hma-cli diagnose --node my-node --destination "https://..." --wait

  # Check status of existing NodeDiagnostic
  hma-cli diagnose --node my-node --status

  # Create, wait, then delete the CR
  hma-cli diagnose --node my-node --destination "https://..." --wait --delete`,
	RunE: runDiagnose,
}

func init() {
	diagnoseCmd.Flags().StringVar(&destination, "destination", "", "Pre-signed S3 URL for log upload (required unless --status)")
	diagnoseCmd.Flags().BoolVar(&wait, "wait", false, "Wait for log collection to complete")
	diagnoseCmd.Flags().BoolVar(&deleteAfter, "delete", false, "Delete the NodeDiagnostic CR after completion")
	diagnoseCmd.Flags().BoolVar(&statusOnly, "status", false, "Check status of existing NodeDiagnostic")
	diagnoseCmd.Flags().DurationVar(&waitTimeout, "timeout", 5*time.Minute, "Timeout when waiting for completion")

	rootCmd.AddCommand(diagnoseCmd)
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	if nodeName == "" {
		return fmt.Errorf("--node is required")
	}

	client, err := diagnose.NewNodeDiagnosticClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx := context.Background()

	// Status check only
	if statusOnly {
		return checkStatus(ctx, client)
	}

	// Create requires destination
	if destination == "" {
		return fmt.Errorf("--destination is required (pre-signed S3 URL)")
	}

	// Check if already exists
	exists, err := client.Exists(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("failed to check existing NodeDiagnostic: %w", err)
	}

	if exists {
		if !force {
			fmt.Printf("NodeDiagnostic for node %s already exists.\n", nodeName)
			fmt.Println("Use --force to delete and recreate, or --status to check status.")
			return nil
		}
		// Delete existing
		fmt.Printf("Deleting existing NodeDiagnostic for node %s...\n", nodeName)
		if err := client.Delete(ctx, nodeName); err != nil {
			return fmt.Errorf("failed to delete existing NodeDiagnostic: %w", err)
		}
		// Wait a moment for deletion to propagate
		time.Sleep(2 * time.Second)
	}

	// Dry run
	if dryRun {
		fmt.Println()
		fmt.Println("DRY RUN: Would create NodeDiagnostic CR")
		fmt.Println()
		fmt.Printf("  Node: %s\n", nodeName)
		fmt.Printf("  Destination: %s\n", destination)
		fmt.Println()
		fmt.Println("YAML that would be applied:")
		fmt.Printf(`
apiVersion: eks.amazonaws.com/v1alpha1
kind: NodeDiagnostic
metadata:
  name: %s
spec:
  logCapture:
    destination: %s
`, nodeName, destination)
		return nil
	}

	// Create NodeDiagnostic
	fmt.Printf("Creating NodeDiagnostic for node %s...\n", nodeName)
	if err := client.Create(ctx, nodeName, destination); err != nil {
		return err
	}
	fmt.Println("NodeDiagnostic created successfully")

	// Wait for completion if requested
	if wait {
		fmt.Printf("Waiting for log collection to complete (timeout: %s)...\n", waitTimeout)
		status, err := client.WaitForCompletion(ctx, nodeName, waitTimeout)
		if err != nil {
			return err
		}
		fmt.Printf("Status: %s\n", status.Phase)
		if status.Message != "" {
			fmt.Printf("Message: %s\n", status.Message)
		}
	}

	// Delete after completion if requested
	if deleteAfter {
		fmt.Printf("Deleting NodeDiagnostic for node %s...\n", nodeName)
		if err := client.Delete(ctx, nodeName); err != nil {
			return fmt.Errorf("failed to delete NodeDiagnostic: %w", err)
		}
		fmt.Println("NodeDiagnostic deleted")
	} else {
		fmt.Println()
		fmt.Println("To check status:")
		fmt.Printf("  hma-cli diagnose --node %s --status\n", nodeName)
		fmt.Println()
		fmt.Println("To delete:")
		fmt.Printf("  kubectl delete nodediagnostics.eks.amazonaws.com %s\n", nodeName)
	}

	return nil
}

func checkStatus(ctx context.Context, client *diagnose.NodeDiagnosticClient) error {
	exists, err := client.Exists(ctx, nodeName)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("no NodeDiagnostic found for node %s", nodeName)
	}

	status, err := client.GetStatus(ctx, nodeName)
	if err != nil {
		return err
	}

	fmt.Printf("NodeDiagnostic for node: %s\n", nodeName)
	fmt.Printf("  Status: %s\n", status.Phase)
	if status.Message != "" {
		fmt.Printf("  Message: %s\n", status.Message)
	}

	return nil
}
