package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_summary() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	version := flag.String("microversion", "3.36", "selected Cinder version")
	all := flag.Bool("all-projects", false, "ask for all projects")
	single := flag.Bool("single-fetch", false, "omit pinned redundant first GET")
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
	value, err := service.Volumes.SummaryRecord(ctx,
		request.Custom(func(c *volumes.SummaryOpts) error { c.AllProjects = *all; return nil }),
		request.Custom(func(c *volumes.SummaryOpts) error { c.SingleFetch = *single; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	raw, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}
