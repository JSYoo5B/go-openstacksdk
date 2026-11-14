# Trunk Address Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
trunk = conn.network.add_trunk_subports(
    trunk_id,
    [{"port_id": child_port_id, "segmentation_type": "vlan", "segmentation_id": 100}],
)
trunk = conn.network.delete_trunk_subports(trunk, [{"port_id": child_port_id}])
group = conn.network.add_addresses_to_address_group(group_id, ["192.0.2.0/24"])
group = conn.network.remove_addresses_from_address_group(group, ["192.0.2.0/24"])
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trunkID := flag.String("trunk", "", "trunk ID")
    addPort := flag.String("add-subport", "", "subport to add")
    removePort := flag.String("remove-subport", "", "subport to remove")
    segmentationID := flag.Int("segmentation-id", 100, "VLAN segmentation ID")
    groupID := flag.String("group", "", "address group ID")
    addAddress := flag.String("add-address", "", "address or CIDR to add")
    removeAddress := flag.String("remove-address", "", "address or CIDR to remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var trunk *trunks.TrunkActionRecord
    if *addPort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        trunk, err = service.Trunks.AddSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *addPort, "segmentation_type": "vlan",
                "segmentation_id": *segmentationID}})
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    if *removePort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        var options []trunks.SubportActionOption
        if trunk != nil {
            options = append(options, trunks.WithSubportActionSeed(trunk.Resource))
        }
        trunk, err = service.Trunks.RemoveSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *removePort}}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    var group *addressgroups.AddressActionRecord
    if *addAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        group, err = service.SecurityAddressGroups.AddAddressesRecord(ctx,
            *groupID, []string{*addAddress})
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
    if *removeAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        var options []addressgroups.AddressActionOption
        if group != nil {
            options = append(options, addressgroups.WithAddressActionSeed(group.Resource))
        }
        group, err = service.SecurityAddressGroups.RemoveAddressesRecord(ctx,
            *groupID, []string{*removeAddress}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
}
```

## References

- [Install](../../docs/install.md)
- [Neutron 27.0.0 trunk policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/trunk.py)
- [address group policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
- [Policy](https://docs.openstack.org/neutron/2026.2/configuration/policy.html)
