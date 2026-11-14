# Instanceactions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.ComputeV2(ctx)
if err != nil { return err }
scope, err := service.InstanceActions.InServer(ctx, resource.Name("worker"))
if err != nil { return err }
for action, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(scope.ServerID(), action.RequestID, action.Action, action.UpdatedAt)
}
action, err := scope.Find(ctx, resource.ID("req-example"), resource.WithIgnoreMissing())
if err != nil { return err }
if action != nil && action.Events != nil {
    for _, event := range *action.Events {
        fmt.Println(event.Event, event.Result, event.Details)
    }
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListServerActions(ctx context.Context, conn *sdk.Connection, serverID string) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	scope, err := service.InstanceActions.InServer(ctx, resource.ID(serverID))
	if err != nil {
		return err
	}
	for action, err := range scope.List(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ServerID, action.RequestID, action.Action)
	}
	return nil
}
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#list-actions-for-server)
- [Nova API version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)
- [action detail API](https://docs.openstack.org/api-ref/compute/#show-server-action-details)
- [pinned ServerAction](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)
- [Resources](resources.go)
- [Instance Actions Scope Test](../../../api/instance_actions_scope_test.go)
- [Listing](../../../docs/listing.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/instanceactions/request.go)
