package v3_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_groupFindFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	value, err := service.Groups.FindRecord(ctx, "sample",
		request.Microversion("3.90"),
		request.Headers(map[string]string{"X-Trace": "group-find"}))

	_ = value
}
