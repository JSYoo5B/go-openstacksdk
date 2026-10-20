# Instanceactions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/instanceactions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.WithListQuery`

- List GETs servers/{id}/os-instance-actions with limit, marker, RFC3339 changes-since/before keeping the caller offset and extension query as one single page ignoring links, decodes zone-less start_time, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors without operation context.
  Tests: `compute/v2/instanceactions/native_contracts_test.go:TestNativeInstanceActionsRoutesQueryAndDecode`, `compute/v2/instanceactions/native_contracts_test.go:TestNativeInstanceActionsStatusesAndDecodeErrors`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/instanceactions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/instanceactions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions.API.Get`

- Get GETs servers/{id}/os-instance-actions/{requestID}, decodes zone-less start/updated/event times (null to zero or nil) and events, rejects zoned timestamps as a decode error and accepts only 200.
  Tests: `compute/v2/instanceactions/native_contracts_test.go:TestNativeInstanceActionsRoutesQueryAndDecode`, `compute/v2/instanceactions/native_contracts_test.go:TestNativeInstanceActionsStatusesAndDecodeErrors`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/instanceactions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

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
