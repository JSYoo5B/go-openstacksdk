# Python Quota Class

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--action", choices=("get", "update"), required=True)
parser.add_argument("--name", default="default")
parser.add_argument("--attrs", default='{"volumes":0}')
parser.add_argument("--microversion", default="")
args = parser.parse_args()
config = {"cloud": args.cloud}
if args.microversion:
    config["block_storage_api_version"] = args.microversion
with openstack.connect(**config) as conn:
    if args.action == "get":
        row = conn.block_storage.get_quota_class_set(args.name)
    else:
        row = conn.block_storage.update_quota_class_set(args.name, **json.loads(args.attrs))
    print(json.dumps(row.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "", "get or update")
    name := flag.String("name", "default", "quota class name")
    attrs := flag.String("attrs", `{"volumes":0}`, "update JSON attributes")
    version := flag.String("microversion", "", "explicit service version, empty uses configured/default policy")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    options := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { options = append(options, sdk.WithMicroversion(sdk.BlockStorage, *version)) }
    conn, err := sdk.Connect(ctx, options...)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := quotasets.New(service.RawClient())
    var row *quotasets.QuotaClassRecord
    switch *action {
    case "get":
        row, err = api.GetQuotaClassRecord(ctx, request.Custom(func(c *quotasets.QuotaClassGetOpts) error { v := *name; c.Name = &v; return nil }))
    case "update":
        var fields map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*attrs), &fields); err != nil { log.Fatal(err) }
        if fields == nil { log.Fatal("attrs must be a JSON object") }
        row, err = api.UpdateQuotaClassRecord(ctx, *name,
            quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: fields}))
    default:
        log.Fatal("action must be get or update")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../../../docs/install.md)
