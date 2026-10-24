# Policies

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/create_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Create`

- POST201 sends a single resource object with required name and spec; library-owned JSON options retain nested false, exact numbers and independent option reuse, while core/identity/header collisions fail before HTTP.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyCreateEnvelopePrecisionAndOptionsSnapshot`, `api/clustering_policies_test.go:TestClusteringPolicyCreateSpecConvenienceSnapshotsObjects`, `api/clustering_policies_test.go:TestClusteringPolicyMutationPreflightRejectsInvalidCoreAndExtensions`, `api/clustering_policies_test.go:TestClusteringPolicyAcceptedMalformedMutationDoesNotResend`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.

### `python:clustering/v1/get_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Get`

- GET200 routes a controller name, UUID or short ID directly and decodes only the documented singular object envelope, preserving null/missing identities and raw HTTP response evidence.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyGetIdentityNullAndEnvelopeEvidence`, `api/clustering_policies_test.go:TestClusteringPolicyRejectsWrongSuccessCodeAndPreservesHTTPError`.
- Missing, null and malformed accepted objects retain the exact HTTP body, status and header without a repeated request; an undeclared 201 response is rejected.
  Tests: `api/clustering_policies_response_test.go:TestClusteringPolicyGetAndValidateRetainMalformedAcceptedResponseEvidence`, `api/clustering_policies_response_test.go:TestClusteringPolicyGetAndValidateRejectUndeclaredSuccessCodes`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.

### `python:clustering/v1/delete_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Delete`

- DELETE204 uses an explicit route or one exact-name lookup; ignored404 remains the default, strict404/403/409 preserve causes, and a missing resolved response ID cannot delete the collection path.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyFindDefaultsAmbiguityAndDeleteConflict`, `api/clustering_policies_test.go:TestClusteringPolicyDeleteSuccessAndMissingResponseIDCannotBeUsed`.
- The shared Collection validates a resolved identity with the binding before Delete, preventing empty, whitespace or a wrong-kind ID from becoming a collection-path request.
  Tests: `resource/delete_identity_test.go:TestDeleteNameValidatesResolvedIdentityBeforeMutation`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.

### `python:clustering/v1/validate_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Validate`

- POST200 to the resource validate endpoint requires a selected numeric microversion>=1.2 before HTTP, sends the spec object and preserves nullable result identity and precise JSON fields.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyValidateVersionGateAndPrecision`, `api/clustering_policies_test.go:TestClusteringPolicyContextAndVersionHeadersPreflight`.
- A custom option that demotes the selected microversion from 1.2 to 1.1 is rejected after body preparation and before the validation POST.
  Tests: `api/clustering_mutation_recheck_test.go:TestClusteringMutationsRecheckConfiguredSourceAfterOptions`.
- Missing, null and malformed accepted objects retain the exact HTTP body, status and header without a repeated request; an undeclared 201 response is rejected.
  Tests: `api/clustering_policies_response_test.go:TestClusteringPolicyGetAndValidateRetainMalformedAcceptedResponseEvidence`, `api/clustering_policies_response_test.go:TestClusteringPolicyGetAndValidateRejectUndeclaredSuccessCodes`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.

### `python:clustering/v1/update_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Track`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Edit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.RemoveName`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Commit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Refresh`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Value`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Response`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.TrackedPolicy.Dirty`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithUpdateName`

- PATCH200 sends a resource object rather than JSON Patch operations; optional mutable fields and headers are prepared before exact-name resolution, explicit IDs skip lookup, and empty updates fail before HTTP.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyUpdateNameResolvedOnceAndInputsFrozen`, `api/clustering_policies_test.go:TestClusteringPolicyUpdateIDSkipsLookupAndSnapshotCanBeReused`, `api/clustering_policies_test.go:TestClusteringPolicyMutationPreflightRejectsInvalidCoreAndExtensions`.
- The SDK-owned tracked handle reuses concrete UpdateOpts and With options; current-value assignment equality gives clean same-value/no-change commits, sticky A-to-B-to-A dirty fields, and null tombstones when a cached mutable field is removed. Only dirty fields reach the PATCH200 object envelope; profile metadata preserves omission, null and empty-object presence.
  Tests: `api/clustering_lifecycle_test.go:TestClusteringTrackedNoOpStickyDirtyMergeAndSeparateResponse`, `api/clustering_lifecycle_test.go:TestClusteringTrackedDeletionNullAndMetadataPresence`, `internal/senlin/tracked_test.go:TestTrackedCurrentComparisonStickyDirtyAndRemoval`.
- Track owns an authoritative cached Body or a typed-only local seed without invented HTTP evidence; Value and Response are deep snapshots of the merged cache and last successful response fields. Explicit Load IDs remain fixed and exact Name lookup occurs once even when response IDs change, are null or are omitted.
  Tests: `api/clustering_lifecycle_test.go:TestClusteringTrackedSnapshotsAndAuthoritativeTrackIdentity`, `api/clustering_lifecycle_test.go:TestClusteringTrackedManualSeedAndInvalidCachedModels`, `api/clustering_lifecycle_test.go:TestClusteringTrackedExplicitLoadKeepsRequestIDAndNameResolvesOnce`, `internal/senlin/tracked_test.go:TestTrackedOwnsAllSnapshotsAndRejectedEditIsAtomic`.
- Valid responses shallow-merge whole fields, replace nested objects, preserve omitted cached values and clean accepted revisions even when an edited field is omitted. Refresh GETs the fixed ID and resets pre-fetch revisions; edits made during Commit or Refresh survive response merge and remain dirty for the next explicit commit.
  Tests: `api/clustering_lifecycle_test.go:TestClusteringTrackedNoOpStickyDirtyMergeAndSeparateResponse`, `api/clustering_lifecycle_test.go:TestClusteringTrackedRefreshMergeResetAndFailurePreservesChanges`, `api/clustering_lifecycle_test.go:TestClusteringTrackedNewEditsSurviveCommitAndRefreshInFlight`, `internal/senlin/tracked_test.go:TestTrackedAcceptShallowMergeAndPreservesNewerEdits`.
- Invalid edits and body/query/auth-header commit options fail atomically before HTTP, including clean commits; canceled contexts and invalid selected service/version remain errors. Native HTTP and transport failures preserve their cause and dirty state. Accepted malformed responses retain raw HTTP evidence in ResponseError, keep the last valid response and never trigger an automatic mutation resend.
  Tests: `api/clustering_lifecycle_test.go:TestClusteringTrackedInvalidEditsAndCommitOptionsStayAtomic`, `api/clustering_lifecycle_test.go:TestClusteringTrackedCommitFailureKeepsDirtyAndResponseEvidence`, `api/clustering_lifecycle_test.go:TestClusteringTrackedRefreshMergeResetAndFailurePreservesChanges`.

Python/Go differences:

- Stateless Update sends explicit fields and rejects an empty request; cached lifecycle is the separate concrete Track/Load plus Edit/Commit/Refresh API, with snapshots instead of mutating or returning the original Python Resource pointer. Apps need no builder or lifecycle interface implementation.
- The official mutable core is name only. Go rejects known readonly fields and case-variant typed collisions; explicit RemoveName sends a null tombstone, whose server acceptance is not assumed. Python permissive descriptors/attrs and implicit Body coercion are replaced by concrete owned options; explicit unknown vendor fields are a documented Go extension.
- Track uses raw cached Body and its decoded ID when present, ignoring caller mutations to exported typed fields; typed-only models are local seeds rather than evidence of an HTTP response. Explicit Load preserves the request ID, exact Name lookup uses one list resolution, and response IDs never retarget subsequent requests.
- JSON numbers compare exactly and booleans remain distinct from numbers, unlike Python True-equals-1 assignment comparison. Unknown response fields are preserved, Value.Body is the merged cache, and Response.Body contains last successful response fields; a manual Track Response remains the local seed until HTTP succeeds.
- Clean Commit validates context, selected service/version and options and returns without HTTP even with valid header-only options. Header dirtiness and allow_empty_commit do not force an empty Go PATCH; newer concurrent edits are revision-protected rather than cleared by an older response.
- The documented object envelope and success200 are strict. Python flat/empty fallback and ignored non-parseable responses are replaced by ResponseError with accepted HTTP evidence; failure keeps dirty state and the SDK does not automatically retry, so callers can Refresh before an explicit resend.
- Endpoint, authentication and numeric microversion belong to the selected client. Arbitrary Python Resource subclasses and inherited per-call base_path, prepend_key, has_body, retry_on_conflict and microversion controls are not options on this concrete Senlin mapping; implicit adapter negotiation and version upgrades are not performed.

### `python:clustering/v1/policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithListMicroversion`

- Lazy typed queries preserve selected filters and explicit false across links and last-wire-ID marker fallback, including short pages; local Body subsets, arrays, aliases and exact numeric filters never become server queries.
  Tests: `api/clustering_policies_test.go:TestClusteringPolicyListShortPagesRetainFiltersAndSnapshot`, `api/clustering_policies_test.go:TestClusteringPolicyLocalFiltersSnapshotExactNumbersAndNull`, `api/clustering_policies_test.go:TestClusteringPolicyListDefaultBreakAndInvalidExtensions`, `api/clustering_policies_test.go:TestClusteringPolicyNameGrammarAndInFlightListCancellation`.
- MaxItems counts validated wire rows before local filtering and zero is unlimited; positive caps supply a limit hint only when no explicit limit exists. Paginated=False consumes the first response, reusable options own bool pointers and later options win. Reaching a cap or an empty page stops before unused rows/continuations while defaults, context and next-page source guards remain active.
  Tests: `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP`, `api/clustering_typed_list_controls_test.go:TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.

Python/Go differences:

- Go takes concrete inputs, context and explicit ID/Name references instead of mutable Python Resource inputs and implicit combined-string identity guesses.
- The documented envelope is strict; raw fields, exact JSON numbers and accepted-response HTTP evidence are preserved rather than Python flat/empty fallback decoding and cached Resource mutation.
- Numeric selected microversions and protected body/query/header extensions replace implicit adapter negotiation; optional JSON values preserve omission, null and empty objects and the deployment validates null support.
- Explicit WithListFilter selects known local Body fields. JSON booleans stay distinct from numbers, decimal numbers compare exactly without float64, and non-object actual values fail object-subset filters without Python attribute errors.
- Go keeps continuation links on the same collection origin/path, retains original filters and selected limits, and guards URL/marker cycles rather than following arbitrary Python next-link parameters.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python policy query keys are name, type, sort and global_project.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/find_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
policy, err := service.Policies.Create(ctx, policies.CreateOpts{Name: "scale_workers"},
    policies.WithCreateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(policy.ID, policy.Spec, policy.Header)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
checked, err := service.Policies.Validate(ctx, policies.ValidateOpts{},
    policies.WithValidateSpec(map[string]any{
        "type": "senlin.policy.scaling", "version": "1.0",
        "properties": map[string]any{
            "event": "CLUSTER_SCALE_OUT",
            "adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
        },
    }))
if err != nil { return err }
fmt.Println(checked.Spec, checked.Body["id"])
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
renamed, err := service.Policies.Update(ctx, resource.Name("scale_workers"), policies.UpdateOpts{},
    policies.WithUpdateName("scale_out_workers"),
    policies.WithUpdateHeader("X-Vendor-Mode", "explicit"))
if err != nil { return err }
fmt.Println(renamed.ID, renamed.Name)
```

```python
policy = conn.clustering.get_policy("POLICY_ID")
policy.name = "scale_out_workers"
policy = conn.clustering.update_policy(policy)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.ID("POLICY_ID"))
if err != nil { return err }
if err := tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName("scale_out_workers")); err != nil {
    return err
}
policy, err := tracked.Commit(ctx)
if err != nil { return err }
fmt.Println(policy.Name, tracked.Dirty(), tracked.Response().Body)
```

```go
service, err := conn.Clustering(ctx)
if err != nil { return err }
for policy, err := range service.Policies.List(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20, Sort: "created_at:desc"}),
    policies.WithListGlobalProject(false),
    policies.WithListFilter("data", map[string]any{"vendor": map[string]any{"enabled": false}})) {
    if err != nil { return err }
    fmt.Println(policy.ID, policy.Name)
}
found, err := service.Policies.Find(ctx, resource.Name("scale_out_workers"))
if err != nil { return err }
if found != nil {
    if err := service.Policies.Delete(ctx, resource.ID(found.ID)); err != nil { return err }
}
```

```go
listingAPI := policies.New(client)
values, err := listingAPI.All(ctx,
    policies.WithListOptions(policies.ListOpts{Limit: 20}),
    policies.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, policies.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
)

func FindPolicyStrict(ctx context.Context, conn *sdk.Connection) (*policies.Policy, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Policies.FindIdentity(ctx, "placement",
        policies.WithFindIgnoreMissing(false))
}
```

## References

- [Scaling policy 1.0](https://docs.openstack.org/senlin/pike/user/policy_types/scaling.html)
- [Tracking](../tracking/README.md)
- [Finding](../finding/README.md)
- [tracked lifecycle](../tracking/README.md)
- [FindIdentity](../finding/README.md)
- [Policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/policy.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py)
- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [Clustering Policies Test](../../../api/clustering_policies_test.go)
- [Clustering Policies Response Test](../../../api/clustering_policies_response_test.go)
- [Clustering Lifecycle Test](../../../api/clustering_lifecycle_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
