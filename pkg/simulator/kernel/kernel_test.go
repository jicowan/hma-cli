package kernel

import (
	"testing"

	"github.com/jicowan/hma-cli/pkg/simulator"
)

func TestZombieSimulator_Interface(t *testing.T) {
	sim := NewZombieSimulator()

	// Verify interface implementation
	var _ simulator.Simulator = sim

	if sim.Name() != "zombies" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "zombies")
	}

	if sim.Category() != simulator.CategoryKernel {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryKernel)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if !sim.IsReversible() {
		t.Error("IsReversible() should return true")
	}
}

func TestZombieSimulator_DryRun(t *testing.T) {
	sim := NewZombieSimulator()

	// Default count
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}

	// Custom count
	result = sim.DryRun(simulator.Options{Count: 50})
	if result == "" {
		t.Error("DryRun with count should return non-empty string")
	}
}

func TestDmesgSimulators_Interface(t *testing.T) {
	tests := []struct {
		name     string
		sim      *DmesgSimulator
		wantName string
	}{
		{
			name:     "ForkOOM",
			sim:      NewForkOOMSimulator(),
			wantName: "fork-oom",
		},
		{
			name:     "KernelBug",
			sim:      NewKernelBugSimulator(),
			wantName: "kernel-bug",
		},
		{
			name:     "SoftLockup",
			sim:      NewSoftLockupSimulator(),
			wantName: "soft-lockup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ simulator.Simulator = tt.sim

			if tt.sim.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", tt.sim.Name(), tt.wantName)
			}

			if tt.sim.Category() != simulator.CategoryKernel {
				t.Errorf("Category() = %q, want %q", tt.sim.Category(), simulator.CategoryKernel)
			}

			if tt.sim.Description() == "" {
				t.Error("Description() should not be empty")
			}

			// Dmesg messages are not reversible
			if tt.sim.IsReversible() {
				t.Error("IsReversible() should return false for dmesg simulators")
			}
		})
	}
}

func TestDmesgSimulator_DryRun(t *testing.T) {
	sim := NewForkOOMSimulator()
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}
	if len(result) < 10 {
		t.Error("DryRun should return meaningful output")
	}
}

func TestPIDExhaustionSimulator_Interface(t *testing.T) {
	sim := NewPIDExhaustionSimulator()

	var _ simulator.Simulator = sim

	if sim.Name() != "pid-exhaustion" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "pid-exhaustion")
	}

	if sim.Category() != simulator.CategoryKernel {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryKernel)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if !sim.IsReversible() {
		t.Error("IsReversible() should return true")
	}
}

func TestPIDExhaustionSimulator_DryRun(t *testing.T) {
	sim := NewPIDExhaustionSimulator()
	result := sim.DryRun(simulator.Options{})
	// May fail on non-Linux, but should not panic
	if result == "" {
		t.Log("DryRun returned empty (may be expected on non-Linux)")
	}
}

func TestDefaultZombieCount(t *testing.T) {
	// NMA threshold is >= 20, so default should be > 20
	if DefaultZombieCount < 20 {
		t.Errorf("DefaultZombieCount should be >= 20, got %d", DefaultZombieCount)
	}
}
