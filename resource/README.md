# Resource

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
resource.ID("server-id")
resource.Name("web-01")
resource.ID(server.ID)
```

```go
for server, err := range service.Servers.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(server.ID)
    if shouldStop { break }
}
```

```go
node, err := service.Nodes.Resources.Wait(ctx, resource.ID("node-uuid"), "power on",
    resource.WithStatusAttribute("power_state"),
    resource.WithFailureStates("error"),
    resource.WithUnlimitedWait(),
    resource.WithProgressCallback(func(progress int) error { fmt.Println(progress); return nil }))
if err != nil { return err }
_ = node
```

```go
package example

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func PrintFirstPage[T any](ctx context.Context, collection *resource.Collection[T]) error {
    for value, err := range collection.List(ctx,
        resource.WithMaxItems(5), resource.WithPaginated(false)) {
        if err != nil { return err }
        fmt.Println(value)
    }
    return nil
}
```

## References

- [Finding Identities](../docs/finding-identities.md)
- [Subnets](../network/v2/subnets/README.md)
- [Listing](../keymanager/v1/secrets/listing/README.md)
- [Listing](../keymanager/v1/containers/listing/README.md)
- [Listing](../keymanager/v1/orders/listing/README.md)
- [Listing](../network/v2/extensions/security/addressgroups/listing/README.md)
- [Listing](../network/v2/extensions/qos/policies/listing/README.md)
- [Listing](../network/v2/extensions/subnetpools/listing/README.md)
- [Listing](../network/v2/networks/listing/README.md)
- [Listing](../network/v2/extensions/layer3/routers/listing/README.md)
- [Listing](../network/v2/extensions/security/groups/listing/README.md)
- [Listing](../network/v2/extensions/trunks/listing/README.md)
- [Listing](../docs/listing.md)
- [Listing](../clustering/v1/listing/README.md)
- [Scoped Resources](../docs/scoped-resources.md)
- [collection_test.go](collection_test.go)
- [Collections Test](../connectiontest/collections_test.go)
