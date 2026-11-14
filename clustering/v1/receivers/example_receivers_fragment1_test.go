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

func Example_receiversFragment1() {
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
	receiver, err := api.Create(ctx, receivers.CreateOpts{Name: "scale_out", Type: "webhook"},
		request.Custom(func(c *request.Config[receivers.CreateOpts]) error {
			c.Options.ClusterID = request.Present("CLUSTER_ID")
			return nil
		}), request.Custom(func(c *request.Config[receivers.CreateOpts]) error {
			c.Options.Action = request.Present("CLUSTER_SCALE_OUT")
			return nil
		}),
		receivers.WithCreateParams(map[string]any{}))
	if err != nil {
		log.Fatal(err)
	}
	message, err := api.Create(ctx, receivers.CreateOpts{Name: "events", Type: "message"})
	if err != nil {
		log.Fatal(err)
	}
	receiver, err = api.Get(ctx, receiver.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(receiver.ID, receiver.Action, receiver.Channel, message.ID)
}
