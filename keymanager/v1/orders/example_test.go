package orders_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
)

func keyManager(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_Create() {
	ctx := context.Background()
	api := orders.New(keyManager(ctx).RawClient())
	order, err := api.Create(ctx, orders.CreateOpts{
		Type: orders.KeyOrder,
		Meta: orders.MetaOpts{Algorithm: "aes", BitLength: 256, Mode: "cbc"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(order.OrderRef)
}
