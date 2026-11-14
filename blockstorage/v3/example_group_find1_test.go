package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_groupsFindRecord() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("identity", "", "group ID or exact name")
	version := flag.String("microversion", "3.38", "selected Cinder default")
	strict := flag.Bool("strict", false, "return an error when not found")
	summary := flag.Bool("summary", false, "use summary fallback list")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.BlockStorage, *version))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	value, err := service.Groups.FindRecord(ctx, *identity,
		request.IgnoreMissing(!*strict),
		request.Custom(func(c *groups.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		fmt.Println("null")
		return
	}
	raw, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}
