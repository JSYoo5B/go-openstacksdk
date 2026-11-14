package ports_test

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
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
	api := service.API.Ports
	if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
		options := []ports.AttributeOption{ports.WithAttribute("name", ""), ports.WithAttribute("binding_host_id", "")}
		if value := os.Getenv("NEUTRON_REVISION"); value != "" {
			revision, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				panic(err)
			}
			options = append(options, ports.WithRevision(revision))
		}
		value, err := api.UpdateWithAttributes(ctx, id, options...)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	if os.Getenv("NEUTRON_CREATE") == "1" {
		value, err := api.CreateWithAttributes(ctx, ports.WithAttribute("network_id", os.Getenv("NETWORK_ID")), ports.WithAttribute("is_admin_state_up", false), ports.WithAttribute("security_group_ids", []string{}))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
