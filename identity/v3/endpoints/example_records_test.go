package endpoints_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_records() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	service := flag.String("service-id", "", "optional service UUID")
	region := flag.String("region-id", "", "optional region ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []endpoints.RecordListOption{
		endpoints.WithRecordListQuery("is_enabled", true),
		request.MaxItems(100),
	}
	if *service != "" {
		options = append(options, endpoints.WithRecordListQuery("service_id", *service))
	}
	if *region != "" {
		options = append(options, endpoints.WithRecordListQuery("region_id", *region))
	}
	for row, err := range identity.Endpoints.ListRecords(ctx, options...) {
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
