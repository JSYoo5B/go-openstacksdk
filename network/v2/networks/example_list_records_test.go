package networks_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func Example_listRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	name := flag.String("name", "", "server-side network name filter")
	maximum := flag.Int("max-items", 0, "raw row cap; zero is unbounded")
	expression := flag.String("jmes", "", "deprecated Python Proxy expression")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []networks.RecordListOption{
		request.MaxItems(*maximum),
		networks.WithRecordListQuery("jmespath_filters", *expression),
	}
	if *name != "" {
		options = append(options, networks.WithRecordListQuery("name", *name))
	}
	result, err := network.Networks.ProxyListRecords(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		fmt.Println(string(result.JSON))
		return
	}
	generator, ok := result.Value.(*networks.RecordListGenerator)
	if !ok {
		// Expressions such as to_array(@) retain nested generator values.
		// They have no JSON representation and do not issue a request here.
		fmt.Printf("opaque expression result: %T\n", result.Value)
		return
	}
	for value, err := range generator.Records {
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
