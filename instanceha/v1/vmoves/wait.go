package vmoves

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/masakari"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// WaitForStatus polls the move UUID under the fixed notification at >=1.3.
func (s *NotificationScope) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*VMove, error) {
	return masakari.WaitForStatus(ctx, rest.Collection(s.spec), ref, status, options...)
}

// WaitForDelete observes disappearance or deleted status without a DELETE API.
func (s *NotificationScope) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return masakari.WaitForDelete(ctx, rest.Collection(s.spec), ref, true, options...)
}
