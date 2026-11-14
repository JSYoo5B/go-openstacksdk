package siteconnections_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func runRecordList(ctx context.Context, cloud string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	for row, err := range service.VPNSiteConnections.ListRecords(ctx,
		request.Query("status", "ACTIVE"),
		request.Query("interval", 30)) {
		if err != nil {
			return err
		}
		if err := json.NewEncoder(os.Stdout).Encode(row.Resource.Body); err != nil {
			return err
		}
	}
	return nil
}

func Example_recordList() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	timeout := flag.Duration("timeout", time.Minute, "whole list deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runRecordList(ctx, *cloud); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
