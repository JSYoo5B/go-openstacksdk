package quotas_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func networkV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	scope, err := quotas.New(networkV2(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	limits, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(scope.ProjectID(), limits.Network, limits.Port)
	defaults, err := scope.Defaults(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaults.Network, defaults.Port)
	usage, err := scope.Detail(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(usage.Network.Used, usage.Network.Reserved, usage.Network.Limit)
	zero, unlimited := 0, -1
	updated, err := scope.Update(ctx, quotas.UpdateOpts{Port: &zero, Network: &unlimited}, quotas.WithUpdateCheckLimit(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Port, updated.Header.Get("X-Openstack-Request-Id"))
	reset, err := scope.Delete(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reset.StatusCode, reset.Header.Get("X-Openstack-Request-Id"))
}

func Example_listProjects() {
	ctx := context.Background()
	api := quotas.New(networkV2(ctx).RawClient())
	for quota, err := range api.ListProjects(ctx, quotas.WithListProjectID("project-id"), request.MaxItems(1)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(quota.ProjectID, quota.Network, quota.Port)
		fmt.Println(quota.Header.Get("X-Openstack-Request-Id"), quota.StatusCode)
	}
}
