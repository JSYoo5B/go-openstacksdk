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
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_cloudFirewallDeletes() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "rule", "rule, policy or group")
	name := flag.String("name", "", "literal resource ID or exact name to delete")
	filters := flag.String("filters", "{}", "JSON Source Find kwargs")
	flag.Parse()
	if *name == "" {
		log.Fatal("name is required")
	}
	if *kind != "rule" && *kind != "policy" && *kind != "group" {
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
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	opts := []network.FirewallDeleteOption{network.WithFirewallDeleteFilters(json.RawMessage(*filters)), request.Custom(func(c *network.FirewallDeleteOpts) error {
		loc := location.Clone()
		c.CurrentLocation = &loc
		return nil
	})}
	var result *network.FirewallDeletion
	switch *kind {
	case "rule":
		result, err = service.DeleteFirewallRuleRecord(ctx, *name, opts...)
	case "policy":
		result, err = service.DeleteFirewallPolicyRecord(ctx, *name, opts...)
	case "group":
		result, err = service.DeleteFirewallGroupRecord(ctx, *name, opts...)
	}
	if err != nil {
		log.Fatal(err)
	}
	value, err := json.Marshal(result.Deleted)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(value))
}
