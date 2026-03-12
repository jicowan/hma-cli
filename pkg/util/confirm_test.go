package util

import (
	"os"
	"testing"
)

func TestIsRoot(t *testing.T) {
	// This just tests that the function runs without panicking
	// The actual result depends on the environment
	isRoot := IsRoot()
	// Verify it returns a boolean (this is mostly a compile-time check)
	if isRoot {
		t.Log("Running as root")
	} else {
		t.Log("Not running as root")
	}
}

func TestRequireRoot(t *testing.T) {
	err := RequireRoot()
	if os.Geteuid() == 0 {
		if err != nil {
			t.Error("RequireRoot should not error when running as root")
		}
	} else {
		if err == nil {
			t.Error("RequireRoot should error when not running as root")
		}
	}
}

func TestPrintDryRun(t *testing.T) {
	// Test that PrintDryRun doesn't panic
	// Capture stdout would be ideal but this is a simple smoke test
	PrintDryRun("Test operation", []string{"Action 1", "Action 2"})
}

func TestPrintResult(t *testing.T) {
	// Test that PrintResult doesn't panic
	PrintResult(true, "Test success", "cleanup-cmd")
	PrintResult(false, "Test failure", "")
}
