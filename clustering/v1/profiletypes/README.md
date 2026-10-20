# Profiletypes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/get_profile_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.Get`

- Dotted/versioned profile type names form one protected route segment; list Name/Version are not synthesized into a different detail ID.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryTypeNamePathsAndIndependentVersion`.
- Native403/404, invalid accepted resource bodies, canceled contexts and changed redirect targets preserve errors and prevent unauthorized extra HTTP.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPAndDecodeErrorsRetainEvidence`, `api/clustering_discovery_test.go:TestClusteringDiscoveryInvalidInputsCancelAndRedirectBeforeSecondRequest`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.

### `python:clustering/v1/list_profile_type_operations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.Operations`

- The flat operations catalog retains arbitrary schemas and requires actual numeric microversion>=1.4 before HTTP without upgrading the shared client.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryOperationsVersionSchemaAndRawEnvelope`.
- Native403/404, invalid accepted resource bodies, canceled contexts and changed redirect targets preserve errors and prevent unauthorized extra HTTP.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPAndDecodeErrorsRetainEvidence`, `api/clustering_discovery_test.go:TestClusteringDiscoveryInvalidInputsCancelAndRedirectBeforeSecondRequest`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.

### `python:clustering/v1/profile_types`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.WithListMicroversion`

- Catalog lists follow inherited body/header links within their origin/path with sticky filters and lazy breaks; no undocumented marker fallback is inferred from type names.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryCatalogLinksPreserveExtensionsAndNoMarkerFallback`, `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPLinksBreakAndTargetGuard`.
- MaxItems supplies a wire-limit hint only when no explicit limit exists, caps validated rows locally and zero is unlimited. Paginated=False consumes the first response; bool pointers/query inputs are owned across lazy repeated iterations and later options win. Cap/single/empty stops skip unused decoding or continuation errors while unbounded row/link errors and per-page selected-version guards retain accepted-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsHintsCapsAndSnapshots`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks`, `api/clustering_catalog_list_controls_test.go:TestClusteringCatalogListControlsPreflightAndPageVersions`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Raw type id filters do not synthesize a name fallback, while the binding scope still requires a valid raw binding ID before local filtering and preserves failure evidence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. The inherited positive-cap limit hint is deployment-dependent because the official catalog/service endpoint declares no limit/marker protocol; Go still does not invent an ID marker fallback.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.

Unresolved boundaries:

- Compare inherited limit/marker fallback, unknown-query omission, automatic Body/query classification and descriptor/alternate-ID normalization, per-call base_path and deprecated proxy JMESPath consumption. Explicit raw id/name/schema/support_status filters are tested, with no raw id-to-name fallback; the official type catalog declares no paging query protocol.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := profiletypes.New(client)
profile, err := api.Get(ctx, "os.nova.server-1.0")
if err != nil {
    return err
}
fmt.Println(profile.Name, profile.Schema)

for profile, err := range api.List(ctx) {
    if err != nil {
        return err
    }
    fmt.Println(profile.Name, profile.Version)
}

// Configure the selected version before concurrent client use.
client.Microversion = "1.4"
ops, err := api.Operations(ctx, "os.nova.server-1.0")
```

```go
values, err := api.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    profiletypes.WithListQuery("vendor_filter", "enabled"),
)
```

```go
listingAPI := profiletypes.New(client)
values, err := listingAPI.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    profiletypes.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, profiletypes.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.Name)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
)

func FilterProfileTypes(ctx context.Context, conn *sdk.Connection, typeName string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for profile, err := range service.ProfileTypes.List(ctx,
		profiletypes.WithListFilter("name", typeName),
		profiletypes.WithListMaxItems(50), profiletypes.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(profile.Name, profile.Version)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#profile-types-profile-types)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
