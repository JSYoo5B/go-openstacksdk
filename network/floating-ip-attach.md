# Floating Ip Attach

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ip_list(
    server, ips=["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server-id", "", "existing Nova server ID")
	ipID := flag.String("ip-id", "", "existing Floating IP ID")
	address := flag.String("address", "", "existing floating IPv4 address")
	fixed := flag.String("fixed", "", "exact destination fixed IPv4")
	execute := flag.Bool("execute", false, "execute the prepared attachment")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *server, *ipID, *address, *fixed, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, server, ipID, address, fixed string, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	ip := resource.Name(address)
	if ipID != "" {
		ip = resource.ID(ipID)
	}
	options := []network.AttachFloatingIPOption{
		network.WithAttachWait(resource.WithTimeout(time.Minute)),
	}
	if fixed != "" {
		options = append(options, network.WithAttachFixedAddress(fixed))
	}
	plan, err := service.FloatingIPs.PrepareAttach(ctx,
		network.AttachFloatingIPRequest{Server: resource.ID(server), IP: ip}, options...)
	if err != nil {
		return err
	}
	fmt.Printf("selection: %+v\n", plan.Selection())
	if !execute {
		return nil
	}
	result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
	if result != nil && result.FloatingIP != nil {
		fmt.Printf("IP: %s address=%s port=%s reused=%t allocated=%t\n",
			result.FloatingIP.ID, result.FloatingIP.FloatingIP, result.FloatingIP.PortID,
			result.Reused, result.Allocated)
	}
	return err
}
```

## References

- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1342-L1391)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
- [Floating Ip Ensure](floating-ip-ensure.md)
- [Implementation Plan](../docs/implementation-plan.md)
