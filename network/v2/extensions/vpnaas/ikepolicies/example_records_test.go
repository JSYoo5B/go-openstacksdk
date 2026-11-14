package ikepolicies_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"os"
	"time"
)

func runRecords(ctx context.Context, cloud, name, units string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	var options []ikepolicies.RecordListOption
	if name != "" {
		options = append(options, request.Query("name", name))
	}
	if units != "" {
		options = append(options, request.Query("units", units))
	}
	for record, err := range service.VPNIKEPolicies.ListRecords(ctx, options...) {
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
	name := flag.String("name", "", "server-side policy name filter")
	units := flag.String("units", "", "local top-level units Body filter")
	timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runRecords(ctx, *cloud, *name, *units); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
