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

func Example_cloudFirewallPolicyRemoveRule() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	policy := flag.String("policy", "", "policy exact name or ID")
	rule := flag.String("rule", "", "rule exact name or ID")
	filters := flag.String("filters", "{}", "policy Find JSON filters")
	flag.Parse()
	if *policy == "" || *rule == "" {
		log.Fatal("policy and rule are required")
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
	result, err := service.RemoveRuleFromPolicyRecord(ctx, *policy, *rule,
		network.WithFirewallPolicyRemoveRuleFilters(json.RawMessage(*filters)))
	if err != nil {
		log.Fatal(err)
	}
	var value any = result.Policy
	if result.NoOp && result.Lookup != nil {
		value = result.Lookup.Resource
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
