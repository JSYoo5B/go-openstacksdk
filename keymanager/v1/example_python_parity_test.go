package v1_test

import (
	"context"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_pythonParity() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	if err := secrets.New(client).Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"}); err != nil {
		log.Fatal(err)
	}
	if _, err := orders.New(client).WaitFor(ctx, resource.ID("o1"), "ACTIVE",
		resource.WithUnlimitedWait(),
		resource.WithFailureStates("ERROR"),
	); err != nil {
		log.Fatal(err)
	}
	if err := orders.New(client).WaitForDeletion(ctx, resource.ID("o1"), resource.WithTimeout(120*time.Second)); err != nil {
		log.Fatal(err)
	}
}
