# Cloud Object Search Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
objects = conn.search_objects("assets", name="logs-*", filters={"content_length": 5})
print([obj.name for obj in objects])
print(conn.search_objects("assets", filters="[].name"))
print(conn.search_objects("assets", filters="length(@)"))
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
    matched, err := conn.SearchObjects(ctx, "assets",
        func(c *objectstorage.ObjectSearchOpts) error { c.Name = "logs-*"; return nil },
        objectstorage.WithObjectSearchFiltersJSON(json.RawMessage(`{"content_length":5}`)))
    if err != nil { log.Fatal(err) }
    for _, row := range matched.Records {
        fmt.Printf("name=%s bytes=%s page_status=%d\n",
            row.Resource.Body["name"], row.Resource.Body["content_length"], row.StatusCode)
    }
    names, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("[].name"))
    if err != nil { log.Fatal(err) }
    total, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("length(@)"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
```

## References

- [Cloud Object Search Test](cloud_object_search_test.go)
- [Connection Swift Object Search Test](../connectiontest/connection_swift_object_search_test.go)
