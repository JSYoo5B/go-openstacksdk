package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_attachments() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	volume := flag.String("volume", "", "volume ID")
	instance := flag.String("instance", "", "optional attaching server ID")
	version := flag.String("microversion", "3.71", "Cinder microversion supported by the deployment")
	reserve := flag.Bool("reserve", false, "create a volume attachment reservation")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.BlockStorage, *version))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.Attachments
	for value, err := range api.ListRecords(ctx,
		request.Query("volume_id", *volume),
		request.MaxItems(100)) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
	if *reserve {
		options := []attachments.RecordCreateOption{}
		if *instance != "" {
			options = append(options, request.Field("instance", *instance))
		}
		value, err := api.CreateRecord(ctx, *volume, options...)
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
