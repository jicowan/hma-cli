package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/jicowan/hma-cli/pkg/simulator"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available failure simulations",
	Long:  `List all available failure simulations organized by category.`,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	categories := []simulator.Category{
		simulator.CategoryKernel,
		simulator.CategoryNetworking,
		simulator.CategoryStorage,
		simulator.CategoryRuntime,
		simulator.CategoryAccelerator,
	}

	fmt.Println("Available failure simulations:")
	fmt.Println()

	for _, category := range categories {
		sims := simulator.ListByCategory(category)
		if len(sims) == 0 {
			continue
		}

		// Sort by name
		sort.Slice(sims, func(i, j int) bool {
			return sims[i].Name() < sims[j].Name()
		})

		fmt.Printf("  %s:\n", category)
		for _, sim := range sims {
			reversible := ""
			if sim.IsReversible() {
				reversible = " [reversible]"
			}
			fmt.Printf("    %-22s %s%s\n", sim.Name(), sim.Description(), reversible)
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  hma-cli <category> <failure-type> [flags]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hma-cli kernel zombies --count 25")
	fmt.Println("  hma-cli networking ipamd-down")
	fmt.Println("  hma-cli accelerator xid-error --code 79")
	fmt.Println("  hma-cli diagnose --node my-node --destination https://...")
}
