package v3_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
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

func Example_resetGroupStatus() {
	ctx := context.Background()
	client := blockStorageV3(ctx).RawClient()
	if _, err := groups.New(client).ResetStatusRecord(ctx, "group-id", "available"); err != nil {
		log.Fatal(err)
	}
	result, err := groupsnapshots.New(client).ResetStatusRecord(ctx, "group-snapshot-id", "error")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
