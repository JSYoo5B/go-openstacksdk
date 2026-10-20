# Flavor Cloud

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/list_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AllFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- AllFlavors eagerly consumes the existing detailed owned Flavor list in source page order, retaining duplicates and producing a completed JSON array, including [] for an empty inventory. Its actual source get_extra default is false; an explicit true enriches falsey inline specs while false makes no child request. It sends the fixed detail list query, never Cloud filters or a name selector, and retains the chosen version and live token through all pages and child reads.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The eager collection reuses the owned reader's guarded paging and conditional enrichment. A late page403 retains prior Inventory but cannot publish completed Value or Flavors. Requested child failure follows the same shared collection path: the Cloud Search composition proves the failing known row is retained, actual child404 is terminal, and an accepted child Close404 keeps the original physical cause plus separate row/child receipt without a completed collection.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The shared Cloud read preparation applies captured typed/raw/header options once, checks the original service/client binding and preserves cancellation causes. AllFlavors rejects any present Filters pointer, including null and {}, before HTTP because its source declaration has no filter argument; nil explicitly clears an earlier filter. The reused inventory stage captures Connection location before callbacks and independently adds it to each projected Resource and child Resource while leaving actual Wire and receipts unchanged.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`.

Python/Go differences:

- The pinned source signature and body use get_extra=False and read no clouds.yaml override, despite the list_flavors docstring saying default True. Go follows the executed signature and exposes a concrete optional bool override.
- Python returns an eager list of connected mutable Flavor Resources. Go returns completed raw JSON Value/Flavors plus partial Inventory on failure; records retain an owned projected Resource, actual Wire/row receipt and separate child enrichment receipt. The owned response view can be modified by its caller; it is not an immutable Python Resource replacement.
- This Cloud composition reuses the separately mapped owned reader's documented JSON-domain descriptors, finite selected/discovered or explicit-versionless policy, route/control/continuation validation and Connection location snapshot. Generic mutable Resource/session/cache facilities and other native/Proxy declarations remain separate parts of the original SDK goal.
- AllFlavors names the eager Cloud operation without replacing Service.ListFlavors, which remains the lazy Proxy-list mapping. The shared options carrier does not add a filters parameter to this source operation: present filters are a Go preflight error and nil clears them.

### `python:cloud/_compute/ComputeCloudMixin/search_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- SearchFlavors defaults requested extra specs to true and completes the entire detailed inventory and conditional child reads before applying any identifier or Cloud filter. Explicit false skips enrichment. Even an ultimately unmatched row is enriched when required, and a dictionary can select newly fetched specs. No name or Cloud dictionary is forwarded into the early Proxy query/Body matcher. Empty successful inventory/selection is [] rather than an inventory error.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- After successful inventory completion, the existing Cloud selector applies per-row Python-shaped id/name exact-or-glob matching, preserving source order and duplicate matches, then ordered dictionary subset or JMESPath selection. Numeric logical IDs participate in Cloud string matching. Reached unknown predicates fail, predicates after an earlier mismatch remain unconsumed, and an empty identifier selection does not consume unknown dictionary keys. Expressions retain arbitrary scalar/object/array/false JSON Value and do not invent associated Flavor records.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`, `internal/cloudfilter/mapping_test.go:TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`.
- Any late inventory, requested child or local filter failure retains consumed Inventory without completed Value/Flavors. Accepted child failures preserve independent raw row/child receipts and their original read/Close causes. Reusable bulk pointer/raw filters and callback header carriers are owned, callbacks execute once, captured location is applied independently before local selection, and actual row Wire/Envelope is unchanged. Source mutation or callback cancellation stops later callbacks/HTTP and retains the custom cancellation cause; selection publication remains guarded.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/results_test.go:TestCloudFilterSelectOwnsViewsFiltersIndicesAndResultsBeforeGuardCallbacks`, `internal/cloudfilter/results_test.go:TestCloudFilterGuardInterruptsTraversalAndRetainsCancellationCauseWithoutPartialSelection`.

Python/Go differences:

- Python annotates a list result, but the executed string-filter branch can return arbitrary JMESPath values. Go exposes that raw JSON in Value and leaves Flavors unset for expression results; ordinary selector indices connect Flavors to the retained Inventory without synthesizing expression records.
- Dictionary filters are local Cloud predicates after complete enriched inventory, not early owned ListRecords query/Body filters. The shared JSON-domain Cloud selector represents Python truthiness, repr/glob, ordered predicate consumption and equality through finite documented Go rules rather than arbitrary Python object execution.
- Search uses the separately mapped reader's owned projected Resource plus captured Connection location, and keeps actual Wire/Envelope/Header/Status and child receipt separate. Partial Inventory is an additive diagnostic result on error. Selected/finite-discovered/explicit-empty version and typed shared-control differences remain documented in the owned guide; generic mutable Resource/session/cache lifecycle remains an independent SDK goal.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- GetFlavor delegates directly to the owned FindFlavor reader with IgnoreMissing(true) and requested extra specs true by default; it does not call Cloud Search or First. Absent, explicit null and every source-falsey filter become an unfiltered Find. Truthy dictionaries retain original member query spelling and known seed attributes. Direct success keeps inline nonempty specs, or enriches the canonical returned ID when requested specs are falsey; explicit false skips the child and an explicit empty microversion remains versionless for both reads without discovery.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`.
- The direct Find delegation keeps its source-shaped clean400/403/404 detail-list fallback. Missing returns nil,nil by default, a second match immediately returns ambiguity without child enrichment, and only a unique fallback result is enriched. Captured location is applied to separate result/child Resources, and member and child status/header/envelope remain distinct. The existing reader preserves late fallback and accepted child failures as terminal partial evidence rather than inventing a successful Get.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Truthy nonobject filters fail before HTTP instead of becoming Cloud expressions. Dictionary keys name_or_id, ignore_missing and get_extra_specs collide with explicit find arguments and are rejected before HTTP; the reused reader protects its explicit id argument and scalar original-query encoding. The shared captured read configuration and final binding check preserve reusable options, selected/live client state and guarded context/location preparation without a second application of Cloud callbacks.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- The source's deprecated filter docstring mentions strings/JMESPath, but its executed body normalizes falsey filters and expands truthy filters as Find kwargs. Go follows that direct delegation: truthy nonobjects and duplicate explicit kwargs are typed HTTP0 errors; it does not turn them into Cloud search. Python's deprecation warning is documented rather than emitted as a Go warning.
- Original member kwargs reuse the separately mapped Find encoder's scalar/null/scalar-array range. Dictionary-valued and nested-array values that Python may stringify or iterate are explicit Go input errors, distinct from SearchFlavors' nested dictionary predicates. The owned Find reader also retains its documented literal route validation, identity/filter/projection, version and continuation differences.
- Go returns an owned FlavorRecord and can retain a known partial record/child receipt with error, rather than returning a connected mutable Python Resource or silently losing physical evidence. Computed current location changes only Resource views, with Wire/Envelope/Header/Status kept actual. Generic mutable Resource/session/cache facilities and other native/Proxy/Cloud declarations remain separate portions of the original SDK goal.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
all_flavors = conn.list_flavors()  # get_extra=False
matched = conn.search_flavors("m1.*", filters={"disk": 20})
found = conn.get_flavor("m1.small")  # get_extra=True, ignore_missing=True
for flavor in matched:
    print(flavor.to_dict())
if found is not None:
    print(found.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "flavor name or ID glob")
    identity := flag.String("flavor", "", "optional flavor ID or name to get")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *identity); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *flavors.FlavorRecord) any {
    if record == nil { return nil }
    var enrichment any
    if child := record.Enrichment; child != nil {
        enrichment = map[string]any{
            "resource": child.Resource, "wire": child.Wire,
            "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
            "header": child.Header, "status_code": child.StatusCode,
        }
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope), "header": record.Header,
        "status_code": record.StatusCode, "enrichment": enrichment,
    }
}

func printQuery(operation string, result *compute.FlavorQueryResult) error {
    if result == nil { return nil }
    inventory := make([]any, len(result.Inventory))
    for i, record := range result.Inventory { inventory[i] = recordValue(record) }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "matched_count": len(result.Flavors), "inventory_count": len(result.Inventory),
        "inventory": inventory,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern, identity string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    all, err := service.AllFlavors(ctx)
    if printErr := printQuery("all", all); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("all flavors: %w", err) }
    matched, err := service.SearchFlavors(ctx, pattern)
    if printErr := printQuery("search", matched); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("search flavors: %w", err) }
    if identity == "" { return nil }
    found, err := service.GetFlavor(ctx, identity)
    if found != nil {
        data, printErr := json.MarshalIndent(map[string]any{"operation": "get", "record": recordValue(found)}, "", "  ")
        if printErr != nil { return errors.Join(err, printErr) }
        fmt.Println(string(data))
    }
    if err != nil { return fmt.Errorf("get flavor: %w", err) }
    if found == nil { fmt.Println("flavor not found") }
    return nil
}
```

## References

- [Flavor Records](flavor-records.md)
- [owned List](flavor-records.md)
- [owned Flavor](flavor-records.md)
- [Cloud list_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L278-L289)
- [search_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L165-L181)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L524-L561)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
