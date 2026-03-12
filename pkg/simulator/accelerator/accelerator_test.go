package accelerator

import (
	"testing"

	"github.com/jicowan/hma-cli/pkg/simulator"
)

func TestXIDSimulator_Interface(t *testing.T) {
	sim := NewXIDSimulator()

	var _ simulator.Simulator = sim

	if sim.Name() != "xid-error" {
		t.Errorf("Name() = %q, want %q", sim.Name(), "xid-error")
	}

	if sim.Category() != simulator.CategoryAccelerator {
		t.Errorf("Category() = %q, want %q", sim.Category(), simulator.CategoryAccelerator)
	}

	if sim.Description() == "" {
		t.Error("Description() should not be empty")
	}

	if sim.IsReversible() {
		t.Error("IsReversible() should return false for dmesg-based simulators")
	}
}

func TestXIDSimulator_DryRun(t *testing.T) {
	sim := NewXIDSimulator()

	// Default code
	result := sim.DryRun(simulator.Options{})
	if result == "" {
		t.Error("DryRun should return non-empty string")
	}

	// Custom code
	result = sim.DryRun(simulator.Options{Code: 31})
	if result == "" {
		t.Error("DryRun with code should return non-empty string")
	}
}

func TestIsFatalXID(t *testing.T) {
	fatalCodes := []int{13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140}
	nonFatalCodes := []int{1, 2, 10, 50, 100}

	for _, code := range fatalCodes {
		if !isFatalXID(code) {
			t.Errorf("XID %d should be fatal", code)
		}
	}

	for _, code := range nonFatalCodes {
		if isFatalXID(code) {
			t.Errorf("XID %d should not be fatal", code)
		}
	}
}

func TestNeuronSimulators_Interface(t *testing.T) {
	tests := []struct {
		name     string
		sim      *NeuronSimulator
		wantName string
	}{
		{
			name:     "SRAM",
			sim:      NewNeuronSRAMSimulator(),
			wantName: "neuron-sram-error",
		},
		{
			name:     "NC",
			sim:      NewNeuronNCSimulator(),
			wantName: "neuron-nc-error",
		},
		{
			name:     "HBM",
			sim:      NewNeuronHBMSimulator(),
			wantName: "neuron-hbm-error",
		},
		{
			name:     "DMA",
			sim:      NewNeuronDMASimulator(),
			wantName: "neuron-dma-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ simulator.Simulator = tt.sim

			if tt.sim.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", tt.sim.Name(), tt.wantName)
			}

			if tt.sim.Category() != simulator.CategoryAccelerator {
				t.Errorf("Category() = %q, want %q", tt.sim.Category(), simulator.CategoryAccelerator)
			}

			if tt.sim.Description() == "" {
				t.Error("Description() should not be empty")
			}

			if tt.sim.IsReversible() {
				t.Error("IsReversible() should return false")
			}
		})
	}
}

func TestDefaultXIDCode(t *testing.T) {
	if DefaultXIDCode != 79 {
		t.Errorf("DefaultXIDCode = %d, want 79", DefaultXIDCode)
	}

	// Verify default code is fatal
	if !isFatalXID(DefaultXIDCode) {
		t.Error("DefaultXIDCode should be a fatal code")
	}
}
