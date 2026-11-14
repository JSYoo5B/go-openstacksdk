package regions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_records() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for row, err := range service.Regions.ListRecords(ctx,
		regions.WithRecordListQuery("parent_region_id", "Root"),
		regions.WithRecordListQuery("description", "operations"),
		request.Limit(100),
		request.MaxItems(100),
	) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", data)
	}
}
