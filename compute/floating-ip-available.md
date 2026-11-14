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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    source := flag.String("source", "", "cloud default, neutron, nova or none")
    pool := flag.String("network", "public", "network name or literal Nova pool")
    server := flag.String("server-id", "", "optional server for new Neutron allocation")
    project := flag.String("project", "", "optional Neutron reuse filter only")
    strict := flag.Bool("strict", false, "omit Nova view compatibility aliases")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *pool, *server, *project, *strict); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, pool, serverID, project string, strict bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil {
        return err
    }
    input := compute.AvailableFloatingIPRequest{}
    if pool != "" {
        input.Networks = []resource.Ref{resource.Name(pool)}
    }
    if serverID != "" {
        input.Server = resource.ID(serverID)
    }
    options := []compute.AvailableFloatingIPOption{
        compute.WithAvailableIPTimeout(time.Minute),
        compute.WithAvailableIPStrict(strict),
    }
    if source != "" {
        options = append(options, compute.WithAvailableIPSource(compute.FloatingIPSource(source)))
    }
    if project != "" {
        options = append(options, compute.WithAvailableIPNetworkOptions(
            network.WithAvailableProject(project)))
    }
    result, err := conn.AvailableFloatingIP(ctx, input, options...)
    if result != nil {
        summary := map[string]any{
            "backend": result.Backend, "id": result.ID, "address": result.Address,
            "reused": result.Reused, "allocated": result.Allocated,
        }
        if record := result.FloatingIP; record != nil {
            summary["normalization_source"] = record.NormalizationSource
            summary["normalized"] = record.Normalized
            if record.Resource != nil {
                summary["resource"] = record.Resource.Body
            }
            if record.Wire != nil {
                summary["wire"] = record.Wire.Body
            }
        }
        if result.FallbackError != nil {
            summary["fallback_error"] = result.FallbackError.Error()
        }
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if encodeErr := encoder.Encode(summary); encodeErr != nil {
            return errors.Join(err, encodeErr)
        }
    }
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
```

## References

- [IP helper](server-ip-helpers.md)
- [Ensure·readiness](server-ip-dispatch.md)
- [Floating Ip Available](../network/floating-ip-available.md)
- [Floating Ip Create](floating-ip-create.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773)
- [Nova allocation+compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [normalization](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Delete](floating-ip-delete.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Connection Floating Ip Available Nova Test](../connectiontest/connection_floating_ip_available_nova_test.go)
- [Connection Floating Ip Available Neutron Test](../connectiontest/connection_floating_ip_available_neutron_test.go)
- [Floating Ip Allocate Test](../network/floating_ip_allocate_test.go)
