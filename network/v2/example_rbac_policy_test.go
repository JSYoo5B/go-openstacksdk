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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printPolicyRbacPolicy(value *rbacpolicies.RBACPolicyRecord) {
	if value == nil {
		fmt.Println("not found")
		return
	}
	raw, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}

func Example_rbacPolicy() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	action := flag.String("action", "list", "list/get/find")
	id := flag.String("id", "", "RBAC policy literal ID")
	fields := flag.String("fields", "", "comma-separated requested fields")
	objectType := flag.String("object-type", "network", "RBAC object type")
	targetProject := flag.String("target-project", "", "target project filter")
	strict := flag.Bool("strict", false, "find reports missing resource")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := rbacpolicies.New(network.RawClient())
	var selected []string
	if *fields != "" {
		selected = strings.Split(*fields, ",")
	}
	switch *action {
	case "get":
		value, err := api.GetRecord(ctx, *id,
			rbacpolicies.WithRecordGetFields(selected...))
		if err != nil {
			log.Fatal(err)
		}
		printPolicyRbacPolicy(value)
	case "list":
		options := []rbacpolicies.RecordListOption{
			request.Query("object_type", *objectType),
		}
		if len(selected) != 0 {
			options = append(options, request.Query("fields", selected))
		}
		if *targetProject != "" {
			options = append(options, request.Query("target_project_id", *targetProject))
		}
		for value, err := range api.ListRecords(ctx, options...) {
			if err != nil {
				log.Fatal(err)
			}
			printPolicyRbacPolicy(value)
		}
	case "find":
		options := []rbacpolicies.RecordFindOption{
			request.IgnoreMissing(!*strict),
			request.Query("object_type", *objectType),
		}
		if len(selected) != 0 {
			options = append(options, request.Query("fields", selected))
		}
		if *targetProject != "" {
			options = append(options, request.Query("target_project_id", *targetProject))
		}
		value, err := api.FindRecord(ctx, *id, options...)
		if err != nil {
			log.Fatal(err)
		}
		printPolicyRbacPolicy(value)
	default:
		log.Fatal("unknown action")
	}
}
