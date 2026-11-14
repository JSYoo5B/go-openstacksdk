package agents_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
)

func printJSONRouterAssignmentDhcpHosts(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_routerAssignmentDhcpHosts() {
	mode := flag.String("mode", "list", "list or update")
	networkID := flag.String("network", "", "network ID for list")
	agentID := flag.String("agent", "", "L3 agent ID for update")
	routerID := flag.String("router", "", "router ID for update")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.API.Agents
	switch *mode {
	case "list":
		for row, err := range api.ListDHCPHostRecords(ctx, *networkID,
			request.Query("is_alive", true), request.MaxItems(20)) {
			if err != nil {
				log.Fatal(err)
			}
			printJSONRouterAssignmentDhcpHosts(row.Resource)
		}
	case "update":
		result, err := api.UpdateRouterInAgent(ctx, *agentID, *routerID,
			agents.WithRouterAssignmentPriority(0))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("HTTP %d\n", result.StatusCode)
		printJSONRouterAssignmentDhcpHosts(result.Value)
	default:
		log.Fatal("mode must be list or update")
	}
}
