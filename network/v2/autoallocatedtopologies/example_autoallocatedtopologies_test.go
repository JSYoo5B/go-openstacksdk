package autoallocatedtopologies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_autoallocatedtopologies() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := autoallocatedtopologies.New(service.RawClient())
	project := request.Custom(func(c *autoallocatedtopologies.Options) error { v := "project-id"; c.ProjectID = &v; return nil })
	validation, err := api.ValidateRecord(ctx, project)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(validation.Resource.Body["id"]), string(validation.Resource.Body["dry_run"]))
	topology, err := api.GetRecord(ctx, project)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(topology.ProjectID, string(topology.Resource.Body["id"]))
}
