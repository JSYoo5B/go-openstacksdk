# Create With Floating Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.create_server(
    name="web-01",
    image="ubuntu",
    flavor="small",
    network="private",
    ip_pool="public",
    auto_ip=False,
    reuse_ips=True,
    wait=True,
    timeout=180,
)
print(server.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud("dev"),
        sdk.WithMicroversion(sdk.Compute, "2.37"),
    )
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    result, err := service.Servers.CreateWithFloatingIP(ctx,
        compute.CreateServerWithFloatingIPRequest{
            Server: compute.CreateServerRequest{
                Name:   "web-01",
                Image:  resource.Name("ubuntu"),
                Flavor: resource.Name("small"),
            },
            FloatingIPNetwork: resource.Name("public"),
        },
        compute.WithWorkflowTimeout(3*time.Minute),
        compute.WithServerOptions(
            compute.WithNetworks(resource.Name("private")),
            compute.WithWait(resource.WithPollInterval(time.Second)),
        ),
        compute.WithFloatingIPOptions(
            network.WithEnsureWait(resource.WithPollInterval(time.Second)),
        ),
    )
    if err != nil {
        if result != nil {
            if result.Server != nil {
                fmt.Fprintf(os.Stderr, "server=%s status=%s\n",
                    result.Server.ID, result.Server.Status)
            }
            if result.Assignment != nil && result.Assignment.FloatingIP != nil {
                assignment := result.Assignment
                fmt.Fprintf(os.Stderr, "floating-ip=%s address=%s reused=%t allocated=%t\n",
                    assignment.FloatingIP.ID, assignment.FloatingIP.FloatingIP,
                    assignment.Reused, assignment.Allocated)
            }
        }
        return err
    }
    fmt.Printf("server=%s floating-ip=%s\n",
        result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
    return nil
}
```

## References

- [Server Addresses](server-addresses.md)
- [Server Default Network](server-default-network.md)
- [Server Network Interfaces](server-network-interfaces.md)
- [Network Roles](../network/network-roles.md)
- [Connection Server Network Roles Test](../connectiontest/connection_server_network_roles_test.go)
- [Floating Ip Ensure](../network/floating-ip-ensure.md)
- [cloud create_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L915-L1245)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1606)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1709)
- [Network Mutations](../network/network-mutations.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Automatic Ip](server-automatic-ip.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
