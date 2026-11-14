package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
)

func Example_addressScopeRecords() {
	mode := flag.String("mode", "find", "get, find or list")
	identity := flag.String("identity", "", "literal ID or exact name")
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	strict := flag.Bool("strict", false, "report logical absence from find")
	flag.Parse()
	if (*mode == "get" || *mode == "find") && *identity == "" {
		log.Fatal("-identity is required for get/find")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	printRecord := func(row *addressscopes.AddressScopeRecord) {
		if row == nil {
			fmt.Println("address scope absent")
			return
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
	switch *mode {
	case "get":
		row, err := network.AddressScopes.GetRecord(ctx, *identity,
			addressscopes.WithRecordReadFields("id", "name", "is_shared", "ip_version"))
		if err != nil {
			log.Fatal(err)
		}
		printRecord(row)
	case "find":
		row, err := network.AddressScopes.FindRecord(ctx, *identity,
			request.IgnoreMissing(!*strict))
		if err != nil {
			log.Fatal(err)
		}
		printRecord(row)
	case "list":
		for row, err := range network.AddressScopes.ListRecords(ctx,
			request.Query("is_shared", false),
			request.Query("max_items", 10)) {
			if err != nil {
				log.Fatal(err)
			}
			printRecord(row)
		}
	default:
		log.Fatal("-mode must be get, find or list")
	}
}
