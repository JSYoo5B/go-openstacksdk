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

func Example_cloudQosRuleCreates() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "bandwidth", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "literal policy ID or exact name")
	value := flag.String("value", "0", "JSON required max_kbps/dscp_mark/min_kbps")
	kwargs := flag.String("kwargs", "{}", "JSON optional cloud kwargs")
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
	required := json.RawMessage(*value)
	option := network.WithQoSRuleCreateJSON(json.RawMessage(*kwargs))
	var result *network.QoSRuleCreateResult
	switch *kind {
	case "bandwidth":
		result, err = service.CreateQoSBandwidthLimitRuleRecord(ctx, *policy, required, option)
	case "dscp":
		result, err = service.CreateQoSDSCPMarkingRuleRecord(ctx, *policy, required, option)
	case "minimum":
		result, err = service.CreateQoSMinimumBandwidthRuleRecord(ctx, *policy, required, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Rule, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
