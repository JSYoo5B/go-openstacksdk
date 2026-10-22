package network

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus polls any SDK Neutron collection with the network Proxy
// wait_for_status defaults: the status attribute when the model declares one,
// exact case-insensitive ERROR failure, a two-second interval and no SDK
// timeout. Models without a status attribute require
// resource.WithStatusAttribute. Caller options apply last.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.ProxyStatus(ctx, collection, ref, status, options...)
}

// WaitForDelete observes deletion with the network Proxy wait_for_delete
// defaults: a two-second interval and 120 seconds. It sends GET requests only;
// a 404, a successful nil observation or a "deleted" status completes it.
// Caller options apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.ProxyDelete(ctx, collection, ref, options...)
}
