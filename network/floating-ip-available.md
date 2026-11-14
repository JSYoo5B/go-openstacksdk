# Floating Ip Available

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    candidates := flag.String("networks", "public", "ordered network names")
    server := flag.String("server-id", "", "optional existing server ID")
    project := flag.String("project", "", "optional reuse filter only")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *candidates, *server, *project); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, candidates, serverID, project string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Network(ctx)
    if err != nil {
        return err
    }
    input := network.AvailableFloatingIPRequest{}
    if candidates != "" {
        for _, name := range strings.Split(candidates, ",") {
            input.Networks = append(input.Networks, resource.Name(strings.TrimSpace(name)))
        }
    }
    if serverID != "" {
        input.Server = resource.ID(serverID)
    }
    options := []network.AvailableFloatingIPOption{
        network.WithAvailableTimeout(time.Minute),
    }
    if project != "" {
        options = append(options, network.WithAvailableProject(project))
    }
    result, err := service.FloatingIPs.Available(ctx, input, options...)
    if result != nil {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if encodeErr := encoder.Encode(result); encodeErr != nil {
            return errors.Join(err, encodeErr)
        }
    }
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")  # floating_ip_source: neutron
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
```

## References

- [Connection·Compute Available](../compute/floating-ip-available.md)
- [Ensure](floating-ip-ensure.md)
- [Attach](floating-ip-attach.md)
- [Network Roles](network-roles.md)
- [Ensure/PrepareEnsure](floating-ip-plan.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773)
- [Neutron create·association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L850-L949)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1608-L1710)
- [normalizer](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
