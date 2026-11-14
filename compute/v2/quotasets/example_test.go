package quotasets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
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
	api := quotasets.New(computeV2(ctx).RawClient())
	scope, err := api.InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	limits, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(scope.ProjectID(), limits.Cores, limits.Header.Get("X-Openstack-Request-Id"))
	usage, err := scope.Detail(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(usage.Cores.Limit, usage.Cores.InUse, usage.Cores.Reserved)
	zero, unlimited := 0, -1
	updated, err := scope.Update(ctx, quotasets.UpdateOpts{Cores: &unlimited, Instances: &zero}, quotasets.WithUpdateForce(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Cores, updated.Instances)
	reset, err := scope.Reset(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
	userScope, err := scope.InUser(ctx, resource.ID("user-id"))
	if err != nil {
		log.Fatal(err)
	}
	userLimits, err := userScope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(userLimits.ProjectID, userLimits.UserID, userLimits.Cores)
	if _, err := userScope.Update(ctx, quotasets.UpdateOpts{Cores: &zero}, quotasets.WithUpdateForce(false)); err != nil {
		log.Fatal(err)
	}
	if _, err := userScope.Reset(ctx); err != nil {
		log.Fatal(err)
	}
}

func ExampleWithQuotaOptions() {
	ctx := context.Background()
	scope, err := quotasets.New(computeV2(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	cores := 7
	snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &cores})
	cores = 99
	if _, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot); err != nil {
		log.Fatal(err)
	}
	fmt.Println(cores)
}
