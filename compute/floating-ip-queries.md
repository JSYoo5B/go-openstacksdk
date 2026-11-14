# Floating Ip Queries

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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    source := flag.String("source", "", "cloud default, neutron, nova or none")
    id := flag.String("id", "", "floating IP identifier; empty lists IPs")
    filters := flag.String("filters", "", "JSON query or local filter")
    direct := flag.Bool("direct", false, "use direct GET for a UUID-shaped ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *id, *filters, *direct); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, id, filters string, direct bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil {
        return err
    }
    options := []compute.FloatingIPQueryOption{
        compute.WithFloatingIPQueryTimeout(time.Minute),
        func(c *compute.FloatingIPQueryOpts) error { c.DirectGet = direct; return nil },
    }
    if source != "" {
        options = append(options,
            func(c *compute.FloatingIPQueryOpts) error { v := compute.FloatingIPSource(source); c.Source = &v; return nil })
    }
    if filters != "" {
        options = append(options,
            compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
    }
    var value json.RawMessage
    var pages int
    if id == "" {
        result, queryErr := conn.ListFloatingIPs(ctx, options...)
        err = queryErr
        if result != nil {
            value, pages = result.Value, len(result.Pages)
        }
    } else {
        result, queryErr := conn.GetFloatingIP(ctx,
            compute.GetFloatingIPRequest{ID: id}, options...)
        err = queryErr
        if result != nil {
            value, pages = result.Value, len(result.Pages)
        }
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    encodeErr := encoder.Encode(map[string]any{"value": value, "pages": pages})
    return errors.Join(err, encodeErr)
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
all_ips = conn.list_floating_ips()
selected = conn.search_floating_ips(id="IP_ID")
ip = conn.get_floating_ip("IP_ID")
direct = conn.get_floating_ip_by_id("IP_ID")
pools = conn.search_floating_ip_pools(name="public*")
```

## References

- [Available](floating-ip-available.md)
- [IP helper](server-ip-helpers.md)
- [Nova model](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L435-L606)
- [ Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L201)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217-L2358)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1766-L1830)
- [Floating Ip Delete](floating-ip-delete.md)
- [CreateFloatingIP](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
