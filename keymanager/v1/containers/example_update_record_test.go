package containers_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
)

func runUpdateRecord(ctx context.Context, cloud, id, attributes string) error {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal([]byte(attributes), &attrs); err != nil {
		return err
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		return err
	}
	result, err := service.Containers.UpdateRecord(ctx, id,
		containers.WithUpdateRecordOpts(containers.UpdateRecordOpts{Attributes: attrs}))
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Committed  bool                       `json:"committed"`
		StatusCode int                        `json:"status_code"`
		Resource   map[string]json.RawMessage `json:"resource"`
	}{result.Committed, result.StatusCode, result.Resource.Body})
}

func Example_updateRecord() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	id := flag.String("container", "c1", "literal container ID")
	attributes := flag.String("attrs-json", "{}", "closed JSON attributes; empty object makes no PUT")
	timeout := flag.Duration("timeout", time.Minute, "whole request deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runUpdateRecord(ctx, *cloud, *id, *attributes); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
