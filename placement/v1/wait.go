package v1

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus polls any SDK Placement collection with the placement Proxy
// wait_for_status defaults: exact case-insensitive ERROR failure, a two-second
// interval and no SDK timeout. No Placement model declares a status
// attribute, so resource.WithStatusAttribute must select an exported string
// field; otherwise the wait fails before HTTP, as Python raises
// AttributeError. Caller options apply last.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.ProxyStatus(ctx, collection, ref, status, options...)
}

// WaitForDelete observes deletion with the placement Proxy wait_for_delete
// defaults: a two-second interval and 120 seconds. It sends GET requests only
// and completes on a 404 or a successful nil observation. Caller options
// apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.ProxyDelete(ctx, collection, ref, options...)
}
