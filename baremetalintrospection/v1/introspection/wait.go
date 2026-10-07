package introspection

import (
	"context"
	"fmt"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// IntrospectionFailureError retains the service's failure message and the last
// response. It unwraps to resource.ErrFailedState for the shared SDK policy.
type IntrospectionFailureError struct {
	ID, State, Message string
	Details            *Introspection
}

func (e *IntrospectionFailureError) Error() string {
	return fmt.Sprintf("introspection %q failed in state %q: %s", e.ID, e.State, e.Message)
}

func (e *IntrospectionFailureError) Unwrap() error { return resource.ErrFailedState }

// WaitUntilFinished polls the same introspection UUID until Finished is true.
// It shares the SDK wait defaults and options, preserving HTTP, cancellation
// and timeout errors. A service Error always fails, even if Finished is true.
// Introspections have no names; explicit Name references are unsupported.
func (a *API) WaitUntilFinished(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) (*Introspection, error) {
	if err := resource.ValidateWaitOptions(options...); err != nil {
		return nil, request.Wrap("WaitUntilFinished", "introspection", err)
	}
	id, err := a.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("WaitUntilFinished", "introspection", err)
	}
	// Progress labels exist only inside this waiter. The native model has no
	// Status field, and the public Resources collection does not acquire one.
	waiter := resource.NewCollection(resource.Adapter[Introspection]{
		Kind:            "introspection",
		FixedWaitStatus: true,
		Get: func(ctx context.Context, _ string) (*Introspection, error) {
			value, err := a.GetIntrospectionStatus(ctx, id)
			if err != nil {
				return nil, err
			}
			if value.Error != "" || strings.EqualFold(value.State, "error") {
				message := value.Error
				if message == "" {
					message = "introspection reported an error state"
				}
				return nil, &IntrospectionFailureError{ID: id, State: value.State, Message: message, Details: value}
			}
			return value, nil
		},
		// The caller's resolved UUID remains authoritative even if a later
		// response omits or changes its UUID; a response never redirects polls.
		ID: func(*Introspection) string { return id },
		Status: func(value *Introspection) string {
			if value.Finished {
				return "finished"
			}
			return "pending"
		},
	})
	value, err := waiter.Wait(ctx, resource.ID(id), "finished", options...)
	return value, request.Wrap("WaitUntilFinished", "introspection", err)
}
