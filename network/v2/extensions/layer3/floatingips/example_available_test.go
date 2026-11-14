package floatingips_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func Example_available() {
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
	result, err := service.API.FloatingIPs.FindAvailableIPRecord(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var value any
	if result != nil {
		value = result.Resource
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}
