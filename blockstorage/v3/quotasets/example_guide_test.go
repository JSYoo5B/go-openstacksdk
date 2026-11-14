package quotasets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

func ExampleAPI_InProject() {
	ctx := context.Background()
	scope, err := quotasets.New(blockStorageV3(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	limits, err := scope.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(scope.ProjectID(), limits.Volumes, limits.Header.Get("X-Openstack-Request-Id"))
	defaults, err := scope.Defaults(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(defaults.Volumes)
	usage, err := scope.Usage(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(usage.Volumes.Limit, usage.Volumes.InUse, usage.Volumes.Reserved)
	zero, unlimited := 0, -1
	updated, err := scope.Update(ctx, quotasets.UpdateOpts{Volumes: &unlimited, Snapshots: &zero},
		quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "SSD", 12))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated.Volumes, string(updated.Body["volumes_SSD"]))
	reset, err := scope.Reset(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
}

func ExampleWithQuotaOptions() {
	ctx := context.Background()
	scope, err := quotasets.New(blockStorageV3(ctx).RawClient()).InProject(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	limit := 5
	nested := map[string]any{"tier": "original"}
	snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Volumes: &limit, Extra: map[string]any{"vendor_quota": nested}})
	limit = 99
	nested["tier"] = "changed"
	if _, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot); err != nil {
		log.Fatal(err)
	}
}
