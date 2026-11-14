package subnets_test

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
)

func Example_mutations() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
	if err != nil {
		panic(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		panic(err)
	}
	api := service.API.Subnets
	if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
		options := []subnets.AttributeOption{subnets.WithAttribute("gateway_ip", nil), subnets.WithAttribute("is_dhcp_enabled", false)}
		if value := os.Getenv("NEUTRON_REVISION"); value != "" {
			revision, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				panic(err)
			}
			options = append(options, subnets.WithRevision(revision))
		}
		value, err := api.UpdateWithAttributes(ctx, id, options...)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	if os.Getenv("NEUTRON_CREATE") == "1" {
		value, err := api.CreateWithAttributes(ctx, subnets.WithAttribute("network_id", os.Getenv("NETWORK_ID")), subnets.WithAttribute("ip_version", 4), subnets.WithAttribute("cidr", "10.61.0.0/24"), subnets.WithAttribute("is_dhcp_enabled", false))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
