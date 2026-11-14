# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.regions(
        parent_region_id="Root", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Regions.ListRecords(ctx,
        regions.WithRecordListQuery("parent_region_id", "Root"),
        regions.WithRecordListQuery("description", "operations"),
        request.Limit(100),
        request.MaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s\n", data)
    }
}
```

## References

- [Region](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/region.py)
