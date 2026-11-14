package receivers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_update() {
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
	receiver, err := api.Update(ctx, resource.Name("scale_out"), receivers.UpdateOpts{},
		request.Custom(func(c *request.Config[receivers.UpdateOpts]) error {
			c.Options.Name = request.Present("scale_in")
			return nil
		}), request.Custom(func(c *request.Config[receivers.UpdateOpts]) error {
			c.Options.Action = request.Present("CLUSTER_SCALE_IN")
			return nil
		}),
		receivers.WithUpdateParams(map[string]any{}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(receiver.ID, receiver.Action)
}
