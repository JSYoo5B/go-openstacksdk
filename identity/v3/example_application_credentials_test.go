package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func printRecordApplicationCredentials(v *applicationcredentials.ApplicationCredentialRecord) {
	if v == nil {
		fmt.Println("null")
		return
	}
	data, err := json.MarshalIndent(v.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_applicationCredentials() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	user := flag.String("user", "", "literal Keystone user ID")
	name := flag.String("name", "ci", "exact credential name or ID")
	credential := flag.String("credential", "", "credential ID to GET")
	rule := flag.String("rule", "", "access rule ID to GET")
	create := flag.Bool("create", false, "create an application credential")
	deleteCredential := flag.String("delete-credential", "", "credential ID to delete")
	deleteRule := flag.String("delete-rule", "", "access rule ID to delete")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.ApplicationCredentials
	scope, err := api.InUser(ctx, resource.ID(*user))
	if err != nil {
		log.Fatal(err)
	}
	rules, err := api.AccessRules(ctx, resource.ID(*user))
	if err != nil {
		log.Fatal(err)
	}
	for v, err := range scope.ListRecords(ctx,
		applicationcredentials.WithRecordListFilter("name", *name),
		request.MaxItems(100)) {
		if err != nil {
			log.Fatal(err)
		}
		printRecordApplicationCredentials(v)
	}
	found, err := scope.FindRecord(ctx, *name)
	if err != nil {
		log.Fatal(err)
	}
	printRecordApplicationCredentials(found)
	if *credential != "" {
		v, err := scope.GetRecord(ctx, *credential)
		if err != nil {
			log.Fatal(err)
		}
		printRecordApplicationCredentials(v)
	}
	for v, err := range rules.ListRecords(ctx) {
		if err != nil {
			log.Fatal(err)
		}
		printRecordApplicationCredentials(v)
	}
	if *rule != "" {
		v, err := rules.GetRecord(ctx, *rule)
		if err != nil {
			log.Fatal(err)
		}
		printRecordApplicationCredentials(v)
	}
	if *create {
		v, err := scope.CreateRecord(ctx, *name,
			request.Field("description", ""),
			request.Field("unrestricted", false),
			request.Field("roles", []map[string]string{{"name": "member"}}))
		if err != nil {
			log.Fatal(err)
		}
		printRecordApplicationCredentials(v)
	}
	if *deleteCredential != "" {
		if err := scope.DeleteRecord(ctx, *deleteCredential); err != nil {
			log.Fatal(err)
		}
	}
	if *deleteRule != "" {
		if err := rules.DeleteRecord(ctx, *deleteRule,
			request.IgnoreMissing(false)); err != nil {
			log.Fatal(err)
		}
	}
}
