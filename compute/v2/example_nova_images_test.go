package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printImageNovaImages(value *images.ImageRecord) {
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

func Example_novaImages() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("id", "", "Nova legacy image ID")
	owner := flag.String("set-owner", "", "metadata owner to POST")
	keys := flag.String("delete-keys", "", "comma-separated metadata keys")
	clear := flag.Bool("clear-metadata", false, "replace metadata with empty object")
	remove := flag.Bool("delete-image", false, "delete image strictly")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.35"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.Images
	for value, err := range api.ListRecords(ctx,
		request.Custom(func(c *images.ListOpts) error { v := false; c.Details = &v; return nil }), request.MaxItems(100)) {
		if err != nil {
			log.Fatal(err)
		}
		printImageNovaImages(value)
	}
	if *id == "" {
		return
	}
	value, err := api.GetRecord(ctx, *id)
	if err != nil {
		log.Fatal(err)
	}
	printImageNovaImages(value)
	value, err = api.FindRecord(ctx, *id, request.IgnoreMissing(false))
	if err != nil {
		log.Fatal(err)
	}
	printImageNovaImages(value)
	value, err = api.FetchMetadata(ctx, *id)
	if err != nil {
		log.Fatal(err)
	}
	printImageNovaImages(value)
	if *owner != "" {
		value, err = api.SetMetadata(ctx, *id, map[string]any{"owner": *owner})
		if err != nil {
			log.Fatal(err)
		}
		printImageNovaImages(value)
	}
	if *keys != "" {
		if err := api.DeleteMetadata(ctx, *id,
			images.WithDeleteMetadataKeys(strings.Split(*keys, ",")...)); err != nil {
			log.Fatal(err)
		}
	}
	if *clear {
		if err := api.DeleteMetadata(ctx, *id); err != nil {
			log.Fatal(err)
		}
	}
	if *remove {
		if err := api.DeleteRecord(ctx, *id,
			request.IgnoreMissing(false)); err != nil {
			log.Fatal(err)
		}
	}
}
