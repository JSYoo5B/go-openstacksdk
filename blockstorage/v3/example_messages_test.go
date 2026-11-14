package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func printMessageMessages(value *resource.RawResource) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}

func Example_messages() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	version := flag.String("microversion", "3.37", "Cinder selected version")
	id := flag.String("message", "", "optional message ID")
	project := flag.String("project", "", "optional server project filter")
	remove := flag.Bool("delete", false, "strictly delete the selected message")
	flag.Parse()
	if *remove && *id == "" {
		log.Fatal("-delete requires -message")
	}
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
	api := service.Messages
	if *id != "" {
		if *remove {
			if err := api.DeleteRecord(ctx, *id,
				request.IgnoreMissing(false)); err != nil {
				log.Fatal(err)
			}
			return
		}
		value, err := api.GetRecord(ctx, *id)
		if err != nil {
			log.Fatal(err)
		}
		printMessageMessages(value.Resource)
		return
	}
	options := []messages.RecordListOption{}
	if *project != "" {
		options = append(options, request.Query("project_id", *project))
	}
	for value, err := range api.ListRecords(ctx, options...) {
		if err != nil {
			log.Fatal(err)
		}
		printMessageMessages(value.Resource)
	}
}
