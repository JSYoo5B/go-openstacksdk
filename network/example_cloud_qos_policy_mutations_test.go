package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func printJSONCloudQosPolicyMutations(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
func Example_cloudQosPolicyMutations() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "create", "create or update")
	identity := flag.String("identity", "", "update: literal ID or exact name")
	attrs := flag.String("attrs", "{}", "JSON cloud mutation kwargs")
	flag.Parse()
	if *mode != "create" && *mode != "update" {
		log.Fatal("mode must be create or update")
	}
	if *mode == "update" && *identity == "" {
		log.Fatal("update identity is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var result *network.QoSPolicyMutation
	option := network.WithQoSPolicyMutationJSON(json.RawMessage(*attrs))
	if *mode == "create" {
		result, err = service.CreateQoSPolicyRecord(ctx, option)
	} else {
		result, err = service.UpdateQoSPolicyRecord(ctx, *identity, option)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Policy == nil {
		printJSONCloudQosPolicyMutations(nil)
		return
	}
	printJSONCloudQosPolicyMutations(result.Policy.Resource)
}
