package manageablevolumes_test

import (
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
)

func Example_records() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for row, err := range service.ManageableVolumes.ListRecords(ctx,
		request.Custom(func(c *manageablevolumes.RecordListOpts) error { c.Details = false; return nil }),
		request.Query("host", "host@backend"),
		request.Limit(20), request.MaxItems(20)) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
	for row, err := range service.ManageableSnapshots.ListRecords(ctx,
		request.Custom(func(c *manageablesnapshots.RecordListOpts) error { c.Details = true; return nil }),
		request.Query("host", "host@backend"),
		request.MaxItems(20)) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
