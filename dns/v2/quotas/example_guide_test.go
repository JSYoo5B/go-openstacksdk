package quotas_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func dns(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.DNSV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	scope, err := quotas.New(dns(ctx).RawClient()).InProject(ctx, resource.ID("project-id"), quotas.WithAllProjects(true))
	if err != nil {
		log.Fatal(err)
	}
	limits, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(limits.ProjectID, limits.Zones, limits.StatusCode)
	zero := 0
	updated, err := scope.Update(ctx, quotas.UpdateOpts{Zones: &zero})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))
	reset, err := scope.Reset(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reset.ProjectID, reset.StatusCode)
	zones := 7
	snapshot := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zones})
	zones = 99
	if _, err := scope.Update(ctx, quotas.UpdateOpts{}, snapshot); err != nil {
		log.Fatal(err)
	}
}
