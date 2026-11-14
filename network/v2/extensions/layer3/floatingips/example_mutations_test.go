package floatingips_test

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
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
	api := service.API.FloatingIPs
	if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
		options := []floatingips.AttributeOption{floatingips.WithAttribute("port_id", nil), floatingips.WithAttribute("fixed_ip_address", "")}
		if value := os.Getenv("NEUTRON_REVISION"); value != "" {
			revision, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				panic(err)
			}
			options = append(options, floatingips.WithRevision(revision))
		}
		value, err := api.UpdateWithAttributes(ctx, id, options...)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	if os.Getenv("NEUTRON_CREATE") == "1" {
		value, err := api.CreateWithAttributes(ctx, floatingips.WithAttribute("floating_network_id", os.Getenv("EXTERNAL_NETWORK_ID")), floatingips.WithAttribute("description", ""))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
