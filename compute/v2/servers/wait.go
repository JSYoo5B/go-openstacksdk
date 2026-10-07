package servers

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/servicewait"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func (a *API) waitResources() *resource.Collection[Server] {
	if a == nil {
		return nil
	}
	return a.Resources
}

// WaitForState observes an explicit target with a two-second polling interval,
// the service's exact ERROR failure state, and no SDK timeout. Options apply last.
func (a *API) WaitForState(ctx context.Context, ref resource.Ref, target string, options ...resource.WaitOption) (*Server, error) {
	return servicewait.State(ctx, a.waitResources(), ref, target, "ERROR", options...)
}

// WaitForDelete observes absence with a two-second interval and a 120-second
// SDK timeout. It sends no DELETE request. Options apply last.
func (a *API) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.Delete(ctx, a.waitResources(), ref, options...)
}

// WaitForServer waits for ACTIVE with ERROR failure and a 120-second timeout.
func (a *API) WaitForServer(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) (*Server, error) {
	return servicewait.Server(ctx, a.waitResources(), ref, "ACTIVE", options...)
}

// WaitForServerState waits for an explicit target with Compute's 120-second timeout.
func (a *API) WaitForServerState(ctx context.Context, ref resource.Ref, target string, options ...resource.WaitOption) (*Server, error) {
	return servicewait.Server(ctx, a.waitResources(), ref, target, options...)
}
