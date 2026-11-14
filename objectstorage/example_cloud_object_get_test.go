package objectstorage_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func runCloudObjectGet(ctx context.Context, cloud, container, object, query, outfile string,
	stream, discard bool, chunkSize int) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	options := []objectstorage.ObjectGetOption{
		func(c *objectstorage.ObjectGetOpts) error { c.QueryString = query; return nil },
		func(c *objectstorage.ObjectGetOpts) error { c.Stream = stream; return nil },
		func(c *objectstorage.ObjectGetOpts) error { v := chunkSize; c.ChunkSize = &v; return nil },
	}
	if outfile != "" {
		options = append(options, objectstorage.WithObjectGetOutfile(outfile))
	}
	if discard {
		if outfile != "" {
			return fmt.Errorf("choose outfile or discard-content")
		}
		options = append(options, objectstorage.WithoutObjectGetContent())
	}
	result, getErr := conn.GetObjectText(ctx, container, object, options...)
	if result != nil {
		summary := struct {
			Headers      map[string]string `json:"headers"`
			Missing      bool              `json:"missing"`
			BytesWritten int64             `json:"bytes_written"`
		}{result.Headers, result.Missing, result.BytesWritten}
		encoded, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, string(encoded))
	}
	if getErr != nil {
		return getErr
	}
	if result != nil && result.Body != nil {
		written, err := io.WriteString(os.Stdout, *result.Body)
		if err != nil {
			return err
		}
		if written != len(*result.Body) {
			return io.ErrShortWrite
		}
	}
	return nil
}

func Example_cloudObjectGet() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	container := flag.String("container", "assets", "literal container name")
	object := flag.String("object", "logs/part.bin", "literal object name")
	query := flag.String("query", "", "ignored by pinned cloud get_object")
	stream := flag.Bool("stream", false, "ignored by pinned cloud get_object")
	outfile := flag.String("outfile", "", "SDK-owned binary output path")
	discard := flag.Bool("discard-content", false, "falsey non-None output: headers only")
	chunkSize := flag.Int("chunk-size", 1024, "output-only decoded chunk size; 1..16777216")
	timeout := flag.Duration("timeout", 5*time.Minute, "whole request deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runCloudObjectGet(ctx, *cloud, *container, *object, *query, *outfile,
		*stream, *discard, *chunkSize); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
