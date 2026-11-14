package orders_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listAESOrdersListing1(ctx context.Context, client *gophercloud.ServiceClient) ([]*orders.Order, error) {
	return orders.New(client).Resources.All(ctx,
		resource.WithFilters(map[string]any{
			"type": "key",
			"meta": map[string]any{"algorithm": "aes", "bit_length": 256},
		}),
		resource.WithMaxItems(20))
}
