package objectstorage_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func runCloudObjectStream(ctx context.Context, cloud, container, object, query string, chunkSize int) (err error) {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	stream := conn.StreamObjectChunks(ctx, container, object,
		func(c *objectstorage.ObjectStreamOpts) error { v := chunkSize; c.ChunkSize = &v; return nil },
		func(c *objectstorage.ObjectStreamOpts) error { c.QueryString = query; return nil },
	)
	defer func() {
		err = errors.Join(err, stream.Close())
		if receipt := stream.Receipt(); receipt != nil {
			fmt.Fprintf(os.Stderr, "status=%d encoding=%q\n",
				receipt.StatusCode, receipt.Header.Get("Content-Encoding"))
		}
	}()
	for chunk, readErr := range stream.Chunks() {
		var writeErr error
		if len(chunk) != 0 {
			var written int
			written, writeErr = os.Stdout.Write(chunk)
			if writeErr == nil && written != len(chunk) {
				writeErr = io.ErrShortWrite
			}
		}
		if readErr != nil || writeErr != nil {
			return errors.Join(readErr, writeErr)
		}
	}
	return stream.Err()
}

func Example_cloudObjectStream() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	container := flag.String("container", "assets", "literal container name")
	object := flag.String("object", "logs/part.bin", "literal object name")
	query := flag.String("query", "", "ignored by the pinned cloud stream_object")
	chunkSize := flag.Int("chunk-size", 1024, "decoded bytes per chunk; 1..16777216")
	timeout := flag.Duration("timeout", 5*time.Minute, "whole download deadline")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := runCloudObjectStream(ctx, *cloud, *container, *object, *query, *chunkSize); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
