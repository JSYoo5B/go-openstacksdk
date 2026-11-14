package domains_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONDomainConfig(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_domainConfig() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	domain := flag.String("domain", "", "parent Domain ID")
	action := flag.String("action", "", "create, get, update or delete")
	attrs := flag.String("attrs", `{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}`, "create/update JSON attributes")
	strict := flag.Bool("strict-delete", false, "do not ignore missing config")
	flag.Parse()
	if *domain == "" {
		log.Fatal("domain is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var row *domains.DomainConfigRecord
	switch *action {
	case "create", "update":
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(*attrs), &fields); err != nil {
			log.Fatal(err)
		}
		if fields == nil {
			log.Fatal("attrs must be a JSON object")
		}
		option := domains.WithDomainConfigOpts(domains.DomainConfigOpts{Attributes: fields})
		if *action == "create" {
			row, err = service.Domains.CreateConfigRecord(ctx, *domain, option)
		} else {
			row, err = service.Domains.UpdateConfigRecord(ctx, *domain, option)
		}
	case "get":
		row, err = service.Domains.GetConfigRecord(ctx, *domain)
	case "delete":
		result, err := service.Domains.DeleteConfigRecord(ctx, *domain, request.Custom(func(c *domains.DomainConfigOpts) error { v := !*strict; c.IgnoreMissing = &v; return nil }))
		if err != nil {
			log.Fatal(err)
		}
		printJSONDomainConfig(result.Deleted)
		return
	default:
		log.Fatal("action must be create, get, update or delete")
	}
	if err != nil {
		log.Fatal(err)
	}
	printJSONDomainConfig(row.Resource)
}
