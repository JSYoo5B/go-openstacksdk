package clusters_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func blockStorageV3(ctx context.Context) *v3.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_Disable() {
	ctx := context.Background()
	api := clusters.New(blockStorageV3(ctx).RawClient())
	cluster, err := api.Get(ctx, "cluster1@lvm")
	if err != nil {
		log.Fatal(err)
	}
	result, err := api.Disable(ctx, cluster.Target(), request.Custom(func(c *clusters.ActionOpts) error { c.Reason = "maintenance"; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
