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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func Example_trunkList() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	name := flag.String("name", "", "server-side trunk name query")
	port := flag.String("port", "", "parent port ID query")
	cap := flag.Int("max-items", 0, "raw row cap; 0 means unlimited")
	single := flag.Bool("single-page", false, "observe first page only")
	version := flag.String("microversion", "", "optional selected service version")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	connOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
	if *version != "" {
		connOptions = append(connOptions, sdk.WithMicroversion(sdk.Network, *version))
	}
	conn, err := sdk.Connect(ctx, connOptions...)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []trunks.RecordListOption{
		request.MaxItems(*cap),
		request.Paginated(!*single),
	}
	if *name != "" {
		options = append(options, trunks.WithRecordListQuery("name", *name))
	}
	if *port != "" {
		options = append(options, trunks.WithRecordListQuery("port_id", *port))
	}
	rows, err := service.Trunks.AllRecords(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		body, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(body))
		fmt.Printf("HTTP %d request-id=%s\n", row.StatusCode,
			row.Header.Get("X-Openstack-Request-Id"))
	}
}
