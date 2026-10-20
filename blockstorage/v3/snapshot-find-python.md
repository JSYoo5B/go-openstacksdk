# Snapshot Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/find_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordFindQuery`

- Find(resource.ID) sends one GET without list fallback and Find(resource.Name) lists snapshots/detail?name=..., so the two phases exist only as separate calls.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotFindHasNoIDThenNameFallback`.
- FindRecord always attempts one bodyless query-free member GET with the literal validated identity. Only clean native400/403/404 permit fallback, including an identical clean RetryFunc rejection;401 and mixed physical/source/cancellation/callback failures do not fall back. Default details uses /snapshots/detail, false uses /snapshots; true all_projects adds all_tenants=True only to the fallback list.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordDefaultFallbackRoutesAndAllProjects`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordMissingAndCleanNativeRetry`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordRejectedPhysicalFailuresNeverFallback`.
- Fallback automatically adds name unless explicitly supplied even as null, matches exact ID OR exact name, and consumes the complete selected search to expose late duplicates and later-page errors. Default missing returns nil,nil after successful list observation, strict missing returns ErrNotFound without attaching suppressed member rejection; duplicate returns ErrAmbiguous.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordFallbackMatchesIDAndDetectsLateDuplicateOrFailure`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordMissingAndCleanNativeRetry`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordFilterDictionaryAndExplicitNullName`.
- Owned SnapshotRecord exposes the15 Source Body attributes plus nullable location in Resource separately from complete Wire and actual whole HTTP Envelope/Header/StatusCode. Alias assignments follow JSON member order; missing member ID seeds only the view while explicit null/nonstring ID is preserved. Source size int conversion preserves bool, Unicode decimal strings and precise integers; BoolStr and dict metadata adaptation apply only to the view. Unknown keys and original values remain lossless in Wire.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordDirectGetAndOwnedDescriptors`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordResponseShapeAndNullableIdentity`.
- Direct member translation accepts singular or flat objects; blank and malformed JSON syntax retain Source seeds. Valid nonobjects/null/invalid UTF8 and descriptor failures retain actual accepted HTTP proof. Fallback plural arrays/singleton objects are supported; null rows or Resource constructor collisions fail with the whole actual page. No native typed Snapshot coercion is imposed.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordResponseShapeAndNullableIdentity`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordAcceptedFaultAndStickySourceProof`.
- Optional Go fallback Query classifies Source server mapping name/status/volume_id/project_id/limit/marker/offset/sort_dir/sort_key/sort, translates all_projects->all_tenants including wire aliases and client-alias-null precedence, and applies declared nonserver Body filters locally. Unknown/wire-only fields are ignored. Requests boolean/iterable/null encoding and Python JSON equality including bool-number equality and recursive dictionary filters are reused; a truthy scalar matches an empty dictionary but nonempty dictionary get on a scalar fails. Positional ID conflict is rejected before HTTP.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordQueryAliasesFiltersAndRawRowCap`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordFilterDictionaryAndExplicitNullName`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordPreflightCancellationAndRetainedOptions`.
- Raw MaxItems counts before local filtering and supplies a limit hint when missing/zero. Cap and SinglePage terminate before parsing next continuation; ordinary search stays unbounded. Shared guarded paging handles snapshots_links, top-level next, HTTP Link, singleton rows, empty-page termination and short-page marker fallback using retained wire ID. Offset-only explicit Go query uses strictly increasing offset continuation. Route/filter/cursor changes retain actual page proof.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordSinglePageMarkerFallbackAndOffsetPaging`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordQueryAliasesFiltersAndRawRowCap`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordResponseShapeAndNullableIdentity`.
- Configured defaults such as3.80 are sent unchanged, reflecting Resource._get_microversion default precedence. With an absent default, shared finite bodyless/versionless Cinder discovery selects the Source Snapshot ceiling3.65 against advertised min/max on a private client. Discovery responses remain separately owned and discovery accepted Read/Close errors produce Discovery-only proof without fabricated snapshot ACK. Fixed negotiated version headers or their absence remain protected across retries and physical attempts.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordDiscoveryAndProtocolHeaderProtection`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordListPhysicalFailureAndDiscoveryOnlyProof`.
- Source/client/option facts are captured and guarded across callbacks, each physical Read/Close and retries. Accepted member Read/Close/source failures return a partial SnapshotRecord plus actual ResponseError; rejected member/list bodies preserve original native status/body/header and read/close causes with one Close, preventing false fallback. Sticky source changes survive restoration; custom cancellation and callback causes remain joined. Maps/raw values/pointers/headers are copied, including retained option callbacks and concurrent reuse.
  Tests: `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordRejectedPhysicalFailuresNeverFallback`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordAcceptedFaultAndStickySourceProof`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordListPhysicalFailureAndDiscoveryOnlyProof`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordPreflightCancellationAndRetainedOptions`, `blockstorage/v3/snapshots/python_find_contracts_test.go:TestPythonSnapshotFindRecordConcurrentOptionReuse`.

Python/Go differences:

- Go takes a validated nonblank UTF8 literal identity and returns an owned Source-shaped SnapshotRecord, rather than accepting mutable Python Resource/Munch/session objects. Resource/Wire/Envelope and Discovery are separate observation channels; Python Connection location synthesis is represented as null. Mutable Resource/subclass/cache/session/base_path behavior remains separately inventoried SDK-R1/C1/S1 scope.
- The named Python find_snapshot proxy accepts only ignore_missing/details/all_projects. Optional Go Query/Limit/Marker/MaxItems/Paginated/Headers are explicit library-owned fallback controls based on inherited Resource.list behavior; member GET remains query-free. Explicit cap/single-page narrows the caller-chosen observed search and cannot guarantee absence of unobserved duplicates; default behavior consumes all pages.
- Go rejects negative typed pagination controls, unsafe identities, positional ID conflicts, invalid UTF8 and forbidden authentication headers before HTTP. Fixed origin/collection/filter/version/physical-request guards intentionally reject Source callback or next-link retargeting. Dictionary links are an explicit Go compatibility extension. Offset-only controls use guarded offset continuation without Source automatic marker fallback.
- A Go map has no Python kwargs order, so multiple local filters are evaluated in sorted attribute order. Literal JSON response member ordering is retained for descriptor alias precedence. Size conversion follows the audited common Unicode16/Python3.14 JSON-domain descriptor helper and arbitrary-precision integer policy.
- Accepted malformed UTF8 fails instead of requests replacement decoding. Blank/malformed JSON syntax matches Source member seed behavior, while wrong valid shapes retain actual ACK errors. The SDK uses shared clean rejection observation so mixed physical or callback failures cannot be mistaken for Source clean400/403/404.
- Absent defaults use finite per-operation shared Cinder discovery at Source Snapshot3.65; configured defaults are never artificially capped. Source generally relies on cached Keystone session endpoint discovery. General session caching/negotiation behavior remains SDK-S1, and authenticated Cinder RBAC deployment validation is separate from local fixtures.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev", block_storage_default_microversion="3.65")
snapshot = conn.block_storage.find_snapshot("daily")
summary = conn.block_storage.find_snapshot("daily", details=False)
strict = conn.block_storage.find_snapshot(
    "daily", details=True, all_projects=True, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("snapshot", "daily", "snapshot name or ID")
    details := flag.Bool("details", true, "use detailed fallback list")
    allProjects := flag.Bool("all-projects", false, "search across projects")
    strict := flag.Bool("strict", false, "return an error when missing")
    version := flag.String("microversion", "3.65", "Cinder selected version; empty enables discovery")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Snapshots.FindRecord(ctx, *identity,
        snapshots.WithRecordFindDetails(*details),
        snapshots.WithRecordFindAllProjects(*allProjects),
        snapshots.WithRecordFindIgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [find_snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L154-L193)
- [Snapshot descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Install](../../docs/install.md)
- [Python Find Contracts Test](snapshots/python_find_contracts_test.go)
