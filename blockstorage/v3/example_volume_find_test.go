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

func Example_volumeFind() {
	identity := flag.String("volume", "data", "Volume name or ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	volume, err := service.Volumes.FindRecord(ctx, *identity,
		request.IgnoreMissing(false),
		request.Custom(func(c *volumes.RecordFindOpts) error { v := false; c.Details = &v; return nil }),
		request.Custom(func(c *volumes.RecordFindOpts) error { v := true; c.AllProjects = &v; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	body, err := json.MarshalIndent(volume.Resource.Body, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}
