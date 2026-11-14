package rules_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func showDscpMarking(record *rules.DSCPMarkingRecord) {
	if record == nil {
		fmt.Println("missing")
		return
	}
	body, err := json.Marshal(record.Resource.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("policy=%s HTTP=%d body=%s\n", record.PolicyID, record.StatusCode, body)
}

func Example_dscpMarking() {
	policy := flag.String("policy", "", "parent policy ID")
	operation := flag.String("operation", "list", "list, get or find")
	rule := flag.String("rule", "", "literal rule ID or find identity")
	pageSize := flag.Int("page-size", 0, "server page size")
	maxItems := flag.Int("max-items", 0, "raw row cap; zero means no cap")
	mark := flag.Int("mark", -1, "optional local DSCP mark filter")
	strict := flag.Bool("strict", false, "find absence is an error")
	flag.Parse()
	if *policy == "" {
		log.Fatal("--policy is required")
	}
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	switch *operation {
	case "get":
		record, err := network.QoSRules.GetDSCPMarkingRecord(ctx, *policy, *rule)
		if err != nil {
			log.Fatal(err)
		}
		showDscpMarking(record)
	case "find":
		record, err := network.QoSRules.FindDSCPMarkingRecord(ctx, *policy, *rule,
			request.IgnoreMissing(!*strict))
		if err != nil {
			log.Fatal(err)
		}
		showDscpMarking(record)
	case "list":
		options := []rules.DSCPMarkingListOption{
			request.Limit(*pageSize),
			request.MaxItems(*maxItems),
		}
		if *mark >= 0 {
			options = append(options, request.Query("dscp_mark", *mark))
		}
		for record, err := range network.QoSRules.ListDSCPMarkingRecords(ctx, *policy, options...) {
			if err != nil {
				log.Fatal(err)
			}
			showDscpMarking(record)
		}
	default:
		log.Fatal("--operation must be list, get or find")
	}
}
