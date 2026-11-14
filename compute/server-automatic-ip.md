# Server Automatic Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None or server.status != "ACTIVE":
    raise RuntimeError("an existing ACTIVE server is required")
server = conn.add_ips_to_server(
    server,
    auto_ip=True,
    reuse=True,
    wait=True,
    timeout=180,
)
print(server.interface_ip)
```

```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	byID := flag.Bool("server-id", false, "treat -server as an ID")
	external := flag.String("network", "", "exact external network name; empty uses roles/router")
	project := flag.String("project-id", "", "explicit reuse/allocation owner")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP before allocation")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	execute := flag.Bool("execute", false, "assign when needed and observe actual Nova addresses")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *server, *byID, *external, *project, *reuse, *enabled, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, nameOrID string, byID bool, external, project string, reuse, enabled, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(nameOrID)
	if byID {
		ref = resource.ID(nameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	request := compute.AutomaticFloatingIPRequest{Server: server}
	if external != "" {
		request.Network = resource.Name(external)
	}
	ipOptions := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
	if project != "" {
		ipOptions = append(ipOptions, network.WithEnsureProject(project))
	}
	options := []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticIPEnabled(enabled),
		compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
		compute.WithAutomaticEnsureOptions(ipOptions...),
		compute.WithAutomaticIPWait(resource.WithTimeout(3*time.Minute), resource.WithPollInterval(time.Second)),
		compute.WithAutomaticIPProgressCallback(func(current *compute.Server) error {
			fmt.Printf("observing server=%s status=%s\n", current.ID, current.Status)
			return nil
		}),
	}
	if !execute {
		decision, err := conn.PlanServerFloatingIP(ctx, request, options...)
		printDecision(decision)
		return err
	}

	result, err := conn.EnsureServerFloatingIP(ctx, request, options...)
	if result != nil {
		printDecision(result.Decision)
		fmt.Printf("observed=%t\n", result.Observed)
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if result.Assignment != nil {
			assignment := result.Assignment
			fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
			if assignment.FloatingIP != nil {
				fmt.Printf("floating4=%s ip-id=%s status=%s\n",
					assignment.FloatingIP.FloatingIP, assignment.FloatingIP.ID,
					assignment.FloatingIP.Status)
			}
		}
	}
	if err != nil {
		if errors.Is(err, resource.ErrUnsupported) {
			log.Printf("the selected backend or scope is not supported by this workflow")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("automatic floating IPv4: %w", err)
	}
	return nil
}

func printDecision(decision *compute.ServerFloatingIPDecision) {
	if decision == nil {
		return
	}
	fmt.Printf("needed=%t reason=%s backend=%s\n",
		decision.Needed, decision.Reason, decision.Backend)
	if decision.Needed && decision.Backend == compute.FloatingIPNeutron {
		selection := decision.Selection
		fmt.Printf("external=%s port-network=%s port=%s fixed4=%s\n",
			selection.NetworkID, selection.PortNetworkID,
			selection.PortID, selection.FixedIPv4)
	}
}
```

## References

- [legacy Nova IP backend](server-nova-floating-ip.md)
- [Floating Ip Plan](../network/floating-ip-plan.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1527-L1606)
- [attach wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1164)
- [legacy Nova mutation](server-nova-floating-ip.md)
- [add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1424)
- [legacy Nova assignment](server-nova-floating-ip.md)
- [Server Addresses](server-addresses.md)
- [Network Roles](../network/network-roles.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
- [Server Ready](server-ready.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
