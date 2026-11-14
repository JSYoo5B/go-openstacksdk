package networkipavailabilities_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"os"
	"time"
)

func runRecords(ctx context.Context, cloud, identity, name string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	api := service.NetworkIPAvailabilities
	if identity != "" {
		record, err := api.FindRecord(ctx, identity, request.IgnoreMissing(false))
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(record.Resource.Body)
	}
	var options []networkipavailabilities.RecordListOption
	if name != "" {
		options = append(options, request.Query("network_name", name))
	}
	for record, err := range api.ListRecords(ctx, options...) {
		if err != nil {
			return err
		}
		if err := json.NewEncoder(os.Stdout).Encode(record.Resource.Body); err != nil {
			return err
		}
	}
	return nil
}

func Example_records() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	identity := flag.String("find", "", "literal member ID or Source fallback id/name")
	name := flag.String("network-name", "", "server-side network name filter")
	timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runRecords(ctx, *cloud, *identity, *name); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
