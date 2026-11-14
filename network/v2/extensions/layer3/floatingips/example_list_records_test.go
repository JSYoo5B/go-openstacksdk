package floatingips_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
)

func Example_listRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	dns := flag.String("dns-name", "app", "local DNS filter")
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
	rows, err := service.API.FloatingIPs.AllRecords(ctx,
		floatingips.WithRecordListQuery("status", "ACTIVE"),
		floatingips.WithRecordListQuery("dns_name", *dns))
	if err != nil {
		log.Fatal(err)
	}
	values := make([]any, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.Resource)
	}
	raw, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}
