package stackresources_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/orchestration/v1"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func orchestration(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.OrchestrationV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_inStack() {
	ctx := context.Background()
	service := orchestration(ctx)
	scope, err := service.StackResources.InStack(ctx, resource.Name("app"))
	if err != nil {
		log.Fatal(err)
	}
	item, err := scope.Find(ctx, resource.Name("web"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(item.Name, item.LogicalID, item.PhysicalID, item.Owner)
	id, err := scope.ResolveID(ctx, resource.Name("web"))
	if err != nil {
		log.Fatal(err)
	}
	detail, err := scope.Get(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(detail.Attributes, detail.Detailed)
}

func Example_forStack() {
	ctx := context.Background()
	service := orchestration(ctx)
	scope, err := service.StackResources.ForStack(stacks.StackIdentity{Name: "app", ID: "stack-uuid"})
	if err != nil {
		log.Fatal(err)
	}
	items, err := scope.All(ctx, stackresources.WithNestedDepth(1), stackresources.WithDetails(true))
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range items {
		identity, err := item.Identity()
		if err != nil {
			log.Fatal(err)
		}
		child, err := service.StackResources.ForResource(identity)
		if err != nil {
			log.Fatal(err)
		}
		metadata, err := child.Metadata(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(identity.Stack.Name, identity.Stack.ID, identity.Name, metadata)
	}
}

func Example_forResource() {
	ctx := context.Background()
	service := orchestration(ctx)
	node, err := service.StackResources.ForResource(stackresources.ResourceIdentity{
		Stack: stacks.StackIdentity{Name: "app", ID: "stack-uuid"},
		Name:  "web",
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := node.MarkUnhealthy(ctx, false, stackresources.WithHealthReason("recovered")); err != nil {
		log.Fatal(err)
	}
	if _, err := node.Wait(ctx, "CHECK_COMPLETE"); err != nil {
		log.Fatal(err)
	}
}

func ExampleAPI() {
	ctx := context.Background()
	api := stackresources.New(orchestration(ctx).RawClient())
	resources, err := api.Find(ctx, "app")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resources)
	for name, err := range api.ListTypes(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(*name)
	}
	schema, err := api.Schema(ctx, "OS::Nova::Server")
	if err != nil {
		log.Fatal(err)
	}
	template, err := api.Template(ctx, "OS::Nova::Server")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(schema, string(template))
	err = api.MarkUnhealthy(ctx, "app", "stack-uuid", "physical-id", stackresources.MarkUnhealthyOpts{MarkUnhealthy: true, ResourceStatusReason: "bad disk"})
	if err != nil {
		log.Fatal(err)
	}
}
