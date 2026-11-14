package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_snapshotFind() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("snapshot", "daily", "snapshot name or ID")
	details := flag.Bool("details", true, "use detailed fallback list")
	allProjects := flag.Bool("all-projects", false, "search across projects")
	strict := flag.Bool("strict", false, "return an error when missing")
	version := flag.String("microversion", "3.65", "Cinder selected version; empty enables discovery")
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
	value, err := service.Snapshots.FindRecord(ctx, *identity,
		request.Custom(func(c *snapshots.RecordFindOpts) error { v := *details; c.Details = &v; return nil }),
		request.Custom(func(c *snapshots.RecordFindOpts) error { v := *allProjects; c.AllProjects = &v; return nil }),
		request.IgnoreMissing(!*strict))
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		fmt.Println("null")
		return
	}
	data, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
