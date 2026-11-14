package orders_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listNamedOrderRowsListing2(ctx context.Context, client *gophercloud.ServiceClient) ([]*orders.Order, error) {
	return orders.New(client).Resources.All(ctx,
		resource.WithFilter("name", "top-level-order-name"),
		resource.WithBodyFilter("status", "ACTIVE"),
		resource.WithPaginated(false))
}
