package quotasets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_quotaQuery() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	scope, err := conn.ProjectQuotas(ctx, resource.ID("project-id"))
	if err != nil {
		log.Fatal(err)
	}
	user, err := scope.InUser(ctx, resource.ID("user-id"))
	if err != nil {
		log.Fatal(err)
	}
	quota, err := user.GetRecord(ctx,
		quotasets.WithQuotaUsage(true),
		request.Query("fields", []string{"cores", "ram"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(quota.Resource.Body["cores"]))
	fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
	fmt.Println(quota.ProjectID, quota.UserID, quota.StatusCode)
}
