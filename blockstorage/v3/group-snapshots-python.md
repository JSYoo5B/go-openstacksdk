# Group Snapshots Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/create_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordCreateMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordCreateLocation`

- Fresh known attrs snapshot is wrapped under group_snapshot and posted once without Resource GET. Explicit id remains dirty in POST because GroupSnapshot has no allow_patch. Unknown attrs are discarded before Marshaler invocation; omitted/nil/empty attrs versus explicit null/false/zero/empty nested values and exact big integers retain distinct JSON. Parsed server descriptors overwrite matching seed values, unknown values remain Wire only.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`.
- Capture source before known attrs MarshalJSON/options and join callback cause with source/context failures. Constructor collisions self/cls/connection/_synchronized fail without HTTP; JSON Content-Type aliases are canonicalized/owned application/json and invalid JSON framing fails before discovery.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotCreateJSONFramingPrecedesDiscovery`.
- Selected volume service is captured before custom options/known attrs Marshaler; immutable private snapshots retain fixed literal route, selected version and owned headers. Unset defaults reuse shared finite Cinder discovery ceiling3.29; explicit service/per-call defaults bypass discovery and explicit empty per-call version means absent version headers. Actual discovery responses remain separate from resource evidence; discovery Read/Close failure cannot reach member/create/list/delete.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDefaultDiscoveryAndExplicitVersion`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDiscoveryPhysicalFailureRetainsActualProof`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Source generic raise_from_response accepts actual100..399 and rejects400..599 retaining native status/body/header. Accepted partial physical failures and rejected native receipts preserve Read/Close/context causes and sticky source changes even if restored during Close. Physical faults prevent replay and successful Resource projection; clean retry retains immutable JSON and owned header/version/route policy, callback denial/mutation retains original native rejection.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`.
- Bound classmethod cls collision is presence-based, including null/false/zero or custom Marshaler values. Reject before serialization/options/HTTP. This is Source Resource.new(cls,**attrs) delegation rather than an ignored unknown Body attr.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotBoundClsPresenceAndGetWireBoundary`.

Python/Go differences:

- This is the four fixed named generic /group_snapshots workflows, distinct from ordinary volume /snapshots, legacy /cgsnapshots and Manila share-group snapshots. Pinned Gophercloud v2.15.0 has no generic group-snapshot leaf; no existing native operation is reimplemented or counted. The optional owned seed/options and actual HTTP receipt channels are a Go SDK extension.
- Only the pinned eight untyped Body descriptors and computed location form the declared view. Unknown JSON remains actual Wire/Envelope evidence; absent descriptors are null. Leaf location reuses recorded authentication project and optional owned CloudLocation without guessing Connection cloud/region/name or issuing project/auth lookups.
- The mapping bounds call inputs to literal UTF-8 IDs or fresh JSON attrs and concrete options. Mutable Python Resource/subclass/dirty/session/Adapter/cache/custom-base-path/__conflicting_attrs overload and deprecated parameter-warning behavior remain outside these named mappings; find_group_snapshot and admin reset_status/reset_state are distinct inventory operations.
- Go guards routing, source, owned headers/body/version and rejects blank/dot/control/invalid-UTF8 literal IDs. Inner whitespace/slash/percent are retained as one escaped fixed member segment. Accepted HTTP evidence or status=creating does not establish backend completion, durability, replication, current state or policy authorization. Fixture validation and real authenticated-cloud/backend acceptance are distinct.
- Shared finite Cinder discovery returns actual response observations and chooses advertised range capped at Source _max_microversion3.29; it does not implement Python adapter/session discovery-cache behavior. Explicit selected defaults are retained even above3.29, and per-call explicit empty version-header absence is a documented Go option.
- CreateRecord accepts a fresh map[string]any in the JSON domain and concrete options; Python Body values are untyped but Python-only/non-JSON values cannot be serialized here. Constructor control microversion/base_path/__conflicting_attrs are not emitted as Body attrs and full overload/session semantics are not claimed.
- Go owns application/json mutation framing and fixes immutable JSON across retries rather than allowing arbitrary Content-Type or retry body mutation. Empty/malformed accepted body retains fresh declared seed while the separate Envelope records only actual POST ACK bytes.

### `python:block_storage/v3/delete_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordDeleteIgnoreMissing`

- One bodyless fixed-ID DELETE accepts generic100..399 opaque ACK without JSON decode or prefetch. Source ignore_missing=True is the owned default; only a fault-free native404 is suppressed, with IgnoredMissing and copied actual404 receipt retained. Explicit false and non404/native/physical/retry failures remain errors.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDeleteDefaultMissingStrictAndOpaqueACK`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`.
- Selected volume service is captured before custom options/known attrs Marshaler; immutable private snapshots retain fixed literal route, selected version and owned headers. Unset defaults reuse shared finite Cinder discovery ceiling3.29; explicit service/per-call defaults bypass discovery and explicit empty per-call version means absent version headers. Actual discovery responses remain separate from resource evidence; discovery Read/Close failure cannot reach member/create/list/delete.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDefaultDiscoveryAndExplicitVersion`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDiscoveryPhysicalFailureRetainsActualProof`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Source generic raise_from_response accepts actual100..399 and rejects400..599 retaining native status/body/header. Accepted partial physical failures and rejected native receipts preserve Read/Close/context causes and sticky source changes even if restored during Close. Physical faults prevent replay and successful Resource projection; clean retry retains immutable JSON and owned header/version/route policy, callback denial/mutation retains original native rejection.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`.

Python/Go differences:

- This is the four fixed named generic /group_snapshots workflows, distinct from ordinary volume /snapshots, legacy /cgsnapshots and Manila share-group snapshots. Pinned Gophercloud v2.15.0 has no generic group-snapshot leaf; no existing native operation is reimplemented or counted. The optional owned seed/options and actual HTTP receipt channels are a Go SDK extension.
- Only the pinned eight untyped Body descriptors and computed location form the declared view. Unknown JSON remains actual Wire/Envelope evidence; absent descriptors are null. Leaf location reuses recorded authentication project and optional owned CloudLocation without guessing Connection cloud/region/name or issuing project/auth lookups.
- The mapping bounds call inputs to literal UTF-8 IDs or fresh JSON attrs and concrete options. Mutable Python Resource/subclass/dirty/session/Adapter/cache/custom-base-path/__conflicting_attrs overload and deprecated parameter-warning behavior remain outside these named mappings; find_group_snapshot and admin reset_status/reset_state are distinct inventory operations.
- Go guards routing, source, owned headers/body/version and rejects blank/dot/control/invalid-UTF8 literal IDs. Inner whitespace/slash/percent are retained as one escaped fixed member segment. Accepted HTTP evidence or status=creating does not establish backend completion, durability, replication, current state or policy authorization. Fixture validation and real authenticated-cloud/backend acceptance are distinct.
- Shared finite Cinder discovery returns actual response observations and chooses advertised range capped at Source _max_microversion3.29; it does not implement Python adapter/session discovery-cache behavior. Explicit selected defaults are retained even above3.29, and per-call explicit empty version-header absence is a documented Go option.
- Python named delete returns None; Go returns an owned actual DELETE receipt, including a clean ignored404, and never invents a fetched GroupSnapshot or absence/backend-completion proof. The API fixes a literal ID rather than mutating caller Resource state.

### `python:block_storage/v3/get_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordReadSeed`

- One bodyless literal member GET without search/prefetch; wrapped group_snapshot and flat JSON object project exactly8 untyped known descriptors plus computed location. Nested attrs, null/false/zero/array/big integer and unknown HTTP fields survive in independent declared/Wire/Envelope/header carriers. A response id overrides required-ID seed only in the declared response view and never rewrites request route.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotGetOwnsUntypedDescriptorsEnvelopeAndLocation`.
- Owned known seed fields are retained for omitted descriptors and empty/malformed accepted JSON, matching _translate_response ValueError behavior. Explicit server null overwrites seed. Parsed null/list/scalar root or wrapper, invalid JSON field/view shape and UTF-8 failures retain actual status/body/header; partial physical errors expose no successful declared Resource.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotGetOpaqueSeedNullAndInvalidShapeProof`.
- Selected volume service is captured before custom options/known attrs Marshaler; immutable private snapshots retain fixed literal route, selected version and owned headers. Unset defaults reuse shared finite Cinder discovery ceiling3.29; explicit service/per-call defaults bypass discovery and explicit empty per-call version means absent version headers. Actual discovery responses remain separate from resource evidence; discovery Read/Close failure cannot reach member/create/list/delete.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDefaultDiscoveryAndExplicitVersion`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDiscoveryPhysicalFailureRetainsActualProof`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Source generic raise_from_response accepts actual100..399 and rejects400..599 retaining native status/body/header. Accepted partial physical failures and rejected native receipts preserve Read/Close/context causes and sticky source changes even if restored during Close. Physical faults prevent replay and successful Resource projection; clean retry retains immutable JSON and owned header/version/route policy, callback denial/mutation retains original native rejection.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`.
- A server cls unknown field is retained as member GET Wire evidence and does not enter declared projection; GET translation does not call Resource.existing. LIST cls is a consumed-row constructor error, even null/false/zero.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotBoundClsPresenceAndGetWireBoundary`.

Python/Go differences:

- This is the four fixed named generic /group_snapshots workflows, distinct from ordinary volume /snapshots, legacy /cgsnapshots and Manila share-group snapshots. Pinned Gophercloud v2.15.0 has no generic group-snapshot leaf; no existing native operation is reimplemented or counted. The optional owned seed/options and actual HTTP receipt channels are a Go SDK extension.
- Only the pinned eight untyped Body descriptors and computed location form the declared view. Unknown JSON remains actual Wire/Envelope evidence; absent descriptors are null. Leaf location reuses recorded authentication project and optional owned CloudLocation without guessing Connection cloud/region/name or issuing project/auth lookups.
- The mapping bounds call inputs to literal UTF-8 IDs or fresh JSON attrs and concrete options. Mutable Python Resource/subclass/dirty/session/Adapter/cache/custom-base-path/__conflicting_attrs overload and deprecated parameter-warning behavior remain outside these named mappings; find_group_snapshot and admin reset_status/reset_state are distinct inventory operations.
- Go guards routing, source, owned headers/body/version and rejects blank/dot/control/invalid-UTF8 literal IDs. Inner whitespace/slash/percent are retained as one escaped fixed member segment. Accepted HTTP evidence or status=creating does not establish backend completion, durability, replication, current state or policy authorization. Fixture validation and real authenticated-cloud/backend acceptance are distinct.
- Shared finite Cinder discovery returns actual response observations and chooses advertised range capped at Source _max_microversion3.29; it does not implement Python adapter/session discovery-cache behavior. Explicit selected defaults are retained even above3.29, and per-call explicit empty version-header absence is a documented Go option.
- GetRecord takes a fixed literal string ID and optional owned declared seed rather than accepting a mutable Python Resource object/deprecated keyword alias. Seed ID/HTTP evidence cannot choose routing or fabricate a response. Per-call version/location/header controls are explicit Go extensions to the named Proxy getter.
- Go requires JSON object after successful JSON parse; malformed/empty JSON retains seed as Source ValueError does, while invalid UTF-8 is an explicit actual-response error.

### `python:block_storage/v3/group_snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordListQuery`

- Lazy Source detail=True chooses /group_snapshots/detail; false chooses summary collection. Known server query mapping limit/marker/offset/sort_dir/sort_key/sort and all_projects→all_tenants plus wire alias preserve client-key precedence including null, bool/number/text/array/object request expansion. Unknown query values are discarded without Marshaler calls; typed/raw scope conflicts and invalid pagination controls fail before HTTP/discovery.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListSourceQueriesDetailsFiltersAndRawCap`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotRawPagingInputPrecedesDiscovery`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Declared Body attrs not server-query mappings are ordered local predicates over Source view. Shared cloudsnapshot.MatchLocal preserves recursive dictionary subset, truthiness and bool/number equality; exact decimal boundary policy is explicitly documented. LIST constructor cls/connection/microversion/_synchronized collisions are actual-page errors while member GET keeps those unknown values in Wire. Raw MaxItems cap is applied before local filters.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListLocalFloatPolicyAndConstructorCollisions`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListSourceQueriesDetailsFiltersAndRawCap`.
- Shared guarded REST pager owns body/HTTP links, singleton plural object, empty termination, raw cap, finite input limit/marker domain, short-page marker fallback and source/origin/collection/filter/cycle restrictions. Markers derive from retained original row data, not public Resource/Wire mutations; paginated=False reads one page. Actual full page Envelope and row/header/status proof remain independent of declared row projection.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListPagingUsesRetainedScalarMarkerAndScope`.
- Selected volume service is captured before custom options/known attrs Marshaler; immutable private snapshots retain fixed literal route, selected version and owned headers. Unset defaults reuse shared finite Cinder discovery ceiling3.29; explicit service/per-call defaults bypass discovery and explicit empty per-call version means absent version headers. Actual discovery responses remain separate from resource evidence; discovery Read/Close failure cannot reach member/create/list/delete.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDefaultDiscoveryAndExplicitVersion`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotDiscoveryPhysicalFailureRetainsActualProof`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Source generic raise_from_response accepts actual100..399 and rejects400..599 retaining native status/body/header. Accepted partial physical failures and rejected native receipts preserve Read/Close/context causes and sticky source changes even if restored during Close. Physical faults prevent replay and successful Resource projection; clean retry retains immutable JSON and owned header/version/route policy, callback denial/mutation retains original native rejection.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`.
- Source Resource.list JSON Accept overrides captured service/default Accept; only explicit method header Accept overlays that default. Selected private Accept survives a clean native retry without mutating the shared client.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListDefaultAcceptOwnsSourceAndExplicitOption`.
- Shared MarkerOrOffsetPagination follows advertised offset links, retaining initial limit/all_tenants/sort and Body-local predicates. Cursor-family switches remove the previous marker/offset; repeated/backward/negative/ambiguous offset, filter/limit changes and foreign scope fail with the actual page receipt before another HTTP.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotListAdvertisedOffsetAndCursorScope`.
- Source bound Resource.existing rejects LIST cls presence including null/false/zero with actual page proof. Direct member GET tolerates cls as unknown Wire evidence.
  Tests: `blockstorage/v3/groupsnapshots/python_records_test.go:TestPythonGroupSnapshotBoundClsPresenceAndGetWireBoundary`.

Python/Go differences:

- This is the four fixed named generic /group_snapshots workflows, distinct from ordinary volume /snapshots, legacy /cgsnapshots and Manila share-group snapshots. Pinned Gophercloud v2.15.0 has no generic group-snapshot leaf; no existing native operation is reimplemented or counted. The optional owned seed/options and actual HTTP receipt channels are a Go SDK extension.
- Only the pinned eight untyped Body descriptors and computed location form the declared view. Unknown JSON remains actual Wire/Envelope evidence; absent descriptors are null. Leaf location reuses recorded authentication project and optional owned CloudLocation without guessing Connection cloud/region/name or issuing project/auth lookups.
- The mapping bounds call inputs to literal UTF-8 IDs or fresh JSON attrs and concrete options. Mutable Python Resource/subclass/dirty/session/Adapter/cache/custom-base-path/__conflicting_attrs overload and deprecated parameter-warning behavior remain outside these named mappings; find_group_snapshot and admin reset_status/reset_state are distinct inventory operations.
- Go guards routing, source, owned headers/body/version and rejects blank/dot/control/invalid-UTF8 literal IDs. Inner whitespace/slash/percent are retained as one escaped fixed member segment. Accepted HTTP evidence or status=creating does not establish backend completion, durability, replication, current state or policy authorization. Fixture validation and real authenticated-cloud/backend acceptance are distinct.
- Shared finite Cinder discovery returns actual response observations and chooses advertised range capped at Source _max_microversion3.29; it does not implement Python adapter/session discovery-cache behavior. Explicit selected defaults are retained even above3.29, and per-call explicit empty version-header absence is a documented Go option.
- Go stable map-key order replaces Python kwargs insertion order. Shared local equality uses exact decimal JSON numbers rather than Python JSON float rounding; actual integer9007199254740992 versus raw filter9007199254740993.0 differs in Go and can equal after Python float decode. Recursive Source dictionary semantics are reused, not replaced by strict mapping-membership matching.
- The finite pager bounds limit to one nonnegative Go integer, marker to one nonblank transport value, max_items to nonnegative Go integer/zero-unbounded and paginated to bool; it does not emulate arbitrary Python control-value truthiness/coercion. Marker JSON uses shared Requests query expansion and must yield exactly one nonempty value; multiple/empty arrays/objects are rejected.
- Owned options support Source mapped server aliases and Body local filters, discarding unknown query. Go independently rejects changed origin/collection/filter/cycles, accepts Dictionary links and uses the shared safe pager rather than full mutable Python Resource/session paging engine. Explicit offset is retained and not automatically incremented; advertised offset links are supported. Shared finite pager permits one cursor family per request and nonnegative uint64 offset, removes the previous cursor on advertised family switches, and rejects repeated/backward offset rather than reproducing arbitrary Python mixed-cursor/session traversal.
- WithRecordListQuery snapshots known JSON at factory creation; iterator captures service on execution. The public declared view/Wire/Envelope are distinct snapshots and retained private fields determine marker behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=["get", "list", "create", "delete"], default="get")
parser.add_argument("--id", default="group-snapshot-id")
parser.add_argument("--group", default="group-id")
args = parser.parse_args()
conn = openstack.connect()
if args.action == "get":
    print(conn.block_storage.get_group_snapshot(args.id).to_dict())
elif args.action == "list":
    for item in conn.block_storage.group_snapshots(details=False, status="available", limit=10):
        print(item.to_dict())
elif args.action == "create":
    print(conn.block_storage.create_group_snapshot(group_id=args.group, name="sdk-example").to_dict())
else:
    print(conn.block_storage.delete_group_snapshot(args.id, ignore_missing=False))
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
)

func show(value *groupsnapshots.GroupSnapshotRecord) {
    if value == nil { return }
    if value.Resource != nil {
        fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    }
    fmt.Printf("actual_http=%d discovery=%d actual_body=%q\n", value.StatusCode, len(value.Discovery), value.Envelope)
}

func main() {
    action := flag.String("action", "get", "get, list, create, or delete")
    id := flag.String("id", "group-snapshot-id", "existing group snapshot ID")
    group := flag.String("group", "group-id", "existing source group ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id)
        show(value)
        if err != nil { log.Fatal(err) }
    case "list":
        for value, err := range api.ListRecords(ctx,
            groupsnapshots.WithRecordListDetails(false),
            groupsnapshots.WithRecordListQuery("status", "available"),
            groupsnapshots.WithRecordListLimit(10)) {
            if err != nil { log.Fatal(err) }
            show(value)
        }
    case "create":
        value, err := api.CreateRecord(ctx, map[string]any{"group_id": *group, "name": "sdk-example"})
        show(value)
        if err != nil { log.Fatal(err) }
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *id, groupsnapshots.WithRecordDeleteIgnoreMissing(false))
        if receipt != nil { fmt.Printf("actual_http=%d ignored404=%t actual_body=%q\n", receipt.StatusCode, receipt.IgnoredMissing, receipt.Envelope) }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown action")
    }
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-snapshots-group-snapshots)
- [Python Records Test](groupsnapshots/python_records_test.go)
