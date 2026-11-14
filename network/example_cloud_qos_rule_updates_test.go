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

func Example_cloudQosRuleUpdates() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "bandwidth", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "literal policy ID or exact name")
	rule := flag.String("rule", "", "literal rule ID")
	kwargs := flag.String("kwargs", "{}", "JSON cloud update kwargs")
	flag.Parse()
	if *policy == "" {
		log.Fatal("policy is required")
	}
	if *kind != "bandwidth" && *kind != "dscp" && *kind != "minimum" {
		log.Fatal("unknown kind")
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
	option := network.WithQoSRuleUpdateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleUpdateResult
	switch *kind {
	case "bandwidth":
		result, err = service.UpdateQoSBandwidthLimitRuleRecord(ctx, *policy, *rule, option)
	case "dscp":
		result, err = service.UpdateQoSDSCPMarkingRuleRecord(ctx, *policy, *rule, option)
	case "minimum":
		result, err = service.UpdateQoSMinimumBandwidthRuleRecord(ctx, *policy, *rule, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	var value any
	if result.Rule != nil {
		value = result.Rule.Resource
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
