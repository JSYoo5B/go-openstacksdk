# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "find", "proxy"), default="list")
parser.add_argument("--extension", default="qos")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--jmes", default="")
parser.add_argument("--list-base-path")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        query = {"skip_cache": True, "error_message": "Extension lookup failed"}
        if args.list_base_path is not None:
            query["list_base_path"] = args.list_base_path
        value = conn.network.find_extension(
            args.extension, ignore_missing=not args.strict, **query
        )
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
    else:
        query = {}
        if args.mode == "list":
            query["alias"] = args.extension
        else:
            query["jmespath_filters"] = args.jmes
        result = conn.network.extensions(**query)
        if isinstance(result, types.GeneratorType):
            for value in result:
                print(json.dumps(value.to_dict(), indent=2))
        else:
            print(result)
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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list, find or proxy")
    identity := flag.String("extension", "qos", "alias or exact display name")
    strict := flag.Bool("strict", false, "return an error when missing")
    expression := flag.String("jmes", "", "deprecated Proxy expression")
    listPath := flag.String("list-base-path", "", "Find fallback service-relative path")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := network.Extensions
    switch *mode {
    case "find":
        options := []extensions.RecordFindOption{
            request.IgnoreMissing(!*strict),
            request.Query("skip_cache", true),
            request.Query("error_message", "Extension lookup failed"),
        }
        if *listPath != "" {
            options = append(options, extensions.WithRecordFindListBasePath(*listPath))
        }
        value, err := api.FindRecord(ctx, *identity, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil); return }
        printJSON(value.Resource)
    case "list":
        for value, err := range api.ListRecords(
            ctx, request.Query("alias", *identity),
        ) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "proxy":
        result, err := api.ProxyListRecords(ctx, extensions.WithRecordListJMESPath(*expression))
        if err != nil { log.Fatal(err) }
        if result.JSON != nil { fmt.Println(string(result.JSON)); return }
        generator, ok := result.Value.(*extensions.RecordListGenerator)
        if !ok { fmt.Printf("opaque expression result: %T\n", result.Value); return }
        for value, err := range generator.Records {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    default:
        log.Fatal("mode must be list, find or proxy")
    }
}
```

## References

- [Extensions](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py#L100-L127)
- [Install](../../../docs/install.md)
- [Extension class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py#L16)
- [find/list proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1588-L1626)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489)
- [list element cast](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L113)
