package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func runQosPolicyRecords(ctx context.Context, cloud, mode, identity, project string) (any, error) {
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
		options := []policies.RecordFindOption{
			request.IgnoreMissing(false),
			request.Query("is_shared", false),
		}
		if project != "" {
			options = append(options, request.Query("project_id", project))
		}
		record, err := network.QoSPolicies.FindRecord(ctx, identity, options...)
		if err != nil || record == nil {
			return nil, err
		}
		return record.Resource, nil
	case "list":
		options := []policies.RecordListOption{
			request.Query("is_shared", false),
		}
		if project != "" {
			options = append(options, request.Query("project_id", project))
		}
		records, err := network.QoSPolicies.AllRecords(ctx, options...)
		if err != nil {
			return nil, err
		}
		values := make([]*resource.RawResource, len(records))
		for i, record := range records {
			values[i] = record.Resource
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
}

func Example_qosPolicyRecords() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	mode := flag.String("mode", "find", "find or list")
	identity := flag.String("identity", "application-qos", "exact policy ID or name")
	project := flag.String("project", "", "optional project filter")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := runQosPolicyRecords(ctx, *cloud, *mode, *identity, *project)
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
