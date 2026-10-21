package queues

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForDeletion polls GET until the queue is missing, like openstacksdk's
// wait_for_delete on a Queue. A native 404 completes the wait; other HTTP and
// transport errors, cancellation and timeouts fail it. The default policy is
// the shared resource wait policy; pass resource.WithTimeout and
// resource.WithPollInterval to select other values. Waiting does not delete
// the queue.
func (a *API) WaitForDeletion(ctx context.Context, queueName string, options ...resource.WaitOption) error {
	collection := resource.NewCollection(resource.Adapter[QueueDetails]{
		Kind: "queues",
		Get: func(ctx context.Context, name string) (*QueueDetails, error) {
			value, err := a.Get(ctx, name)
			if err != nil {
				return nil, err
			}
			return &value, nil
		},
		ID: func(*QueueDetails) string { return queueName },
	})
	return collection.WaitDeleted(ctx, resource.ID(queueName), options...)
}
