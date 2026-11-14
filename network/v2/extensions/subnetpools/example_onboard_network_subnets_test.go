package subnetpools_test

import (
	"context"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"log"
)

func Example_onboardNetworkSubnets() {
	pool := flag.String("pool", "", "subnet pool ID")
	network := flag.String("network", "", "network ID")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.API.SubnetPools.OnboardNetworkSubnetsRecord(ctx, *pool, *network)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("HTTP %d\n%s\n", result.StatusCode, result.Value)
}
