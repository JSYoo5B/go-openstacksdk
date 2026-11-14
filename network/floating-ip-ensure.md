# Floating Ip Ensure

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ips_to_server(
    server,
    ip_pool="public",
    nat_destination="private",
    reuse=True,
    wait=True,
    timeout=60,
)
print(server.interface_ip)
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	assignment, err := service.FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{
		Server: resource.Name("web-01"), Network: resource.Name("public"),
	}, network.WithEnsureNATDestination(resource.Name("private")),
		network.WithEnsureWait(resource.WithTimeout(time.Minute)))
	if err != nil {
		if assignment != nil && assignment.FloatingIP != nil {
			log.Printf("floating IP remains: id=%s reused=%t allocated=%t",
				assignment.FloatingIP.ID, assignment.Reused, assignment.Allocated)
		}
		return err
	}
	floating := assignment.FloatingIP
	fmt.Println(floating.ID, floating.FloatingIP, floating.PortID, floating.FixedIP, floating.Status)
	fmt.Println("reused", assignment.Reused, "allocated", assignment.Allocated)
	return nil
}
```

## References

- [PrepareEnsure·EnsurePrepared](floating-ip-plan.md)
- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [FloatingIPs.Available](floating-ip-available.md)
- [Connection AvailableFloatingIP](../compute/floating-ip-available.md)
- [Network Roles](network-roles.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394-L1487)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)
- [Compute IP dispatch](../compute/server-ip-dispatch.md)
- [Network Mutations](network-mutations.md)
- [Floating Ip Roles Test](floating_ip_roles_test.go)
- [Floating Ip Ensure Test](floating_ip_ensure_test.go)
- [Floating Ip Selection Test](floating_ip_selection_test.go)
- [Connection Floating Ip Ensure Test](../connectiontest/connection_floating_ip_ensure_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
