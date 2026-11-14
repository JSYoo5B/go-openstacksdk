package stackcloud_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/templateloader"
)

func Example() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := conn.CreateStack(ctx, stackcloud.CreateRequest{
		Name: "app",
		StackRequest: stackcloud.StackRequest{
			Template:   templateloader.StackInput{Template: templateloader.Source{File: "stack.yaml"}},
			Parameters: map[string]any{"flavor": "m1.small"},
			Wait:       true,
		},
	})
	if err != nil && result != nil && result.Accepted != nil {
		log.Printf("stack accepted, then failed while waiting or reading it back: %v", err)
	} else if err != nil {
		log.Fatal(err)
	}
	if result != nil && result.WaitStatus == "CREATE_FAILED" {
		log.Print("stack reached CREATE_FAILED")
	}
	found, err := conn.SearchStacks(ctx, stackcloud.SearchRequest{NameOrID: "app-*"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(found)
	deleted, err := conn.DeleteStack(ctx, stackcloud.DeleteRequest{NameOrID: "app", Wait: true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(deleted)
}
