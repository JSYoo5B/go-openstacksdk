# Connector Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
connector = {
    "host": "compute-1",
    "wwnns": ["1000000000000001", "1000000000000002"],
    "nqn": "nqn.2014-08.org.example:compute-1",
    "do_local_attach": False,
    "extensions": {"optional": None, "number": 0, "devices": []},
}
connection_info = conn.block_storage.init_volume_attachment("volume-id", connector)
conn.block_storage.terminate_volume_attachment("volume-id", connector)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    connector := map[string]any{
        "host": "compute-1",
        "wwnns": []string{"1000000000000001", "1000000000000002"},
        "nqn": "nqn.2014-08.org.example:compute-1",
        "do_local_attach": false,
        "extensions": map[string]any{"optional": nil, "number": 0, "devices": []any{}},
    }
    initialized, err := cinder.Volumes.InitializeWithConnector(ctx, "volume-id", connector,
        request.Header("X-Client-Request-Id", "connector-init"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("connection_info=%s action_status=%d microversion=%s discovery=%d\n",
        initialized.ConnectionInfo, initialized.Response.StatusCode,
        initialized.Microversion, len(initialized.Discovery))

    terminated, err := cinder.Volumes.TerminateWithConnector(ctx, "volume-id", connector)
    if err != nil { log.Fatal(err) }
    fmt.Printf("terminate action_status=%d body=%q\n",
        terminated.Response.StatusCode, terminated.Response.Body)
}
```

## References

- [Python Connector Test](volumes/python_connector_test.go)
