# Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
action = conn.clustering.get_action(action_id)
actions = conn.clustering.actions(target_id=node_id, limit=20)
```

```go
api := actions.New(client)
action, err := api.Get(ctx, actionID)
if err != nil {
    return err
}
fmt.Println(action.ID, action.Status)

for action, err := range api.List(ctx,
    actions.WithListOptions(actions.ListOpts{TargetID: nodeID, Limit: 20}),
) {
    if err != nil {
        return err
    }
    fmt.Println(action.ID, action.Status)
}
```

```python
action = conn.clustering.update_action("ACTION_ID", status="CANCELLED")
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
accepted, err := service.Actions.Cancel(ctx, resource.ID("ACTION_ID"),
    request.Custom(func(c *request.Config[actions.UpdateOpts]) error { v := false; c.Options.Force = &v; return nil }))
if err != nil { return err }
fmt.Println(accepted.StatusCode, string(accepted.Body))
action, err := service.Actions.Get(ctx, "ACTION_ID")
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```go
listingAPI := actions.New(client)
values, err := listingAPI.All(ctx,
    actions.WithListOptions(actions.ListOpts{Limit: 20}),
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
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterActions(ctx context.Context, conn *sdk.Connection, ownerID string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for action, err := range service.Actions.List(ctx,
		actions.WithListFilter("owner_id", ownerID),
		actions.WithListFilter("inputs", map[string]any{"region": "region-one"}),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ID, action.Status)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#actions)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
- [tracked handle](../tracking/async/README.md)
