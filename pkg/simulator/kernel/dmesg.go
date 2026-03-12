package kernel

import (
	"context"
	"fmt"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// DmesgPattern represents a dmesg pattern to inject
type DmesgPattern string

const (
	PatternForkOOM    DmesgPattern = "fork-oom"
	PatternKernelBug  DmesgPattern = "kernel-bug"
	PatternSoftLockup DmesgPattern = "soft-lockup"
)

// DmesgSimulator injects kernel log messages to trigger conditions
type DmesgSimulator struct {
	pattern DmesgPattern
	message string
}

// NewForkOOMSimulator creates a simulator for fork OOM errors
func NewForkOOMSimulator() *DmesgSimulator {
	return &DmesgSimulator{
		pattern: PatternForkOOM,
		message: util.KernelLogPatterns.ForkOOM,
	}
}

// NewKernelBugSimulator creates a simulator for kernel BUG errors
func NewKernelBugSimulator() *DmesgSimulator {
	return &DmesgSimulator{
		pattern: PatternKernelBug,
		message: util.KernelLogPatterns.KernelBug,
	}
}

// NewSoftLockupSimulator creates a simulator for CPU soft lockup errors
func NewSoftLockupSimulator() *DmesgSimulator {
	return &DmesgSimulator{
		pattern: PatternSoftLockup,
		message: util.KernelLogPatterns.SoftLockup,
	}
}

func (d *DmesgSimulator) Name() string {
	return string(d.pattern)
}

func (d *DmesgSimulator) Description() string {
	switch d.pattern {
	case PatternForkOOM:
		// NOTE: NMA watches kubelet journal for fork failures, not dmesg
		// This simulation won't trigger NMA detection - use pid-exhaustion instead
		return "[NOT WORKING] Injects to dmesg but NMA watches kubelet journal for 'fork/exec.*resource temporarily unavailable'"
	case PatternKernelBug:
		return "Inject '[timestamp] BUG:' message to dmesg to trigger KernelBug event (EVENT-level, Warning only)"
	case PatternSoftLockup:
		return "Inject soft lockup message to dmesg to trigger SoftLockup event (EVENT-level, Warning only)"
	default:
		return "Inject pattern to dmesg"
	}
}

func (d *DmesgSimulator) Category() simulator.Category {
	return simulator.CategoryKernel
}

func (d *DmesgSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	if err := util.WriteKmsg(d.message); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to write to kmsg: %v", err),
		}, err
	}

	var msg string
	switch d.pattern {
	case PatternForkOOM:
		msg = fmt.Sprintf("Injected '%s' to dmesg. WARNING: NMA won't detect this - it watches kubelet journal, not dmesg", d.pattern)
	case PatternKernelBug, PatternSoftLockup:
		msg = fmt.Sprintf("Injected '%s' to dmesg. NMA should create a Warning event (EVENT-level only, status stays True)", d.pattern)
	default:
		msg = fmt.Sprintf("Injected '%s' pattern to kernel log", d.pattern)
	}
	return &simulator.Result{
		Success: true,
		Message: msg,
	}, nil
}

func (d *DmesgSimulator) Cleanup(ctx context.Context) error {
	// Dmesg messages cannot be removed, but NMA uses a time window
	// so the condition will clear after some time
	return nil
}

func (d *DmesgSimulator) DryRun(opts simulator.Options) string {
	return fmt.Sprintf("Would write to /dev/kmsg: %s", d.message)
}

func (d *DmesgSimulator) IsReversible() bool {
	// Messages persist in dmesg but NMA condition will clear over time
	return false
}

func (d *DmesgSimulator) ShellCommand(opts simulator.Options) []string {
	return []string{
		fmt.Sprintf("echo '%s' > /dev/kmsg", d.message),
		fmt.Sprintf("echo 'Injected %s pattern to kernel log'", d.pattern),
	}
}

func (d *DmesgSimulator) CleanupCommand() []string {
	// Dmesg messages cannot be removed
	return []string{"echo 'Dmesg messages cannot be removed, but NMA condition will clear over time'"}
}

func init() {
	simulator.Register(NewForkOOMSimulator())
	simulator.Register(NewKernelBugSimulator())
	simulator.Register(NewSoftLockupSimulator())
}
