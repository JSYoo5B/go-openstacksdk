package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_cloudQosBandwidth() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "rule ID or search pattern")
	mode := flag.String("mode", "list", "list, search or get")
	filters := flag.String("filters", "null", "Source cloud filters JSON")
	flag.Parse()
	if *policy == "" {
		log.Fatal("--policy required")
	}
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSBandwidthLimitResult
	switch *mode {
	case "list":
		result, err = service.ListQoSBandwidthLimitRules(ctx, *policy, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "search":
		result, err = service.SearchQoSBandwidthLimitRules(ctx, *policy, *rule, network.WithQoSBandwidthLimitFilters(json.RawMessage(*filters)))
	case "get":
		if *rule == "" {
			log.Fatal("--rule required for get")
		}
		result, err = service.GetQoSBandwidthLimitRule(ctx, *policy, *rule)
	default:
		log.Fatal("--mode must be list, search or get")
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Value == nil {
		log.Fatal("opaque result: consume QoSBandwidthLimitGenerator.Records explicitly")
	}
	output, err := json.MarshalIndent(result.Value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(output))
}
