package v3_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_resourceFilters() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	version := flag.String("microversion", "3.33", "selected Cinder version")
	resourceName := flag.String("resource", "", "resource name; empty omits query")
	single := flag.Bool("single-page", false, "stop after the first page")
	expression := flag.String("expression", "", "optional JMESPath result")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.BlockStorage, *version))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	filters := resourcefilters.New(service.RawClient())
	options := []resourcefilters.RecordListOption{
		request.Custom(func(c *resourcefilters.RecordListOpts) error {
			c.Control = resource.ListControl{SinglePage: *single}
			return nil
		}),
		resourcefilters.WithRecordListJMESPath(*expression),
	}
	if *resourceName != "" {
		options = append(options, resourcefilters.WithRecordListResource(*resourceName))
	}
	result, err := filters.ListResult(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(result.Value))
}
