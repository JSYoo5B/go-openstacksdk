package snapshots

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (a *API) waitResources() *resource.Collection[Snapshot] {
	if a == nil {
		return nil
	}
	return a.Resources
}

// WaitForState observes an explicit target with a two-second polling interval,
// the service's exact error failure state, and no SDK timeout. Options apply last.
func (a *API) WaitForState(ctx context.Context, ref resource.Ref, target string, options ...resource.WaitOption) (*Snapshot, error) {
	return servicewait.State(ctx, a.waitResources(), ref, target, "error", options...)
}

// WaitForDelete observes absence with a two-second interval and a 120-second
// SDK timeout. It sends no DELETE request. Options apply last.
func (a *API) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.Delete(ctx, a.waitResources(), ref, options...)
}

// WaitForAvailable uses the Cinder available target, error failure and unlimited
// SDK wait. Options can select a timeout or a different failure-state list.
func (a *API) WaitForAvailable(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) (*Snapshot, error) {
	return servicewait.State(ctx, a.waitResources(), ref, "available", "error", options...)
}
