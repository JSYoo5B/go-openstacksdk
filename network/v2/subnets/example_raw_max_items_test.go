package subnets_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
)

func Example_rawMaxItems() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	capJSON := flag.String("max-items-json", "null", "raw JSON max_items")
	flag.Parse()
	raw := json.RawMessage(*capJSON)
	if !json.Valid(raw) {
		log.Fatal("max-items-json must be complete JSON")
	}
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
	options := []subnets.RecordListOption{
		subnets.WithRecordListQuery("max_items", raw),
		request.Limit(10),
	}
	proxy, err := network.Subnets.ProxyListRecords(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	generator, ok := proxy.Value.(*subnets.RecordListGenerator)
	if !ok {
		log.Fatal("expected a Resource list generator")
	}
	for row, err := range generator.Records {
		if err != nil {
			log.Fatal(err)
		}
		encoded, err := json.Marshal(row.Resource)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(encoded))
	}
}
