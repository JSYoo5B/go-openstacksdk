package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func runRouterReadRecords(ctx context.Context, cloud, mode, identity string) (any, error) {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return nil, err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return nil, err
	}
	switch mode {
	case "find":
		value, err := network.Routers.FindRecord(
			ctx, identity, request.IgnoreMissing(false),
		)
		if err != nil || value == nil {
			return nil, err
		}
		return value.Resource, nil
	case "list":
		values, err := network.Routers.AllRecords(
			ctx, routers.WithRecordListQuery("name", identity),
			request.MaxItems(100),
		)
		if err != nil {
			return nil, err
		}
		rows := make([]*resource.RawResource, 0, len(values))
		for _, value := range values {
			rows = append(rows, value.Resource)
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
}

func Example_routerReadRecords() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	identity := flag.String("router", "edge", "router name or literal ID")
	mode := flag.String("mode", "list", "find or list")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := runRouterReadRecords(ctx, *cloud, *mode, *identity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
