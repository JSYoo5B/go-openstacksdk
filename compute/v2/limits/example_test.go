package limits_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func computeV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_InProject() {
	ctx := context.Background()
	api := limits.New(computeV2(ctx).RawClient())
	current, err := api.Fetch(ctx, limits.WithGetReserved(true))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(current.Absolute.MaxTotalCores, current.StatusCode)
	scope, err := api.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	value, err := scope.Get(ctx, limits.WithGetReserved(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(value.ProjectID, value.Absolute.TotalCoresUsed)
	for _, group := range value.Rate {
		for _, rule := range group.Limits {
			fmt.Println(group.URI, rule.Verb, rule.Remaining, string(rule.NextAvailable))
		}
	}
}
