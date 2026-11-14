package instanceactions_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_instanceactionsFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	scope, err := service.InstanceActions.InServer(ctx, resource.Name("worker"))
	if err != nil {
		log.Fatal(err)
	}
	for action, err := range scope.List(ctx, resource.WithPageSize(100)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(scope.ServerID(), action.RequestID, action.Action, action.UpdatedAt)
	}
	action, err := scope.Find(ctx, resource.ID("req-example"), resource.WithIgnoreMissing())
	if err != nil {
		log.Fatal(err)
	}
	if action != nil && action.Events != nil {
		for _, event := range *action.Events {
			fmt.Println(event.Event, event.Result, event.Details)
		}
	}
}
