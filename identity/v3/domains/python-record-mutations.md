# Python Record Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), required=True)
parser.add_argument("--identity", default="")
parser.add_argument("--name", default="example-domain")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("--identity is required for update")
with openstack.connect(cloud=args.cloud) as conn:
    attrs = {"description": None, "is_enabled": False,
             "options": {"immutable": False}}
    if args.mode == "create":
        row = conn.identity.create_domain(name=args.name, **attrs)
    else:
        row = conn.identity.update_domain(args.identity, **attrs)
        unchanged = conn.identity.update_domain(args.identity)
        print(json.dumps(unchanged.to_dict(), indent=2))
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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create or update")
    id := flag.String("identity", "", "Domain ID for update")
    name := flag.String("name", "example-domain", "Domain create name")
    flag.Parse()
    if *mode != "create" && *mode != "update" { log.Fatal("mode must be create or update") }
    if *mode == "update" && *id == "" { log.Fatal("identity is required for update") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    options := []domains.RecordMutationOption{
        request.Field("description", nil),
        request.Field("is_enabled", false),
        request.Field("options", map[string]any{"immutable": false}),
    }
    var row *domains.DomainRecord
    if *mode == "create" {
        options = append(options, request.Field("name", *name))
        row, err = service.Domains.CreateRecord(ctx, options...)
    } else {
        row, err = service.Domains.UpdateRecord(ctx, *id, options...)
    }
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource)
    if *mode == "update" {
        unchanged, err := service.Domains.UpdateRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printJSON(unchanged.Resource)
    }
}
```

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
