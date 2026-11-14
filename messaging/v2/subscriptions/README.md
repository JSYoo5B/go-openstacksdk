# Subscriptions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package subscriptionexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions"
)

func Subscribe(ctx context.Context, client *gophercloud.ServiceClient) error {
    queue, err := subscriptions.New(client).InQueue(ctx, "events")
    if err != nil { return err }
    created, err := queue.Create(ctx, subscriptions.CreateOpts{
        Subscriber: "mailto:example@example.test",
    }, subscriptions.WithCreateDeliveryOptions(map[string]any{
        "from": "Zaqar", "subject": "new event",
    }))
    if err != nil { return err }
    current, err := queue.Get(ctx, created.SubscriptionID)
    if err != nil { return err }
    fmt.Println(current.Subscriber, string(current.TTL))
    for item, err := range queue.List(ctx,
        subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 10}),
        request.MaxItems(5)) {
        if err != nil { return err }
        fmt.Println(item.ID)
    }
    return queue.Delete(ctx, created.SubscriptionID,
        request.IgnoreMissing(false))
}
```

## References

- [Message](https://docs.openstack.org/api-ref/message/#subscriptions-subscriptions)
- [Messaging Subscriptions Test](../../../api/messaging_subscriptions_test.go)
