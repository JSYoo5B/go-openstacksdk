# Cloud Containers Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
container = conn.create_container("photos", public=True)
if container is not None:
    print(container.name, container.read_ACL)

updated = conn.set_container_access("photos", "private")
observed = conn.set_container_access("photos", "public", refresh=True)
print(conn.get_container_access("photos"))
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorage(ctx)
    if err != nil { log.Fatal(err) }

    created, err := swift.Cloud.CreateContainer(ctx, "photos",
        func(c *objectstorage.ContainerCreateOpts) error { c.Public = true; return nil })
    if err != nil { log.Fatal(err) }
    if created.Container != nil {
        fmt.Printf("existing=%t name=%s ACL=%s\n", created.Existing,
            created.Container.Resource.Body["name"],
            created.Container.Resource.Body["read_ACL"])
    }

    updated, err := swift.Cloud.SetContainerAccess(ctx, "photos", "private")
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d requested=%s\n", updated.Acknowledgement.StatusCode, updated.Requested)

    observed, err := swift.Cloud.SetContainerAccess(ctx, "photos", "public",
        func(c *objectstorage.ContainerAccessOpts) error { c.Refresh = true; return nil })
    if err != nil { log.Fatal(err) }
    fmt.Printf("POST=%d HEAD=%d observed ACL=%s\n",
        observed.Acknowledgement.StatusCode, observed.Refreshed.StatusCode,
        observed.Container.Resource.Body["read_ACL"])

    access, err := swift.Cloud.GetContainerAccess(ctx, "photos")
    if err != nil { log.Fatal(err) }
    fmt.Printf("access=%s HEAD=%d\n", access.Access, access.Container.StatusCode)
}
```

## References

- [Cloud Containers Test](cloud_containers_test.go)
- [Cloud Records Test](v1/containers/cloud_records_test.go)
