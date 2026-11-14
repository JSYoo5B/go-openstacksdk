package network_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printJSONCloudQosRuleTypes(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func Example_cloudQosRuleTypes() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	filters := flag.String("filters", "null", "original Source cloud filters JSON")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.ListQoSRuleTypeValues(ctx, network.WithQoSRuleTypeListFilters(json.RawMessage(*filters)))
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(result.JSON))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			log.Fatal(err)
		}
		printJSONCloudQosRuleTypes(decoded)
		return
	}
	for _, value := range result.Values {
		if generator, ok := value.(*ruletypes.ProxyListGenerator); ok {
			for record, err := range generator.Records {
				if err != nil {
					log.Fatal(err)
				}
				printJSONCloudQosRuleTypes(record.Resource)
			}
		} else {
			printJSONCloudQosRuleTypes(value)
		}
	}
}
