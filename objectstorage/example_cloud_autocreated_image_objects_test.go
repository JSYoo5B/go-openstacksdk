package objectstorage_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func runCloudAutocreatedImageObjects(ctx context.Context, cloud, container, prefix string, tasks bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	swift, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	result, err := swift.Cloud.DeleteAutocreatedImageObjects(ctx,
		func(c *objectstorage.AutocreatedImageObjectsOpts) error { c.ImageAPIUseTasks = tasks; return nil },
		func(c *objectstorage.AutocreatedImageObjectsOpts) error { v := container; c.Container = &v; return nil },
		func(c *objectstorage.AutocreatedImageObjectsOpts) error {
			v := prefix
			c.SegmentPrefix = &v
			return nil
		})
	if err != nil {
		return err
	}
	fmt.Printf("deleted=%t observed=%d\n", result.Deleted, len(result.Observations))
	return nil
}

func Example_cloudAutocreatedImageObjects() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	container := flag.String("container", "images", "Swift container")
	prefix := flag.String("prefix", "", "server object prefix")
	tasks := flag.Bool("tasks", false, "enable image task upload cleanup")
	timeout := flag.Duration("timeout", time.Minute, "whole workflow timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runCloudAutocreatedImageObjects(ctx, *cloud, *container, *prefix, *tasks); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
