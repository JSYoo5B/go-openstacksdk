package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func Example_addressGroupList() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	project := flag.String("project", "", "optional server project filter")
	address := flag.String("address", "", "optional exact one-address list filter")
	cap := flag.Int("max-items", 20, "raw row cap; zero is unbounded")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []addressgroups.RecordListOption{
		request.MaxItems(*cap),
	}
	if *project != "" {
		options = append(options, addressgroups.WithRecordListQuery("project_id", *project))
	}
	if *address != "" {
		options = append(options, addressgroups.WithRecordListQuery("addresses", []string{*address}))
	}
	for row, err := range network.SecurityAddressGroups.ListRecords(ctx, options...) {
		if err != nil {
			log.Fatal(err)
		}
		raw, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
	}
}
