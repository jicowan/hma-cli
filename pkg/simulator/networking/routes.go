package networking

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jicowan/hma-cli/pkg/simulator"
	"github.com/jicowan/hma-cli/pkg/util"
)

// RoutesSimulator simulates missing routes
type RoutesSimulator struct {
	deletedRoutes []string
}

// NewRoutesSimulator creates a new routes simulator
func NewRoutesSimulator() *RoutesSimulator {
	return &RoutesSimulator{
		deletedRoutes: make([]string, 0),
	}
}

func (r *RoutesSimulator) Name() string {
	return "routes-missing"
}

func (r *RoutesSimulator) Description() string {
	return "Delete pod-specific routes to trigger MissingIPRoutes event (NMA checks routes from IPAMD checkpoint)"
}

func (r *RoutesSimulator) Category() simulator.Category {
	return simulator.CategoryNetworking
}

func (r *RoutesSimulator) Simulate(ctx context.Context, opts simulator.Options) (*simulator.Result, error) {
	if err := util.RequireRoot(); err != nil {
		return nil, err
	}

	// Get current routes
	cmd := exec.CommandContext(ctx, "ip", "route", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	// Find VPC routes (typically 10.x.x.x/x)
	lines := strings.Split(string(output), "\n")
	var vpcRoutes []string
	for _, line := range lines {
		// Look for routes that start with 10. (VPC CIDR)
		if strings.HasPrefix(line, "10.") && !strings.Contains(line, "blackhole") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				vpcRoutes = append(vpcRoutes, parts[0])
			}
		}
	}

	if len(vpcRoutes) == 0 {
		return &simulator.Result{
			Success: false,
			Message: "No VPC routes (10.x.x.x) found to delete",
		}, nil
	}

	// Delete the first VPC route (to minimize impact)
	routeToDelete := vpcRoutes[0]
	if opts.Target != "" {
		routeToDelete = opts.Target
	}

	deleteCmd := exec.CommandContext(ctx, "ip", "route", "del", routeToDelete)
	if err := deleteCmd.Run(); err != nil {
		return &simulator.Result{
			Success: false,
			Message: fmt.Sprintf("Failed to delete route %s: %v", routeToDelete, err),
		}, err
	}

	r.deletedRoutes = append(r.deletedRoutes, routeToDelete)

	return &simulator.Result{
		Success:    true,
		Message:    fmt.Sprintf("Deleted route %s. NMA should detect NetworkingReady=False", routeToDelete),
		CleanupCmd: fmt.Sprintf("ip route add %s", routeToDelete),
	}, nil
}

func (r *RoutesSimulator) Cleanup(ctx context.Context) error {
	// Note: Route restoration requires knowing the original route details
	// This is a simplified cleanup - in practice you'd need to restore with proper gateway/device
	for _, route := range r.deletedRoutes {
		// Try to re-add the route via default interface
		cmd := exec.CommandContext(ctx, "ip", "route", "add", route, "dev", "eth0")
		cmd.Run() // Best effort
	}
	r.deletedRoutes = nil
	return nil
}

func (r *RoutesSimulator) DryRun(opts simulator.Options) string {
	if opts.Target != "" {
		return fmt.Sprintf("Would delete route: %s", opts.Target)
	}
	return "Would delete a VPC route (10.x.x.x/x) using 'ip route del'"
}

func (r *RoutesSimulator) IsReversible() bool {
	return true // But requires proper route details
}

func (r *RoutesSimulator) ShellCommand(opts simulator.Options) []string {
	// NMA checks routes from IPAMD checkpoint file, not VPC CIDR routes.
	// It looks for routes like: <pod-ip> dev <veth> scope link
	// We need to delete a pod-specific route to trigger MissingIPRoutes.
	if opts.Target != "" {
		return []string{
			fmt.Sprintf("ip route del %s && echo 'Deleted route %s' || echo 'Failed to delete route %s'", opts.Target, opts.Target, opts.Target),
		}
	}
	script := `#!/bin/sh
echo "=== Route Missing Simulation ==="
echo "NMA checks for pod-specific routes from IPAMD checkpoint"
echo "Looking for routes like: <pod-ip> dev <veth> scope link"
echo ""

# Find pod-specific routes (routes to individual IPs with scope link)
# These are typically created by IPAMD for pod IPs
POD_ROUTES=$(ip route show | grep "scope link" | grep -E "^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+ " | head -5)

if [ -z "$POD_ROUTES" ]; then
  echo "No pod-specific routes found (routes with 'scope link')."
  echo ""
  echo "Checking IPAMD checkpoint for allocated pod IPs..."
  CHECKPOINT="/var/run/aws-node/ipam.json"
  if [ -f "$CHECKPOINT" ]; then
    echo "IPAMD checkpoint exists. Checking for allocated IPs..."
    # Try to extract IPs from checkpoint
    if command -v jq >/dev/null 2>&1; then
      ALLOC_IPS=$(jq -r '.allocations[].ipv4 // empty' "$CHECKPOINT" 2>/dev/null | head -3)
      if [ -n "$ALLOC_IPS" ]; then
        echo "Found allocated pod IPs:"
        echo "$ALLOC_IPS"
        # Delete route for first allocated IP
        FIRST_IP=$(echo "$ALLOC_IPS" | head -1)
        echo ""
        echo "Deleting route for pod IP: $FIRST_IP"
        # Save current route for cleanup
        FULL_ROUTE=$(ip route show "$FIRST_IP" 2>/dev/null | head -1)
        echo "$FULL_ROUTE" > /tmp/deleted_route.txt
        ip route del "$FIRST_IP" 2>/dev/null && echo "Deleted: $FULL_ROUTE" || echo "Failed to delete route"
        exit 0
      fi
    fi
    echo "Could not parse IPAMD checkpoint (jq not available or no allocations)"
  else
    echo "IPAMD checkpoint not found at $CHECKPOINT"
  fi
  echo ""
  echo "Falling back to deleting any scope link route..."
  FALLBACK_ROUTE=$(ip route show | grep "scope link" | head -1)
  if [ -n "$FALLBACK_ROUTE" ]; then
    ROUTE_IP=$(echo "$FALLBACK_ROUTE" | awk '{print $1}')
    echo "Deleting: $FALLBACK_ROUTE"
    echo "$FALLBACK_ROUTE" > /tmp/deleted_route.txt
    ip route del "$ROUTE_IP" && echo "Deleted route $ROUTE_IP" || echo "Failed"
  else
    echo "No routes with 'scope link' found. Cannot simulate."
  fi
  exit 0
fi

echo "Found pod-specific routes:"
echo "$POD_ROUTES"
echo ""

# Delete the first one
FIRST_ROUTE=$(echo "$POD_ROUTES" | head -1)
ROUTE_IP=$(echo "$FIRST_ROUTE" | awk '{print $1}')

# Save full route for cleanup
echo "$FIRST_ROUTE" > /tmp/deleted_route.txt

echo "Deleting route: $FIRST_ROUTE"
ip route del "$ROUTE_IP" && echo "SUCCESS: Deleted route for $ROUTE_IP" || echo "FAILED to delete route"

echo ""
echo "NMA will detect MissingIPRoutes if this pod IP is in IPAMD checkpoint."
echo "Restore with: ip route add $(cat /tmp/deleted_route.txt)"`
	return []string{script}
}

func (r *RoutesSimulator) CleanupCommand() []string {
	return []string{
		`#!/bin/sh
if [ -f /tmp/deleted_route.txt ]; then
  ROUTE=$(cat /tmp/deleted_route.txt)
  echo "Restoring route: $ROUTE"
  ip route add $ROUTE 2>/dev/null && echo "Route restored" || echo "Failed to restore (may already exist)"
  rm -f /tmp/deleted_route.txt
else
  echo "No saved route found. Manual restoration may be needed."
  echo "Use: ip route add <ip> dev <interface> scope link"
fi`,
	}
}

func init() {
	simulator.Register(NewRoutesSimulator())
}
