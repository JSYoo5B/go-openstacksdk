package notifications

import (
	"context"

	"gophercloudsdk/internal/masakari"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// WaitForStatus polls the notification UUID with the instance-ha defaults.
func (a *API) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Notification, error) {
	return masakari.WaitForStatus(ctx, rest.Collection(a.spec), ref, status, options...)
}

// WaitForDelete waits for absence or deleted status without submitting DELETE.
func (a *API) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return masakari.WaitForDelete(ctx, rest.Collection(a.spec), ref, true, options...)
}
