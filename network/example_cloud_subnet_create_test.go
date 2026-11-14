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

func runCloudSubnetCreate(ctx context.Context, cloud, networkName, keywords string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	result, err := conn.CreateSubnetRecord(ctx, networkName,
		network.WithSubnetCreateJSON(json.RawMessage(keywords)))
	if result != nil {
		if result.Network != nil {
			fmt.Fprintf(os.Stderr, "network lookup status=%d\n", result.Network.StatusCode)
		}
		if result.SubnetPool != nil {
			fmt.Fprintf(os.Stderr, "subnetpool lookup status=%d\n", result.SubnetPool.StatusCode)
		}
		if result.StatusCode != 0 {
			fmt.Fprintf(os.Stderr, "subnet POST status=%d request-id=%q\n",
				result.StatusCode, result.Header.Get("X-Request-Id"))
		}
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func Example_cloudSubnetCreate() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	networkName := flag.String("network", "", "literal network ID or exact name")
	keywords := flag.String("kwargs", `{"cidr":"10.61.0.0/24"}`, "Source cloud keyword JSON object")
	flag.Parse()
	if *networkName == "" {
		fmt.Fprintln(os.Stderr, "network is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runCloudSubnetCreate(ctx, *cloud, *networkName, *keywords); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
