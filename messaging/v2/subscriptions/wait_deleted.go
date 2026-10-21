package subscriptions

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForDeletion polls GET on one subscription of the scoped queue until it
// is missing, like openstacksdk's wait_for_delete on a Subscription. A native
// 404 completes the wait; other HTTP and transport errors, cancellation and
// timeouts fail it. Each poll uses the scope's configured Client-ID and
// project headers. The shared wait policy applies; pass resource.WithTimeout
// and resource.WithPollInterval for other bounds. Waiting does not delete the
// subscription.
func (s *QueueScope) WaitForDeletion(ctx context.Context, id string, options ...resource.WaitOption) error {
	collection := resource.NewCollection(resource.Adapter[Subscription]{
		Kind: kind,
		Get:  func(ctx context.Context, id string) (*Subscription, error) { return s.Get(ctx, id) },
		ID:   func(*Subscription) string { return id },
	})
	return collection.WaitDeleted(ctx, resource.ID(id), options...)
}
