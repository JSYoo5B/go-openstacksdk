package quotas_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func containerInfra(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ContainerInfraV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := quotas.New(containerInfra(ctx).RawClient())
	project, err := api.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	scope, err := project.ForResource(quotas.Cluster)
	if err != nil {
		log.Fatal(err)
	}
	quota, err := scope.Create(ctx,
		request.Custom(func(c *request.Config[quotas.QuotaCreateOpts]) error { v := 0; c.Options.HardLimit = &v; return nil }),
		request.Field("vendor", map[string]string{"tier": "small"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quota.RequestProjectID, quota.RequestResource, quota.ID, quota.HardLimit, quota.Header.Get("X-Openstack-Request-Id"))
	current, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(current.HardLimit)
	updated, err := scope.Update(ctx, request.Custom(func(c *request.Config[quotas.QuotaCreateOpts]) error { v := 2; c.Options.HardLimit = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.StatusCode)
	if _, err := scope.Delete(ctx); err != nil {
		log.Fatal(err)
	}
	for quota, err := range api.List(ctx,
		quotas.WithListOptions(quotas.ListOpts{Limit: 100}),
		request.Custom(func(c *request.Config[quotas.ListOpts]) error { v := true; c.Options.AllTenants = &v; return nil })) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(quota.ID, quota.ProjectID, quota.Resource, quota.HardLimit)
	}
}
