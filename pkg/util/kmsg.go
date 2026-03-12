package util

import (
	"fmt"
	"os"
)

const kmsgPath = "/dev/kmsg"

// KmsgWriter writes messages to the kernel log
type KmsgWriter struct {
	file *os.File
}

// NewKmsgWriter creates a new kernel message writer
func NewKmsgWriter() (*KmsgWriter, error) {
	f, err := os.OpenFile(kmsgPath, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w (are you running as root?)", kmsgPath, err)
	}
	return &KmsgWriter{file: f}, nil
}

// Write writes a message to the kernel log
func (w *KmsgWriter) Write(message string) error {
	_, err := w.file.WriteString(message + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to kmsg: %w", err)
	}
	return nil
}

// Close closes the kmsg file
func (w *KmsgWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// WriteKmsg is a convenience function to write a single message to kmsg
func WriteKmsg(message string) error {
	w, err := NewKmsgWriter()
	if err != nil {
		return err
	}
	defer w.Close()
	return w.Write(message)
}

// KernelLogPatterns contains common kernel log patterns that the NMA monitors
// Based on NMA source: https://github.com/aws/eks-node-monitoring-agent/blob/main/monitors/kernel/monitor.go
var KernelLogPatterns = struct {
	ForkOOM       string
	KernelBug     string
	SoftLockup    string
	ConntrackFull string
}{
	// ForkOOM: NMA doesn't watch dmesg for this - it watches kubelet journal for:
	// `fork/exec.*resource temporarily unavailable` - this pattern is a placeholder
	ForkOOM: "fork_oom: fork rejected due to memory constraints",
	// KernelBug: NMA regex: \[.*?] BUG: (.*)
	// Must include timestamp in brackets followed by "BUG:"
	KernelBug: "[12345.678901] BUG: kernel memory corruption detected at mm/memory.c:1234",
	// SoftLockup: NMA regex: watchdog: BUG: soft lockup - .* stuck for (.*)! \[(.*?).*\]
	// Must include process name in brackets at the end
	SoftLockup: "watchdog: BUG: soft lockup - CPU#0 stuck for 22s! [kworker/0:0:1234]",
	// ConntrackFull: NMA regex: (ip|nf)_conntrack: table full, dropping packet
	ConntrackFull: "nf_conntrack: table full, dropping packet",
}

// NvidiaXIDPatterns contains NVIDIA XID error patterns
// Fatal XID codes: 13, 31, 48, 63, 64, 74, 79, 94, 95, 119, 120, 121, 140
var NvidiaXIDPatterns = map[int]string{
	13:  "NVRM: Xid (PCI:0000:00:1e.0): 13, Graphics Exception: ESR 0x123456",
	31:  "NVRM: Xid (PCI:0000:00:1e.0): 31, GPU has fallen off the bus",
	48:  "NVRM: Xid (PCI:0000:00:1e.0): 48, DBE (Double Bit Error) ECC detected",
	63:  "NVRM: Xid (PCI:0000:00:1e.0): 63, Row Remapper: An uncorrectable error has occurred",
	64:  "NVRM: Xid (PCI:0000:00:1e.0): 64, Row Remapper: A critical error has occurred",
	74:  "NVRM: Xid (PCI:0000:00:1e.0): 74, GPU has encountered a NVLink error",
	79:  "NVRM: Xid (PCI:0000:00:1e.0): 79, GPU has encountered an uncorrectable error",
	94:  "NVRM: Xid (PCI:0000:00:1e.0): 94, Contained ECC error",
	95:  "NVRM: Xid (PCI:0000:00:1e.0): 95, Uncontained ECC error",
	119: "NVRM: Xid (PCI:0000:00:1e.0): 119, GSP RPC timeout",
	120: "NVRM: Xid (PCI:0000:00:1e.0): 120, GSP communication timeout",
	121: "NVRM: Xid (PCI:0000:00:1e.0): 121, GSP communication error",
	140: "NVRM: Xid (PCI:0000:00:1e.0): 140, GPU encountered an uncorrectable memory error",
}

// NeuronErrorPatterns contains AWS Neuron hardware error patterns
var NeuronErrorPatterns = map[string]string{
	"SRAM_UNCORRECTABLE": "neuron0: NEURON_HW_ERR=SRAM_UNCORRECTABLE_ERROR device=0",
	"NC_UNCORRECTABLE":   "neuron0: NEURON_HW_ERR=NC_UNCORRECTABLE_ERROR device=0",
	"HBM_UNCORRECTABLE":  "neuron0: NEURON_HW_ERR=HBM_UNCORRECTABLE_ERROR device=0",
	"DMA_ERROR":          "neuron0: NEURON_HW_ERR=DMA_ERROR device=0",
}
