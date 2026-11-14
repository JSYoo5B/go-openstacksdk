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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func runTrunkAddressFind(ctx context.Context, cloud, kind, identity, project string) (*resource.RawResource, error) {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return nil, err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "trunk":
		options := []trunks.RecordFindOption{
			request.IgnoreMissing(false),
			request.Query("is_admin_state_up", false),
		}
		if project != "" {
			options = append(options, request.Query("project_id", project))
		}
		record, err := network.Trunks.FindRecord(ctx, identity, options...)
		if err != nil || record == nil {
			return nil, err
		}
		return record.Resource, nil
	case "address-group":
		options := []addressgroups.RecordFindOption{
			request.IgnoreMissing(false),
		}
		if project != "" {
			options = append(options, request.Query("project_id", project))
		}
		record, err := network.SecurityAddressGroups.FindRecord(ctx, identity, options...)
		if err != nil || record == nil {
			return nil, err
		}
		return record.Resource, nil
	default:
		return nil, fmt.Errorf("unknown kind %q", kind)
	}
}

func Example_trunkAddressFind() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	kind := flag.String("kind", "trunk", "trunk or address-group")
	identity := flag.String("identity", "application-trunk", "exact ID or name")
	project := flag.String("project", "", "optional project filter")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := runTrunkAddressFind(ctx, *cloud, *kind, *identity, *project)
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
