package groups_test

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
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
	api := service.API.SecurityGroups
	if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
		options := []groups.AttributeOption{groups.WithAttribute("name", ""), groups.WithAttribute("stateful", false)}
		if value := os.Getenv("NEUTRON_REVISION"); value != "" {
			revision, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				panic(err)
			}
			options = append(options, groups.WithRevision(revision))
		}
		value, err := api.UpdateWithAttributes(ctx, id, options...)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	if os.Getenv("NEUTRON_CREATE") == "1" {
		value, err := api.CreateWithAttributes(ctx, groups.WithAttribute("name", ""), groups.WithAttribute("stateful", false), groups.WithAttribute("is_shared", false))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n", value)
		return
	}
	fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
