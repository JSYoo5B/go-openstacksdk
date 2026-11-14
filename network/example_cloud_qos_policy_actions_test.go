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

func printJSONCloudQosPolicyActions(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
func Example_cloudQosPolicyActions() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "get", "get or delete")
	identity := flag.String("identity", "", "literal ID or exact name")
	filters := flag.String("filters", `{"skip_cache":true,"error_message":"QoS policy lookup failed"}`, "get-only JSON kwargs")
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
	switch *mode {
	case "get":
		result, err := service.GetQoSPolicyRecord(
			ctx, *identity, network.WithQoSPolicyFilters(json.RawMessage(*filters)),
		)
		if err != nil {
			log.Fatal(err)
		}
		if result.Policy == nil {
			printJSONCloudQosPolicyActions(nil)
			return
		}
		printJSONCloudQosPolicyActions(result.Policy.Resource)
	case "delete":
		result, err := service.DeleteQoSPolicyRecord(ctx, *identity)
		if err != nil {
			log.Fatal(err)
		}
		printJSONCloudQosPolicyActions(result.Deleted)
	default:
		log.Fatal("mode must be get or delete")
	}
}
