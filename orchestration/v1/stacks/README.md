# Stacks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:orchestration/v1/stacks.Abandon`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Abandon`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Abandon`

- Abandon sends DELETE /stacks/{name}/{id}/abandon, requires HTTP 200 and preserves every AbandonedStack field including nested nulls, files and environment; 202, 403, 404, 500 and malformed JSON fail, and canceled contexts send nothing.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksAbandonRequiresHTTP200AndPreservesPayload`.

Python/Go differences:

- Go exposes the fixed-identity native typed AbandonedStack response rather than a mutable Resource.

### `gophercloud:orchestration/v1/stacks.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Delete`

- DELETE uses the fixed name/ID without a lookup; default 404 is ignored and explicit strict 404 returns resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.
- Native DELETE accepts 202 and 204 and returns 200, 401, 403, 409 and 500 as HTTP errors; the scoped call propagates those errors even with the strict missing policy, and a canceled context sends nothing.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksDeleteAcceptsOnly202And204AndKeepsHTTPErrors`.

Python/Go differences:

- Scoped Go deletion fixes name/ID and ignores 404 by default; native Delete directly returns its HTTP error.

### `gophercloud:orchestration/v1/stacks.Find`

Status: `go_mapping`.

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
- Native Find sends a single GET /stacks/{identity} with no query, follows the canonical redirect for both a name and an ID while keeping auth and client headers, decodes the stack (false, zero, null output values) and keeps 403/404, malformed JSON, null object and cancellation behavior.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksFindUsesSingleIdentityRedirectAndKeepsErrors`.

Python/Go differences:

- Go explicit Name references resolve exact list summaries and ID references fetch details, whereas native Find passes a single identity string to Heat.

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

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithListQuery`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Positive limit advances last canonical ID as marker until an empty page; break avoids the next request and unchanged markers produce a cycle error before refetch.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerMarkerFallbackCycleAndBreak`.
- An identical relative next URL produces a cycle error after one request; break skips next-link processing.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerRepeatedLinkUsesCommonGuard`.
- Native List sends no query for zero options and serializes every ListOpts filter (tenant_id, id, status, name, marker, limit, sort_keys, sort_dir, global_tenant, show_deleted, show_nested, show_hidden, tags, tags_any, not_tags, not_tags_any) plus extension queries, omits false booleans, performs a single request, and handles 204, bad arrays, 403 and cancellation.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksListSerializesEveryTypedFilter`.

Python/Go differences:

- Go SDK resources use linked pages with relative-link resolution and positive-limit marker fallback rather than pinned native SinglePageBase; summaries share a StackResource model and explicit Detailed=false.
- Native API.List is a single-page SinglePageBase request that never follows links; the scoped collection pages by link or marker.

### `gophercloud:orchestration/v1/stacks.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdateField`

- Update sends PUT with the required template, environment, parameters (including false), timeout, comma-joined tags, referenced child-template files and extension fields; omitted options stay omitted; 201/202 succeed; 200/204/400/403/409/500 fail with their code; a missing template, unparseable template, protected-field override and canceled context fail before HTTP.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksUpdateSendsPUTTemplateEnvironmentFilesAndExtensions`.

Python/Go differences:

- The scope retains name/ID and delegates to concrete native PUT options; a PATCH is a separate operation with different required fields and success codes.

### `gophercloud:orchestration/v1/stacks.UpdatePatch`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.UpdatePatch`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.UpdatePatch`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdatePatchField`

- PATCH can omit a template and sends parameters, comma-joined tags and explicit false extension at the fixed name/ID endpoint; 204 succeeds.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.
- Canceled scoped PATCH is rejected without a request.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.
- UpdatePatch accepts 200, 202 and 204 and fails 201/400/403/409/500; nil tags and empty parameters are omitted while an empty non-nil tag list is sent as an empty string; template and environment are optional, parsed and merged with referenced files; extension fields, unparseable templates, protected-field overrides and cancellation are rejected before HTTP.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksUpdatePatchAcceptsSuccessCodesAndPreservesEmptyTags`.

Python/Go differences:

- Go supplies concrete options and field extensions with a fixed canonical pair and returns error; the native template-optional PATCH behavior is preserved in the exercised subset.

### `python:orchestration/v1/abandon_stack`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Abandon`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Abandon`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.InStack`

- DELETE /stacks/{name}/{id}/abandon uses the fixed identity without a lookup, requires HTTP 200, returns the abandoned-stack payload (name, id, status, resources and unknown fields decode) and does not replay; 202 and canceled contexts fail without success.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinScopedAbandonAndDeleteUseFixedIdentity`.
- The native API.Abandon requires HTTP 200 and preserves the payload and native errors.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksAbandonRequiresHTTP200AndPreservesPayload`.

Python/Go differences:

- Python Stack.abandon builds /stacks/{name}/{id}/abandon from the Resource; for an ID string the name is empty and urljoin produces stacks//{id}/abandon. It returns resp.json() without raise_from_response, so an error body is returned as data. Go always uses both canonical components, requires HTTP 200 and reports every other status as an error.
- Go returns typed values and errors instead of Python Resource objects or None; a Stack, Resource or ID argument is replaced by an explicit resource.Ref resolved once to a fixed name/ID scope.

### `python:orchestration/v1/delete_stack`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

- DELETE uses the fixed name/ID without a lookup; default 404 is ignored and explicit strict 404 returns resource.ErrNotFound.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy`.
- Scoped DELETE succeeds on 202 and 204, keeps 403 and 500 as native errors and sends exactly one request per call (no replay, no lookup).
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinScopedAbandonAndDeleteUseFixedIdentity`.
- Native API.Delete accepts only 202 and 204 and keeps HTTP errors.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksDeleteAcceptsOnly202And204AndKeepsHTTPErrors`.

Python/Go differences:

- Python Proxy._delete(Stack, stack, ignore_missing=True) issues DELETE /stacks/{id} from the Resource, and any status below 400 succeeds; Go issues DELETE /stacks/{name}/{id} for the canonical identity and accepts 202 and 204 only. Python ignores 404 by default and raises NotFoundException with ignore_missing=False; Go ignores by default and uses WithMissingError for the strict policy.
- Python returns None and gives no completion signal; deletion waiting is the separate wait_for_delete helper (see that review).
- Go returns typed values and errors instead of Python Resource objects or None; a Stack, Resource or ID argument is replaced by an explicit resource.Ref resolved once to a fixed name/ID scope.

### `python:orchestration/v1/find_stack`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.FindNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithFindResolveOutputs`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Find`

- One GET /stacks/{nameOrID} follows Heat's redirect to the canonical location, issues no list request, appends resolve_outputs=False exactly once when requested and returns the typed model plus the complete raw stack object (deletion_time, stack_owner, parent, stack_user_project_id, files_container, environment_files, files, template and extension fields, including an exact large integer).
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinFindNameOrIDUsesOneGETKeepsRawFieldsAndHidesDeleted`.
- 404, DELETE_COMPLETE and ADOPT_COMPLETE are reported as resource.ErrNotFound or nil with WithFindIgnoreMissing; DELETE_FAILED stays visible; 403 is never ignored; a missing stack envelope or identity, an empty reference, a nil option and a canceled context fail before a second request.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinFindNameOrIDUsesOneGETKeepsRawFieldsAndHidesDeleted`.
- The native single-identity Find keeps the same redirect and error behavior, and exact-name list resolution remains available through StackResources.Find with duplicate detection.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksFindUsesSingleIdentityRedirectAndKeepsErrors`, `api/heat_stacks_contracts_test.go:TestHeatStackExactDuplicateNamesAcrossPagesAndMissingPolicies`, `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.

Python/Go differences:

- Python find_stack defaults ignore_missing=True and returns None; Go FindNameOrID returns resource.ErrNotFound by default and uses WithFindIgnoreMissing for nil.
- Python Stack.find fetches into a mutable Resource and falls back to the whole response body when the stack key is missing; Go requires a stack object with a complete canonical name and ID and returns a StackDetail value.
- Python resolve_outputs is a find_stack argument; Go uses WithFindResolveOutputs. Neither sends a list request or detects duplicates in this path, because Heat enforces unique names.
- Go returns typed values and errors instead of Python Resource objects or None; a Stack, Resource or ID argument is replaced by an explicit resource.Ref resolved once to a fixed name/ID scope.

### `python:orchestration/v1/get_stack`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.FindNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithFindResolveOutputs`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithResolveOutputs`

- A strict FindNameOrID is Python get_stack: one GET (with resolve_outputs=False only when requested, a followed redirect), missing, DELETE_COMPLETE and ADOPT_COMPLETE stacks are ErrNotFound, and the raw object retains every declared Python Stack field absent from the native model.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinFindNameOrIDUsesOneGETKeepsRawFieldsAndHidesDeleted`.
- A fixed name/ID GET returns the detailed native model; resolve_outputs=False is written once and the canonical redirect keeps headers.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`, `api/heat_stacks_contracts_test.go:TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders`.
- Changed identity, missing or null objects, bad timestamps and HTTP 403/404 fail with preserved causes, and invalid identities or canceled contexts send nothing.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackIdentityMismatchesAndHTTPErrorsArePreserved`, `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Python Stack.fetch mutates and returns the passed Resource, honors skip_cache and treats DELETE_COMPLETE and ADOPT_COMPLETE as NotFound; Go FindNameOrID returns an independent StackDetail with the same two states missing, while the fixed-identity StackScope.Get and StackResources.Get retain those states as data.
- Python accepts an ID or a Stack object (GET /stacks/{id}); Go FindNameOrID accepts a name or ID reference string and the scope variants require a complete canonical identity. The typed StackResource still lacks Python-only fields such as deletion_time, owner and parent IDs; StackDetail.Raw carries them without coercion.
- Go returns typed values and errors instead of Python Resource objects or None; a Stack, Resource or ID argument is replaced by an explicit resource.Ref resolved once to a fixed name/ID scope.

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

### `python:orchestration/v1/stacks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackResources.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithListOptions`

- Exact local name lookup covers relative next links, avoids additional detail GETs, preserves canonical name/ID and distinguishes summaries from detailed ID GETs.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails`.
- Positive limit advances last canonical ID as marker until an empty page; break avoids the next request and unchanged markers produce a cycle error before refetch.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerMarkerFallbackCycleAndBreak`.
- An identical relative next URL produces a cycle error after one request; break skips next-link processing.
  Tests: `api/heat_stacks_contracts_test.go:TestHeatStackSDKPagerRepeatedLinkUsesCommonGuard`.
- Native List serializes every typed filter (tenant_id, id, status, name, marker, limit, sort_keys, sort_dir, global_tenant, show_deleted, show_nested, show_hidden, tags, tags_any, not_tags, not_tags_any), omits false booleans and zero ListOpts sends no query.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksListSerializesEveryTypedFilter`.
- Python filter names with no typed Go field (action, owner_id, username) and wire names such as tenant_id, tags_any and not_tags_any reach Heat verbatim through explicit queries; the Python aliases project_id, any_tags and not_any_tags are not translated.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinStackListForwardsPythonFilterNamesThroughExplicitQueries`.

Python/Go differences:

- Go returns typed summaries with Detailed=false rather than mutable Stack Resources; query names are HTTP names and Python project_id/tag aliases are not automatic Go aliases.
- Python Stack query mapping accepts action, name, status, project_id (sent as tenant_id), owner_id, username and CSV tags/any_tags/not_tags/not_any_tags (sent as tags, tags_any, not_tags, not_tags_any) plus limit and marker; Go has typed fields for the Heat names it declares and sends other documented filters (action, owner_id, username) only through explicit WithQuery/WithListQuery values, without Python aliases.
- Python Resource.list discards unknown keywords and turns declared Body attributes (for example status_reason or description) into client filters; Go does not discard: unknown names are sent only when the caller supplies an explicit query, and there is no implicit local Body filtering for stacks.
- Python rows pass through Stack descriptors with defaults (tags defaults to an empty list, added/deleted/replaced fields and others); Go list summaries are typed ListedStack/StackResource values with Detailed=false and carry no Python defaults.
- Python also honors per-call base_path and jmespath_filters through Proxy._list; Go does not expose either.

### `python:orchestration/v1/update_stack`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.StackScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks.WithUpdateOptions`

- PUT /stacks/{name}/{id} sends template, environment, parameters (including false), timeout, tags and an explicit false extension; a template is required, omitted optional values stay omitted and referenced child templates merge into files.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksUpdateSendsPUTTemplateEnvironmentFilesAndExtensions`.
- Scoped Update keeps the fixed identity, honors cancellation before HTTP and reports statuses other than 201/202 with their native code without replay.
  Tests: `api/heat_native_stacks_test.go:TestHeatNativeStacksUpdateSendsPUTTemplateEnvironmentFilesAndExtensions`, `api/heat_stacks_contracts_test.go:TestHeatStackScopeValidationAndCancellationBeforeHTTP`.

Python/Go differences:

- Python Stack.commit builds PUT /stacks/{name}/{id} (or /stacks/{id} when only one is known) from the dirty attributes of a mutable Resource, translating any status below 400 into the returned Resource; Go sends only its concrete UpdateOpts to the canonical identity and returns an error value.
- In the pinned proxy, update_stack(preview=True) passes preview positionally to Stack.commit, where it becomes the ignored prepend_key (preview is keyword-only), so Python always performs an ordinary PUT. Go has no preview update and no claim of one; the separate PATCH operation UpdatePatch is Go-only.
- Python sends the template as a dict and tags as a list; Go sends strings and a comma-joined tags value through the native builder and requires a template.
- Go returns typed values and errors instead of Python Resource objects or None; a Stack, Resource or ID argument is replaced by an explicit resource.Ref resolved once to a fixed name/ID scope.

### `python:orchestration/v1/wait_for_delete`

Status: `go_mapping`.

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
- Transient DELETE_IN_PROGRESS responses are polled until DELETE_COMPLETE, nonterminal responses report zero progress and the terminal one reports none; ADOPT_COMPLETE is not inferred as deletion and a bounded wait ends with context.DeadlineExceeded.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinStackWaitDeletedTransientAdoptAndCallback`.

Python/Go differences:

- Go deletion wait returns error, uses a five-minute default and treats 404 or DELETE_COMPLETE as completion while ADOPT_COMPLETE is not inferred as deletion. Python returns the Resource and its Stack.fetch treats both complete states as NotFound.
- Python wait_for_delete returns the original Resource, defaults to a 120 second wait and treats a NotFound from Stack.fetch (404, DELETE_COMPLETE or ADOPT_COMPLETE) or a status equal to deleted as completion; Go StackScope.WaitDeleted returns only an error, defaults to the shared five-minute policy and recognizes HTTP 404 and DELETE_COMPLETE but not ADOPT_COMPLETE.
- Go returns ErrFailedState immediately for DELETE_FAILED, while Python keeps polling until its timeout.

### `python:orchestration/v1/wait_for_status`

Status: `go_mapping`.

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
- Transient responses are polled until the target, callbacks receive 0 for every nonterminal response (Stack has no progress field), bounded waits end with context.DeadlineExceeded, WithFailureStates("ERROR") replaces the *_FAILED default so a CREATE_FAILED stack keeps polling, WithStatusAttribute waits on another string field and parent cancellation stops an unlimited wait in flight.
  Tests: `api/py_heat_senlin_stacks_test.go:TestPyHeatSenlinStackWaitTransientTimeoutCustomPolicyAndCancellation`.

Python/Go differences:

- Go stack wait resolves once and polls canonical name/ID, defaults to five minutes/two seconds and treats *_FAILED or ERROR as terminal. Python Proxy delegates a generic Resource with custom failures/attribute/callback and wait=None.
- Python wait_for_status returns the passed Resource immediately, with no request, when its cached status already equals the target; Go StackScope.Wait always begins with a request on the fixed identity.
- Python default failures are only ERROR, so a stack in CREATE_FAILED is polled until the timeout (forever when wait=None); the Go default also stops on any *_FAILED state and on ERROR, and WithFailureStates reproduces the Python list.
- Python fetches with Stack.fetch, which raises NotFound for DELETE_COMPLETE and ADOPT_COMPLETE, so waiting for those targets raises instead of matching; Go returns the status as data.
- Python raises ResourceTimeout and ResourceFailure and returns the updated mutable Resource; Go returns a StackResource and context/ErrFailedState errors. Attribute names are Go JSON tags or field names of the typed model, not Python descriptor names.

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
