package util

import (
	"testing"
)

func TestNewProcessManager(t *testing.T) {
	pm := NewProcessManager()
	if pm == nil {
		t.Fatal("NewProcessManager should not return nil")
	}
	if pm.pids == nil {
		t.Error("pids slice should be initialized")
	}
	if len(pm.pids) != 0 {
		t.Errorf("pids slice should be empty, got %d", len(pm.pids))
	}
}

func TestProcessManager_KillAll_Empty(t *testing.T) {
	pm := NewProcessManager()
	// KillAll on empty manager should not error
	err := pm.KillAll()
	if err != nil {
		t.Errorf("KillAll on empty manager should not error: %v", err)
	}
}

func TestGetPIDMax(t *testing.T) {
	// This test may fail on non-Linux systems
	pidMax, err := GetPIDMax()
	if err != nil {
		t.Skipf("Skipping on non-Linux system: %v", err)
	}
	if pidMax <= 0 {
		t.Errorf("pid_max should be positive, got %d", pidMax)
	}
	// Typical Linux systems have pid_max between 32768 and 4194304
	if pidMax < 1000 {
		t.Errorf("pid_max seems too low: %d", pidMax)
	}
}

func TestGetThreadsMax(t *testing.T) {
	// This test may fail on non-Linux systems
	threadsMax, err := GetThreadsMax()
	if err != nil {
		t.Skipf("Skipping on non-Linux system: %v", err)
	}
	if threadsMax <= 0 {
		t.Errorf("threads-max should be positive, got %d", threadsMax)
	}
}

func TestGetFileMax(t *testing.T) {
	// This test may fail on non-Linux systems
	fileMax, err := GetFileMax()
	if err != nil {
		t.Skipf("Skipping on non-Linux system: %v", err)
	}
	if fileMax <= 0 {
		t.Errorf("file-max should be positive, got %d", fileMax)
	}
}

func TestCountCurrentPIDs(t *testing.T) {
	// This test may fail on non-Linux systems
	count, err := CountCurrentPIDs()
	if err != nil {
		t.Skipf("Skipping on non-Linux system: %v", err)
	}
	// There should be at least 1 process (this process)
	if count < 1 {
		t.Errorf("should have at least 1 PID, got %d", count)
	}
}

func TestCountZombies(t *testing.T) {
	// This test may fail on non-Linux systems
	count, err := CountZombies()
	if err != nil {
		t.Skipf("Skipping on non-Linux system: %v", err)
	}
	// Count should be non-negative
	if count < 0 {
		t.Errorf("zombie count should not be negative, got %d", count)
	}
}
