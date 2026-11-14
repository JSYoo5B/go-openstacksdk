# Policytypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := policytypes.New(client)
policy, err := api.Get(ctx, "senlin.policy.scaling-1.0")
if err != nil {
    return err
}
fmt.Println(policy.Name, policy.Schema)

for policy, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(policy.Name)
}
```

```go
listingAPI := policytypes.New(client)
values, err := listingAPI.All(ctx,
    policytypes.WithListOptions(policytypes.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.Name)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterPolicyTypes(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.PolicyTypes.All(ctx,
		policytypes.WithListFilter("name", typeName),
		request.MaxItems(50), request.Paginated(false))
	if err != nil {
		return err
	}
	for _, policy := range values {
		fmt.Println(policy.Name, policy.Version)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#policy-types-policy-types)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
