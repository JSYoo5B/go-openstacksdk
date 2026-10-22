package v1

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus polls any SDK Inspector collection, such as
// Introspection.Resources or rules.API.Resources, with the inspector Proxy
// wait_for_status defaults: exact case-insensitive ERROR failure, a two-second
// interval and no SDK timeout. Neither Introspection nor Rule declares a status
// attribute, so resource.WithStatusAttribute must select an exported string
// field such as "state"; otherwise the wait fails before HTTP, as Python
// raises AttributeError. Use introspection.API.WaitUntilFinished for the
// finished-flag waiter. Caller options apply last.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.ProxyStatus(ctx, collection, ref, status, options...)
}

// WaitForDelete observes absence with the inspector Proxy wait_for_delete
// defaults: a two-second interval and 120 seconds. It sends GET requests only,
// never DELETE or abort, and completes on a 404 or a successful nil
// observation. The inspector keeps an aborted or finished introspection
// readable, so such a status usually reaches the deadline, as in Python.
// Caller options apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.ProxyDelete(ctx, collection, ref, options...)
}
