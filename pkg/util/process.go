package util

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// ProcessManager handles process creation and management for simulations
type ProcessManager struct {
	pids []int
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		pids: make([]int, 0),
	}
}

// CreateZombies creates zombie processes by forking children that exit immediately
// without the parent calling wait()
func (pm *ProcessManager) CreateZombies(count int) ([]int, error) {
	zombiePids := make([]int, 0, count)

	for i := 0; i < count; i++ {
		// Fork a child that exits immediately
		cmd := exec.Command("/bin/sh", "-c", "exit 0")
		if err := cmd.Start(); err != nil {
			return zombiePids, fmt.Errorf("failed to create zombie %d: %w", i, err)
		}
		zombiePids = append(zombiePids, cmd.Process.Pid)
		pm.pids = append(pm.pids, cmd.Process.Pid)
		// Intentionally NOT calling cmd.Wait() to create zombie
	}

	return zombiePids, nil
}

// CreateSleepingProcesses creates sleeping processes to exhaust PIDs
func (pm *ProcessManager) CreateSleepingProcesses(count int) ([]int, error) {
	sleepPids := make([]int, 0, count)

	for i := 0; i < count; i++ {
		cmd := exec.Command("sleep", "infinity")
		if err := cmd.Start(); err != nil {
			return sleepPids, fmt.Errorf("failed to create sleeping process %d: %w", i, err)
		}
		sleepPids = append(sleepPids, cmd.Process.Pid)
		pm.pids = append(pm.pids, cmd.Process.Pid)
	}

	return sleepPids, nil
}

// KillAll kills all processes tracked by this manager
func (pm *ProcessManager) KillAll() error {
	var lastErr error
	for _, pid := range pm.pids {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			// Ignore "no such process" errors (process already dead)
			if err != syscall.ESRCH {
				lastErr = fmt.Errorf("failed to kill PID %d: %w", pid, err)
			}
		}
	}
	pm.pids = nil
	return lastErr
}

// GetPIDMax returns the maximum number of PIDs allowed on the system
func GetPIDMax() (int, error) {
	data, err := os.ReadFile("/proc/sys/kernel/pid_max")
	if err != nil {
		return 0, fmt.Errorf("failed to read pid_max: %w", err)
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// GetThreadsMax returns the maximum number of threads allowed on the system
func GetThreadsMax() (int, error) {
	data, err := os.ReadFile("/proc/sys/kernel/threads-max")
	if err != nil {
		return 0, fmt.Errorf("failed to read threads-max: %w", err)
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// GetFileMax returns the maximum number of file handles allowed on the system
func GetFileMax() (int, error) {
	data, err := os.ReadFile("/proc/sys/fs/file-max")
	if err != nil {
		return 0, fmt.Errorf("failed to read file-max: %w", err)
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// CountCurrentPIDs returns the current number of PIDs in use
func CountCurrentPIDs() (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := strconv.Atoi(entry.Name()); err == nil {
				count++
			}
		}
	}
	return count, nil
}

// CountZombies returns the current number of zombie processes
func CountZombies() (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := strconv.Atoi(entry.Name()); err == nil {
				statPath := fmt.Sprintf("/proc/%s/stat", entry.Name())
				data, err := os.ReadFile(statPath)
				if err != nil {
					continue
				}
				// Process state is the third field, 'Z' indicates zombie
				fields := strings.Fields(string(data))
				if len(fields) >= 3 && fields[2] == "Z" {
					count++
				}
			}
		}
	}
	return count, nil
}
