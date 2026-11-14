# Profiletypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := profiletypes.New(client)
profile, err := api.Get(ctx, "os.nova.server-1.0")
if err != nil {
    return err
}
fmt.Println(profile.Name, profile.Schema)

for profile, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(profile.Name, profile.Version)
}

// Configure the selected version before concurrent client use.
client.Microversion = "1.4"
ops, err := api.Operations(ctx, "os.nova.server-1.0")
```

```go
values, err := api.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    request.Query("vendor_filter", "enabled"),
)
```

```go
listingAPI := profiletypes.New(client)
values, err := listingAPI.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
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
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterProfileTypes(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for profile, err := range service.ProfileTypes.List(ctx,
		profiletypes.WithListFilter("name", typeName),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(profile.Name, profile.Version)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#profile-types-profile-types)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
