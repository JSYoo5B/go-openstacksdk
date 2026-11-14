package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"log"
	"os"
	"time"
)

func Example_cloudRouterInterfaces() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	router := flag.String("router", "", "Source router dictionary JSON")
	kind := flag.String("interface-type", "null", "JSON discriminator")
	flag.Parse()
	if *router == "" {
		log.Fatal("-router required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var connect []sdk.ConnectionOption
	if *cloud != "" {
		connect = append(connect, sdk.WithCloud(*cloud))
	}
	conn, err := sdk.Connect(ctx, connect...)
	if err != nil {
		log.Fatal(err)
	}
	result, err := conn.ListRouterInterfaces(ctx, json.RawMessage(*router), network.WithRouterInterfaceTypeJSON(json.RawMessage(*kind)))
	if err != nil {
		if result != nil {
			fmt.Fprintf(os.Stderr, "observed ports=%d\n", len(result.Inventory))
		}
		log.Fatal(err)
	}
	fmt.Println(string(result.Value))
}
