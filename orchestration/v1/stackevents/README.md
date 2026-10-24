# Stackevents

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:orchestration/v1/stackevents.Find`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.API.InStack`

- Native Find requests GET /stacks/{name}/events without a query, follows the canonical redirect keeping auth and client headers, returns the single non-paged events array (null physical ID, both timestamp forms, properties, links) without following next links, and keeps 403/404, invalid timestamp, malformed JSON and canceled-context failures.
  Tests: `api/heat_native_stackevents_test.go:TestHeatNativeStackEventsFindFollowsNameRedirectAndKeepsErrors`.

Python/Go differences:

- Native Find takes the stack name only and relies on Heat's redirect, whereas the scope resolves a stack once and lists at the canonical /stacks/{name}/{id}/events path.

### `gophercloud:orchestration/v1/stackevents.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.ResourceEventScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithGetHeader`

- Both stack-scope and resource-scope GET use the complete stack name/ID/resource/event path, preserve request and wire identities separately, return native fields/properties/links with Detailed=true, and retain raw null/large integers and response headers.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsGetUsesCompleteResourcePathAndPreservesRequestIdentity`.
- Invalid event/resource segments, nil/unsupported options and attempts to override auth/Accept/microversion fail before HTTP; canceled GET also performs no request.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsPreflightValidatesIdentityPathsAndOptions`.
- Missing/null/wrong-type detail objects, missing IDs, bad timestamps and bad resource_type fail instead of returning a usable event.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsMalformedResponsesAndLaterErrorsRemainObservable`.
- GET 400/401/403/404/500 preserve original status, body, header and URL; only 404 is classified as resource.ErrNotFound.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPreservesGetAndPaginationHTTPErrors`.
- Unicode stack/resource/event segments are escaped and simultaneous scope GETs retain independent caller-mutable raw bodies and headers.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsScopesEscapeUnicodeSegmentsAndRemainConcurrent`.

Python/Go differences:

- Go fixes stack and optional resource parents, accepts a concrete event ID and returns EventResource instead of GetResult. Request identity, raw body/header and optional resource_type are preserved separately; path/options and response object/ID are validated before a usable result is returned.

### `gophercloud:orchestration/v1/stackevents.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.All`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/resource.Pages`

- Stack list uses its canonical events path; typed marker/limit, repeated actions/statuses/resource names, resource_type and sort_keys/sort_dir plus nested_depth, escaped extension query and headers are asserted. Native fields, request identity, null/omission, large raw integers and summary/detail distinction are retained.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists`.
- Last event ID becomes the next marker while page size, filters and headers remain; an empty page ends traversal and page response headers are independently retained.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationUsesLastEventMarkerAndEmptyPage`.
- Break skips later item decoding and next page; cancellation between resource-list items is returned without another HTTP request.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationStopsOnBreakAndCancellation`.
- Both stack and resource list reject a repeated marker URL before a third request.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationRejectsRepeatedMarkerBeforeRefetch`.
- Malformed pages/items and later decode or HTTP errors make All return nil plus error rather than a partial complete list.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsMalformedResponsesAndLaterErrorsRemainObservable`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPreservesGetAndPaginationHTTPErrors`.
- Nil/unsupported field/argument/header options and canceled list fail before HTTP; in-flight cancellation reaches the list HTTP request.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsPreflightValidatesIdentityPathsAndOptions`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsInFlightCancellationReachesHTTPRequest`.

Python/Go differences:

- Go returns a lazy typed EventResource sequence or non-nil collected slice rather than Pager. The native last-event-ID marker protocol is retained, with library-owned raw decoding, parent identity, query/header extensions, cancellation and cycle protection.

### `gophercloud:orchestration/v1/stackevents.ListResourceEvents`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.ResourceEventScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.ResourceEventScope.All`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListResourceEventsOptions`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListResourceEventsQuery`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.WithListResourceEventsHeader`
- `github.com/JSYoo5B/go-openstacksdk/resource.Pages`

- Resource list uses the distinct /resources/{logicalName}/events path, typed resource_name filter, nested_depth query and extension header; the next last-ID marker request ends on 204. Request resource name remains independent of the response name.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists`.
- Resource-list repeated marker is rejected; between-item cancellation, malformed options and HTTP errors remain observable.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationRejectsRepeatedMarkerBeforeRefetch`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationStopsOnBreakAndCancellation`, `api/heat_stackevents_scope_test.go:TestHeatStackEventsPreflightValidatesIdentityPathsAndOptions`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPreservesGetAndPaginationHTTPErrors`.
- The resource-scoped list serializes every ListResourceEventsOpts input (explicit initial Marker, Limit, repeated ResourceActions/Statuses/Names/Types, SortKey, SortDir) plus an extension query and header, repeats them on the next page with only marker replaced by the last event ID, and ends on 204.
  Tests: `api/heat_native_stackevents_test.go:TestHeatNativeStackEventsResourceListSerializesEveryTypedOption`.

Python/Go differences:

- The logical resource parent is fixed by ForResource rather than repeated as a string; event response name remains separate from the request name.

### `python:orchestration/v1/stack_events`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.API.InStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.API.ForStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.ForResource`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.ResourceEventScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.StackEventScope.All`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents.ResourceEventScope.All`

- Exact stack-name binding consumes all parent pages once, fixes both canonical components and keeps stack-wide and resource-specific event paths separate.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists`.
- Explicit ID parent binding reuses canonical GET/redirect exactly once; later event reads issue no parent lookup and empty All results are non-nil.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsInStackIDReusesCanonicalLookup`.
- Missing, duplicate and forbidden parents are reported rather than treated as empty event lists.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsParentErrorsAndDuplicateNamesRemainObservable`.
- Stack list uses its canonical events path; typed marker/limit, repeated actions/statuses/resource names, resource_type and sort_keys/sort_dir plus nested_depth, escaped extension query and headers are asserted. Native fields, request identity, null/omission, large raw integers and summary/detail distinction are retained.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists`.
- Last event ID becomes the next marker while page size, filters and headers remain; an empty page ends traversal and page response headers are independently retained.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationUsesLastEventMarkerAndEmptyPage`.
- Resource list uses the distinct /resources/{logicalName}/events path, typed resource_name filter, nested_depth query and extension header; the next last-ID marker request ends on 204. Request resource name remains independent of the response name.
  Tests: `api/heat_stackevents_scope_test.go:TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists`.
- Resource-list repeated marker is rejected; between-item cancellation, malformed options and HTTP errors remain observable.
  Tests: `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationRejectsRepeatedMarkerBeforeRefetch`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPaginationStopsOnBreakAndCancellation`, `api/heat_stackevents_scope_test.go:TestHeatStackEventsPreflightValidatesIdentityPathsAndOptions`, `api/heat_stackevents_pagination_test.go:TestHeatStackEventsPreservesGetAndPaginationHTTPErrors`.

Python/Go differences:

- Go fixes canonical stack identity and resource logical name once, provides typed immutable request metadata plus raw body/header and uses native marker traversal with cycle protection.
- Go uses Heat sort_keys through native SortKey and HTTP query extensions; pinned Python StackEvent advertises sort_key. Empty resource name is rejected by ForResource instead of Python falsey resource_name selecting a whole-stack list.
- Python stack_events fetches a non-Stack argument with Proxy._get(Stack, stack), a GET /stacks/{id} in which DELETE_COMPLETE and ADOPT_COMPLETE become NotFound, and then lists /stacks/{name}/{id}/events or the resource-specific /resources/{resource_name}/events path; Go resolves the parent once with InStack and has distinct stack and resource scopes.
- Python StackEvent declares resource_action, resource_status, resource_name, resource_type, nested_depth, sort_key and sort_dir plus limit and marker; it sends sort_key, while Heat's documented parameter and the Go native option are sort_keys. Go does not send sort_key unless the caller supplies it as an explicit query.
- A falsy resource_name selects the whole-stack list in Python; Go ForResource rejects an empty name.
- Python builds StackEvent Resources with descriptors; Go returns EventResource values with typed fields plus raw body and headers.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { panic(err) }
    service, err := conn.OrchestrationV1(ctx)
    if err != nil { panic(err) }
    events, err := service.StackEvents.InStack(ctx, resource.Name("app"))
    if err != nil { panic(err) }

    for event, err := range events.List(ctx,
        stackevents.WithListOptions(stackevents.ListOpts{Limit: 100}),
        stackevents.WithListQuery("nested_depth", "1")) {
        if err != nil { panic(err) }
        fmt.Println(event.ID, event.ResourceName, event.ResourceStatus)
    }

    resourceEvents, err := events.ForResource("server")
    if err != nil { panic(err) }
    values, err := resourceEvents.All(ctx,
        stackevents.WithListResourceEventsOptions(
            stackevents.ListResourceEventsOpts{Limit: 100}))
    if err != nil { panic(err) }
    if len(values) > 0 {
        detail, err := resourceEvents.Get(ctx, values[0].ID)
        if err != nil { panic(err) }
        fmt.Println(detail.ID, detail.ResourceProperties, detail.Header)
    }
}
```

## References

- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#show-event-details)
- [Pinned Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py)
- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#list-stack-events)
- [Stacks](../stacks/README.md)
- [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/orchestration/v1/stackevents/requests.go)
- [Python StackEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/stack_event.py)

## Native API calls

`API.Find` issues `GET /stacks/{name}/events`, follows Heat's redirect to the canonical stack and returns the `events` array of that single response without following any next link. It is covered by `api/heat_native_stackevents_test.go`.

```go
events, err := stackevents.New(client).Find(ctx, "app")
```

For resource events, `ResourceEventScope.List` serializes every `ListResourceEventsOpts` field (`Marker`, `Limit`, repeated `ResourceActions`, `ResourceStatuses`, `ResourceNames`, `ResourceTypes`, `SortKey`, `SortDir`) and repeats them on later pages, replacing only `marker` with the last event ID. A 204 response ends the listing.
