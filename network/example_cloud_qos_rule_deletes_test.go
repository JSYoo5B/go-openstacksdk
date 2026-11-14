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

func Example_cloudQosRuleDeletes() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	kind := flag.String("kind", "", "bandwidth, dscp or minimum")
	policy := flag.String("policy", "", "QoS policy ID or name")
	rule := flag.String("rule", "", "literal QoS rule ID")
	flag.Parse()
	if *policy == "" || *rule == "" || *kind == "" {
		log.Fatal("--kind, --policy and --rule required")
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
	var result *network.QoSRuleDeleteResult
	switch *kind {
	case "bandwidth":
		result, err = service.DeleteQoSBandwidthLimitRule(ctx, *policy, *rule)
	case "dscp":
		result, err = service.DeleteQoSDSCPMarkingRule(ctx, *policy, *rule)
	case "minimum":
		result, err = service.DeleteQoSMinimumBandwidthRule(ctx, *policy, *rule)
	default:
		log.Fatal("--kind must be bandwidth, dscp or minimum")
	}
	if err != nil {
		if result != nil && result.Receipt != nil {
			log.Printf("actual DELETE response: %d %s", result.Receipt.StatusCode, string(result.Receipt.Envelope))
		}
		log.Fatal(err)
	}
	output := map[string]any{"completed": result.Completed, "deleted": result.Deleted}
	if result.Receipt != nil {
		output["status"] = result.Receipt.StatusCode
		output["body"] = string(result.Receipt.Envelope)
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
