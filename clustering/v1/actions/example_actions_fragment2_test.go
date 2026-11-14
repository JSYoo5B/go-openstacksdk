package actions_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_actionsCancel() {
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
	accepted, err := service.Actions.Cancel(ctx, resource.ID("ACTION_ID"),
		request.Custom(func(c *request.Config[actions.UpdateOpts]) error { v := false; c.Options.Force = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(accepted.StatusCode, string(accepted.Body))
	action, err := service.Actions.Get(ctx, "ACTION_ID")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(action.ID, action.Status)
}
