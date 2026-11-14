# Cloud Container Search Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
print([c.name for c in conn.search_containers(name="logs-*", filters={"count": 1})])
print(conn.search_containers(filters="[].name"))
print(conn.search_containers(filters="length(@)"))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    matched, err := conn.SearchContainers(ctx,
        func(c *objectstorage.ContainerSearchOpts) error { c.Name = "logs-*"; return nil },
        objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"count":1}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s page_status=%d\n", row.Resource.Body["name"], row.StatusCode)
    }
    names, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

## References

- [Cloud Container Search Test](cloud_container_search_test.go)
