# Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
container = conn.object_store.create_container(
    "assets", read_ACL=".r:*", is_content_type_detected=False,
    history_location="", metadata={"owner": "team", "disabled": False},
)
print(container.name, container.metadata)
conn.object_store.delete_container("assets")
conn.object_store.delete_container("assets", ignore_missing=False)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    created, err := swift.Containers.CreateRecord(ctx, "assets", map[string]any{
        "read_ACL": ".r:*",
        "is_content_type_detected": false,
        "history_location": "",
        "metadata": map[string]any{"owner": "team", "disabled": false},
    }, request.Header("X-Client-Request-Id", "create-assets"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT status=%d name=%s observed_metadata=%v\n",
        created.StatusCode, created.Resource.Body["name"], created.Metadata)

    deleted, err := swift.Containers.DeleteRecord(ctx, "assets")
    if err != nil { log.Fatal(err) }
    fmt.Printf("DELETE status=%d ignored_missing=%t\n", deleted.StatusCode, deleted.IgnoredMissing)

    // To reproduce ignore_missing=False, select the concrete option explicitly.
    _, err = swift.Containers.DeleteRecord(ctx, "assets",
        request.IgnoreMissing(false))
    if err != nil { fmt.Println("strict deletion:", err) }
}
```

## References

- [Python Record Test](python_record_test.go)
