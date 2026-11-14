package ports_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func printJSONUserHelpers(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}

func Example_userHelpers() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	mode := flag.String("mode", "values", "values, subnet or bulk")
	filters := flag.String("filters", "{}", "Source cloud filters JSON")
	subnetID := flag.String("subnet", "", "literal subnet ID")
	data := flag.String("data", "[]", "nonempty JSON array of port attribute objects")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	if *mode == "values" {
		result, err := conn.ListPortValues(ctx, ports.WithSourceCloudListFilters(json.RawMessage(*filters)))
		if err != nil {
			log.Fatal(err)
		}
		if result.JSON == nil {
			log.Fatal("opaque result; consume RecordListGenerator.Records explicitly")
		}
		printJSONUserHelpers(result.JSON)
		return
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *mode {
	case "subnet":
		if *subnetID == "" {
			log.Fatal("--subnet required")
		}
		result, err := network.Ports.GetSubnetPorts(ctx, ports.SubnetPortID(*subnetID))
		if err != nil {
			log.Fatal(err)
		}
		for _, record := range result.Ports {
			printJSONUserHelpers(record.Resource.Body)
		}
	case "bulk":
		var rows []json.RawMessage
		if err := json.Unmarshal([]byte(*data), &rows); err != nil {
			log.Fatal(err)
		}
		inputs := make([]ports.BulkPortInput, len(rows))
		for i, row := range rows {
			inputs[i] = ports.BulkPortJSON(row)
		}
		result, err := network.Ports.CreatePorts(ctx, inputs)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("POST completed: %d\n", result.StatusCode)
		for record, err := range result.Records {
			if err != nil {
				log.Fatal(err)
			}
			printJSONUserHelpers(record.Resource.Body)
		}
	default:
		log.Fatal("--mode must be values, subnet or bulk")
	}
}
