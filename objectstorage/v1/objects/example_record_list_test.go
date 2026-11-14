package objects_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func runRecordList(ctx context.Context, cloud, container string, prefix *string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	swift, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		return err
	}
	var options []objects.RecordListOption
	if prefix != nil {
		options = append(options, request.Custom(func(c *objects.RecordListOpts) error { v := *prefix; c.Prefix = &v; return nil }))
	}
	rows := make([]*resource.RawResource, 0)
	for record, err := range swift.Objects.ListRecords(ctx, container, options...) {
		if err != nil {
			if record != nil {
				fmt.Fprintf(os.Stderr, "page status=%d body=%s\n", record.StatusCode, record.Envelope)
			}
			return err
		}
		rows = append(rows, record.Resource)
	}
	output, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func Example_recordList() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	container := flag.String("container", "assets", "literal container name")
	prefix := flag.String("prefix", "", "optional prefix")
	flag.Parse()
	var selectedPrefix *string
	flag.Visit(func(value *flag.Flag) {
		if value.Name == "prefix" {
			selectedPrefix = prefix
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := runRecordList(ctx, *cloud, *container, selectedPrefix); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
