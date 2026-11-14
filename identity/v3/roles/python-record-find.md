# Python Record Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--domain", required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    role = conn.identity.find_role(
        args.identity, ignore_missing=False, domain_id=args.domain,
        skip_cache=True, error_message="Role lookup failed",
    )
    print(json.dumps(role.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "Role exact name or ID")
    domain := flag.String("domain", "", "explicit domain ID")
    flag.Parse()
    if *identity == "" || *domain == "" { log.Fatal("identity and domain are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.Roles.FindRecord(ctx, *identity,
        request.IgnoreMissing(false),
        request.Query("domain_id", *domain),
        request.Query("skip_cache", true),
        request.Query("error_message", "Role lookup failed"))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
