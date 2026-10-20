# Group Snapshot Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/find_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.WithRecordFindMaxItems`

- Pinned named default ignore_missing=True/details=True; one escaped literal member GET, clean400/403/404 fallback, no automatic server/local name query, exact ID-or-name scan, default nil only after complete successful list; strict logical absence has no suppressed native GET cause.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindDefaultsCleanFallbackAndNoAutomaticNameFilter`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindDefaultMissingStrictAndListRejection`.
- Successful GET retains eight untyped descriptors/constructor seed, accepts passive null/nonstring/different response ID, tolerates cls as unknown Wire; ordinary fields/JSON expansion and special all_projects are bounded underlying Resource.find Go extensions, with constructor collisions/null/false/zero and invalid cap/query rejected before discovery/HTTP.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindDirectSuccessUntypedIDAndSeed`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindQuerySeedFieldsScopeAndOwnedSnapshots`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindReservedAndMalformedInputPrecedesHTTP`.
- Common identity engine observes every page before unique success; second match returns Ambiguous, late HTTP/row/unsafe link/physical failure returns only actual terminal receipt, no candidate success. Shared marker-or-offset pager and raw cap reuse existing reader; single-page/raw-cap options document bounded search.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindCompletePagesDuplicateAndLateFailure`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindLatePhysicalListFailureKeepsActualPage`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindPagingControlsOffsetAndRawCap`.
- GET-only explicit microversion, including empty absence, never leaks into separately prepared LIST service/default ceiling3.29; copied private source headers/aliases and one original sticky guard protect both phases. Actual discovery observations remain separate and original client remains unchanged.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindGetOnlyMicroversionAndIndependentDiscovery`.
- Captured service precedes callback, immutable configs are copied around options, location/descriptor constructor preflight precedes discovery/GET; successful declared/wire/envelope/header evidence is independent, nil/invalid contexts and source/cancel errors preserve their causes.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindQuerySeedFieldsScopeAndOwnedSnapshots`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindCapturedSourceOptionCancellationAndNil`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindDeclaredLocationAndEvidenceDoNotAlias`.
- Read/Close/source/context faults on accepted and rejected statuses cannot replay/fallback/ignore/call native retry callback; late physical LIST faults preserve actual partial page. Clean native retry retains headers/version/route; attempted owned header/source mutation keeps native rejection and policy causes.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindLatePhysicalListFailureKeepsActualPage`, `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Find completes captured GET policy/discovery before entering shared identity lookup. A clean discovery400/403 remains a preparation failure with actual native status/body/header and receipt, exactly one discovery HTTP and zero member/list requests; it cannot become member absence, repeat fallback discovery or successful missing. Public GetRecord still owns its prior preparation and transport behavior.
  Tests: `blockstorage/v3/groupsnapshots/python_find_test.go:TestPythonGroupSnapshotFindDiscoveryRejectionNeverBecomesMemberFallback`.

Python/Go differences:

- The exact named Source Proxy accepts only name_or_id, ignore_missing and details. Headers/location/GET-only microversion/JSON Query/bounded paging are explicit Go extensions exposing a finite underlying Resource.find profile; named Python **query parity is not claimed.
- One original cloudread capture owns both phase copies. Defaults independently discover ceiling3.29 rather than implement Python Adapter/session discovery cache; explicit service/per-call defaults are retained and empty GET-only header absence is a documented Go option.
- Go validates blank/dot/control/invalid-UTF8 literal identity and owns escaping fixed member segments. Full mutable Resource/subclass/session/Adapter/cache/dirty/custom-routing/method-control/constructor overloads and Python default-warning behavior are outside this mapping.
- Only eight untyped descriptors and computed recorded-auth/owned location form the declared model; unknown attrs remain Wire/Envelope. Passive response IDs never rewrite routing. Go returns actual terminal HTTP receipt as an extension, not a successful Resource or current backend proof.
- Underlying JSON Query uses stable Go key order, Requests scalar/list/object expansion, known Body seeds/local predicates and known server aliases; all_projects is bool/null and GET-excluded. Reserved constructor/control keys reject presence even null/false/zero. Query factories snapshot/marshal at creation before API capture; API callback-time source changes remain guarded.
- Shared local predicate uses exact decimal equality, not Python JSON binary64 rounding. Actual9007199254740992 versus filter9007199254740993.0 can differ from Source after floating decode. Recursive dictionary/truthiness/bool-number Source matching remains shared.
- Finite shared pager rejects changed origin/collection/filter/cycles, supports advertised marker/offset with one cursor family and nonnegative uint64 offset, does not increment explicit offset automatically, and bounds raw cap/limit; standard RFC Link support is an explicit improvement over pinned Source Requests uri lookup. Paginated=false/raw cap bound search and do not establish global uniqueness.
- Fixture tests and authenticated-cloud/backend acceptance are separate. Existing four GroupSnapshot get/list/create/delete and native operations are reused, not recounted; no admin status/reset operation is promoted.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("identity")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
value = conn.block_storage.find_group_snapshot(
    args.identity,
    ignore_missing=not args.strict,
    details=not args.summary,
)
print(None if value is None else value.to_dict())
```

```go
package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    identity := flag.String("identity", "group-snapshot-id", "group snapshot ID or exact name")
    strict := flag.Bool("strict", false, "report completed search absence as an error")
    summary := flag.Bool("summary", false, "use summary list only on fallback")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    value, err := api.FindRecord(ctx, *identity,
        groupsnapshots.WithRecordFindIgnoreMissing(!*strict),
        groupsnapshots.WithRecordFindDetails(!*summary))
    if err != nil {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
        }
        if errors.Is(err, resource.ErrNotFound) { log.Fatal("completed search found no group snapshot") }
        if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("more than one ID/name match") }
        log.Fatal(err)
    }
    if value == nil { fmt.Println("no matching group snapshot"); return }
    fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2690-L2719)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
