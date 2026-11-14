# Python Raw Get

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
parser.add_argument("--stream", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    response = conn.get_object_raw(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
    )
    try:
        print(f"status={response.status_code}", file=sys.stderr)
        for chunk in response.iter_content(chunk_size=32768):
            sys.stdout.buffer.write(chunk)
    finally:
        response.close()
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
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func run(ctx context.Context, cloud, container, object, query string, stream bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    response, requestErr := swift.Objects.GetObjectRaw(ctx, container, object,
        func(c *objects.RawObjectOpts) error { c.QueryString = query; return nil },
        func(c *objects.RawObjectOpts) error { c.Stream = stream; return nil },
    )
    if response == nil { return requestErr }
    fmt.Fprintf(os.Stderr, "status=%d\n", response.StatusCode)
    _, readErr := io.Copy(os.Stdout, response.Body)
    closeErr := response.Body.Close()
    return errors.Join(requestErr, readErr, closeErr)
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name; empty selects container")
    query := flag.String("query", "", "already encoded query string")
    stream := flag.Bool("stream", false, "transfer a guarded stream")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *stream); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [`get_object_raw(container, obj, query_string=None, stream=False)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L449-L484)
- [Install](../../../docs/install.md)
- [Python Raw Get Test](python_raw_get_test.go)
