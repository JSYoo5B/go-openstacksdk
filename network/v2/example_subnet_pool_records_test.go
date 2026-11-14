package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_subnetPoolRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	action := flag.String("action", "find", "find, list, update")
	pool := flag.String("pool", "", "find identity or update literal ID")
	name := flag.String("name", "", "optional update name")
	quota := flag.Int("quota", 0, "optional update quota; supply -quota to send zero")
	flag.Parse()
	sent := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { sent[f.Name] = true })
	if (*action == "find" || *action == "update") && *pool == "" {
		log.Fatal("-pool is required")
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
	var rows []*subnetpools.SubnetPoolRecord
	switch *action {
	case "find":
		row, findErr := network.SubnetPools.FindRecord(ctx, *pool)
		err = findErr
		if row != nil {
			rows = append(rows, row)
		}
	case "list":
		rows, err = network.SubnetPools.AllRecords(ctx)
	case "update":
		options := []subnetpools.RecordMutationOption{}
		if sent["name"] {
			options = append(options, request.Field("name", *name))
		}
		if sent["quota"] {
			options = append(options, request.Field("default_quota", *quota))
		}
		row, updateErr := network.SubnetPools.UpdateRecord(ctx, *pool, options...)
		err = updateErr
		if row != nil {
			rows = append(rows, row)
		}
	default:
		log.Fatal("select find, list or update")
	}
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		raw, marshalErr := json.MarshalIndent(row.Resource.Body, "", "  ")
		if marshalErr != nil {
			log.Fatal(marshalErr)
		}
		fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
	}
}
