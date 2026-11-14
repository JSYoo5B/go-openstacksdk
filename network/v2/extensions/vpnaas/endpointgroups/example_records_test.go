package endpointgroups_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_records() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	flag.Parse()
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
	values, err := service.API.VPNEndpointGroups.AllRecords(ctx,
		request.Query("endpoints", []string{"10.0.0.0/24"}))
	if err != nil {
		log.Fatal(err)
	}
	for _, value := range values {
		raw, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(raw))
	}
}
