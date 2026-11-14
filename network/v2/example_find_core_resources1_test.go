package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResourceFindCoreResources1(kind string, value *resource.RawResource) {
	fmt.Println(kind)
	if value == nil {
		fmt.Println("null")
		return
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}

func Example_networksFindRecord() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	networkName := flag.String("network", "app-net", "network ID or exact name")
	portName := flag.String("port", "", "optional port ID or exact name")
	subnetName := flag.String("subnet", "", "optional subnet ID or exact name")
	project := flag.String("project", "", "optional project filter")
	strict := flag.Bool("strict", false, "fail on logical absence")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if *networkName != "" {
		options := []networks.RecordFindOption{request.IgnoreMissing(!*strict)}
		if *project != "" {
			options = append(options, request.Query("project_id", *project))
		}
		result, err := service.Networks.FindRecord(ctx, *networkName, options...)
		if err != nil {
			log.Fatal(err)
		}
		if result == nil {
			printResourceFindCoreResources1("network", nil)
		} else {
			printResourceFindCoreResources1("network", result.Resource)
		}
	}
	if *portName != "" {
		options := []ports.RecordFindOption{request.IgnoreMissing(!*strict)}
		if *project != "" {
			options = append(options, request.Query("project_id", *project))
		}
		result, err := service.Ports.FindRecord(ctx, *portName, options...)
		if err != nil {
			log.Fatal(err)
		}
		if result == nil {
			printResourceFindCoreResources1("port", nil)
		} else {
			printResourceFindCoreResources1("port", result.Resource)
		}
	}
	if *subnetName != "" {
		options := []subnets.RecordFindOption{request.IgnoreMissing(!*strict)}
		if *project != "" {
			options = append(options, request.Query("project_id", *project))
		}
		result, err := service.Subnets.FindRecord(ctx, *subnetName, options...)
		if err != nil {
			log.Fatal(err)
		}
		if result == nil {
			printResourceFindCoreResources1("subnet", nil)
		} else {
			printResourceFindCoreResources1("subnet", result.Resource)
		}
	}
}
