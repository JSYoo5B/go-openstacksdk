package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func runCloudServerConsole1(ctx context.Context, cloud, server, suppliedServer, length string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	var options []compute.ServerConsoleOption
	if suppliedServer != "" {
		options = append(options, compute.WithServerConsoleServerJSON(json.RawMessage(suppliedServer)))
	}
	if length != "" {
		options = append(options, compute.WithServerConsoleLengthJSON(json.RawMessage(length)))
	}
	result, err := conn.GetServerConsole(ctx, server, options...)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("console result is missing")
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Output      json.RawMessage `json:"output"`
		HasOutput   bool            `json:"has_output"`
		Unsupported bool            `json:"unsupported"`
		StatusCode  int             `json:"status_code"`
	}{result.Value, result.HasOutput, result.Unsupported, result.Action.StatusCode})
}

func Example_cloudServerConsole1() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	server := flag.String("server", "vm", "literal name or ID; ignored when server-json is supplied")
	suppliedServer := flag.String("server-json", "", "owned server dictionary, for example {\"id\":\"s1\"}")
	length := flag.String("length-json", "", "optional literal JSON length; empty flag omits it")
	timeout := flag.Duration("timeout", time.Minute, "whole lookup and action deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runCloudServerConsole1(ctx, *cloud, *server, *suppliedServer, *length); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
