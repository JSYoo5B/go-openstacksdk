package compute

import (
	"context"

	"gophercloudsdk/internal/servicewait"
	"gophercloudsdk/resource"
)

// WaitForState polls an SDK collection with Compute's ERROR failure default,
// a two-second interval and no SDK timeout. Caller options apply last.
func WaitForState[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.State(ctx, collection, ref, status, "ERROR", options...)
}

// WaitForDelete observes deletion for up to 120 seconds by default. It does not
// send a delete request. Caller options apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.Delete(ctx, collection, ref, options...)
}

func (s *Servers) waitCollection() *resource.Collection[Server] {
	if s == nil {
		return nil
	}
	return s.Collection
}

// WaitForState uses the generic Compute status defaults and requires a target.
func (s *Servers) WaitForState(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Server, error) {
	return WaitForState(ctx, s.waitCollection(), ref, status, options...)
}

// WaitForServer waits for ACTIVE, failing on ERROR, with a 120-second timeout.
// Use WaitForServerState to select another target; caller options apply last.
func (s *Servers) WaitForServer(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) (*Server, error) {
	return s.WaitForServerState(ctx, ref, "ACTIVE", options...)
}

// WaitForServerState uses the server-specific 120-second timeout and ERROR
// failure default with an explicit target. Caller options apply last.
func (s *Servers) WaitForServerState(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Server, error) {
	return servicewait.Server(ctx, s.waitCollection(), ref, status, options...)
}

// WaitForDelete observes deletion using the Compute proxy's 120-second default.
func (s *Servers) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return WaitForDelete(ctx, s.waitCollection(), ref, options...)
}
