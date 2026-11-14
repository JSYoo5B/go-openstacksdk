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
)

func Example_cloudPortUpdate() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("identity", "", "port ID or exact name")
	attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
	flag.Parse()
	if *identity == "" {
		log.Fatal("identity is required")
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
	result, err := service.UpdatePortRecord(ctx, *identity,
		network.WithPortUpdateJSON(json.RawMessage(*attrs)))
	if err != nil {
		log.Fatal(err)
	}
	var value any = result.Port
	if result.Response != nil && result.Response.Merged != nil {
		value = result.Response.Merged // the Python post-commit resource state
	}
	if result.NoOp {
		value = result.Lookup.Resource
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
