package snapshots_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"log"
)

func Example_manage() {
	action := flag.String("action", "", "manage or unmanage")
	identity := flag.String("identity", "", "volume ID for manage, snapshot ID for unmanage")
	reference := flag.String("reference", `{"source-name":"backend-snapshot"}`, "backend reference JSON")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *action {
	case "manage":
		row, err := service.Snapshots.ManageRecord(ctx, *identity, json.RawMessage(*reference))
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	case "unmanage":
		receipt, err := service.Snapshots.UnmanageRecord(ctx, *identity)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(receipt.StatusCode)
	default:
		log.Fatal("action must be manage or unmanage")
	}
}
