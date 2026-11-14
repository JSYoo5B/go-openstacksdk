# Python Record List

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--prefix")
parser.add_argument("--mode", choices=("proxy", "cloud"), default="proxy")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "cloud":
        values = conn.list_objects(args.container, prefix=args.prefix)
    else:
        query = {} if args.prefix is None else {"prefix": args.prefix}
        values = conn.object_store.objects(args.container, **query)
    print(json.dumps([value.to_dict() for value in values], indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, container string, prefix *string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    var options []objects.RecordListOption
    if prefix != nil { options = append(options, request.Custom(func(c *objects.RecordListOpts) error { v := *prefix; c.Prefix = &v; return nil })) }
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
    if err != nil { return err }
    fmt.Println(string(output))
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    prefix := flag.String("prefix", "", "optional prefix")
    flag.Parse()
    var selectedPrefix *string
    flag.Visit(func(value *flag.Flag) { if value.Name == "prefix" { selectedPrefix = prefix } })
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, selectedPrefix); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [`conn.object_store.objects(container, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L271)
- [`conn.list_objects(container, full_listing=True, prefix=None)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L347-L366)
- [Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)
- [Install](../../../docs/install.md)
- [Python Record List Test](python_record_list_test.go)
