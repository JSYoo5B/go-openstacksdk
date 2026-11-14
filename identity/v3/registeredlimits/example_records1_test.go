package registeredlimits_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_registeredLimitsAllRecords() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	identity, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// service_id/resource_name use Keystone query parameters.
	// description is a declared Body filter, evaluated locally.
	rows, err := identity.RegisteredLimits.AllRecords(ctx,
		registeredlimits.WithRecordListQuery("service_id", "compute-service-id"),
		registeredlimits.WithRecordListQuery("resource_name", "cores"),
		registeredlimits.WithRecordListQuery("description", "team quota"),
		request.MaxItems(100),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		encoded, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", encoded)
	}
}
