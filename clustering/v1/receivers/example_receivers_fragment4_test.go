package receivers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_find() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	api := receivers.New(client)
	receiver, err := api.Find(ctx, resource.Name("scale_out"))
	if err != nil {
		log.Fatal(err)
	}
	if receiver != nil {
		fmt.Println(receiver.ID)
	}
	if err := api.Delete(ctx, resource.Name("scale_out")); err != nil {
		log.Fatal(err)
	}
}
