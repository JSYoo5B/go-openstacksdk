package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_cloudServerGroupReads() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	operation := flag.String("operation", "search", "search or get")
	name := flag.String("name", "", "name, literal ID or search glob")
	filters := flag.String("filters", "", "complete JSON filter; omitted means None")
	flag.Parse()
	if *operation != "search" && *operation != "get" {
		log.Fatal("unknown operation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []compute.ServerGroupCloudReadOption{request.Custom(func(c *compute.ServerGroupCloudReadOpts) error {
		loc := location.Clone()
		c.CurrentLocation = &loc
		return nil
	})}
	if *filters != "" {
		options = append(options, compute.WithServerGroupCloudFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *operation == "search" {
		result, cause := service.SearchServerGroups(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	} else {
		result, cause := service.GetServerGroup(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		value = json.RawMessage("null")
	}
	data, err := json.MarshalIndent(json.RawMessage(value), "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
