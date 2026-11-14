package actions_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
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
	api := actions.New(clustering(ctx).RawClient())
	action, err := api.Get(ctx, "action-id")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(action.ID, action.Status)
	for action, err := range api.List(ctx, actions.WithListOptions(actions.ListOpts{TargetID: "node-id", Limit: 20})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(action.ID, action.Status)
	}
}
