package events

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// WaitForStatus polls with the shared Senlin defaults: no deadline and ERROR
// as a failure state. Caller context and wait options can override defaults.
func (a *API) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Event, error) {
	return senlin.WaitForStatus(ctx, rest.Collection(spec(a.RawClient())), ref, status, options...)
}

// WaitForDelete polls for absence with a 120-second default timeout. It does
// not initiate deletion; caller context and wait options retain control.
func (a *API) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return senlin.WaitForDelete(ctx, rest.Collection(spec(a.RawClient())), ref, true, options...)
}
