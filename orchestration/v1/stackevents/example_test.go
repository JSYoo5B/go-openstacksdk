package stackevents_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.OrchestrationV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	events, err := service.StackEvents.InStack(ctx, resource.Name("app"))
	if err != nil {
		log.Fatal(err)
	}
	for event, err := range events.List(ctx,
		stackevents.WithListOptions(stackevents.ListOpts{Limit: 100}),
		request.Query("nested_depth", "1")) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(event.ID, event.ResourceName, event.ResourceStatus)
	}
	resourceEvents, err := events.ForResource("server")
	if err != nil {
		log.Fatal(err)
	}
	values, err := resourceEvents.All(ctx,
		stackevents.WithListResourceEventsOptions(stackevents.ListResourceEventsOpts{Limit: 100}))
	if err != nil {
		log.Fatal(err)
	}
	if len(values) > 0 {
		detail, err := resourceEvents.Get(ctx, values[0].ID)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(detail.ID, detail.ResourceProperties, detail.Header)
	}
}

func ExampleAPI_Find() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.OrchestrationV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	events, err := stackevents.New(service.RawClient()).Find(ctx, "app")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(events)
}
