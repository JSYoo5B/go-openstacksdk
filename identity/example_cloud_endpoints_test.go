package identity_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func Example_cloudEndpoints() {
	cloud := flag.String("cloud", "system-reader", "clouds.yaml cloud")
	id := flag.String("id", "", "literal ID or search glob")
	filters := flag.String("filters", "", "JSON dict or JSON encoded JMESPath string")
	get := flag.Bool("get", false, "get one endpoint")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	api, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	reader := identity.NewEndpointReadCloud(api.Endpoints)
	var options []identity.EndpointReadOption
	if *filters != "" {
		options = append(options, identity.WithEndpointReadFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *get {
		result, err := reader.GetEndpoint(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	} else {
		result, err := reader.SearchEndpoints(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
