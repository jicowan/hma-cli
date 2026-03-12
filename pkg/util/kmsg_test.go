package util

import (
	"testing"
)

func TestKernelLogPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "ForkOOM pattern",
			pattern: KernelLogPatterns.ForkOOM,
			want:    "fork_oom",
		},
		{
			name:    "KernelBug pattern",
			pattern: KernelLogPatterns.KernelBug,
			want:    "BUG:", // NMA regex: \[.*?] BUG: (.*)
		},
		{
			name:    "SoftLockup pattern",
			pattern: KernelLogPatterns.SoftLockup,
			want:    "soft lockup",
		},
		{
			name:    "ConntrackFull pattern",
			pattern: KernelLogPatterns.ConntrackFull,
			want:    "table full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pattern == "" {
				t.Error("pattern should not be empty")
			}
			if !contains(tt.pattern, tt.want) {
				t.Errorf("pattern %q should contain %q", tt.pattern, tt.want)
			}
		})
	}
}

func TestNvidiaXIDPatterns(t *testing.T) {
	// Fatal XID codes that the NMA monitors
	fatalCodes := []int{13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140}

	for _, code := range fatalCodes {
		t.Run("XID code "+string(rune('0'+code%10)), func(t *testing.T) {
			pattern, ok := NvidiaXIDPatterns[code]
			if !ok {
				t.Errorf("missing pattern for fatal XID code %d", code)
				return
			}
			if pattern == "" {
				t.Errorf("pattern for XID code %d should not be empty", code)
			}
			// Verify pattern contains the XID code
			if !contains(pattern, "Xid") {
				t.Errorf("pattern for XID code %d should contain 'Xid'", code)
			}
		})
	}
}

func TestNeuronErrorPatterns(t *testing.T) {
	expectedErrors := []string{"SRAM_UNCORRECTABLE", "NC_UNCORRECTABLE", "HBM_UNCORRECTABLE", "DMA_ERROR"}

	for _, errType := range expectedErrors {
		t.Run(errType, func(t *testing.T) {
			pattern, ok := NeuronErrorPatterns[errType]
			if !ok {
				t.Errorf("missing pattern for Neuron error type %s", errType)
				return
			}
			if pattern == "" {
				t.Errorf("pattern for error type %s should not be empty", errType)
			}
			if !contains(pattern, "NEURON_HW_ERR") {
				t.Errorf("pattern should contain 'NEURON_HW_ERR'")
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
