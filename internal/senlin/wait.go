package senlin

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus applies Senlin's proxy defaults without changing Collection's
// common five-minute policy. Caller options are applied after these defaults.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	defaults := []resource.WaitOption{
		resource.WithUnlimitedWait(),
		resource.WithFailureStates("ERROR"),
		resource.WithStatusAttribute("status"),
	}
	return collection.Wait(ctx, ref, status, append(defaults, options...)...)
}

// WaitForDelete polls GET, without submitting DELETE. Only models which declare
// status use it by default; other models can complete through absence alone.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, hasStatus bool, options ...resource.WaitOption) error {
	defaults := []resource.WaitOption{resource.WithTimeout(120 * time.Second)}
	if hasStatus {
		defaults = append(defaults, resource.WithStatusAttribute("status"))
	}
	return collection.WaitDeleted(ctx, ref, append(defaults, options...)...)
}
