package identity_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func Example_cloudProjectGroupGet() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	kind := flag.String("kind", "project", "project or group")
	name := flag.String("name", "", "literal name or ID")
	domain := flag.String("domain-id", "", "optional domain UUID")
	filters := flag.String("filters", "", "optional JSON filters")
	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	var value json.RawMessage
	switch *kind {
	case "project":
		var options []identity.ProjectGetOption
		if *domain != "" {
			options = append(options, identity.WithProjectGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithProjectGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetProjectRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	case "group":
		var options []identity.GroupGetOption
		if *domain != "" {
			options = append(options, identity.WithGroupGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithGroupGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetGroupRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	default:
		log.Fatal("-kind must be project or group")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
