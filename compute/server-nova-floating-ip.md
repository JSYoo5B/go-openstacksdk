# Server Nova Floating Ip

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

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
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    id := flag.String("server-id", "", "existing Nova server ID")
    mode := flag.String("mode", "dispatch", "dispatch or list")
    pool := flag.String("pool", "", "literal Nova pool")
    ips := flag.String("ips", "", "comma-separated floating IPv4 addresses")
    fixed := flag.String("fixed", "", "fixed IPv4 for addFloatingIp")
    reuse := flag.Bool("reuse", true, "reuse a free IP for pool/auto")
    wait := flag.Bool("wait", false, "observe each address in raw Nova")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *id, *mode, *pool, *ips, *fixed, *reuse, *wait); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, id, mode, pool, ips, fixed string, reuse, wait bool) error {
    if id == "" || (mode != "dispatch" && mode != "list") {
        return fmt.Errorf("provide -server-id and -mode dispatch or list")
    }
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"),
        sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNova)),
    )
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
        return fmt.Errorf("server GET did not return %q", id)
    }
    addresses := []string{}
    if ips != "" {
        for _, address := range strings.Split(ips, ",") {
            addresses = append(addresses, strings.TrimSpace(address))
        }
    }
    destination := []network.EnsureFloatingIPOption{network.WithEnsureReuse(reuse)}
    if fixed != "" {
        destination = append(destination, network.WithEnsureFixedAddress(fixed))
    }
    automatic := []compute.AutomaticFloatingIPOption{
        compute.WithAutomaticEnsureOptions(destination...),
    }
    if pool != "" {
        automatic = append(automatic, compute.WithFloatingIPPool(resource.Name(pool)))
    }
    if len(addresses) != 0 {
        automatic = append(automatic, compute.WithFloatingIPAddresses(addresses...))
    }
    options := []compute.ServerIPOption{
        compute.WithServerIPAutomaticOptions(automatic...),
        compute.WithServerIPWait(wait),
    }
    var result *compute.AutomaticServerIPResult
    if mode == "list" {
        result, err = conn.AddIPList(ctx, server, addresses, options...)
    } else {
        result, err = conn.AddIPsToServer(ctx, compute.AutomaticFloatingIPRequest{Server: server}, options...)
    }
    if result != nil {
        fmt.Printf("mode=%s observed=%t\n", result.Mode, result.Observed)
        printAssignment("latest", result.NovaAssignment)
        for _, attempt := range result.Attempts {
            fmt.Printf("attempt=%d requested=%s completed=%t observed=%t error=%v\n",
                attempt.Index, attempt.RequestedAddress, attempt.Completed, attempt.Observed, attempt.Error)
            printAssignment("attempt", attempt.NovaAssignment)
        }
    }
    return err
}

func printAssignment(label string, assignment *compute.NovaFloatingIPAssignment) {
    if assignment == nil {
        return
    }
    fmt.Printf("%s reused=%t allocated=%t already_attached=%t action_accepted=%t\n",
        label, assignment.Reused, assignment.Allocated, assignment.AlreadyAttached, assignment.ActionAccepted)
    if ip := assignment.FloatingIP; ip != nil {
        fmt.Printf("ip=%s address=%s pool=%s fixed=%s instance=%s\n",
            ip.ID, ip.Address, ip.Pool, optional(ip.FixedAddress), optional(ip.InstanceID))
    }
    if response := assignment.ActionResponse; response != nil {
        fmt.Printf("action_status=%d\n", response.StatusCode)
    }
}

func optional(value *string) string {
    if value == nil {
        return "<null>"
    }
    return *value
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")  # configured floating_ip_source: nova
server = conn.get_server("web-01")
if server is None:
    raise RuntimeError("server not found")

# standalone defaults: timeout=60, wait=False, reuse=True
server = conn.add_ips_to_server(server, ip_pool="public", reuse=True)
server = conn.add_ip_list(
    server, ["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

## References

- [Server Ip Helpers](server-ip-helpers.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Server Ready](server-ready.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [API Microversion History](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html#microversion-2-36)
- [action reference](https://docs.openstack.org/api-ref/compute/#add-associate-floating-ip-addfloatingip-action-deprecated)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1284-L1525)
- [Nova available/create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L673-L712)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1080-L1212)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [AvailableFloatingIP](floating-ip-available.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Floating Ip Delete](floating-ip-delete.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
