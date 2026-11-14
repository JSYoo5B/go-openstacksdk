package extensions_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONRecords(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_records() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "list", "list, find or proxy")
	identity := flag.String("extension", "qos", "alias or exact display name")
	strict := flag.Bool("strict", false, "return an error when missing")
	expression := flag.String("jmes", "", "deprecated Proxy expression")
	listPath := flag.String("list-base-path", "", "Find fallback service-relative path")
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
	api := network.Extensions
	switch *mode {
	case "find":
		options := []extensions.RecordFindOption{
			request.IgnoreMissing(!*strict),
			request.Query("skip_cache", true),
			request.Query("error_message", "Extension lookup failed"),
		}
		if *listPath != "" {
			options = append(options, extensions.WithRecordFindListBasePath(*listPath))
		}
		value, err := api.FindRecord(ctx, *identity, options...)
		if err != nil {
			log.Fatal(err)
		}
		if value == nil {
			printJSONRecords(nil)
			return
		}
		printJSONRecords(value.Resource)
	case "list":
		for value, err := range api.ListRecords(
			ctx, request.Query("alias", *identity),
		) {
			if err != nil {
				log.Fatal(err)
			}
			printJSONRecords(value.Resource)
		}
	case "proxy":
		result, err := api.ProxyListRecords(ctx, extensions.WithRecordListJMESPath(*expression))
		if err != nil {
			log.Fatal(err)
		}
		if result.JSON != nil {
			fmt.Println(string(result.JSON))
			return
		}
		generator, ok := result.Value.(*extensions.RecordListGenerator)
		if !ok {
			fmt.Printf("opaque expression result: %T\n", result.Value)
			return
		}
		for value, err := range generator.Records {
			if err != nil {
				log.Fatal(err)
			}
			printJSONRecords(value.Resource)
		}
	default:
		log.Fatal("mode must be list, find or proxy")
	}
}
