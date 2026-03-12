package accelerator

import (
	"context"
	"fmt"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// Default XID code that causes fatal error
const DefaultXIDCode = 79

// XIDSimulator injects NVIDIA XID errors to dmesg
type XIDSimulator struct{}

// NewXIDSimulator creates a new XID error simulator
func NewXIDSimulator() *XIDSimulator {
	return &XIDSimulator{}
}

func (x *XIDSimulator) Name() string {
	return "xid-error"
}

func (x *XIDSimulator) Description() string {
	return "Inject NVIDIA XID error via DCGM to trigger AcceleratedHardwareReady=False"
}

func (x *XIDSimulator) Category() simulator.Category {
	return simulator.CategoryAccelerator
}

func (x *XIDSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	code := opts.Code
	if code == 0 {
		code = DefaultXIDCode
	}

	pattern, ok := util.NvidiaXIDPatterns[code]
	if !ok {
		// Generate a generic pattern for unknown codes
		pattern = fmt.Sprintf("NVRM: Xid (PCI:0000:00:1e.0): %d, GPU error occurred", code)
	}

	if err := util.WriteKmsg(pattern); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to write XID error to kmsg: %v", err),
		}, err
	}

	isFatal := isFatalXID(code)
	severity := "warning"
	if isFatal {
		severity = "FATAL"
	}

	return &simulator.Result{
		Success: true,
		Message: fmt.Sprintf("Injected XID %d (%s) error to kernel log. NMA should detect AcceleratedHardwareReady=False", code, severity),
	}, nil
}

func (x *XIDSimulator) Cleanup(ctx context.Context) error {
	// Dmesg messages cannot be removed
	return nil
}

func (x *XIDSimulator) DryRun(opts simulator.Options) string {
	code := opts.Code
	if code == 0 {
		code = DefaultXIDCode
	}
	return fmt.Sprintf("Would inject XID %d via DCGM: dcgmi test --inject --gpuid 0 -f 230 -v %d", code, code)
}

func (x *XIDSimulator) IsReversible() bool {
	return false
}

func (x *XIDSimulator) ShellCommand(opts simulator.Options) []string {
	code := opts.Code
	if code == 0 {
		code = DefaultXIDCode
	}
	// Field ID 230 is xid_errors in DCGM
	// Use dcgmi test --inject to inject XID error directly into DCGM cache
	// This triggers NMA's DCGM health monitoring
	return []string{
		fmt.Sprintf(`if command -v dcgmi >/dev/null 2>&1; then
  dcgmi test --inject --gpuid 0 -f 230 -v %d
  echo "Injected XID %d error via DCGM (field 230)"
else
  echo "ERROR: dcgmi not found. Install datacenter-gpu-manager package."
  echo "  dnf install -y datacenter-gpu-manager-4-cuda12  # or cuda13"
  echo "  systemctl enable --now nvidia-dcgm"
  exit 1
fi`, code, code),
	}
}

func (x *XIDSimulator) CleanupCommand() []string {
	return []string{"echo 'Dmesg messages cannot be removed, but NMA condition will clear over time'"}
}

// isFatalXID returns true if the XID code is considered fatal by NMA
func isFatalXID(code int) bool {
	fatalCodes := map[int]bool{
		13: true, 31: true, 48: true, 63: true, 64: true,
		74: true, 79: true, 94: true, 95: true, 119: true,
		120: true, 121: true, 140: true,
	}
	return fatalCodes[code]
}

func init() {
	simulator.Register(NewXIDSimulator())
}
