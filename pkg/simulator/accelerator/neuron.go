package accelerator

import (
	"context"
	"fmt"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// NeuronErrorType represents a Neuron hardware error type
type NeuronErrorType string

const (
	NeuronSRAM NeuronErrorType = "SRAM_UNCORRECTABLE"
	NeuronNC   NeuronErrorType = "NC_UNCORRECTABLE"
	NeuronHBM  NeuronErrorType = "HBM_UNCORRECTABLE"
	NeuronDMA  NeuronErrorType = "DMA_ERROR"
)

// NeuronSimulator injects AWS Neuron hardware errors to dmesg
type NeuronSimulator struct {
	errorType NeuronErrorType
}

// NewNeuronSRAMSimulator creates a simulator for SRAM errors
func NewNeuronSRAMSimulator() *NeuronSimulator {
	return &NeuronSimulator{errorType: NeuronSRAM}
}

// NewNeuronNCSimulator creates a simulator for NC errors
func NewNeuronNCSimulator() *NeuronSimulator {
	return &NeuronSimulator{errorType: NeuronNC}
}

// NewNeuronHBMSimulator creates a simulator for HBM errors
func NewNeuronHBMSimulator() *NeuronSimulator {
	return &NeuronSimulator{errorType: NeuronHBM}
}

// NewNeuronDMASimulator creates a simulator for DMA errors
func NewNeuronDMASimulator() *NeuronSimulator {
	return &NeuronSimulator{errorType: NeuronDMA}
}

func (n *NeuronSimulator) Name() string {
	switch n.errorType {
	case NeuronSRAM:
		return "neuron-sram-error"
	case NeuronNC:
		return "neuron-nc-error"
	case NeuronHBM:
		return "neuron-hbm-error"
	case NeuronDMA:
		return "neuron-dma-error"
	default:
		return "neuron-error"
	}
}

func (n *NeuronSimulator) Description() string {
	switch n.errorType {
	case NeuronSRAM:
		return "Inject Neuron SRAM uncorrectable error to dmesg"
	case NeuronNC:
		return "Inject Neuron NC uncorrectable error to dmesg"
	case NeuronHBM:
		return "Inject Neuron HBM uncorrectable error to dmesg"
	case NeuronDMA:
		return "Inject Neuron DMA error to dmesg"
	default:
		return "Inject Neuron hardware error to dmesg"
	}
}

func (n *NeuronSimulator) Category() simulator.Category {
	return simulator.CategoryAccelerator
}

func (n *NeuronSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	pattern, ok := util.NeuronErrorPatterns[string(n.errorType)]
	if !ok {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Unknown Neuron error type: %s", n.errorType),
		}, fmt.Errorf("unknown error type")
	}

	if err := util.WriteKmsg(pattern); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to write Neuron error to kmsg: %v", err),
		}, err
	}

	return &simulator.Result{
		Success: true,
		Message: fmt.Sprintf("Injected Neuron %s error to kernel log. NMA should detect AcceleratedHardwareReady=False", n.errorType),
	}, nil
}

func (n *NeuronSimulator) Cleanup(ctx context.Context) error {
	return nil
}

func (n *NeuronSimulator) DryRun(opts simulator.Options) string {
	pattern := util.NeuronErrorPatterns[string(n.errorType)]
	return fmt.Sprintf("Would write to /dev/kmsg: %s", pattern)
}

func (n *NeuronSimulator) IsReversible() bool {
	return false
}

func (n *NeuronSimulator) ShellCommand(opts simulator.Options) []string {
	pattern := util.NeuronErrorPatterns[string(n.errorType)]
	return []string{
		fmt.Sprintf("echo '%s' > /dev/kmsg", pattern),
		fmt.Sprintf("echo 'Injected Neuron %s error to kernel log'", n.errorType),
	}
}

func (n *NeuronSimulator) CleanupCommand() []string {
	return []string{"echo 'Dmesg messages cannot be removed, but NMA condition will clear over time'"}
}

func init() {
	simulator.Register(NewNeuronSRAMSimulator())
	simulator.Register(NewNeuronNCSimulator())
	simulator.Register(NewNeuronHBMSimulator())
	simulator.Register(NewNeuronDMASimulator())
}
