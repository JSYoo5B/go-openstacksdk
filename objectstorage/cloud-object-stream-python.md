# Cloud Object Stream Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--chunk-size", type=int, default=1024)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    stream = conn.stream_object(
        args.container, args.object,
        query_string=args.query, resp_chunk_size=args.chunk_size,
    )
    try:
        for chunk in stream:
            sys.stdout.buffer.write(chunk)
    finally:
        stream.close()
```

```go
package main

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

func run(ctx context.Context, cloud, container, object, query string, chunkSize int) (err error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
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
            if writeErr == nil && written != len(chunk) { writeErr = io.ErrShortWrite }
        }
        if readErr != nil || writeErr != nil {
            return errors.Join(readErr, writeErr)
        }
    }
    return stream.Err()
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name")
    query := flag.String("query", "", "ignored by the pinned cloud stream_object")
    chunkSize := flag.Int("chunk-size", 1024, "decoded bytes per chunk; 1..16777216")
    timeout := flag.Duration("timeout", 5*time.Minute, "whole download deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [cloud stream_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L484)
- [Object.stream/_download](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L319)
- [Install](../docs/install.md)
- [Python Cloud Stream Test](v1/objects/python_cloud_stream_test.go)
- [Cloud Object Stream Test](cloud_object_stream_test.go)
