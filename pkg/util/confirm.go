package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for confirmation
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// ConfirmDangerous prompts the user with a warning for dangerous operations
func ConfirmDangerous(operation string, details string) bool {
	fmt.Println()
	fmt.Println("WARNING: This operation may impact node stability!")
	fmt.Println()
	fmt.Printf("Operation: %s\n", operation)
	if details != "" {
		fmt.Printf("Details: %s\n", details)
	}
	fmt.Println()
	return Confirm("Are you sure you want to proceed?")
}

// PrintDryRun prints what would happen in a dry run
func PrintDryRun(operation string, actions []string) {
	fmt.Println()
	fmt.Printf("DRY RUN: %s\n", operation)
	fmt.Println()
	fmt.Println("The following actions would be performed:")
	for i, action := range actions {
		fmt.Printf("  %d. %s\n", i+1, action)
	}
	fmt.Println()
}

// PrintResult prints the result of a simulation
func PrintResult(success bool, message string, cleanupCmd string) {
	fmt.Println()
	if success {
		fmt.Println("SUCCESS:", message)
	} else {
		fmt.Println("FAILED:", message)
	}
	if cleanupCmd != "" {
		fmt.Println()
		fmt.Println("To cleanup, run:")
		fmt.Printf("  %s\n", cleanupCmd)
	}
	fmt.Println()
}

// IsRoot checks if the current process is running as root
func IsRoot() bool {
	return os.Geteuid() == 0
}

// RequireRoot returns an error if not running as root
func RequireRoot() error {
	if !IsRoot() {
		return fmt.Errorf("this operation requires root privileges (try running with sudo)")
	}
	return nil
}
