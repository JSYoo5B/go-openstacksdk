# Subnets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
subnets = conn.network.subnets(
    network_id="network-a",
    is_dhcp_enabled=False,
    prefix_length=24,
    dns_nameservers=["192.0.2.53"],
    max_items=100,
)
for subnet in subnets:
    print(subnet.id, subnet.cidr)
```

```go
package examples

import (
    "context"
    "fmt"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListSubnets(ctx context.Context, service *networkv2.Service) error {
    for subnet, err := range service.Subnets.Resources.List(ctx,
        resource.WithFilters(map[string]any{
            "network_id": "network-a",
            "is_dhcp_enabled": false,
            "prefix_length": 24,
            "dns_nameservers": []string{"192.0.2.53"},
        }),
        resource.WithMaxItems(100),
    ) {
        if err != nil { return err }
        fmt.Println(subnet.ID, subnet.CIDR)
    }
    return nil
}
```

## References

- [Python Raw Max Items](python-raw-max-items.md)
- [AST manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)
- [SDKgen](../../../internal/cmd/sdkgen/README.md)
- [Listing](../../../docs/listing.md)
- [Finding Identities](../../../docs/finding-identities.md)
- [Network Subnet Semantic Filters Test](../../../api/network_subnet_semantic_filters_test.go)
- [Network Subnet Raw Body Filters Test](../../../api/network_subnet_raw_body_filters_test.go)
