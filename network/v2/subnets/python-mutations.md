# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.network.create_subnet(network_id=network_id, ip_version=4, cidr="10.61.0.0/24", is_dhcp_enabled=False)
conn.network.update_subnet(subnet_id, gateway_ip=None, is_dhcp_enabled=False, if_revision=0)
conn.network.update_subnet(resource_id)
conn.network.update_subnet(resource_id, if_revision=0)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.Subnets
    if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
        options := []subnets.AttributeOption{subnets.WithAttribute("gateway_ip", nil), subnets.WithAttribute("is_dhcp_enabled", false)}
        if value := os.Getenv("NEUTRON_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, subnets.WithRevision(revision))
        }
        value, err := api.UpdateWithAttributes(ctx, id, options...)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    if os.Getenv("NEUTRON_CREATE") == "1" {
        value, err := api.CreateWithAttributes(ctx, subnets.WithAttribute("network_id", os.Getenv("NETWORK_ID")), subnets.WithAttribute("ip_version", 4), subnets.WithAttribute("cidr", "10.61.0.0/24"), subnets.WithAttribute("is_dhcp_enabled", false))
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
```
