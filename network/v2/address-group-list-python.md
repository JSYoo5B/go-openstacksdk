# Address Group List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
rows = list(conn.network.address_groups(
    project_id="PROJECT_ID",
    addresses=["10.0.0.0/24"],
    max_items=20,
))
for row in rows:
    print(row.to_dict())
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    project := flag.String("project", "", "optional server project filter")
    address := flag.String("address", "", "optional exact one-address list filter")
    cap := flag.Int("max-items", 20, "raw row cap; zero is unbounded")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []addressgroups.RecordListOption{
        request.MaxItems(*cap),
    }
    if *project != "" {
        options = append(options, addressgroups.WithRecordListQuery("project_id", *project))
    }
    if *address != "" {
        options = append(options, addressgroups.WithRecordListQuery("addresses", []string{*address}))
    }
    for row, err := range network.SecurityAddressGroups.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        raw, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
    }
}
```

## References

- [Install](../../docs/install.md)
- [Address Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
