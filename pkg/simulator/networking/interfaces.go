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
	return "Bring down secondary ENI (auto-detects eth1/ens6) to trigger NetworkingReady=False"
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
	if opts.Target != "" {
		return fmt.Sprintf("Would bring down interface %s using 'ip link set %s down'", opts.Target, opts.Target)
	}
	return "Would auto-detect and bring down secondary ENI (eth1/ens6/enp*s*) using 'ip link set <iface> down'"
}

func (i *InterfaceSimulator) IsReversible() bool {
	return true
}

func (i *InterfaceSimulator) ShellCommand(opts simulator.Options) []string {
	if opts.Target != "" {
		// User specified interface explicitly
		iface := opts.Target
		return []string{
			fmt.Sprintf("ip link set %s down && echo 'Interface %s brought down' || echo 'Failed to bring down interface %s'", iface, iface, iface),
		}
	}
	// Auto-detect secondary ENI (works for both eth1 and ens6 naming schemes)
	return []string{
		`# Find secondary ENI (not lo, not primary, not veth/eni interfaces)
SECONDARY=$(ip -o link show | awk -F': ' '{print $2}' | grep -E '^(eth[1-9]|ens[6-9]|enp[0-9]+s[1-9])$' | head -1)
if [ -z "$SECONDARY" ]; then
  echo "ERROR: No secondary ENI found. Available interfaces:"
  ip -o link show | awk -F': ' '{print "  " $2}'
  echo ""
  echo "Use --target <interface> to specify manually"
  exit 1
fi
echo "Detected secondary ENI: $SECONDARY"
ip link set $SECONDARY down && echo "Interface $SECONDARY brought down" || echo "Failed to bring down interface $SECONDARY"`,
	}
}

func (i *InterfaceSimulator) CleanupCommand() []string {
	// Auto-detect and bring up the secondary ENI
	return []string{
		`# Find secondary ENI that is down
SECONDARY=$(ip -o link show | grep -E '(eth[1-9]|ens[6-9]|enp[0-9]+s[1-9])' | grep -i 'state DOWN' | awk -F': ' '{print $2}' | head -1)
if [ -z "$SECONDARY" ]; then
  # Try finding any secondary ENI (might already be up)
  SECONDARY=$(ip -o link show | awk -F': ' '{print $2}' | grep -E '^(eth[1-9]|ens[6-9]|enp[0-9]+s[1-9])$' | head -1)
fi
if [ -z "$SECONDARY" ]; then
  echo "No secondary ENI found to bring up"
  exit 0
fi
echo "Bringing up interface: $SECONDARY"
ip link set $SECONDARY up && echo "Interface $SECONDARY brought up" || echo "Failed to bring up interface $SECONDARY"`,
	}
}

func init() {
	simulator.Register(NewInterfaceSimulator())
}
