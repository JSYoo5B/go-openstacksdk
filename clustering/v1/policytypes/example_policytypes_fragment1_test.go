package policytypes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
)

func Example_get() {
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
	api := policytypes.New(client)
	policy, err := api.Get(ctx, "senlin.policy.scaling-1.0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(policy.Name, policy.Schema)

	for policy, err := range api.List(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(policy.Name)
	}
}
