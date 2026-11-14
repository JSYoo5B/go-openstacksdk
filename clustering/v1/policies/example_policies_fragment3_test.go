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

func Example_policiesFragment3() {
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
	renamed, err := service.Policies.Update(ctx, resource.Name("scale_workers"), policies.UpdateOpts{},
		request.Custom(func(c *request.Config[policies.UpdateOpts]) error {
			v := "scale_out_workers"
			c.Options.Name = &v
			return nil
		}),
		request.Header("X-Vendor-Mode", "explicit"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(renamed.ID, renamed.Name)
}
