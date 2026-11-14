package nodes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

func Example_forceDelete() {
	ctx := context.Background()
	client := clustering(ctx).RawClient()
	submission, err := nodes.New(client).Delete(ctx, resource.ID("NODE_ID"), request.Custom(func(c *request.Config[nodes.DeleteOpts]) error { c.Options.Force = true; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	if submission == nil {
		return
	}
	action, err := actions.New(client).Get(ctx, submission.ActionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(action.ID, action.Status)
}
