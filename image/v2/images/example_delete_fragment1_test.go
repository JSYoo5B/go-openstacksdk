package images_test

import (
	"context"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/image/v2"
	"log"
)

func Example_deleteFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ImageV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	_, _, _ = conn, service, client
	api, err := conn.ImageV2(ctx)
	if err != nil {
		log.Fatal(err)
	}

	_ = api
}
