package secretstores_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func ExampleWithListFilter() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stores, err := service.SecretStores.All(ctx,
		secretstores.WithListFilter("name", "backend-pattern"),
		secretstores.WithListFilter("secret_store_id", "store-id"),
		secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(stores)
}
