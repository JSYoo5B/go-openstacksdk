# Cloud Object Get Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
parser.add_argument("--chunk-size", type=int, default=1024)
output = parser.add_mutually_exclusive_group()
output.add_argument("--outfile")
output.add_argument("--discard-content", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    result = conn.get_object(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
        resp_chunk_size=args.chunk_size,
        outfile="" if args.discard_content else args.outfile,
    )
    if result is None:
        print("missing", file=sys.stderr)
    else:
        headers, body = result
        print(json.dumps(headers, indent=2), file=sys.stderr)
        if body is not None:
            sys.stdout.write(body)
```

```go
package main

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

func run(ctx context.Context, cloud, container, object, query, outfile string,
    stream, discard bool, chunkSize int) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    options := []objectstorage.ObjectGetOption{
        func(c *objectstorage.ObjectGetOpts) error { c.QueryString = query; return nil },
        func(c *objectstorage.ObjectGetOpts) error { c.Stream = stream; return nil },
        func(c *objectstorage.ObjectGetOpts) error { v := chunkSize; c.ChunkSize = &v; return nil },
    }
    if outfile != "" { options = append(options, objectstorage.WithObjectGetOutfile(outfile)) }
    if discard {
        if outfile != "" { return fmt.Errorf("choose outfile or discard-content") }
        options = append(options, objectstorage.WithoutObjectGetContent())
    }
    result, getErr := conn.GetObjectText(ctx, container, object, options...)
    if result != nil {
        summary := struct {
            Headers map[string]string `json:"headers"`
            Missing bool `json:"missing"`
            BytesWritten int64 `json:"bytes_written"`
        }{result.Headers, result.Missing, result.BytesWritten}
        encoded, err := json.MarshalIndent(summary, "", "  ")
        if err != nil { return err }
        fmt.Fprintln(os.Stderr, string(encoded))
    }
    if getErr != nil { return getErr }
    if result != nil && result.Body != nil {
        written, err := io.WriteString(os.Stdout, *result.Body)
        if err != nil { return err }
        if written != len(*result.Body) { return io.ErrShortWrite }
    }
    return nil
}

func main() {
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
    if err := run(ctx, *cloud, *container, *object, *query, *outfile,
        *stream, *discard, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [cloud get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L512)
- [Proxy.get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288)
- [Install](../docs/install.md)
- [Python Cloud Get Test](v1/objects/python_cloud_get_test.go)
- [Cloud Object Get Test](cloud_object_get_test.go)
