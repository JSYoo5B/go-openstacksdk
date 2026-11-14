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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func printJSONSecurityFind(value *resource.RawResource) {
	if value == nil {
		fmt.Println("null")
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_securityFind() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	group := flag.String("group", "", "security group exact name or ID")
	rule := flag.String("rule", "", "security group rule exact ID or name")
	project := flag.String("project", "", "optional project_id query")
	groupFilter := flag.String("security-group", "", "optional security_group_id rule query")
	strict := flag.Bool("strict", false, "return an error when missing")
	flag.Parse()
	if *group == "" && *rule == "" {
		log.Fatal("-group or -rule is required")
	}
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
	if *group != "" {
		options := []groups.RecordFindOption{request.IgnoreMissing(!*strict)}
		if *project != "" {
			options = append(options, request.Query("project_id", *project))
		}
		value, err := service.SecurityGroups.FindRecord(ctx, *group, options...)
		if err != nil {
			log.Fatal(err)
		}
		if value == nil {
			printJSONSecurityFind(nil)
		} else {
			printJSONSecurityFind(value.Resource)
		}
	}
	if *rule != "" {
		options := []rules.RecordFindOption{request.IgnoreMissing(!*strict)}
		if *project != "" {
			options = append(options, request.Query("project_id", *project))
		}
		if *groupFilter != "" {
			options = append(options, request.Query("security_group_id", *groupFilter))
		}
		value, err := service.SecurityRules.FindRecord(ctx, *rule, options...)
		if err != nil {
			log.Fatal(err)
		}
		if value == nil {
			printJSONSecurityFind(nil)
		} else {
			printJSONSecurityFind(value.Resource)
		}
	}
}
