// Package servicewait supplies the SDK-owned Compute, Cinder and Glance wait
// defaults without changing shared collection or native wait APIs.
package servicewait

import (
	"context"
	"fmt"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func validate[T any](ctx context.Context, collection *resource.Collection[T]) error {
	if ctx == nil {
		return fmt.Errorf("%w: wait context is required", resource.ErrInvalidOption)
	}
	if collection == nil {
		return fmt.Errorf("%w: wait collection is required", resource.ErrUnsupported)
	}
	return nil
}

// State requires a target and uses one exact, case-insensitive failure state,
// a two-second interval and no SDK timeout. Caller options apply last.
func State[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, target, failure string, options ...resource.WaitOption) (*T, error) {
	if err := validate(ctx, collection); err != nil {
		return nil, err
	}
	policy := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(2 * time.Second), resource.WithFailureStates(failure)}
	return collection.Wait(ctx, ref, target, append(policy, options...)...)
}

// Server uses Compute's ERROR failure and a 120-second SDK timeout.
func Server[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, target string, options ...resource.WaitOption) (*T, error) {
	if err := validate(ctx, collection); err != nil {
		return nil, err
	}
	policy := []resource.WaitOption{resource.WithTimeout(120 * time.Second), resource.WithPollInterval(2 * time.Second), resource.WithFailureStates("ERROR")}
	return collection.Wait(ctx, ref, target, append(policy, options...)...)
}

// Delete observes absence with a two-second interval and a 120-second timeout.
// It sends GET/list requests only; caller options apply last.
func Delete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	if err := validate(ctx, collection); err != nil {
		return err
	}
	policy := []resource.WaitOption{resource.WithTimeout(120 * time.Second), resource.WithPollInterval(2 * time.Second), resource.WithFailureStates()}
	return collection.WaitDeleted(ctx, ref, append(policy, options...)...)
}
