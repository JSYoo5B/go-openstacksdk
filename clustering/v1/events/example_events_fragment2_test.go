package events_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_eventsFragment2() {
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
	listingAPI := events.New(client)
	values, err := listingAPI.All(ctx,
		events.WithListOptions(events.ListOpts{Limit: 20}),
		request.MaxItems(50))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(values))
	for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value.ID)
	}
}
