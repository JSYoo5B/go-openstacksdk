# Python Source Mapping

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "find", "update"), required=True)
parser.add_argument("--id", default="example-mapping")
args = parser.parse_args()
rules = [{"local": [{"user": {"name": "{0}", "type": "ephemeral"}}],
          "remote": [{"type": "REMOTE_USER", "any_one_of": ["alice"]}]}]
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.identity.create_mapping(id=args.id, rules=rules,
                                           schema_version="2.0")
    elif args.mode == "update":
        row = conn.identity.update_mapping(args.id, schema_version="2.0")
    else:
        row = conn.identity.find_mapping(args.id)
    print(json.dumps(None if row is None else row.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create, find or update")
    id := flag.String("id", "example-mapping", "Mapping ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    var row *federation.SourceMappingRecord
    switch *mode {
    case "create":
        rules := []any{map[string]any{
            "local": []any{map[string]any{"user": map[string]any{"name": "{0}", "type": "ephemeral"}}},
            "remote": []any{map[string]any{"type": "REMOTE_USER", "any_one_of": []string{"alice"}}},
        }}
        row, err = service.Federation.CreateSourceMappingRecord(ctx,
            request.Field("id", *id),
            request.Field("rules", rules),
            request.Field("schema_version", "2.0"))
    case "update":
        row, err = service.Federation.UpdateSourceMappingRecord(ctx, *id,
            request.Field("schema_version", "2.0"))
    case "find":
        row, err = service.Federation.FindSourceMappingRecord(ctx, *id)
    default:
        log.Fatal("mode must be create, find or update")
    }
    if err != nil { log.Fatal(err) }
    var value any
    if row != nil { value = row.Resource }
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Install](../../../docs/install.md)
