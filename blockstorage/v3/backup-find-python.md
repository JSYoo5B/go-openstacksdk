# Backup Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/find_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRecordFindPaginated`

- Find(resource.ID) sends one GET without fallback and Find(resource.Name) lists backups/detail without a name query and matches locally.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupFindHasNoIDThenNameFallback`.
- FindRecord sends GET first and only clean400/403/404 permit fallback; default detail and explicit summary both add the Source name query.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordFallbackCodesAndDetailsDefaults`.
- Full exact string ID-or-name observation detects late duplicates and HTTP failures; raw numeric IDs are not equated with string identities and missing defaults to nil with an explicit strict option.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordExactIdentityAndLateDuplicateOrError`, `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordMissingAndAcceptedTranslation`.
- Owned Resource projects all23 declared Body fields plus nullable location, preserving untyped IDs/timestamps/large JSON, Source bool/int/list/dict descriptors, ordered project aliases and original Wire/Envelope/header/status.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordGetFirstOwnsSourceViewAndRawEvidence`, `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordMissingAndAcceptedTranslation`.
- Source explicit session default bypasses discovery without downgrading; absent default uses shared finite Cinder discovery and min(server maximum,3.64), keeping discovery proof separate and shared client policy unchanged.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordDiscoveryDefaultAndExplicitVersion`, `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordDiscoveryRejectionPreservesPhysicalCauses`.
- Explicit Go query extensions classify declared server filters versus Body-local JSON filters, preserve Requests iterable query expansion, snapshot caller maps and cap raw rows before local filtering.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordQueryClassesOwnedOptsAndRawCap`.
- Shared linked and short-marker paging observes the complete search by default; explicit first-page/cap and offset policies prevent unintended continuation and retain scope.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordPagingShortMarkersAndOffset`.
- Every member/list400..599 rejection retains physical Read/Close faults and exact native proof; accepted list faults keep ResponseError evidence, with no observation retry, suppression or fallback after mixed causes.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordRejectedAndAcceptedPhysicalCauses`.
- Invalid identities/options/JSON, source-changing callbacks and cancellation fail before member/discovery HTTP, preserving caller cancellation causes.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordSelectionAndOptionFailures`.
- Short-page fallback derives the marker from the retained original last-row fields decoded by the shared pager, independently of public Resource projection; untyped large integer IDs become exact Requests marker strings without changing Resource/Wire ID JSON.
  Tests: `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordPagingShortMarkersAndOffset`, `blockstorage/v3/backups/python_find_test.go:TestPythonBackupFindRecordMarkerUsesRetainedUntypedWireID`.

Python/Go differences:

- BackupResourceRecord owns a declared Python-shaped Resource and literal Wire/Envelope/header/status rather than a mutable session-bound Python Resource. Nullable location is null; unknown fields remain in Wire.
- Query/Headers/MaxItems/Paginated are explicit Go extensions beyond the pinned proxy ignore_missing/details arguments. The JSON-domain Resource query/descriptor policy is documented separately.
- Go performs finite discovery per call only when no selected client default exists, rather than reusing a Python adapter discovery cache. Shared client microversion remains unchanged.
- Initial offset opts into the guarded shared offset pager and disables marker fallback; unsafe origin/path/filter changes, negative limits and mixed marker/offset are rejected. Dictionary links compatibility is a Go extension.
- Non-object actual values do not match an object local filter instead of raising a Python membership/get type error. Invalid UTF-8 response bytes fail with HTTP evidence.
- The four Python Backup MetadataMixin child-route operations remain unresolved because stock Cinder does not register /backups/{id}/metadata. This row covers only find_backup.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
backup = conn.block_storage.find_backup(
    "nightly", ignore_missing=False, details=False
)
print(backup.id, backup.name, backup.created_at)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
)

func main() {
    identity := flag.String("backup", "nightly", "Backup name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    backup, err := service.Backups.FindRecord(ctx, *identity,
        backups.WithRecordFindIgnoreMissing(false),
        backups.WithRecordFindDetails(false),
    )
    if err != nil { log.Fatal(err) }
    body, err := json.MarshalIndent(backup.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Backup Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py)
