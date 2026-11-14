# Python Read Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    selected, err := service.API.Networks.GetWithFields(ctx, os.Getenv("NETWORK_ID"),
        "id", "name", "is_admin_state_up", "revision_number")
    if err != nil { panic(err) }
    fmt.Printf("network: %s %s revision=%d\n", selected.ID, selected.Name, selected.RevisionNumber)

    if id := os.Getenv("DELETE_NETWORK_ID"); id != "" {
        var options []networks.DeleteOption
        if value := os.Getenv("DELETE_NETWORK_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, request.WithNeutronDeleteRevision(revision))
        }
        if err := service.API.Networks.DeleteWithOptions(ctx, id, options...); err != nil { panic(err) }
    }
}
```

## References

- [..](../README.md)
