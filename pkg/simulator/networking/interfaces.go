package networking

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// InterfaceSimulator simulates network interface issues
type InterfaceSimulator struct {
	downedInterface string
}

// NewInterfaceSimulator creates a new interface simulator
func NewInterfaceSimulator() *InterfaceSimulator {
	return &InterfaceSimulator{}
}

func (i *InterfaceSimulator) Name() string {
	return "interface-down"
}

func (i *InterfaceSimulator) Description() string {
	return "Bring down secondary ENI (eth1) to trigger NetworkingReady=False"
}

func (i *InterfaceSimulator) Category() simulator.Category {
	return simulator.CategoryNetworking
}

func (i *InterfaceSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	// Default to eth1 (secondary ENI) unless specified
	iface := "eth1"
	if opts.Target != "" {
		iface = opts.Target
	}

	// Check if interface exists
	checkCmd := exec.CommandContext(ctx, "ip", "link", "show", iface)
	if err := checkCmd.Run(); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Interface %s not found. This node may not have a secondary ENI.", iface),
		}, nil
	}

	// Bring interface down
	cmd := exec.CommandContext(ctx, "ip", "link", "set", iface, "down")
	if err := cmd.Run(); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to bring down interface %s: %v", iface, err),
		}, err
	}

	i.downedInterface = iface

	return &simulator.Result{
		Success:    true,
		Message:    fmt.Sprintf("Interface %s brought down. NMA should detect NetworkingReady=False", iface),
		CleanupCmd: fmt.Sprintf("hma-cli networking interface-down --cleanup"),
	}, nil
}

func (i *InterfaceSimulator) Cleanup(ctx context.Context) error {
	if i.downedInterface == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "ip", "link", "set", i.downedInterface, "up")
	err := cmd.Run()
	i.downedInterface = ""
	return err
}

func (i *InterfaceSimulator) DryRun(opts simulator.Options) string {
	iface := "eth1"
	if opts.Target != "" {
		iface = opts.Target
	}
	return fmt.Sprintf("Would bring down interface %s using 'ip link set %s down'", iface, iface)
}

func (i *InterfaceSimulator) IsReversible() bool {
	return true
}

func (i *InterfaceSimulator) ShellCommand(opts simulator.Options) []string {
	iface := "eth1"
	if opts.Target != "" {
		iface = opts.Target
	}
	return []string{
		fmt.Sprintf("ip link set %s down && echo 'Interface %s brought down' || echo 'Failed to bring down interface %s'", iface, iface, iface),
	}
}

func (i *InterfaceSimulator) CleanupCommand() []string {
	// Bring up eth1 by default; user should use --target if different interface was used
	return []string{
		"ip link set eth1 up && echo 'Interface eth1 brought up' || echo 'Failed to bring up interface eth1'",
	}
}

func init() {
	simulator.Register(NewInterfaceSimulator())
}
