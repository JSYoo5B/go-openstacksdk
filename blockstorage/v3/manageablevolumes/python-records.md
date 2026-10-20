# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/manageable_volumes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.WithRecordListQuery`

- The SDK defaults details to false and reads the underscore summary/detail route while extracting the literal hyphenated response collection; nullable Source descriptors cast reference dictionaries, safe_to_manage JSON truthiness and integer size, preserving physical page evidence and unknown wire fields.
  Tests: `blockstorage/v3/manageablevolumes/python_records_test.go:TestPythonManageableVolumesListDefaultsDetailsAndViews`.
- Owned options classify server query and local Body filters, cap raw rows before local matching, preserve the unconsumed-generator JMESPath return shape, reject bound route argument collisions and stop on selected-client changes. No version discovery or inferred cluster minimum is added.
  Tests: `blockstorage/v3/manageablevolumes/python_records_test.go:TestPythonManageableVolumesFiltersCapsProxyAndGuard`.

Python/Go differences:

- Go exposes finite UTF-8 JSON and concrete typed details, headers/version and paging options, rather than arbitrary Python objects or runtime classes. Amounts are nonnegative Go integers; map option order is deterministic and ordinary unknown kwargs are ignored after the JSON option boundary.
- Records own Resource/Wire/Envelope/Header/StatusCode independently; Source generator single-use state, generic Resource/session cache and automatic Connection location are outside this profile. Unsafe or cyclic continuation routes fail with actual response evidence.
- JMESPath operates on the opaque unconsumed generator; ProxyListRecords preserves polymorphic Value and only materializes JSON when serializable. Typed ListRecords requires that callers select ProxyListRecords for truthy string expressions.

### `python:block_storage/v3/manageable_snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots.WithRecordListQuery`

- The SDK defaults details to false and reads the underscore summary/detail route while extracting the literal hyphenated response collection; nullable Source descriptors cast reference dictionaries, safe_to_manage JSON truthiness and integer size, preserving physical page evidence and unknown wire fields.
  Tests: `blockstorage/v3/manageablesnapshots/python_records_test.go:TestPythonManageableSnapshotsListDefaultsDetailsAndViews`.
- Owned options classify server query and local Body filters, cap raw rows before local matching, preserve the unconsumed-generator JMESPath return shape, reject bound route argument collisions and stop on selected-client changes. No version discovery or inferred cluster minimum is added.
  Tests: `blockstorage/v3/manageablesnapshots/python_records_test.go:TestPythonManageableSnapshotsFiltersCapsProxyAndGuard`.

Python/Go differences:

- Go exposes finite UTF-8 JSON and concrete typed details, headers/version and paging options, rather than arbitrary Python objects or runtime classes. Amounts are nonnegative Go integers; map option order is deterministic and ordinary unknown kwargs are ignored after the JSON option boundary.
- Records own Resource/Wire/Envelope/Header/StatusCode independently; Source generator single-use state, generic Resource/session cache and automatic Connection location are outside this profile. Unsafe or cyclic continuation routes fail with actual response evidence.
- JMESPath operates on the opaque unconsumed generator; ProxyListRecords preserves polymorphic Value and only materializes JSON when serializable. Typed ListRecords requires that callers select ProxyListRecords for truthy string expressions.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

with openstack.connect() as conn:
    for row in conn.block_storage.manageable_volumes(details=False, host="host@backend", limit=20, max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
    for row in conn.block_storage.manageable_snapshots(details=True, host="host@backend", max_items=20):
        print(json.dumps(row.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablesnapshots"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.ManageableVolumes.ListRecords(ctx,
        manageablevolumes.WithRecordListDetails(false),
        manageablevolumes.WithRecordListQuery("host", "host@backend"),
        manageablevolumes.WithRecordListLimit(20), manageablevolumes.WithRecordListMaxItems(20)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    for row, err := range service.ManageableSnapshots.ListRecords(ctx,
        manageablesnapshots.WithRecordListDetails(true),
        manageablesnapshots.WithRecordListQuery("host", "host@backend"),
        manageablesnapshots.WithRecordListMaxItems(20)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```
