package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func runCloudRouterInterfaceRemove(ctx context.Context, cloud, router, subnetJSON, portJSON string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	result, err := service.RemoveRouterInterface(ctx, router,
		network.WithRouterInterfaceRemoveSubnetID(json.RawMessage(subnetJSON)),
		network.WithRouterInterfaceRemovePortID(json.RawMessage(portJSON)))
	if result != nil {
		fmt.Fprintf(os.Stderr, "PUT status=%d request-id=%q\n",
			result.StatusCode, result.Header.Get("X-Request-Id"))
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(result.Value))
	return err
}

func Example_cloudRouterInterfaceRemove() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	router := flag.String("router", "", "literal router ID, not a name lookup")
	subnet := flag.String("subnet-json", "null", "subnet_id as JSON")
	port := flag.String("port-json", "null", "port_id as JSON; a truthy port wins")
	flag.Parse()
	if *router == "" {
		fmt.Fprintln(os.Stderr, "router is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runCloudRouterInterfaceRemove(ctx, *cloud, *router, *subnet, *port); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
