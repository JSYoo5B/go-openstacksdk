# Python Record List

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.InContainer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListPrefix`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxPages`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListSinglePage`

- The fixed-container listing preserves escaped opaque container names, marker query, object metadata and delimiter subdirectory rows without HEAD; subdirectory rows count toward the raw cap.
  Tests: `api/native_scope_list_controls_test.go:TestNativeScopeListControlsFiveBindingsRawCapBeforeNameAndHostFilters`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsSinglePageSkipsForeignAndCyclicContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsCrossPageQueryFixedParentAndIteratorReuse`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsZeroDefaultsFollowNativeContinuation`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsKeepWholePageDecodeErrors`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsParentCancellationAndConsumerBreak`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsPreflightNegativeAndTroveQueryPolicy`, `api/native_scope_list_controls_test.go:TestNativeScopeListControlsSwiftSubdirectoryConsumesRawCap`.
- Pinned Object MRO projection owns six Body, 28 Header, updated_at aka and null SDK location; ordered canonical/wire aliases, alternate name ID, untyped RawMessage metadata, container URI from call, unknowns and subdir retained in Wire, actual page receipt cloned independently. No object HEAD or synthetic metadata observation.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsDefaultRequestAndProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`.
- GET escaped fixed container with format=json and operation Accept JSON; Object query mapping client alias wins wire alias, unknown queries discarded before Marshaler/DTO JSON validation, declared canonical Body filters remain local, no transport-wrapper counting. CPython loaded integer/float/bool recursive equality and float truthiness are explicit shared policy opt-ins. Empty dictionary filter matches every truthy scalar/list/mapping; falsey actual is excluded and nonempty dictionary against truthy nonmapping remains a real page error, including recursive empty filters.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsLocalFiltersUseLoadedFloatsAndDeclaredHeaderTruthiness`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsEmptyDictionaryFilterMatchesEveryTruthyType`.
- Default full lazy pagination uses raw consumed count against X-Container-Object-Count, last raw row ID/name marker, truthy-limit continuation, falsey limit sent once, source body next and RFC Link interoperability, fixed origin/container/filters and SDK cycle/page bounds. Consumer changes cannot alter raw marker; raw cap includes rejected rows; empty page ends.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCountAndMarkerContinuation`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsMaxItemsSinglePageAndPageBound`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`.
- Named cloud list_objects delegates same owned Source-shaped listing; prefix absent/explicit-empty/selected value remain distinct, full_listing is ignored by pinned Python and does not disable paging. Go iterator is collected by caller.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`.
- Actual 100..399 admitted only after page JSON/row projection; malformed accepted singleton/scalar/constructor and every 400..599 (including 404) remain terminal with actual status/header/body and OperationError, no ignore-missing.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsStatusesAndOperationContext`.
- Mixed physical Read/Close/sticky restored source/custom cancellation faults on 200/203/401/404/429/498/503 preserve whole actual receipt, one HTTP and one Close and zero willing caller retry/reauth/backoff callbacks. Native pointer ErrUnableToReauthenticate ErrOriginal and ErrReauth are retained separately. Clean native retry/reauth/backoff preserve fixed route/headers/conditions/selected version and live token.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsPhysicalFaultsPreserveReceiptNative401AndStopHooks`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCleanRetriesPreserveRouteHeadersAndLiveToken`.
- Owned concrete options clone raw bytes/maps/pointers and callback resets; invalid/reserved/unsupported Source controls fail before Marshaler/HTTP, nil Headers reset composes safely, repeated iterator consumption is independent, consumer break avoids continuation and custom cancellation during yield preserves page proof.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsIteratorReuseConsumerBreakAndCancellation`.

Python/Go differences:

- Go exposes these local controls through the common Collection or fixed-parent scope; existing native typed List option signatures remain unchanged. Zero means unlimited and negative values fail lazily before HTTP.
- The native path does not infer a wire limit from max_items and stops exactly at the raw row cap. Native whole-page extraction or IsEmpty may still reject unused trailing rows; continuation retains the native pager and SDK cycle policy.
- The pinned Swift proxy explicitly forces paginated=True. First-page consumption is an explicit Go scope/Collection extension rather than a proxy keyword default mapping.
- Go accepts literal string container names, escapes one path segment and preflights empty/slash/dot/control names; pinned Python also accepts a Container resource/name conversion and interpolates its raw name, including str(None) for an empty Resource name. No mutable Python Resource object identity/cache is exposed.
- Go exposes concrete DTO/functional options and iter.Seq2; callers collect to a slice for named cloud list_objects. Pinned full_listing is ignored. Unknown queries are discarded before serialization; per-call headers/microversion/max_items have dedicated controls. Unsafe cls/self/resource_type/session and __conflicting_attrs, arbitrary base_path and deprecated JMESPath are explicitly rejected before HTTP rather than silently claiming those callable/route branches.
- Go SDK-owned location is null and each row metadata map owns raw JSON values. Pinned Connection location calculation and BaseResource class-level shared metadata/dirty state remain common SDK-R1/C1/S1 contracts; actual remote fields remain in Wire and full Envelope.
- Go max_items stops immediately at the raw cap while pinned Python may request the next page before stopping on its first row. Go follows interoperable RFC Link rel=next URLs while pinned _get_next_link indexes requests response.links next uri, a missing key in normal Requests output; Go rejects foreign/changed/cyclic continuations and defaults to 10000-page bound.
- Go sends captured or explicit selected Swift microversion without Python server-max negotiation, and sorts query keys. Requests query/str conversion retains the reviewed shared JSON-domain representation policy and its documented primitive/runtime limits; custom Python objects are outside the Go JSON-domain API.
- Fixture tests and guide builds do not prove a real Swift account/container ACL, deployment middleware/backend, auth renewal or replace-free install; actual-cloud acceptance and release gates remain pending.

### `python:cloud/_object_store/ObjectStoreCloudMixin/list_objects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.InContainer`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListPrefix`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListMaxPages`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRecordListSinglePage`

- Pinned Object MRO projection owns six Body, 28 Header, updated_at aka and null SDK location; ordered canonical/wire aliases, alternate name ID, untyped RawMessage metadata, container URI from call, unknowns and subdir retained in Wire, actual page receipt cloned independently. No object HEAD or synthetic metadata observation.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsDefaultRequestAndProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`.
- GET escaped fixed container with format=json and operation Accept JSON; Object query mapping client alias wins wire alias, unknown queries discarded before Marshaler/DTO JSON validation, declared canonical Body filters remain local, no transport-wrapper counting. CPython loaded integer/float/bool recursive equality and float truthiness are explicit shared policy opt-ins. Empty dictionary filter matches every truthy scalar/list/mapping; falsey actual is excluded and nonempty dictionary against truthy nonmapping remains a real page error, including recursive empty filters.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsLocalFiltersUseLoadedFloatsAndDeclaredHeaderTruthiness`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsEmptyDictionaryFilterMatchesEveryTruthyType`.
- Default full lazy pagination uses raw consumed count against X-Container-Object-Count, last raw row ID/name marker, truthy-limit continuation, falsey limit sent once, source body next and RFC Link interoperability, fixed origin/container/filters and SDK cycle/page bounds. Consumer changes cannot alter raw marker; raw cap includes rejected rows; empty page ends.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCountAndMarkerContinuation`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsMaxItemsSinglePageAndPageBound`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`.
- Named cloud list_objects delegates same owned Source-shaped listing; prefix absent/explicit-empty/selected value remain distinct, full_listing is ignored by pinned Python and does not disable paging. Go iterator is collected by caller.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectCloudListObjectsPrefixAndCountPaging`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsQueryMappingHeadersAndLocalFilters`.
- Actual 100..399 admitted only after page JSON/row projection; malformed accepted singleton/scalar/constructor and every 400..599 (including 404) remain terminal with actual status/header/body and OperationError, no ignore-missing.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsStatusesAndOperationContext`.
- Mixed physical Read/Close/sticky restored source/custom cancellation faults on 200/203/401/404/429/498/503 preserve whole actual receipt, one HTTP and one Close and zero willing caller retry/reauth/backoff callbacks. Native pointer ErrUnableToReauthenticate ErrOriginal and ErrReauth are retained separately. Clean native retry/reauth/backoff preserve fixed route/headers/conditions/selected version and live token.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsPhysicalFaultsPreserveReceiptNative401AndStopHooks`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsCleanRetriesPreserveRouteHeadersAndLiveToken`.
- Owned concrete options clone raw bytes/maps/pointers and callback resets; invalid/reserved/unsupported Source controls fail before Marshaler/HTTP, nil Headers reset composes safely, repeated iterator consumption is independent, consumer break avoids continuation and custom cancellation during yield preserves page proof.
  Tests: `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsReservedControlsRejectBeforeMarshalAndUnknownDTOValuesAreDiscarded`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsOwnedOptionsNilHeaderResetAndImmutableProjection`, `objectstorage/v1/objects/python_record_list_test.go:TestPythonObjectListRecordsIteratorReuseConsumerBreakAndCancellation`.

Python/Go differences:

- Go accepts literal string container names, escapes one path segment and preflights empty/slash/dot/control names; pinned Python also accepts a Container resource/name conversion and interpolates its raw name, including str(None) for an empty Resource name. No mutable Python Resource object identity/cache is exposed.
- Go exposes concrete DTO/functional options and iter.Seq2; callers collect to a slice for named cloud list_objects. Pinned full_listing is ignored. Unknown queries are discarded before serialization; per-call headers/microversion/max_items have dedicated controls. Unsafe cls/self/resource_type/session and __conflicting_attrs, arbitrary base_path and deprecated JMESPath are explicitly rejected before HTTP rather than silently claiming those callable/route branches.
- Go SDK-owned location is null and each row metadata map owns raw JSON values. Pinned Connection location calculation and BaseResource class-level shared metadata/dirty state remain common SDK-R1/C1/S1 contracts; actual remote fields remain in Wire and full Envelope.
- Go max_items stops immediately at the raw cap while pinned Python may request the next page before stopping on its first row. Go follows interoperable RFC Link rel=next URLs while pinned _get_next_link indexes requests response.links next uri, a missing key in normal Requests output; Go rejects foreign/changed/cyclic continuations and defaults to 10000-page bound.
- Go sends captured or explicit selected Swift microversion without Python server-max negotiation, and sorts query keys. Requests query/str conversion retains the reviewed shared JSON-domain representation policy and its documented primitive/runtime limits; custom Python objects are outside the Go JSON-domain API.
- Fixture tests and guide builds do not prove a real Swift account/container ACL, deployment middleware/backend, auth renewal or replace-free install; actual-cloud acceptance and release gates remain pending.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--prefix")
parser.add_argument("--mode", choices=("proxy", "cloud"), default="proxy")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "cloud":
        values = conn.list_objects(args.container, prefix=args.prefix)
    else:
        query = {} if args.prefix is None else {"prefix": args.prefix}
        values = conn.object_store.objects(args.container, **query)
    print(json.dumps([value.to_dict() for value in values], indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, container string, prefix *string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    var options []objects.RecordListOption
    if prefix != nil { options = append(options, objects.WithRecordListPrefix(*prefix)) }
    rows := make([]*resource.RawResource, 0)
    for record, err := range swift.Objects.ListRecords(ctx, container, options...) {
        if err != nil {
            if record != nil {
                fmt.Fprintf(os.Stderr, "page status=%d body=%s\n", record.StatusCode, record.Envelope)
            }
            return err
        }
        rows = append(rows, record.Resource)
    }
    output, err := json.MarshalIndent(rows, "", "  ")
    if err != nil { return err }
    fmt.Println(string(output))
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    prefix := flag.String("prefix", "", "optional prefix")
    flag.Parse()
    var selectedPrefix *string
    flag.Visit(func(value *flag.Flag) { if value.Name == "prefix" { selectedPrefix = prefix } })
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, selectedPrefix); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [`conn.object_store.objects(container, **query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L271)
- [`conn.list_objects(container, full_listing=True, prefix=None)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L347-L366)
- [Object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)
- [Install](../../../docs/install.md)
- [Python Record List Test](python_record_list_test.go)
