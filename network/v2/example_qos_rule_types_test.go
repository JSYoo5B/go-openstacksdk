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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printTypeQosRuleTypes(value *ruletypes.QoSRuleTypeRecord) {
	if value == nil {
		fmt.Println("not found")
		return
	}
	data, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_qosRuleTypes() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	operation := flag.String("operation", "list", "get/find/list")
	name := flag.String("name", "bandwidth_limit", "literal rule type name")
	fields := flag.String("fields", "", "comma-separated fields")
	typeFilter := flag.String("type", "", "server type filter")
	allRules := flag.Bool("all-rules", false, "server all_rules control")
	allSupported := flag.Bool("all-supported", false, "server all_supported control")
	maxItems := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
	strict := flag.Bool("strict", false, "find reports missing")
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
	api := service.QoSRuleTypes
	var selected []string
	if *fields != "" {
		selected = strings.Split(*fields, ",")
	}
	switch *operation {
	case "get":
		value, err := api.GetRecord(ctx, *name, ruletypes.WithRecordGetFields(selected...))
		if err != nil {
			log.Fatal(err)
		}
		printTypeQosRuleTypes(value)
	case "find":
		value, err := api.FindRecord(ctx, *name, request.IgnoreMissing(!*strict))
		if err != nil {
			log.Fatal(err)
		}
		printTypeQosRuleTypes(value)
	case "list":
		options := []ruletypes.RecordListOption{request.MaxItems(*maxItems)}
		if len(selected) != 0 {
			options = append(options, request.Query("fields", selected))
		}
		if *typeFilter != "" {
			options = append(options, request.Query("type", *typeFilter))
		}
		if *allRules {
			options = append(options, request.Query("all_rules", true))
		}
		if *allSupported {
			options = append(options, request.Query("all_supported", true))
		}
		for value, err := range api.ListRecords(ctx, options...) {
			if err != nil {
				log.Fatal(err)
			}
			printTypeQosRuleTypes(value)
		}
	default:
		log.Fatal("unknown operation")
	}
}
