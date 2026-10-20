package masakari

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus applies the instance-ha proxy defaults. Explicit caller
// options take precedence; cancellation always follows the caller context.
func WaitForStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, status string, options ...resource.WaitOption) (*T, error) {
	defaults := []resource.WaitOption{
		resource.WithUnlimitedWait(),
		resource.WithFailureStates("ERROR"),
		resource.WithStatusAttribute("status"),
	}
	return collection.Wait(ctx, ref, status, append(defaults, options...)...)
}

// WaitForDelete polls existing GET capabilities, independently of whether
// submitting DELETE is supported. Models without status complete on HTTP 404.
func WaitForDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, hasStatus bool, options ...resource.WaitOption) error {
	defaults := []resource.WaitOption{resource.WithTimeout(120 * time.Second)}
	if hasStatus {
		defaults = append(defaults, resource.WithStatusAttribute("status"))
	}
	return collection.WaitDeleted(ctx, ref, append(defaults, options...)...)
}
