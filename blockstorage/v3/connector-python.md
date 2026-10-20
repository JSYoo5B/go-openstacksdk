# Connector Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/init_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.InitializeWithConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeaders`

- The owned direct-ID action sends the exact Source envelope and arbitrary nested connector JSON, including wwnns arrays/nqn/extensions and nil/null/false/zero/empty values. Source is captured before custom MarshalJSON and JSON is snapshotted before options, so callbacks cannot change transmitted caller data; headers are concrete guarded inputs and native typed connector APIs remain unchanged.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNestedInputAndOptionSnapshots`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNullEmptyAndRawConnectionInfo`.
- Action200..399 follows Source _action. Initialization requires the JSON response connection_info field and retains its raw object/null/scalar/array without Source cast coercion; termination accepts opaque success bytes. Actual discovery/action body/header/status are separate receipts, never Resource state or invented connector input.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorResponseShapeAndSourceStatuses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.
- The shared Cinder negotiation honors a selected version without GET, otherwise selects server bounds and Source Volume3.71 ceiling with actual discovery receipts. Guards preserve accepted/rejected physical read/Close/cancel/source errors, sticky route changes, ordinary native retries and original HTTP causes while rejecting mutated retry JSON/version. Source and typed options remain library-owned.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorCapturesSourceBeforeMarshalAndOptions`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorPhysicalFaultsKeepReceiptsAndCauses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNativeRetryOwnsBodyAndVersion`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.

Python/Go differences:

- Go accepts a literal validated ID and a map[string]any JSON connector instead of mutable Python Resource; arbitrary connector contents remain untyped. Unsupported Go JSON values fail before HTTP.
- Go returns explicit action/discovery receipts; Python initialize returns its unchecked connection_info JSON value and terminate returns None. Neither proves local attachment completion.
- The SDK reuses finite Cinder discovery when no client version is selected, capped at Source Volume3.71; explicitly selected versions retain existing session/client policy. JSON/route/version ownership and physical handling causes are guarded Go behavior.

### `python:block_storage/v3/terminate_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.TerminateWithConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeaders`

- The owned direct-ID action sends the exact Source envelope and arbitrary nested connector JSON, including wwnns arrays/nqn/extensions and nil/null/false/zero/empty values. Source is captured before custom MarshalJSON and JSON is snapshotted before options, so callbacks cannot change transmitted caller data; headers are concrete guarded inputs and native typed connector APIs remain unchanged.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNestedInputAndOptionSnapshots`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNullEmptyAndRawConnectionInfo`.
- Action200..399 follows Source _action. Initialization requires the JSON response connection_info field and retains its raw object/null/scalar/array without Source cast coercion; termination accepts opaque success bytes. Actual discovery/action body/header/status are separate receipts, never Resource state or invented connector input.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorResponseShapeAndSourceStatuses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.
- The shared Cinder negotiation honors a selected version without GET, otherwise selects server bounds and Source Volume3.71 ceiling with actual discovery receipts. Guards preserve accepted/rejected physical read/Close/cancel/source errors, sticky route changes, ordinary native retries and original HTTP causes while rejecting mutated retry JSON/version. Source and typed options remain library-owned.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorCapturesSourceBeforeMarshalAndOptions`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorPhysicalFaultsKeepReceiptsAndCauses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNativeRetryOwnsBodyAndVersion`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.

Python/Go differences:

- Go accepts a literal validated ID and a map[string]any JSON connector instead of mutable Python Resource; arbitrary connector contents remain untyped. Unsupported Go JSON values fail before HTTP.
- Go returns explicit action/discovery receipts; Python initialize returns its unchecked connection_info JSON value and terminate returns None. Neither proves local attachment completion.
- The SDK reuses finite Cinder discovery when no client version is selected, capped at Source Volume3.71; explicitly selected versions retain existing session/client policy. JSON/route/version ownership and physical handling causes are guarded Go behavior.

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
        volumes.WithConnectorHeader("X-Client-Request-Id", "connector-init"))
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
