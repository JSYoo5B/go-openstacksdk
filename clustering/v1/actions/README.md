# Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/actions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.WithListMicroversion`

- Concrete/default query values and defensive snapshots preserve service-specific filters, raw responses and actual list capability.
  Tests: `api/clustering_read_test.go:TestClusteringReadActionListMarkersDefaultsAndOptionsSnapshot`.
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
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python maps the keyword target_id to the wire key target (query keys: name, action, status, sort, global_project, cluster_id); Go uses the wire name target and applies no alias.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/get_action`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.Get`

- Direct GET200 retains resource ID distinct from target/object ID, exact typed/raw fields and accepted response evidence.
  Tests: `api/clustering_read_test.go:TestClusteringReadActionNumericTimesAndResponseIdentity`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.

### `python:clustering/v1/update_action`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.Cancel`

- PATCH202 action cancellation sends {action:{status:CANCELLED}} at numeric microversion>=1.12, optional force is a query parameter, and original accepted Body/Header/StatusCode are retained without assuming an Action response schema.
  Tests: `api/clustering_action_update_test.go:TestClusteringActionUpdateForceQueryAndUnspecifiedResponseBody`, `api/clustering_action_update_test.go:TestClusteringActionUpdateFreezesBodyQueryAndHeadersBeforeNameLookup`, `api/clustering_action_update_test.go:TestClusteringActionUpdatePreflightAndVersionRecheck`, `api/clustering_action_update_test.go:TestClusteringActionUpdateOnlyAcceptsPublished202AndPreservesNativeErrors`.
- Connection shares the selected Senlin client, current authentication token, numeric version and reverse-proxy prefix across cluster scale-out, node check and action cancellation, without implicit action fetch or resubmission.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`.

Python/Go differences:

- Go Update/Cancel are stateless requests with explicit ID/Name/context, a required published CANCELLED status and optional owned Force query. Pinned Python Action has no Force field and drops unknown force attrs; Go adds the published query capability.
- The published update success is202 with no declared response object schema. Go UpdateResult retains even an empty or opaque original Body/Header/StatusCode instead of fabricating an updated Action, requiring an assumed action envelope, or automatically fetching status. Location remains raw header evidence rather than command Submission identity.
- Input body/query/header and selected>=1.12 requirement are frozen before exact-name resolution and revalidated before PATCH. Typed/response body collisions, force/status query replacement and authentication/version header replacement fail before HTTP.
- Python Proxy._update commits arbitrary changed Action attributes with PATCH /actions/{id} and returns the Resource holding the submitted values as if they were applied; a call with no attributes sends nothing. Go sends only the published {"action": {"status": "CANCELLED"}} request, requires a status and returns the accepted response evidence.
- Python uses the session-negotiated microversion; Go requires an explicitly selected numeric version of 1.12 or later and rechecks it before the PATCH.
- No live Senlin evidence was collected for the accepted response body; Go retains whatever the 202 response contains rather than assuming an Action envelope.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
action = conn.clustering.get_action(action_id)
actions = conn.clustering.actions(target_id=node_id, limit=20)
```

```go
api := actions.New(client)
action, err := api.Get(ctx, actionID)
if err != nil {
    return err
}
fmt.Println(action.ID, action.Status)

for action, err := range api.List(ctx,
    actions.WithListOptions(actions.ListOpts{TargetID: nodeID, Limit: 20}),
) {
    if err != nil {
        return err
    }
    fmt.Println(action.ID, action.Status)
}
```

```python
action = conn.clustering.update_action("ACTION_ID", status="CANCELLED")
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
accepted, err := service.Actions.Cancel(ctx, resource.ID("ACTION_ID"),
    actions.WithUpdateForce(false))
if err != nil { return err }
fmt.Println(accepted.StatusCode, string(accepted.Body))
action, err := service.Actions.Get(ctx, "ACTION_ID")
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```go
listingAPI := actions.New(client)
values, err := listingAPI.All(ctx,
    actions.WithListOptions(actions.ListOpts{Limit: 20}),
    actions.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, actions.WithListPaginated(false)) {
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
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
)

func FilterActions(ctx context.Context, conn *sdk.Connection, ownerID string) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for action, err := range service.Actions.List(ctx,
		actions.WithListFilter("owner_id", ownerID),
		actions.WithListFilter("inputs", map[string]any{"region": "region-one"}),
		actions.WithListMaxItems(50), actions.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(action.ID, action.Status)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#actions)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
- [tracked handle](../tracking/async/README.md)
