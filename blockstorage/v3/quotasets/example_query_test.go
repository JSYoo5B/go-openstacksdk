package quotasets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_query() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	scope, err := conn.BlockStorageProjectQuotas(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	quota, err := scope.GetRecord(ctx,
		quotasets.WithQuotaUsage(true),
		request.Query("fields", []string{"volumes", "gigabytes"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(quota.Resource.Body["volumes"]))
	fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
	fmt.Println(quota.ProjectID, quota.StatusCode)
}
