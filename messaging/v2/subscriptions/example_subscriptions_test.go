package subscriptions_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func SubscribeSubscriptions(ctx context.Context, client *gophercloud.ServiceClient) error {
	queue, err := subscriptions.New(client).InQueue(ctx, "events")
	if err != nil {
		return err
	}
	created, err := queue.Create(ctx, subscriptions.CreateOpts{
		Subscriber: "mailto:example@example.test",
	}, subscriptions.WithCreateDeliveryOptions(map[string]any{
		"from": "Zaqar", "subject": "new event",
	}))
	if err != nil {
		return err
	}
	current, err := queue.Get(ctx, created.SubscriptionID)
	if err != nil {
		return err
	}
	fmt.Println(current.Subscriber, string(current.TTL))
	for item, err := range queue.List(ctx,
		subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 10}),
		request.MaxItems(5)) {
		if err != nil {
			return err
		}
		fmt.Println(item.ID)
	}
	return queue.Delete(ctx, created.SubscriptionID,
		request.IgnoreMissing(false))
}
