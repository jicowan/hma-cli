package cmd

import (
	"context"
	"testing"

	"github.com/jicowan/hma-cli/pkg/simulator"
)

// mockSimulator implements simulator.Simulator for testing
type mockSimulator struct {
	name         string
	category     simulator.Category
	shellCmds    []string
	cleanupCmds  []string
	reversible   bool
}

func (m *mockSimulator) Name() string                                              { return m.name }
func (m *mockSimulator) Description() string                                       { return "Mock simulator" }
func (m *mockSimulator) Category() simulator.Category                              { return m.category }
func (m *mockSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) { return nil, nil }
func (m *mockSimulator) Cleanup(ctx context.Context) error                         { return nil }
func (m *mockSimulator) DryRun(opts simulator.Options) string                      { return "dry run" }
func (m *mockSimulator) IsReversible() bool                                        { return m.reversible }
func (m *mockSimulator) ShellCommand(opts simulator.Options) []string              { return m.shellCmds }
func (m *mockSimulator) CleanupCommand() []string                                  { return m.cleanupCmds }

func TestRunRemoteDryRun(t *testing.T) {
	sim := &mockSimulator{
		name:     "test-sim",
		category: simulator.CategoryKernel,
		shellCmds: []string{
			"echo 'test command 1'",
			"echo 'test command 2'",
		},
	}

	opts := simulator.Options{Count: 10}

	// This should not panic and should print output
	runRemoteDryRun("test-node", sim, opts)
}

func TestShellCommandInterface(t *testing.T) {
	sim := &mockSimulator{
		name:     "test-sim",
		category: simulator.CategoryKernel,
		shellCmds: []string{
			"echo hello",
		},
		cleanupCmds: []string{
			"echo cleanup",
		},
		reversible: true,
	}

	opts := simulator.Options{}

	cmds := sim.ShellCommand(opts)
	if len(cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(cmds))
	}
	if cmds[0] != "echo hello" {
		t.Errorf("Expected 'echo hello', got %s", cmds[0])
	}

	cleanupCmds := sim.CleanupCommand()
	if len(cleanupCmds) != 1 {
		t.Errorf("Expected 1 cleanup command, got %d", len(cleanupCmds))
	}
	if cleanupCmds[0] != "echo cleanup" {
		t.Errorf("Expected 'echo cleanup', got %s", cleanupCmds[0])
	}
}
