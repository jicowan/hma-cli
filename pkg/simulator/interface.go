package simulator

import (
	"context"
)

// Category represents the health check category
type Category string

const (
	CategoryKernel      Category = "kernel"
	CategoryNetworking  Category = "networking"
	CategoryStorage     Category = "storage"
	CategoryRuntime     Category = "runtime"
	CategoryAccelerator Category = "accelerator"
)

// Options contains options for running a simulation
type Options struct {
	// Force skips confirmation prompts
	Force bool

	// DryRun shows what would happen without executing
	DryRun bool

	// Count specifies the number of items to create (e.g., zombie processes)
	Count int

	// Code specifies an error code (e.g., XID error code)
	Code int

	// Target specifies a target (e.g., interface name, service name)
	Target string
}

// Result contains the result of a simulation
type Result struct {
	// Success indicates if the simulation succeeded
	Success bool

	// Message contains a human-readable result message
	Message string

	// CleanupCmd contains the command to run for manual cleanup
	CleanupCmd string
}

// Simulator defines the interface for failure simulators
type Simulator interface {
	// Name returns the unique identifier for this simulator
	Name() string

	// Description returns a human-readable description
	Description() string

	// Category returns the health category (kernel, networking, etc.)
	Category() Category

	// Simulate executes the failure simulation
	Simulate(ctx context.Context, opts Options) (*Result, error)

	// Cleanup reverts the simulation if possible
	Cleanup(ctx context.Context) error

	// DryRun shows what would happen without executing
	DryRun(opts Options) string

	// IsReversible indicates if cleanup is possible
	IsReversible() bool

	// ShellCommand returns shell commands to execute remotely on the target node
	ShellCommand(opts Options) []string

	// CleanupCommand returns shell commands to cleanup remotely on the target node
	CleanupCommand() []string
}
