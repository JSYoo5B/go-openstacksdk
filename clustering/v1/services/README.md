# Services

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/services.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadServicesVersionGateAndLocalStatusFilter`.
- MaxItems supplies a wire-limit hint only when no explicit limit exists, caps validated rows locally and zero is unlimited. Paginated=False consumes the first response; bool pointers/query inputs are owned across lazy repeated iterations and later options win. Cap/single/empty stops skip unused decoding or continuation errors while unbounded row/link errors and per-page selected-version guards retain accepted-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsHintsCapsAndSnapshots`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsPreflightAndPageVersions`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links. Services retains its numeric1.7 preflight gate even with list controls.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsServicesVersionGateBeforeHTTP`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- An entirely filtered first page cannot bypass the selected numeric 1.7 service gate before advertised continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersServicesVersionGateOnFilteredContinuation`, `internal/senlin/catalog_filter_test.go:TestBodyFilterCatalogFacadesUseRawFieldsAndOwnNamespaces`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.
- Services>=1.7 and Receiver user>=1.4 gates evaluate the effective per-call client, allowing documented explicit upgrades and rejecting downgrades/default resets without changing source configuration.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestEffectiveVersionGates`, `internal/senlin/list_request_test.go:TestListRequestRevalidatesSourceBeforeContinuationAndGatesEffectiveVersion`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. The inherited positive-cap limit hint is deployment-dependent because the official catalog/service endpoint declares no limit/marker protocol; Go still does not invent an ID marker fallback.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python type-catalog and service resources forward limit and marker and, with an explicit limit, continue with the last row's Resource.id (the alternate name ID); Go sends limit hints only through opt-in local caps, has no marker fallback because these endpoints document no paging protocol, and follows only links the server returns.
- Python Service declares no query keys beyond limit and marker, so every other keyword is a client-side filter on Service attributes; Go provides explicit local filters for id, name, state, binary, host, updated_at, disabled_reason and status and the numeric 1.7 version gate.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
services = conn.clustering.services()
```

```go
// Configure before sharing the client between goroutines.
client.Type = "clustering"
client.Microversion = "1.7"
api := services.New(client)

for service, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(service.ID, service.Host, service.Status, service.State)
}
```

```go
listingAPI := services.New(client)
values, err := listingAPI.All(ctx,
    services.WithListOptions(services.ListOpts{Limit: 20}),
    services.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, services.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
)

func FilterServices(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	values, err := service.Services.All(ctx,
		services.WithListFilter("state", "up"),
		services.WithListFilter("disabled_reason", nil),
		services.WithListMaxItems(50), services.WithListPaginated(false))
	if err != nil {
		return err
	}
	for _, value := range values {
		fmt.Println(value.ID, value.Host, value.State)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#services-services)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
