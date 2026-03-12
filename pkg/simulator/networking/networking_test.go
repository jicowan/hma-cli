package networking

import (
	"testing"

	"github.com/jicowan/hma-cli/pkg/simulator"
)

func TestIPAMDSimulator_Interface(t *testing.T) {
	sim := NewIPAMDSimulator()

	var _ simulator.Simulator = sim

	if sim.Name() != "ipamd-down" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "ipamd-down")
	}

	if sim.Category() != simulator.CategoryNetworking {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryNetworking)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if !sim.IsReversible() {
		t.Error("IsReversible() should return true (systemd restarts it)")
	}
}

func TestIPAMDSimulator_DryRun(t *testing.T) {
	sim := NewIPAMDSimulator()
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}
}

func TestRoutesSimulator_Interface(t *testing.T) {
	sim := NewRoutesSimulator()

	var _ simulator.Simulator = sim

	if sim.Name() != "routes-missing" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "routes-missing")
	}

	if sim.Category() != simulator.CategoryNetworking {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryNetworking)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if !sim.IsReversible() {
		t.Error("IsReversible() should return true")
	}
}

func TestRoutesSimulator_DryRun(t *testing.T) {
	sim := NewRoutesSimulator()

	// Default target
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}

	// Custom target
	result = sim.DryRun(simulator.Options{Target: "10.0.0.0/16"})
	if result == "" {
		t.Error("DryRun with target should return non-empty string")
	}
}

func TestInterfaceSimulator_Interface(t *testing.T) {
	sim := NewInterfaceSimulator()

	var _ simulator.Simulator = sim

	if sim.Name() != "interface-down" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "interface-down")
	}

	if sim.Category() != simulator.CategoryNetworking {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryNetworking)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if !sim.IsReversible() {
		t.Error("IsReversible() should return true")
	}
}

func TestInterfaceSimulator_DryRun(t *testing.T) {
	sim := NewInterfaceSimulator()

	// Default interface (eth1)
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}

	// Custom interface
	result = sim.DryRun(simulator.Options{Target: "eth2"})
	if result == "" {
		t.Error("DryRun with target should return non-empty string")
	}
}
