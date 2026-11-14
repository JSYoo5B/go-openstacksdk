# Server Ip Helpers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Choosing a floating IP helper

`AddIPsToServer` with a pool or in automatic mode is idempotent: a server that already has a matching floating IP keeps it, and no further IP is allocated or attached. Use `AddIPFromPool` or `AddAutoIP` when you need Python's `_add_ip_from_pool` or `_add_auto_ip` behavior instead, including reuse of the first unattached pool IP, the Nova fallback after a Neutron NotFound, and, for `AddAutoIP`, deletion of an IP it created when the wait times out. An address list follows Python's `add_ip_list` steps in every helper.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, auto_ip=True, reuse=True
added = conn.add_ips_to_server(server, ip_pool="public", auto_ip=False)
added = conn.add_ip_list(server, ["198.51.100.10", "198.51.100.11"], wait=True)
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    id := flag.String("server-id", "", "existing Nova server ID")
    mode := flag.String("mode", "dispatch", "dispatch or list")
    pool := flag.String("pool", "", "exact external network name")
    ips := flag.String("ips", "", "comma-separated existing IPv4 addresses")
    wait := flag.Bool("wait", false, "observe each address in raw Nova")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *pool, *ips, *wait); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, mode, pool, ips string, wait bool) error {
    if id == "" || (mode != "dispatch" && mode != "list") {
        return fmt.Errorf("provide -server-id and -mode dispatch or list")
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
    if server == nil || server.ID != id {
        return fmt.Errorf("Nova returned a different or missing server")
    }
    var addresses []string
    if ips != "" {
        addresses = strings.Split(ips, ",")
        for i := range addresses {
            addresses[i] = strings.TrimSpace(addresses[i])
        }
    }
    var automatic []compute.AutomaticFloatingIPOption
    if len(addresses) != 0 {
        automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
    }
    if pool != "" {
        automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    options := []compute.ServerIPOption{
        compute.WithServerIPAutomaticOptions(automatic...),
        compute.WithServerIPWait(wait),
    }
    var result *compute.AutomaticServerIPResult
    if mode == "list" {
        result, err = conn.AddIPList(ctx, server, addresses, options...)
    } else {
        result, err = conn.AddIPsToServer(ctx,
            compute.AutomaticFloatingIPRequest{Server: server}, options...)
    }
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        if result.Assignment != nil && result.Assignment.FloatingIP != nil {
            ip := result.Assignment.FloatingIP
            fmt.Printf("ip=%s address=%s status=%s\n", ip.ID, ip.FloatingIP, ip.Status)
        }
        for _, attempt := range result.Attempts {
            // ShortCircuited: the server already showed this address.
            // AttachmentNotFound: Neutron returned 404 after selection and
            // the item continued, so no association is confirmed.
            fmt.Printf("step=%d requested=%s completed=%t short=%t observed=%t notfound=%v error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed,
                attempt.ShortCircuited, attempt.Observed,
                attempt.AttachmentNotFound, attempt.Error)
        }
    }
    return err
}
```

## References

- [Server Ip Dispatch](server-ip-dispatch.md)
- [Attach](../network/floating-ip-attach.md)
- [Ensure](../network/floating-ip-ensure.md)
- [Server Ready](server-ready.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [add_ips_to_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1489-L1525)
- [add_ip_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1343-L1392)
- [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1163)
- [legacy Nova backend](server-nova-floating-ip.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
