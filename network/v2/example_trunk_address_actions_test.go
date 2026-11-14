package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func printJSONTrunkAddressActions(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_trunkAddressActions() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	trunkID := flag.String("trunk", "", "trunk ID")
	addPort := flag.String("add-subport", "", "subport to add")
	removePort := flag.String("remove-subport", "", "subport to remove")
	segmentationID := flag.Int("segmentation-id", 100, "VLAN segmentation ID")
	groupID := flag.String("group", "", "address group ID")
	addAddress := flag.String("add-address", "", "address or CIDR to add")
	removeAddress := flag.String("remove-address", "", "address or CIDR to remove")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var trunk *trunks.TrunkActionRecord
	if *addPort != "" {
		if *trunkID == "" {
			log.Fatal("-trunk is required")
		}
		trunk, err = service.Trunks.AddSubportsRecord(ctx, *trunkID,
			[]map[string]any{{"port_id": *addPort, "segmentation_type": "vlan",
				"segmentation_id": *segmentationID}})
		if err != nil {
			log.Fatal(err)
		}
		printJSONTrunkAddressActions(trunk.Resource)
	}
	if *removePort != "" {
		if *trunkID == "" {
			log.Fatal("-trunk is required")
		}
		var options []trunks.SubportActionOption
		if trunk != nil {
			options = append(options, trunks.WithSubportActionSeed(trunk.Resource))
		}
		trunk, err = service.Trunks.RemoveSubportsRecord(ctx, *trunkID,
			[]map[string]any{{"port_id": *removePort}}, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONTrunkAddressActions(trunk.Resource)
	}
	var group *addressgroups.AddressActionRecord
	if *addAddress != "" {
		if *groupID == "" {
			log.Fatal("-group is required")
		}
		group, err = service.SecurityAddressGroups.AddAddressesRecord(ctx,
			*groupID, []string{*addAddress})
		if err != nil {
			log.Fatal(err)
		}
		printJSONTrunkAddressActions(group.Resource)
	}
	if *removeAddress != "" {
		if *groupID == "" {
			log.Fatal("-group is required")
		}
		var options []addressgroups.AddressActionOption
		if group != nil {
			options = append(options, addressgroups.WithAddressActionSeed(group.Resource))
		}
		group, err = service.SecurityAddressGroups.RemoveAddressesRecord(ctx,
			*groupID, []string{*removeAddress}, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONTrunkAddressActions(group.Resource)
	}
}
