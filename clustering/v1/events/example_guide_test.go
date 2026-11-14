package events_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
)

func clustering(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := events.New(clustering(ctx).RawClient())
	event, err := api.Get(ctx, "event-id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(event.ObjectID, event.GeneratedAt, event.Level)
	records, err := api.All(ctx, events.WithListOptions(events.ListOpts{
		Limit:       20,
		ObjectIDs:   []string{"node-id", "other-node-id"},
		ObjectTypes: []string{"NODE", "CLUSTER"},
		Sort:        "timestamp:desc",
	}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(records))
}
