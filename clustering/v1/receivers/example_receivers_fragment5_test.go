package receivers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_all() {
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
	listingAPI := receivers.New(client)
	values, err := listingAPI.All(ctx,
		receivers.WithListOptions(receivers.ListOpts{Limit: 20}),
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
