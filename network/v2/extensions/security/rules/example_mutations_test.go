package rules_test

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
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
	api := service.API.SecurityRules
	if os.Getenv("NEUTRON_CREATE") != "1" {
		fmt.Println("set NEUTRON_CREATE=1 and SECURITY_GROUP_ID (optionally NEUTRON_BULK=1)")
		return
	}
	first := []rules.AttributeOption{
		rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
		rules.WithAttribute("direction", "ingress"), rules.WithAttribute("ether_type", "IPv4"),
		rules.WithAttribute("protocol", nil), rules.WithAttribute("remote_address_group_id", ""),
	}
	if os.Getenv("NEUTRON_BULK") == "1" {
		second := []rules.AttributeOption{
			rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
			rules.WithAttribute("direction", "egress"), rules.WithAttribute("ether_type", "IPv6"),
			rules.WithAttribute("remote_address_group_id", nil),
		}
		values, err := api.CreateBulkWithAttributes(ctx, first, second)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", values)
		return
	}
	value, err := api.CreateWithAttributes(ctx, first...)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", value)
}
