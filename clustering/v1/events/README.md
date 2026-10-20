# Events

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/events`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadEventRepeatedFiltersSnapshotAndExactBody`.
- With an explicit positive limit, every nonempty page including a short page advances by its retained raw response ID; consumer model mutation cannot replace the wire marker and an empty page ends the stream.
  Tests: `api/clustering_read_test.go:TestClusteringReadShortPagesContinueUsingUnchangedWireIDs`, `internal/rest/list_short_page_test.go:TestListShortPageMarkerPolicyUsesWireRowsAndStopsOnEmptyPage`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
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
- A numeric raw level matches the corresponding exact numeric filter and does not match the typed-model string spelling.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersEventLevelUsesRawJSONRatherThanTypedString`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.

Unresolved boundaries:

- Compare automatic Body/query classification and descriptor defaults/coercion, per-call base_path and deprecated proxy JMESPath controls before closing the whole list declaration. Raw Event.level filtering intentionally does not reuse the typed string conversion.
- Verify the controller default/capped limit and fallback policy against an actual negotiated server contract; typed explicit-limit wire-ID fallback and local cap/single-page behavior are tested.

### `python:clustering/v1/get_event`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.Get`

- Direct GET200 retains resource ID distinct from target/object ID, exact typed/raw fields and accepted response evidence.
  Tests: `api/clustering_read_test.go:TestClusteringReadEventGetHTTPAndDecodeEvidence`.
- Event levels accept JSON string and exact number forms; null/omission and the original JSON type remain visible without float64 conversion.
  Tests: `api/clustering_read_test.go:TestClusteringReadEventLevelsPreserveStringNumberNullAndOmission`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.
- Python level is an untyped Body value; Go Level is its exact string/number lexeme and Body retains the original JSON type.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
event = conn.clustering.get_event(event_id)
events = conn.clustering.events(obj_id=node_id, limit=20)
```

```go
api := events.New(client)
event, err := api.Get(ctx, eventID)
if err != nil {
    return err
}
fmt.Println(event.ObjectID, event.GeneratedAt, event.Level)

records, err := api.All(ctx, events.WithListOptions(events.ListOpts{
    Limit:       20,
    ObjectIDs:   []string{nodeID, otherNodeID},
    ObjectTypes: []string{"NODE", "CLUSTER"},
    Sort:        "timestamp:desc",
}))
```

```go
listingAPI := events.New(client)
values, err := listingAPI.All(ctx,
    events.WithListOptions(events.ListOpts{Limit: 20}),
    events.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, events.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package examples

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
)

func FilterEvents(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for event, err := range service.Events.List(ctx,
		events.WithListFilter("status", "ERROR"),
		events.WithListFilter("meta_data", json.RawMessage(`{"count":9007199254740993}`)),
		events.WithListMaxItems(50), events.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(event.ID, event.Level, event.Status)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#events-events)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
