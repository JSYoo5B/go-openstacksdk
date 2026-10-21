package messages

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForDeletion polls GET until the message is missing, like openstacksdk's
// wait_for_delete on a Message. A native 404 completes the wait; other HTTP and
// transport errors, cancellation and timeouts fail it. The default policy is
// the shared resource wait policy; pass resource.WithTimeout and
// resource.WithPollInterval to select other values. Waiting does not delete
// the message.
func (a *API) WaitForDeletion(ctx context.Context, queueName, messageID string, options ...resource.WaitOption) error {
	collection := resource.NewCollection(resource.Adapter[Message]{
		Kind: "messages",
		Get: func(ctx context.Context, id string) (*Message, error) {
			value, err := a.Get(ctx, queueName, id)
			if err != nil {
				return nil, err
			}
			return &value, nil
		},
		ID: func(*Message) string { return messageID },
	})
	return collection.WaitDeleted(ctx, resource.ID(messageID), options...)
}
