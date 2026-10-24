# Receivers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/create_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Create`

- POST201 receiver creation preserves required name/type, webhook cluster/action, message omission, explicit null and exact raw actor/params; input/options are snapshotted and source/context rechecked before one request.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Go uses stateless owned concrete options/context and explicit ID/Name references instead of Python mutable Resource input. Nonempty vendor receiver types pass through to the controller, known webhook inputs require cluster/action and no Python CLI default type or unproved message microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.

### `python:clustering/v1/get_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Get`

- GET200 sends a controller name, short ID or UUID directly without an extra lookup, retains typed/raw receiver identity and ownership, and malformed or native HTTP responses preserve evidence.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Go uses stateless owned concrete options/context and explicit ID/Name references instead of Python mutable Resource input. Nonempty vendor receiver types pass through to the controller, known webhook inputs require cluster/action and no Python CLI default type or unproved message microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.

### `python:clustering/v1/delete_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Delete`

- DELETE204 preserves direct ID and exact Name resolution, defaults to ignoring only404, supports an explicit strict override and rejects invalid resolved identities without a mutation.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Go uses stateless owned concrete options/context and explicit ID/Name references instead of Python mutable Resource input. Nonempty vendor receiver types pass through to the controller, known webhook inputs require cluster/action and no Python CLI default type or unproved message microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.

### `python:clustering/v1/update_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Track`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.TrackedReceiver.Edit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.TrackedReceiver.Commit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.TrackedReceiver.Refresh`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.AtBasePath`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.UpdateScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.UpdateScope.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.UpdateScope.Track`

- Object PATCH200 accepts owned name/action/params, preserves omission/null/empty/exact numbers and freezes body/headers before Name lookup; source validation is repeated before mutation.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.
- Owned TrackedReceiver maps cached input through atomic offline edits, sticky current-value dirty tracking, absence/null/empty and removal tombstones, dirty-only PATCH200, whole-field response merge and separate actual response snapshots. Clean Commit validates context/source/version/header without HTTP; canonical raw ID/name or an explicit requested ID fixes the route. Native and malformed accepted errors retain whole HTTP evidence and state without resending; Refresh resets older revisions and HTTP-in-flight edits survive. Action remains a command name and incidental Location/channel URLs are not followed; cached user does not impose the list user1.4 gate.
  Tests: `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedCleanStickyDirtyShallowMergeAndActualResponse`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedEmptyNullAndRemovalPresence`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedAtomicEditAndHeaderOnlyCommitProtection`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedOwnsInputOptionsMapsPointersAndHTTPViews`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedCanonicalNameAllPagesAndExplicitFixedID`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedNativeAndMalformedResponseEvidenceNoResend`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedRefreshResetsOlderDirtyAndPreservesFailures`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedNewEditsSurviveCommitAndRefreshInFlight`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverUpdateScopeRoutesSnapshotsLiveTokensAndIsolation`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedAndUpdateScopeOfflineSourceAndCleanValidation`.
- AtBasePath uses the common validated literal collection path and escapes Unicode once under the selected ResourceBase. Scoped stateless name lookup freezes body/header before all pages; Load/Track capture collection and ID for Commit/Refresh. Concurrent scopes retain the same API/client/authentication and leave default routes and other scopes unchanged. Nil/zero scopes, protected options and invalid source fail before HTTP.
  Tests: `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverUpdateScopeRoutesSnapshotsLiveTokensAndIsolation`, `api/clustering_receivers_lifecycle_test.go:TestClusteringReceiverTrackedAndUpdateScopeOfflineSourceAndCleanValidation`, `internal/senlin/collection_path_test.go:TestCollectionPathOwnsLiteralRelativeSegments`.

Python/Go differences:

- Go maps cached Resource input through SDK-owned Track/Load/Edit/Commit/Refresh and immutable snapshots instead of the same mutable Python Resource pointer. It retains owned concrete options/context and explicit ID/Name references. Create/type validation remains separate from tracked updates; no unproved update microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.
- The direct base_path control is mapped to a literal relative UpdateScope with validated unescaped segments, rejecting URLs, templates, percent escapes, dot/empty segments, whitespace and controls. Python overrides the current commit only; the Go handle fixes collection+ID for Load/Commit/Refresh. Other operations keep default routes and the shared client remains live.
- Python permissive descriptor/attrs coercion and cached ID/readonly-looking field mutation are deliberately mapped to mutable concrete Name/Action/Params, explicit vendor fields, exact raw tracked identity and core/readonly protection. Separate inherited Resource.commit controls are not exposed by the direct update_receiver proxy.

### `python:clustering/v1/receivers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithListMicroversion`

- Lazy/reusable GET200 preserves typed query, explicit global_project=false, user_id-to-user at selected>=1.4 in API preparation, public Resources and subsequent pages, local exact JSON filters and pre-filter wire-ID continuation on linked/short pages.
  Tests: `api/clustering_receivers_test.go:TestClusteringReceiversSynchronousCRUDAndResponseEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversMessagePresenceSnapshotsAndReuse`, `api/clustering_receivers_test.go:TestClusteringReceiversUpdateNameLookupFreezesInputsAndRechecksSource`, `api/clustering_receivers_test.go:TestClusteringReceiversUserGateResourcesBypassAndPageRecheck`, `api/clustering_receivers_test.go:TestClusteringReceiversLazyPagingLocalFiltersAndWireMarkers`, `api/clustering_receivers_test.go:TestClusteringReceiversFindDeleteMissingAmbiguousAndInvalidResolvedIDs`, `api/clustering_receivers_test.go:TestClusteringReceiversFindContinuationKeepsNameAndLaterPageOutcomes`, `api/clustering_receivers_test.go:TestClusteringReceiversAcceptedMalformedAndWrongCodesKeepEvidence`, `api/clustering_receivers_test.go:TestClusteringReceiversPreflightOwnedFieldsOptionsAndContext`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.
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
- Services>=1.7 and Receiver user>=1.4 gates evaluate the effective per-call client, allowing documented explicit upgrades and rejecting downgrades/default resets without changing source configuration.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestEffectiveVersionGates`, `internal/senlin/list_request_test.go:TestListRequestRevalidatesSourceBeforeContinuationAndGatesEffectiveVersion`.

Python/Go differences:

- Go uses stateless owned concrete options/context and explicit ID/Name references instead of Python mutable Resource input. Nonempty vendor receiver types pass through to the controller, known webhook inputs require cluster/action and no Python CLI default type or unproved message microversion gate is invented.
- Published201/200 strict receiver/receivers envelopes and204 deletion are enforced. Exact JSON Body, Header and StatusCode are retained, nullable action/cluster fields remain distinct from a Senlin action UUID, and channel or incidental Location URLs are never followed.
- Owned extension fields/query/headers are explicit Go capabilities rather than Python silently consuming unknown Resource attributes. Concrete/readonly fields and SDK authentication/version/transport headers are protected; null and deployment plugin semantics stay server decisions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python maps user_id to the wire key user (keys: name, type, cluster_id, action, sort, global_project) and accepts an arbitrary sort string; Go validates the published sort key/direction set and sends user through the typed UserID option.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/find_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindMicroversion`

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

```python
receiver = conn.clustering.create_receiver(
    name="scale_out", type="webhook", cluster_id="CLUSTER_ID",
    action="CLUSTER_SCALE_OUT", params={},
)
message = conn.clustering.create_receiver(name="events", type="message")
receiver = conn.clustering.get_receiver(receiver.id)
```

```go
api := receivers.New(client)
receiver, err := api.Create(ctx, receivers.CreateOpts{Name: "scale_out", Type: "webhook"},
    receivers.WithCreateClusterID("CLUSTER_ID"), receivers.WithCreateAction("CLUSTER_SCALE_OUT"),
    receivers.WithCreateParams(map[string]any{}))
if err != nil { return err }
message, err := api.Create(ctx, receivers.CreateOpts{Name: "events", Type: "message"})
if err != nil { return err }
receiver, err = api.Get(ctx, receiver.ID)
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action, receiver.Channel, message.ID)
```

```python
receiver = conn.clustering.update_receiver(
    "scale_out", name="scale_in", action="CLUSTER_SCALE_IN", params={},
)
```

```go
api := receivers.New(client)
receiver, err := api.Update(ctx, resource.Name("scale_out"), receivers.UpdateOpts{},
    receivers.WithUpdateName("scale_in"), receivers.WithUpdateAction("CLUSTER_SCALE_IN"),
    receivers.WithUpdateParams(map[string]any{}))
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action)
```

```python
receivers = conn.clustering.receivers(type="message", user_id="USER_ID", limit=20)
```

```go
api := receivers.New(client)
for receiver, err := range api.List(ctx,
    receivers.WithListOptions(receivers.ListOpts{Type: "message", Limit: 20}),
    receivers.WithListUserID("USER_ID"), receivers.WithListGlobalProject(false),
    receivers.WithListFilter("params", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(receiver.ID, receiver.Params)
}
```

```python
receiver = conn.clustering.find_receiver("scale_out", ignore_missing=True)
conn.clustering.delete_receiver("scale_out", ignore_missing=True)
```

```go
api := receivers.New(client)
receiver, err := api.Find(ctx, resource.Name("scale_out"))
if err != nil { return err }
if receiver != nil { fmt.Println(receiver.ID) }
if err := api.Delete(ctx, resource.Name("scale_out")); err != nil { return err }
```

```go
listingAPI := receivers.New(client)
values, err := listingAPI.All(ctx,
    receivers.WithListOptions(receivers.ListOpts{Limit: 20}),
    receivers.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, receivers.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
)

func FindReceiverStrict(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Receivers.FindIdentity(ctx, "scale_out",
        receivers.WithFindIgnoreMissing(false))
}
```

## References

- [Receivers](../tracking/receivers/README.md)
- [Finding](../finding/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#receivers-receivers)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
