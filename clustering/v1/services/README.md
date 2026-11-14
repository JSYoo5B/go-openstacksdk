# Services

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
services = conn.clustering.services()
```

```go
// Configure before sharing the client between goroutines.
client.Type = "clustering"
client.Microversion = "1.7"
api := services.New(client)

for service, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(service.ID, service.Host, service.Status, service.State)
}
```

```go
listingAPI := services.New(client)
values, err := listingAPI.All(ctx,
    services.WithListOptions(services.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterServices(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.Services.All(ctx,
		services.WithListFilter("state", "up"),
		services.WithListFilter("disabled_reason", nil),
		request.MaxItems(50), request.Paginated(false))
	if err != nil {
		return err
	}
	for _, value := range values {
		fmt.Println(value.ID, value.Host, value.State)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#services-services)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
