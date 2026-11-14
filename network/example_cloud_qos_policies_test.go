package network_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_cloudQosPolicies() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "list", "list or search")
	name := flag.String("name", "", "literal server name query")
	filters := flag.String("filters", "null", "JSON filters")
	flag.Parse()
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
	options := []network.QoSPolicyReadOption{
		network.WithQoSPolicyFilters(json.RawMessage(*filters)),
	}
	var result *network.QoSPolicyRecords
	switch *mode {
	case "list":
		result, err = service.ListQoSPolicyRecords(ctx, options...)
	case "search":
		result, err = service.SearchQoSPolicyRecords(ctx, *name, options...)
	default:
		log.Fatal("mode must be list or search")
	}
	if err != nil {
		if result != nil {
			log.Printf("completed extension records: %d", len(result.Extensions))
		}
		log.Fatal(err)
	}
	if result.JSON == nil {
		fmt.Printf("opaque iterable result: %T\n", result.Value)
		return
	}
	var readable any
	decoder := json.NewDecoder(bytes.NewReader(result.JSON))
	decoder.UseNumber()
	if err := decoder.Decode(&readable); err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(readable, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
