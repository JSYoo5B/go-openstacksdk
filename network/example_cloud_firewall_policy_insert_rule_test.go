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

func Example_cloudFirewallPolicyInsertRule() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("identity", "", "firewall policy ID or exact name")
	rule := flag.String("rule", "", "rule ID or exact name")
	after := flag.String("after", "", "optional preceding rule")
	before := flag.String("before", "", "optional following rule")
	filters := flag.String("filters", "{}", "JSON Source Find kwargs")
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
	options := []network.FirewallPolicyInsertOption{
		network.WithFirewallPolicyInsertFilters(json.RawMessage(*filters)),
	}
	if *after != "" {
		options = append(options, network.WithFirewallPolicyInsertAfter(*after))
	}
	if *before != "" {
		options = append(options, network.WithFirewallPolicyInsertBefore(*before))
	}
	result, err := service.InsertFirewallRuleIntoPolicyRecord(ctx, *identity, *rule, options...)
	if err != nil {
		log.Fatal(err)
	}
	var value any = result.Action
	if result.NoOp {
		value = nil
		if result.Lookup != nil {
			value = result.Lookup.Resource
		}
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
