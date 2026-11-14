package v3_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_transfers() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithCloud("source"))
	if err != nil {
		log.Fatal(err)
	}
	storage, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	created, err := storage.Transfers.CreateRecord(ctx,
		transfers.WithRecordCreateVolumeID("v1"),
		transfers.WithRecordCreateName("handoff"),
		transfers.WithRecordCreateNoSnapshots(true),
		request.Custom(func(c *transfers.RecordCreateOpts) error { c.SuppressLegacySnapshots = true; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(created.Resource.Body["id"]))
	for candidate, err := range storage.Transfers.ListRecords(ctx,
		request.Custom(func(c *transfers.RecordListOpts) error { v := false; c.Details = &v; return nil }),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(candidate.Resource.Body["id"]), string(candidate.Resource.Body["name"]))
	}
	found, err := storage.Transfers.FindRecord(ctx, "handoff",
		request.IgnoreMissing(false),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(found.Resource.Body["id"]))
}
