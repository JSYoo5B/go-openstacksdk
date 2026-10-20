# Stacks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:orchestration/v1/stacks.Abandon`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Abandon`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Abandon`

Python/Go differences:

- Go exposes the fixed-identity native typed AbandonedStack response rather than a mutable Resource.

Unresolved boundaries:

- Add DELETE /stacks/{name}/{id}/abandon with required HTTP 200 and an AbandonedStack payload fixture, plus errors/cancellation/decoding and preservation of all returned fields. No existing scope HTTP test calls Abandon.

### `gophercloud:orchestration/v1/stacks.Delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- DELETE uses the fixed name/ID without a lookup; default 404 is ignored and explicit strict 404 returns resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.

Python/Go differences:

- Scoped Go deletion fixes name/ID and ignores 404 by default; native Delete directly returns its HTTP error.

Unresolved boundaries:

- Add positive native DELETE 202/204 fixtures plus forbidden/server-error/cancellation assertions for the mutating request itself. Current scoped delete fixture only returns 404; deletion waiter GET cases do not verify DELETE success or errors.

### `gophercloud:orchestration/v1/stacks.Find`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithResolveOutputs`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Duplicate exact names across pages fail, strict absence fails and explicit ignore-missing returns nil; empty collection is non-nil.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackExactDuplicateNamesAcrossPagesAndMissingPolicies`.
- The ID lookup follows a canonical redirect with the false query written once and retains authentication and client headers.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.

Python/Go differences:

- Go explicit Name references resolve exact list summaries and ID references fetch details, whereas native Find passes a single identity string to Heat.

Unresolved boundaries:

- Test the unchanged native Find name-or-ID /stacks/{identity} redirect entry point and its result/errors separately; exact-name SDK list resolution plus the canonical ID fixture do not exercise its name-string variant.

### `gophercloud:orchestration/v1/stacks.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithResolveOutputs`

- A fixed name/ID GET returns a detailed native field model; parameter values, output presence, rollback=false, timeout and query omission/resolve_outputs=False are asserted.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- The ID lookup follows a canonical redirect with the false query written once and retains authentication and client headers.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.
- Changed name/ID, missing/null objects, incomplete identity and bad timestamps fail; 403/404 retain an unexpected-response cause and strict 404 is resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIdentityMismatchesAndHTTPErrorsArePreserved`.
- Invalid compound identities, nil GET option and canceled GET fail without HTTP.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Go returns StackResource with Detailed=true instead of GetResult; the scope owns canonical name/ID, validates response identity and optionally sends resolve_outputs=False without changing the original native API. Context/errors are explicit.

### `gophercloud:orchestration/v1/stacks.List`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Positive limit advances last canonical ID as marker until an empty page; break avoids the next request and unchanged markers produce a cycle error before refetch.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerMarkerFallbackCycleAndBreak`.
- An identical relative next URL produces a cycle error after one request; break skips next-link processing.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerRepeatedLinkUsesCommonGuard`.

Python/Go differences:

- Go SDK resources use linked pages with relative-link resolution and positive-limit marker fallback rather than pinned native SinglePageBase; summaries share a StackResource model and explicit Detailed=false.

Unresolved boundaries:

- Audit and add assertions for native list typed query filters and defaults (tenant/id/status/name/sort, global_tenant, show_deleted/show_nested/show_hidden, tags/tags_any/not_tags/not_tags_any). Current scoped fixtures prove page size/name lookup/paging and selected summary fields, not every declared ListOpts serialization.

### `gophercloud:orchestration/v1/stacks.Update`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdateField`

Python/Go differences:

- The scope retains name/ID and delegates to concrete native PUT options; no successful PUT contract is established by a PATCH fixture.

Unresolved boundaries:

- Add an actual PUT fixture asserting the required template, Template/Environment/file handling, body/extensions, native 201/202 success codes and errors. Method existence alone is not completion evidence.

### `gophercloud:orchestration/v1/stacks.UpdatePatch`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.UpdatePatch`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.UpdatePatch`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdatePatchField`

- PATCH can omit a template and sends parameters, comma-joined tags and explicit false extension at the fixed name/ID endpoint; 204 succeeds.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.
- Canceled scoped PATCH is rejected without a request.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Go supplies concrete options and field extensions with a fixed canonical pair and returns error; the native template-optional PATCH behavior is preserved in the exercised subset.

Unresolved boundaries:

- Audit Template/Environment parsing and referenced-file merging, explicit empty versus omitted Tags/Parameters, all native success statuses 200/202/204 and HTTP/decode/option errors. The present PATCH fixture proves template omission, parameters/tags/false extension and 204, not the complete helper surface.

### `python:orchestration/v1/abandon_stack`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Abandon`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Abandon`

Python/Go differences:

- Python Stack.abandon returns resp.json without its own raise_from_response; Go uses native HTTP-200 validation and typed AbandonedStack extraction.

Unresolved boundaries:

- Audit Proxy._get_resource and Stack.abandon HTTP/JSON/error behavior including a Python Resource argument; add an actual scoped abandon fixture and reconcile unknown response fields. Binding or method existence does not verify abandonment.

### `python:orchestration/v1/delete_stack`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- DELETE uses the fixed name/ID without a lookup; default 404 is ignored and explicit strict 404 returns resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.

Python/Go differences:

- Go explicit references/complete identity and concrete missing-policy options replace Python string-or-Stack input and exceptions.

Unresolved boundaries:

- Audit Proxy._delete and inherited Resource.delete/existing identity, microversion, URL preparation and error policy; exercise native successful DELETE and authorization/server errors as well as the 404 policy before mapping the whole operation.

### `python:orchestration/v1/find_stack`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.InStack`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithResolveOutputs`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Duplicate exact names across pages fail, strict absence fails and explicit ignore-missing returns nil; empty collection is non-nil.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackExactDuplicateNamesAcrossPagesAndMissingPolicies`.
- The ID lookup follows a canonical redirect with the false query written once and retains authentication and client headers.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.

Python/Go differences:

- Python defaults ignore_missing=True and returns a detailed Resource through Stack.find/fetch. Go requires explicit ID versus Name, defaults missing lookup to error and returns a summary for Name; duplicates are rejected using a full exact local scan.

Unresolved boundaries:

- Audit Proxy._find and the Resource existing/request/cache/response path used by direct Stack.find/fetch, including resolve_outputs forwarded through find_stack.
- Add behavioral evidence for Python ambiguous/name redirect and DELETE_COMPLETE/ADOPT_COMPLETE filtering versus Go policies. Name Find does not accept WithResolveOutputs; obtaining details is a separate scoped Get.

### `python:orchestration/v1/get_stack`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithResolveOutputs`

- A fixed name/ID GET returns a detailed native field model; parameter values, output presence, rollback=false, timeout and query omission/resolve_outputs=False are asserted.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- The ID lookup follows a canonical redirect with the false query written once and retains authentication and client headers.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.
- Changed name/ID, missing/null objects, incomplete identity and bad timestamps fail; 403/404 retain an unexpected-response cause and strict 404 is resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIdentityMismatchesAndHTTPErrorsArePreserved`.
- Invalid compound identities, nil GET option and canceled GET fail without HTTP.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Go returns a new typed StackResource and explicit error, retains DELETE_COMPLETE/ADOPT_COMPLETE data in normal GET and validates canonical identity. Python Stack.fetch mutates a Resource and treats those two complete states as NotFound.

Unresolved boundaries:

- Audit Proxy._get and Stack.fetch inherited request preparation, microversion/cache/skip_cache and response translation; cover existing Python Stack input as well as ID input.
- Reconcile declared Python Stack response fields absent from the native model, including deletion_time, owner/parent IDs, environment/files/template and preview-change fields, before declaring the whole operation mapped.

### `python:orchestration/v1/stack_events`

Status: `unresolved`.

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

Unresolved boundaries:

- Audit inherited Proxy._get/_list and StackEvent Resource/MRO query mapping, pagination, descriptors/cache/response hydration and Resource input behavior.
- Add assertions for complete resource-scoped typed options and explicitly resolve Python sort_key versus Heat sort_keys compatibility; no inherited StackEvent.fetch parity is implied by the Go resource-scoped GET feature.

### `python:orchestration/v1/stacks`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Positive limit advances last canonical ID as marker until an empty page; break avoids the next request and unchanged markers produce a cycle error before refetch.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerMarkerFallbackCycleAndBreak`.
- An identical relative next URL produces a cycle error after one request; break skips next-link processing.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerRepeatedLinkUsesCommonGuard`.

Python/Go differences:

- Go returns typed summaries with Detailed=false rather than mutable Stack Resources; query names are HTTP names and Python project_id/tag aliases are not automatic Go aliases.

Unresolved boundaries:

- Audit inherited Proxy._list/Resource.list query transformation and descriptor/default handling, particularly project_id->tenant_id and CSV tag aliases any_tags/not_any_tags.
- Verify all declared Python Stack summary/body fields and MRO list hydration/defaults; current tests prove selected native fields and linked/marker traversal, not complete Python Resource equivalence.

### `python:orchestration/v1/update_stack`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.UpdatePatch`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.UpdatePatch`

- PATCH can omit a template and sends parameters, comma-joined tags and explicit false extension at the fixed name/ID endpoint; 204 succeeds.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.
- Canceled scoped PATCH is rejected without a request.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Go separates PUT from template-optional PATCH and returns error, while Python Stack.commit mutates Resource dirty state and translates a response.
- Pinned Proxy.update_stack passes preview positionally to Stack.commit prepend_key, but Stack.commit preview is keyword-only; the source wiring must not be assumed to exercise a preview endpoint.

Unresolved boundaries:

- Audit inherited _get_resource/_prepare_request, dirty-body/id removal and response translation; provide PUT versus PATCH and preview behavior fixtures rather than treating the PATCH test as Python commit evidence.
- Define and test compatibility for the pinned positional preview wiring, direct Stack.commit keyword-only preview, template/environment/files and incomplete name-or-ID paths.

### `python:orchestration/v1/wait_for_delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`

- Stack deletion wait succeeds on HTTP 404 or DELETE_COMPLETE, rejects DELETE_FAILED and preserves 403 as an HTTP error.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeDeletionWaitRecognizesCompletionFailureAndHTTP`.
- Canceled stack deletion waiting performs no HTTP.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.
- Shared Go WaitDeleted completes on 404, nil successful fetch or case-insensitive deleted status, skips terminal callbacks, stops further GET after callback cancellation and honors explicit unlimited wait with parent deadlines.
  Tests: `resource/wait_attributes_test.go:TestWaitDeletedProgressStopsAtDeletedStatusOr404`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`, `resource/wait_policy_test.go:TestWaitUnlimitedAndBoundedOptionsPreserveParentContext`.

Python/Go differences:

- Go deletion wait returns error, uses a five-minute default and treats 404 or DELETE_COMPLETE as completion while ADOPT_COMPLETE is not inferred as deletion. Python returns the Resource and its Stack.fetch treats both complete states as NotFound.

Unresolved boundaries:

- Audit Stack.fetch and cached/mutated Resource return/error semantics, including Python ADOPT_COMPLETE behavior. Unlimited wait and progress callbacks now exist in the shared Go helper; Heat-specific deletion errors remain authoritative.
- Add Heat transient polling, timeout, in-flight cancellation and callback evidence before whole-operation mapping. Go returns error rather than a Python Resource.

### `python:orchestration/v1/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Wait`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`

- Stack polling keeps the fixed name/ID, returns detailed CREATE_COMPLETE and terminates on CREATE_FAILED, UPDATE_FAILED or ROLLBACK_FAILED after exactly two requests.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeWaitUsesFixedPairAndDetectsFailedStates`.
- Empty target status fails before HTTP.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.
- Shared Go Wait policy supports exact replacement/disabled failure states, model-owned string attribute selection, unlimited wait with parent cancellation, and integer progress callbacks on initial and later nonterminal HTTP results; target/failure/HTTP results omit callbacks.
  Tests: `resource/wait_policy_test.go:TestWaitFailurePolicyReplacementAndTargetPrecedence`, `resource/wait_policy_test.go:TestWaitUnlimitedAndBoundedOptionsPreserveParentContext`, `resource/wait_attributes_test.go:TestWaitStatusAttributeSelectsNativeNodeFields`, `resource/wait_attributes_test.go:TestWaitProgressIncludesEachNonterminalNativeServerResponse`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`.

Python/Go differences:

- Go stack wait resolves once and polls canonical name/ID, defaults to five minutes/two seconds and treats *_FAILED or ERROR as terminal. Python Proxy delegates a generic Resource with custom failures/attribute/callback and wait=None.

Unresolved boundaries:

- Audit cached/mutated Python Resource, None target status, Stack.fetch, skip_cache and Adapter error/timeout semantics against mandatory Go lookup/context deadlines. Configurable failure states, attribute selection, callbacks and unlimited wait now exist in the shared Go helper.
- Add stack-specific timeout, in-flight cancellation, custom failure/attribute/callback evidence before whole-operation mapping; existing stack default success/failure fixtures and shared Node/Server tests cover different contracts.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Stacks.InStack(ctx, resource.Name("app"))
if err != nil { return err }
identity := scope.Identity()
fmt.Println(identity.Name, identity.ID)

stack, err := scope.Get(ctx, stacks.WithResolveOutputs(false))
if err != nil { return err }
fmt.Println(stack.Name, stack.ID, stack.Status, stack.Detailed)
```

```go
scope, err := service.Stacks.ForStack(stacks.StackIdentity{
    Name: "app",
    ID: "stack-uuid",
})
if err != nil { return err }
if err := scope.UpdatePatch(ctx, stacks.UpdateOpts{
    Parameters: map[string]any{"size": "large"},
    Tags: []string{"ops", "test"},
}); err != nil { return err }
if _, err := scope.Wait(ctx, "UPDATE_COMPLETE"); err != nil { return err }
```

```go
if err := scope.Delete(ctx); err != nil { return err }
if err := scope.WaitDeleted(ctx); err != nil { return err }
```

## References

- [..](../README.md)
- [Heat Stacks Contracts Test](../../../api/heat_stacks_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)

## Native API calls

The generated `API` methods are the unchanged native entry points and take the stack name and ID, or a single identity string, as separate path segments. They are covered by `api/heat_native_stacks_test.go`.

```go
api := stacks.New(client)

// Name-or-ID lookup: one GET /stacks/{identity}; Heat's redirect is followed.
stack, err := api.Find(ctx, "app")

// Single-page list with every typed filter; booleans that are false are omitted.
for stack, err := range api.List(ctx, stacks.WithListOptions(stacks.ListOpts{Name: "app", AllTenants: true, Tags: "a,b"})) {
    if err != nil { panic(err) }
    fmt.Println(stack.ID)
}

// PUT requires a template; PATCH does not. Update accepts HTTP 201/202, UpdatePatch accepts 200/202/204.
err = api.Update(ctx, "app", id, stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: tpl}}})
err = api.UpdatePatch(ctx, "app", id, stacks.UpdateOpts{Parameters: map[string]any{"size": 2}})

// Delete accepts HTTP 202/204 and returns every other status as an error. Abandon requires HTTP 200.
err = api.Delete(ctx, "app", id)
abandoned, err := api.Abandon(ctx, "app", id)
```

Limitations: an empty non-nil `Tags` slice is sent as `"tags": ""`, while `nil` tags and an empty `Parameters` map are omitted. Native `Delete` does not ignore 404. A canceled context fails before any request. Mutating calls are never retried.
