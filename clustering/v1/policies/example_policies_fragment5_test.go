package policies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_policiesFragment5() {
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
	service, err = conn.Clustering(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for policy, err := range service.Policies.List(ctx,
		policies.WithListOptions(policies.ListOpts{Limit: 20, Sort: "created_at:desc"}),
		request.Custom(func(c *request.Config[policies.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
		policies.WithListFilter("data", map[string]any{"vendor": map[string]any{"enabled": false}})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(policy.ID, policy.Name)
	}
	found, err := service.Policies.Find(ctx, resource.Name("scale_out_workers"))
	if err != nil {
		log.Fatal(err)
	}
	if found != nil {
		if err := service.Policies.Delete(ctx, resource.ID(found.ID)); err != nil {
			log.Fatal(err)
		}
	}
}
