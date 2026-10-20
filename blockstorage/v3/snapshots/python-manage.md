# Python Manage

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/manage_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ManageRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithManageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithManageRecordMetadata`

- Manage checks the advertised and selected-default 3.8 capability to choose the modern or legacy route, then posts with the Source fixed 3.65 version. All five snapshot envelope fields are present including optional nulls; discovery and snapshot physical responses are separate.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManageRouteCapabilityAndFixedVersion`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementRetriesOwnFixedVersionAndHeaders`.
- Raw reference and metadata plus optional text are cloned before callbacks; response fields are projected from the actual response without echoing submitted attributes, and missing response id remains Source null. Invalid JSON fails before HTTP; accepted descriptor errors retain the actual response and never expose a successful Resource.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManageResponseMissingIDAndOptionsOwned`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementGuardsAndAcceptedProjectionError`.

Python/Go differences:

- Go accepts a string volume ID and finite raw JSON reference/metadata with concrete optional text/header options instead of arbitrary Python kwargs or Resource objects. Unknown formal kwargs cannot be injected.
- The SDK reuses bounded guarded Cinder discovery for supports_microversion; actual discovery observations accompany results rather than Python adapter endpoint-data caching. The Source mismatch between legacy route selection and fixed 3.65 submission is intentionally preserved.
- The result is an owned SnapshotRecord, not a mutable Python Snapshot. Generic dirty lifecycle/cache and automatic Connection location remain global SDK work; no live cloud or storage backend completion is claimed.

### `python:block_storage/v3/unmanage_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.UnmanageRecord`

- Unmanage POSTs the literal os-unmanage:null action to one escaped snapshot member with the Source fixed 3.65 header, ignores response JSON and returns actual opaque acknowledgement. A physical 404 remains a native error, never ignored missing or a successful state transition.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotUnmanageOpaqueAckFixedHeaderAndMissing`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementRetriesOwnFixedVersionAndHeaders`.
- Nil options, invalid identifiers and cancelled context stop before HTTP; selected API/source and physical response guards remain terminal rather than falling back or inventing a successful resource.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementGuardsAndAcceptedProjectionError`.

Python/Go differences:

- Python returns None; Go returns an owned ManagementReceipt for actual opaque body/header/status. Submission acknowledgement does not prove a completed backend transition.
- Go uses a validated literal string ID escaped as one path segment and owned header options; arbitrary Python Resource/URI/session objects are not accepted. Source fixed 3.65 action behavior is preserved without version discovery.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=("manage", "unmanage"), required=True)
parser.add_argument("--identity", required=True)
parser.add_argument("--reference", default='{"source-name":"backend-snapshot"}')
args = parser.parse_args()
with openstack.connect() as conn:
    if args.action == "manage":
        row = conn.block_storage.manage_snapshot(volume_id=args.identity, ref=json.loads(args.reference))
        print(json.dumps(row.to_dict(), indent=2))
    else:
        conn.block_storage.unmanage_snapshot(args.identity)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    action := flag.String("action", "", "manage or unmanage")
    identity := flag.String("identity", "", "volume ID for manage, snapshot ID for unmanage")
    reference := flag.String("reference", `{"source-name":"backend-snapshot"}`, "backend reference JSON")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    switch *action {
    case "manage":
        row, err := service.Snapshots.ManageRecord(ctx, *identity, json.RawMessage(*reference))
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    case "unmanage":
        receipt, err := service.Snapshots.UnmanageRecord(ctx, *identity)
        if err != nil { log.Fatal(err) }
        fmt.Println(receipt.StatusCode)
    default:
        log.Fatal("action must be manage or unmanage")
    }
}
```
