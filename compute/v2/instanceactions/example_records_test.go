package instanceactions_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_records() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	server := flag.String("server", "", "literal server UUID")
	since := flag.String("changes-since", "", "optional ISO timestamp")
	flag.Parse()
	if *server == "" {
		log.Fatal("-server is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var options []instanceactions.RecordListOption
	if *since != "" {
		options = append(options, request.Query("changes_since", *since))
	}
	for value, err := range compute.InstanceActions.ListRecords(ctx, *server, options...) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
