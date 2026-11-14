# Server Ip Dispatch

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

ready = conn.wait_for_server(
    server,
    ip_pool="public",
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    reuse=True,
    timeout=120,
)

ready = conn.wait_for_server(
    server,
    ips=["198.51.100.10", "198.51.100.11"],
    auto_ip=False,
    timeout=120,
)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    serverID := flag.String("server-id", "", "existing Nova server ID")
    pool := flag.String("pool", "", "exact external network name")
    ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
    execute := flag.Bool("execute", false, "execute assignment and observation")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *serverID, *pool, *ips, *execute); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, pool, ips string, execute bool) error {
    if id == "" {
        return fmt.Errorf("-server-id is required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    server, err := service.Servers.Get(ctx, id)
    if err != nil {
        return err
    }
    opts := []compute.AutomaticFloatingIPOption{
        compute.WithAutomaticIPWait(resource.WithTimeout(2 * time.Minute)),
        compute.WithAutomaticEnsureOptions(network.WithEnsureWait(
            resource.WithPollInterval(time.Second))),
    }
    if ips != "" {
        addresses := strings.Split(ips, ",")
        for i := range addresses {
            addresses[i] = strings.TrimSpace(addresses[i])
        }
        opts = append(opts, compute.WithFloatingIPAddresses(addresses...))
    }
    if pool != "" {
        opts = append(opts, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    input := compute.AutomaticFloatingIPRequest{Server: server}
    if !execute {
        decision, err := conn.PlanServerFloatingIP(ctx, input, opts...)
        if decision != nil {
            encoder := json.NewEncoder(os.Stdout)
            encoder.SetIndent("", "  ")
            if printErr := encoder.Encode(decision); printErr != nil {
                return printErr
            }
        }
        return err
    }
    result, err := conn.EnsureServerFloatingIP(ctx, input, opts...)
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        if result.Server != nil {
            fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
        }
        for _, attempt := range result.Attempts {
            fmt.Printf("step=%d requested=%s completed=%t observed=%t error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed,
                attempt.Observed, attempt.Error)
            if attempt.Assignment != nil && attempt.Assignment.FloatingIP != nil {
                ip := attempt.Assignment.FloatingIP
                fmt.Printf("ip=%s address=%s port=%s\n", ip.ID, ip.FloatingIP, ip.PortID)
            }
        }
    }
    return err
}
```

## References

- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1527)
- [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392)
- [_add_ip_from_pool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1342)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1481)
- [Ensure](../network/floating-ip-ensure.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L671)
- [Attach](../network/floating-ip-attach.md)
- [legacy Nova backend](server-nova-floating-ip.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
