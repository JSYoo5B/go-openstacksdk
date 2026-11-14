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

func Example_receiversFragment3() {
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
	for receiver, err := range api.List(ctx,
		receivers.WithListOptions(receivers.ListOpts{Type: "message", Limit: 20}),
		request.Custom(func(c *request.Config[receivers.ListOpts]) error { c.Options.UserID = "USER_ID"; return nil }), request.Custom(func(c *request.Config[receivers.ListOpts]) error {
			v := false
			c.Options.GlobalProject = &v
			return nil
		}),
		receivers.WithListFilter("params", map[string]any{"team": "infra"}),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(receiver.ID, receiver.Params)
	}
}
