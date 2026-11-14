package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func runQosMinimumBandwidthRecords(ctx context.Context, cloud, mode, parent, identity string) (any, error) {
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
		value, err := network.QoSRules.FindMinimumBandwidthRecord(
			ctx, parent, identity, request.IgnoreMissing(false),
		)
		if err != nil || value == nil {
			return nil, err
		}
		return value.Resource, nil
	case "get":
		value, err := network.QoSRules.GetMinimumBandwidthRecord(ctx, parent, identity)
		if err != nil {
			return nil, err
		}
		return value.Resource, nil
	case "list":
		values, err := network.QoSRules.AllMinimumBandwidthRecords(
			ctx, parent, request.Query("direction", "egress"),
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

func Example_qosMinimumBandwidthRecords() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	parent := flag.String("policy", "", "literal QoS policy ID")
	identity := flag.String("rule", "", "literal rule ID or Find identity")
	mode := flag.String("mode", "list", "find, get or list")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := runQosMinimumBandwidthRecords(ctx, *cloud, *mode, *parent, *identity)
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
