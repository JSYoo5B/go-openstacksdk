# Python Resource Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
ag = conn.network.create_address_group(name="web", addresses=[])
scope = conn.network.create_address_scope(name="scope", ip_version=4, is_shared=False)
pool = conn.network.create_subnet_pool(
    name="pool", prefixes=["10.60.0.0/16"], default_prefix_length=24,
    minimum_prefix_length=16, maximum_prefix_length=28, is_shared=False,
)
trunk = conn.network.create_trunk(port_id=parent_port_id, is_admin_state_up=False, sub_ports=[])
policy = conn.network.create_qos_policy(name="gold", is_shared=False, is_default=False)

unchanged = conn.network.update_address_group(ag.id)
conn.network.update_address_group(ag.id, description="")
conn.network.update_address_scope(scope.id, is_shared=False)
conn.network.delete_address_group(ag.id)
conn.network.delete_address_scope(scope.id, ignore_missing=False)
```

```go
err := api.DeleteWithOptions(ctx, id)
err = api.DeleteWithOptions(ctx, id, request.IgnoreMissing(false))
strict := false
option := addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    kind := os.Getenv("NEUTRON_MUTATION_KIND")
    deleteID, updateID := os.Getenv("NEUTRON_DELETE_ID"), os.Getenv("NEUTRON_UPDATE_ID")
    switch kind {
    case "address-group":
        api := service.API.SecurityAddressGroups
        value, err := api.CreateWithAttributes(ctx, addressgroups.WithAttribute("name", "web"), addressgroups.WithAttribute("addresses", []string{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressgroups.WithAttribute("description", "")); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "address-scope":
        api := service.API.AddressScopes
        value, err := api.CreateWithAttributes(ctx, addressscopes.WithAttribute("name", "scope"), addressscopes.WithAttribute("ip_version", 4), addressscopes.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if updateID != "" {
            if _, err := api.UpdateWithAttributes(ctx, updateID, addressscopes.WithAttribute("is_shared", false)); err != nil { panic(err) }
        }
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID, request.IgnoreMissing(false)); err != nil { panic(err) }
        }
    case "subnet-pool":
        api := service.API.SubnetPools
        value, err := api.CreateWithAttributes(ctx, subnetpools.WithAttribute("name", "pool"), subnetpools.WithAttribute("prefixes", []string{"10.60.0.0/16"}), subnetpools.WithAttribute("default_prefix_length", 24), subnetpools.WithAttribute("minimum_prefix_length", 16), subnetpools.WithAttribute("maximum_prefix_length", 28), subnetpools.WithAttribute("is_shared", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "trunk":
        api := service.API.Trunks
        value, err := api.CreateWithAttributes(ctx, trunks.WithAttribute("port_id", os.Getenv("PARENT_PORT_ID")), trunks.WithAttribute("is_admin_state_up", false), trunks.WithAttribute("sub_ports", []map[string]any{}))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    case "qos-policy":
        api := service.API.QoSPolicies
        value, err := api.CreateWithAttributes(ctx, policies.WithAttribute("name", "gold"), policies.WithAttribute("is_shared", false), policies.WithAttribute("is_default", false))
        if err != nil { panic(err) }
        fmt.Println(value.ID)
        if deleteID != "" {
            if err := api.DeleteWithOptions(ctx, deleteID); err != nil { panic(err) }
        }
    default:
        fmt.Println("set NEUTRON_MUTATION_KIND to address-group, address-scope, subnet-pool, trunk or qos-policy")
    }
}
```

## References

- [Python Read Delete](python-read-delete.md)
