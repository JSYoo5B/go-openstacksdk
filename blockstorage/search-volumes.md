# Search Volumes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
)

func Search(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumes(ctx,
        blockstorage.SearchVolumesRequest{NameOrID: "data*"},
        blockstorage.WithVolumeSearchFilters(json.RawMessage(
            `{"metadata":{"owner":"worker"},"is_bootable":false}`)))
    if err != nil { return err }
    fmt.Println(string(result.Value), len(result.Volumes), len(result.Pages))

    ids, err := conn.SearchVolumes(ctx, blockstorage.SearchVolumesRequest{},
        blockstorage.WithVolumeSearchExpression("[?size >= `10`].id"))
    if err != nil { return err }
    fmt.Println(string(ids.Value))

    volume, err := conn.GetVolume(ctx,
        blockstorage.GetVolumeRequest{NameOrID: "data"})
    if err != nil { return err }
    if volume.Value == nil { return nil }
    fmt.Println(string(volume.Value))
    return nil
}
```

```python
volumes = conn.search_volumes(
    "data*", filters={"metadata": {"owner": "worker"}, "is_bootable": False}
)
ids = conn.search_volumes(filters="[?size >= `10`].id")
volume = conn.get_volume("data")
```
