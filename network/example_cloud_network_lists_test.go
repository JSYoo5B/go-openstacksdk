package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_cloudNetworkLists() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "list", "list or search")
	name := flag.String("name", "", "literal server-side network name for search")
	filters := flag.String("filters", "{}", "JSON filters; search also accepts key/value pairs")
	flag.Parse()
	if *mode != "list" && *mode != "search" {
		log.Fatal("mode must be list or search")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []network.NetworkCloudListOption{
		network.WithNetworkCloudFilters(json.RawMessage(*filters)),
		request.Custom(func(c *network.NetworkCloudListOpts) error {
			loc := location.Clone()
			c.CurrentLocation = &loc
			return nil
		}),
	}
	var result *network.NetworkCloudRecords
	if *mode == "search" {
		result, err = service.SearchNetworkRecords(ctx, *name, options...)
	} else {
		result, err = service.ListNetworkRecords(ctx, options...)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON == nil {
		log.Fatal("result contains an opaque lazy generator; inspect result.Values instead")
	}
	var formatted json.RawMessage = result.JSON
	data, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
