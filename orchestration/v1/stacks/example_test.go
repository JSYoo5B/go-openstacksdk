package stacks_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/orchestration/v1"
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
	scope, err := orchestration(ctx).Stacks.InStack(ctx, resource.Name("app"))
	if err != nil {
		log.Fatal(err)
	}
	identity := scope.Identity()
	fmt.Println(identity.Name, identity.ID)
	stack, err := scope.Get(ctx, stacks.WithResolveOutputs(false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(stack.Name, stack.ID, stack.Status, stack.Detailed)
}

func Example_forStack() {
	ctx := context.Background()
	scope, err := orchestration(ctx).Stacks.ForStack(stacks.StackIdentity{Name: "app", ID: "stack-uuid"})
	if err != nil {
		log.Fatal(err)
	}
	if err := scope.UpdatePatch(ctx, stacks.UpdateOpts{
		Parameters: map[string]any{"size": "large"},
		Tags:       []string{"ops", "test"},
	}); err != nil {
		log.Fatal(err)
	}
	if _, err := scope.Wait(ctx, "UPDATE_COMPLETE"); err != nil {
		log.Fatal(err)
	}
	if err := scope.Delete(ctx); err != nil {
		log.Fatal(err)
	}
	if err := scope.WaitDeleted(ctx); err != nil {
		log.Fatal(err)
	}
}

func ExampleAPI() {
	ctx := context.Background()
	api := stacks.New(orchestration(ctx).RawClient())
	stack, err := api.Find(ctx, "app")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(stack.ID)
	for stack, err := range api.List(ctx, stacks.WithListOptions(stacks.ListOpts{Name: "app", AllTenants: true, Tags: "a,b"})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stack.ID)
	}
	template := []byte("heat_template_version: 2018-08-31\n")
	if err := api.Update(ctx, "app", stack.ID, stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: template}}}); err != nil {
		log.Fatal(err)
	}
	if err := api.UpdatePatch(ctx, "app", stack.ID, stacks.UpdateOpts{Parameters: map[string]any{"size": 2}}); err != nil {
		log.Fatal(err)
	}
	abandoned, err := api.Abandon(ctx, "app", stack.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(abandoned)
	if err := api.Delete(ctx, "app", stack.ID); err != nil {
		log.Fatal(err)
	}
}
