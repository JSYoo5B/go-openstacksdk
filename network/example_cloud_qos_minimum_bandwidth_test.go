package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_cloudQosMinimumBandwidth() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "rule ID or fnmatch pattern")
	mode := flag.String("mode", "list", "list, search or get")
	filters := flag.String("filters", "null", "JSON-domain Source filters")
	flag.Parse()
	if *policy == "" {
		log.Fatal("-policy required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var connect []sdk.ConnectionOption
	if *cloud != "" {
		connect = append(connect, sdk.WithCloud(*cloud))
	}
	conn, err := sdk.Connect(ctx, connect...)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []network.MinimumBandwidthCloudOption{func(c *network.MinimumBandwidthCloudOpts) error {
		loc := location.Clone()
		c.CurrentLocation = &loc
		return nil
	}}
	var result *network.MinimumBandwidthCloudResult
	switch *mode {
	case "list":
		options = append(options, network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
		result, err = service.ListQoSMinimumBandwidthRules(ctx, *policy, options...)
	case "search":
		options = append(options, network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
		result, err = service.SearchQoSMinimumBandwidthRules(ctx, *policy, *rule, options...)
	case "get":
		if *rule == "" {
			log.Fatal("-rule required for get")
		}
		result, err = service.GetQoSMinimumBandwidthRule(ctx, *policy, *rule, options...)
	default:
		log.Fatal("mode must be list, search or get")
	}
	if err != nil {
		if result != nil {
			fmt.Fprintf(os.Stderr, "observed extensions=%d rules=%d\n", len(result.Extensions), len(result.Inventory))
		}
		log.Fatal(err)
	}
	if result.Value == nil {
		fmt.Printf("opaque result: %d values\n", len(result.Values))
		return
	}
	fmt.Println(string(result.Value))
}
