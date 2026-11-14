# Share Attachments Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--share", default="")
parser.add_argument("--mode", choices=("list", "get", "create", "delete"), default="list")
parser.add_argument("--tag", default=None)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_default_microversion="2.97") as conn:
    if args.mode == "list":
        for value in conn.compute.share_attachments(args.server):
            print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "get":
        value = conn.compute.get_share_attachment(args.server, args.share)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "create":
        attrs = {} if args.tag is None else {"tag": args.tag}
        value = conn.compute.create_share_attachment(args.server, args.share, **attrs)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.compute.delete_share_attachment(args.server, args.share)
        print(json.dumps({"delete_requested": args.share}, indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "server UUID")
    share := flag.String("share", "", "Manila share UUID")
    mode := flag.String("mode", "list", "list, get, create or delete")
    tag := flag.String("tag", "", "optional guest mount tag for create")
    strict := flag.Bool("strict", false, "return missing attachment errors on delete")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    if *mode != "list" && *share == "" { log.Fatal("-share is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.97"))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "list":
        for value, err := range compute.ShareAttachments.ListRecords(ctx, *server) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "get":
        value, err := compute.ShareAttachments.GetRecord(ctx, *server, *share)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    case "create":
        var options []shareattachments.CreateOption
        if *tag != "" {
            options = append(options, shareattachments.WithCreateAttribute("tag", *tag))
        }
        value, err := compute.ShareAttachments.CreateRecord(ctx, *server, *share, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d\n", value.StatusCode)
    case "delete":
        value, err := compute.ShareAttachments.DeleteRecord(ctx, *server, *share,
            request.IgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printJSON(value)
    default:
        log.Fatal("unknown -mode")
    }
}
```

## References

- [Compute Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3422)
- [ShareMapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_share.py)
- [Install](../../docs/install.md)
- [Server Shares](https://github.com/openstack/nova/blob/31.0.0/nova/policies/server_shares.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/server_shares.py)
- [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/server_shares.py)
- [Base](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
