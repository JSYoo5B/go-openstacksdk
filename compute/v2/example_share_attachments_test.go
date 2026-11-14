package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONShareAttachments(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_shareAttachments() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	server := flag.String("server", "", "server UUID")
	share := flag.String("share", "", "Manila share UUID")
	mode := flag.String("mode", "list", "list, get, create or delete")
	tag := flag.String("tag", "", "optional guest mount tag for create")
	strict := flag.Bool("strict", false, "return missing attachment errors on delete")
	flag.Parse()
	if *server == "" {
		log.Fatal("-server is required")
	}
	if *mode != "list" && *share == "" {
		log.Fatal("-share is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.97"))
	if err != nil {
		log.Fatal(err)
	}
	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *mode {
	case "list":
		for value, err := range compute.ShareAttachments.ListRecords(ctx, *server) {
			if err != nil {
				log.Fatal(err)
			}
			printJSONShareAttachments(value.Resource)
		}
	case "get":
		value, err := compute.ShareAttachments.GetRecord(ctx, *server, *share)
		if err != nil {
			log.Fatal(err)
		}
		printJSONShareAttachments(value.Resource)
	case "create":
		var options []shareattachments.CreateOption
		if *tag != "" {
			options = append(options, shareattachments.WithCreateAttribute("tag", *tag))
		}
		value, err := compute.ShareAttachments.CreateRecord(ctx, *server, *share, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONShareAttachments(value.Resource)
		fmt.Printf("HTTP %d\n", value.StatusCode)
	case "delete":
		value, err := compute.ShareAttachments.DeleteRecord(ctx, *server, *share,
			request.IgnoreMissing(!*strict))
		if err != nil {
			log.Fatal(err)
		}
		printJSONShareAttachments(value)
	default:
		log.Fatal("unknown -mode")
	}
}
