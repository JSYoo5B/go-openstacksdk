# Server Network Interfaces

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.42")
port = conn.network.find_port("app-port", ignore_missing=False)
server = conn.create_server(
    name="web-01", image="ubuntu", flavor="small",
    nics=[
        {"net-name": "private", "fixed_ip": "10.0.0.10", "tag": "app"},
        {"port-id": port.id, "tag": "existing-port"},
    ],
    auto_ip=False, wait=False,
)
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.42"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name("ubuntu"), Flavor: resource.Name("small"),
	}, compute.WithNetworkInterfaces(
		compute.ServerNetworkInterface{Network: resource.Name("private"), FixedIP: "10.0.0.10", Tag: request.Present("app")},
		compute.ServerNetworkInterface{Port: resource.Name("app-port"), Tag: request.Present("existing-port")},
	))
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
```

## References

- [Reference](README.md)
- [Server Default Network](server-default-network.md)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [WithMicroversionRange/WithLatestMicroversion](../docs/microversions.md)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1074-L1154)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L242-L312)
- [Network Roles](../network/network-roles.md)
- [Create With Floating Ip](create-with-floating-ip.md)
- [Network Mutations](../network/network-mutations.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
