package image

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/servicewait"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// WaitForState polls an SDK collection with Glance's exact ERROR failure
// default, a two-second interval and no SDK timeout. Caller options apply last.
func WaitForState[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	return servicewait.State(ctx, collection, ref, status, "ERROR", options...)
}

// WaitForDelete observes deletion for up to 120 seconds by default. It does not
// send a delete request. Caller options apply last.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	return servicewait.Delete(ctx, collection, ref, options...)
}
