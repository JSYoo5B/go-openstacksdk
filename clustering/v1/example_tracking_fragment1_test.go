package v1_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_trackingFragment1() {
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
	tracked, err := service.Policies.Load(ctx, resource.Name("existing-policy"))
	if err != nil {
		log.Fatal(err)
	}
	current := tracked.Value().Name
	if err := tracked.Edit(policies.UpdateOpts{}, request.Custom(func(c *request.Config[policies.UpdateOpts]) error { v := current; c.Options.Name = &v; return nil })); err != nil {
		log.Fatal(err)
	}
	_, err = tracked.Commit(ctx)
	log.Fatal(err)
}
