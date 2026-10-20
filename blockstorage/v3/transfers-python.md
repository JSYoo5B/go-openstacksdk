# Transfers Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/create_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateVolumeID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateNoSnapshots`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateExtension`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordCreateSuppressLegacySnapshots`

- CreateRecord discovers advertised3.55 support and selected default independently, uses /volume-transfers or /os-volume-transfer, caps an absent default at3.55 and preserves an explicit default including3.71.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferDiscoveryAndVersionRouting`.
- Declared attrs preserve omitted/null/false/zero/empty values; unknown attrs are discarded, explicit extensions are owned, response fields overlay the input seed and untyped dates/large integers retain exact JSON.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferCreateRawSnapshotsExtensionsAndResponseViews`.
- Pinned proxy preconstructs no_snapshots on Resource, so default legacy create retains that key; SuppressLegacySnapshots explicitly removes the wire key without erasing the input Resource seed.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferCreateRawSnapshotsExtensionsAndResponseViews`.
- Invalid options/JSON and selected-source mutations fail before discovery or transfer HTTP.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferInvalidOptionsAndStickySelection`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.
- WithRecordCreateExtension is an explicit Go extension for otherwise unknown Body keys. Default Source attrs ignore unknown keys. WithRecordCreateSuppressLegacySnapshots(true) intentionally fixes the pinned legacy no_snapshots cleanup defect.

### `python:block_storage/v3/get_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordReadHeaders`

- Transfers.Get sends GET os-volume-transfer/{id} and decodes name, volume_id and created_at.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- GetRecord sends bodyless GET to the discovered transfer route, seeds literal ID only for the declared Resource, retains timezone dates/unknown Wire fields and actual HTTP response proof.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Advertised bounds and explicit/default microversion selection remain independent without mutating the shared client.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferDiscoveryAndVersionRouting`.
- Invalid route IDs/options and sticky client mutations fail before HTTP.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferInvalidOptionsAndStickySelection`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Go decodes created_at only without a time zone suffix.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.

### `python:block_storage/v3/transfers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordListServerQuery`

- ListRecords defaults to the discovered /detail route, while WithRecordListDetails(false) selects summary. Source all_projects and unknown kwargs are discarded; declared Body attrs filter locally.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferListDefaultsSummaryQueriesAndRawFilters`.
- Only limit/marker are Source server queries; explicit Go server-query extensions can send all_tenants. Raw JSON fields, nested local filters and default Accept header are retained.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferListDefaultsSummaryQueriesAndRawFilters`.
- Lazy pages, raw-row cap before filtering, zero-limit/max-items hint, short-page marker fallback, early break and HTTP Link continuations reuse the guarded shared pager with actual late-page proof.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferListLazinessCapsAndContinuation`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.
- WithRecordListServerQuery is an explicit Go extension for server filters missing from the pinned query map. The Source all_projects parameter is ignored because Transfer has no corresponding Body/query descriptor.
- The shared pager supports dictionary links:{next:...} as a Go compatibility extension and restricts continuations to the fixed service collection. Invalid/negative limits and malformed page resources are rejected with HTTP proof.

### `python:block_storage/v3/find_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordFindIgnoreMissing`

- FindRecord attempts discovered-route GET and falls back only on clean400/403/404; fallback is exact Source Transfer.list summary /volume-transfers even below3.55.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferFindGetFallbackSummaryAndMissingDefaults`.
- Fallback consumes all rows to resolve exact ID or name, reports ambiguity and defaults to nil missing with an explicit strict option.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferFindGetFallbackSummaryAndMissingDefaults`.
- Rejected body Read/Close faults, cancellation and changed sources retain the native rejection and physical cause, prevent list fallback and do not retry an observation.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferFindDeleteRejectedReadCloseAndCancellation`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.
- The pinned legacy GET fallback still lists modern /volume-transfers: preserving this Source route can lead to404 on older deployments, rather than silently substituting legacy summary.

### `python:block_storage/v3/accept_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Accept`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.AcceptRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordAcceptAuthKey`

- Transfers.Accept posts {"accept": {"auth_key": ...}} to os-volume-transfer/{id}/accept and returns the transfer from the response.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- AcceptRecord uses discovered modern/legacy route, sends the exact accept/auth_key envelope and constructs the response Resource without seeding the input ID.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Option ownership and invalid preflight prevent HTTP on nil options or invalid input.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferInvalidOptionsAndStickySelection`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.

### `python:block_storage/v3/delete_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordDeleteIgnoreMissing`

- Remove sends DELETE os-volume-transfer/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- DeleteRecord uses the discovered route, ignores only clean404 by default and preserves an identical native rejection returned by a retry hook; strict missing retains the original native404.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Rejected body Read/Close faults, cancellation and changed sources are never suppressed as missing and retain native proof and physical causes.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferFindDeleteRejectedReadCloseAndCancellation`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.
- Strict deletion returns the native Gophercloud404 cause (ResponseCodeIs) rather than synthesizing a missing-resource sentinel.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="source")
transfer = conn.block_storage.create_transfer(
    volume_id="v1", name="handoff", no_snapshots=True
)
for candidate in conn.block_storage.transfers(details=False):
    print(candidate.id, candidate.name)
found = conn.block_storage.find_transfer("handoff", ignore_missing=False)
print(found.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud("source"))
    if err != nil { log.Fatal(err) }
    storage, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    created, err := storage.Transfers.CreateRecord(ctx,
        transfers.WithRecordCreateVolumeID("v1"),
        transfers.WithRecordCreateName("handoff"),
        transfers.WithRecordCreateNoSnapshots(true),
        transfers.WithRecordCreateSuppressLegacySnapshots(true),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(created.Resource.Body["id"]))
    for candidate, err := range storage.Transfers.ListRecords(ctx,
        transfers.WithRecordListDetails(false),
    ) {
        if err != nil { log.Fatal(err) }
        fmt.Println(string(candidate.Resource.Body["id"]), string(candidate.Resource.Body["name"]))
    }
    found, err := storage.Transfers.FindRecord(ctx, "handoff",
        transfers.WithRecordFindIgnoreMissing(false),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(found.Resource.Body["id"]))
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Transfer Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/transfer.py)
