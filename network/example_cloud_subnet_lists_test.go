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

func runCloudSubnetLists(ctx context.Context, cloud, mode, name, filters string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	options := []network.SubnetReadOption{network.WithSubnetReadFiltersJSON(json.RawMessage(filters))}
	var result *network.SubnetRecords
	switch mode {
	case "list":
		result, err = conn.ListSubnetRecords(ctx, options...)
	case "search":
		result, err = conn.SearchSubnetRecords(ctx, name, options...)
	default:
		return fmt.Errorf("mode must be list or search")
	}
	if result != nil && result.LastResponse != nil {
		fmt.Fprintf(os.Stderr, "last observed resource/error status=%d request-id=%q partial-resources=%d\n",
			result.LastResponse.StatusCode, result.LastResponse.Header.Get("X-Request-Id"), len(result.Inventory))
	}
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result.Value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(encoded))
	return err
}

func Example_cloudSubnetLists() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	mode := flag.String("mode", "list", "list or search")
	name := flag.String("name", "", "literal search name, including UUID or glob characters")
	filters := flag.String("filters", `{}`, "Source cloud filter JSON")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runCloudSubnetLists(ctx, *cloud, *mode, *name, *filters); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
