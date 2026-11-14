package quotas_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func loadBalancer(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.LoadBalancerV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := quotas.New(loadBalancer(ctx).RawClient())
	scope, err := api.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	limits, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(limits.ProjectID, limits.Loadbalancer, limits.Listener)
	defaults, err := api.Defaults(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaults.Loadbalancer, defaults.Listener)
	zero, unlimited := 0, -1
	updated, err := scope.Update(ctx, quotas.UpdateOpts{Listener: &zero, Loadbalancer: &unlimited}, quotas.WithDefaultLimit(quotas.LimitMembers))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))
	for quota, err := range api.ListProjects(ctx, quotas.WithProjectListOptions(quotas.ProjectListOpts{Limit: 100})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(quota.ProjectID, quota.Pool)
	}
	if _, err := scope.Reset(ctx); err != nil {
		log.Fatal(err)
	}
}

func ExampleWithLegacyQuotaKeys() {
	ctx := context.Background()
	scope, err := quotas.New(loadBalancer(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	ten, unlimited := 10, -1
	if _, err := scope.Update(ctx, quotas.UpdateOpts{Loadbalancer: &ten, Healthmonitor: &unlimited}, quotas.WithLegacyQuotaKeys(true)); err != nil {
		log.Fatal(err)
	}
}
