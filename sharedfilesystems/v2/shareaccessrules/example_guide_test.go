package shareaccessrules_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	v2 "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func sharedFileSystems(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.SharedFileSystemV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleWithListBodyFilter() {
	ctx := context.Background()
	scope, err := sharedFileSystems(ctx).ShareAccessRules.InShare(ctx, resource.Name("data"))
	if err != nil {
		log.Fatal(err)
	}
	rules, err := scope.All(ctx,
		shareaccessrules.WithListBodyFilter("access_level", "rw"),
		shareaccessrules.WithListBodyFilter("state", "active"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(rules))
}

func ExampleAPI_ListAccessAction() {
	ctx := context.Background()
	client := sharedFileSystems(ctx).RawClient()
	client.Microversion = "2.40"
	result, err := shareaccessrules.New(client).ListAccessAction(ctx, "share-id")
	if err != nil {
		log.Fatal(err)
	}
	for _, rule := range result.Rules {
		fmt.Println(result.Action, rule.ID, rule.AccessTo, rule.State)
	}
}
