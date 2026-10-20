# Nodes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/create_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Create`

- POST202 node creation sends required name/profile_id, omitted or explicit null/empty cluster_id/role and raw metadata, requires an action Location, and keeps Senlin ID distinct from physical_id.
  Tests: `api/clustering_nodes_test.go:TestClusteringNodeCreateAcceptedEnvelopeSnapshotsAndSubmission`, `api/clustering_nodes_test.go:TestClusteringNodeCreateDefaultsDoNotInventClusterOrRole`, `api/clustering_nodes_test.go:TestClusteringNodeAcceptedMalformedEvidenceNeverResends`, `api/clustering_nodes_test.go:TestClusteringNodePreflightRejectsImmutableFieldsAndWrongOptions`, `api/clustering_async_location_case_test.go:TestClusteringNodeMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Accepted Location must identify one action at the selected origin and collection, including reverse-proxy prefixes; missing required, empty, ambiguous, foreign or malformed references return ResponseError without following or resending. Submission holds action identity and original response, not an invented action status or completed result.

### `python:clustering/v1/get_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Get`

- GET200 routes the controller identity directly; omitted/true/false details policy uses show_details and preserves precise index, physical_id, nullable ownership/timestamps and raw physical details.
  Tests: `api/clustering_nodes_test.go:TestClusteringNodeGetDirectIdentityAndPhysicalDetails`, `api/clustering_nodes_test.go:TestClusteringNodeAcceptedMalformedEvidenceNeverResends`, `api/clustering_nodes_test.go:TestClusteringNodeWrongSuccessCodesRemainHTTPErrors`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Omitted details sends no query and explicit true/false sends that bool; the pinned Python default details=False enters its false branch. Senlin validates deployment-specific physical details.

### `python:clustering/v1/delete_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Delete`

- DELETE202 returns accepted action evidence without fetching or polling; ordinary deletion has no body and ignores404 by default, force sends {"force":true} with strict404 by default, explicit ignore overrides both, and403/409 or invalid resolved IDs remain errors.
  Tests: `api/clustering_nodes_test.go:TestClusteringNodeDeleteActionDefaultForceAndResponseOwnership`, `api/clustering_nodes_test.go:TestClusteringNodeDeleteNameLookupDefaultsMissingAndConflict`, `api/clustering_nodes_test.go:TestClusteringNodeMalformedActionLocationsRetainAcceptedEvidence`, `api/clustering_nodes_test.go:TestClusteringNodePreflightRejectsImmutableFieldsAndWrongOptions`, `api/clustering_async_location_case_test.go:TestClusteringNodeMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Accepted Location must identify one action at the selected origin and collection, including reverse-proxy prefixes; missing required, empty, ambiguous, foreign or malformed references return ResponseError without following or resending. Submission holds action identity and original response, not an invented action status or completed result.
- Go WithDeleteIgnoreMissing deliberately overrides either ordinary or force defaults. Shared Resources.Delete returns ErrUnsupported because its error-only signature would discard the asynchronous Submission.

### `python:clustering/v1/update_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Track`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.TrackedNode.Edit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.TrackedNode.Commit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.TrackedNode.Refresh`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.Submission.Snapshot`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.AtBasePath`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.UpdateScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.UpdateScope.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.UpdateScope.Track`

- Object PATCH202 prepares optional name/profile_id/role/metadata/tainted and headers before one exact-name lookup; explicit tainted false/true/null requires numeric>=1.13 before lookup and again before mutation.
  Tests: `api/clustering_nodes_test.go:TestClusteringNodeUpdateNameResolutionFreezesNullableInputsAndHeaders`, `api/clustering_nodes_test.go:TestClusteringNodeUpdateIDNoLookupAndOptionalNullEmptyObject`, `api/clustering_nodes_test.go:TestClusteringNodeTaintedFalseAndNullRequireExplicitVersion`, `api/clustering_nodes_test.go:TestClusteringNodeVersionRevalidatedAfterNameLookup`, `api/clustering_async_location_case_test.go:TestClusteringNodeMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
- Owned tracked handles snapshot models and accepted Operation evidence, compare current values with sticky dirty state, preserve zero/false/empty/null and deletion tombstones, validate Edit atomically offline, and PATCH pending fields only. Clean Commit validates source/context/header without HTTP; accepted responses merge whole fields while actual Response stays separate. Fixed explicit ID or one canonical-name resolution owns the route; pending-only numeric field gates, failures and concurrent edits preserve state; successful Refresh resets the older revision and accepted Operation.
  Tests: `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNoOpStickyDirtyShallowMergeAndActualResponse`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedScalarPresenceAndRemovalTombstones`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedInvalidEditsAndHeaderOnlyCommitAreAtomic`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedModelAndOperationDeepOwnership`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedFixedIDAndNameLookupOnce`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedFailuresKeepDirtyAndAcceptedEvidence`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedRefreshResetsOperationAndFailureKeepsChanges`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNewEditsSurviveCommitAndRefreshInFlight`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedPendingPresenceVersionGates`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedOfflineEditAndCleanSourceContextValidation`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNameLoadAbsenceDuplicatesAndNilHandles`, `clustering/v1/actions/submission_test.go:TestSubmissionSnapshotOwnsAcceptedResponseEvidence`, `internal/senlin/tracked_test.go:TestTrackedCurrentComparisonStickyDirtyAndRemoval`, `internal/senlin/tracked_test.go:TestTrackedOwnsAllSnapshotsAndRejectedEditIsAtomic`, `internal/senlin/tracked_test.go:TestTrackedAcceptShallowMergeAndPreservesNewerEdits`.
- AtBasePath owns a validated literal relative collection and escapes segments once without altering the shared client, default API routes, ResourceBase, headers or original action collection. Scoped Update freezes inputs before all-page exact-name lookup and PATCH; explicit ID is direct. Scoped Load/Track captures both collection and canonical/fixed resource identity for Commit/Refresh. Nil/zero scopes and unsafe paths fail without HTTP; distinct concurrent scopes, live token, escaped reverse prefix, foreign continuation, pending-null gates/source rechecks, native and accepted-response failures preserve routing and dirty/evidence ownership.
  Tests: `api/clustering_update_scope_test.go:TestClusteringUpdateScopeRejectsUnsafePathsAndNilSourcesWithoutHTTP`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeLiteralRoutingLiveTokenAndConcurrentIsolation`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeNameLookupUsesAllScopedPagesAndFrozenInputs`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeTrackedLoadAndTrackKeepScopedIdentity`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopePendingNullGatesAndSourceRecheck`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeFailuresRetainAcceptedEvidenceDirtyAndFixedRoute`, `internal/senlin/collection_path_test.go:TestCollectionPathOwnsLiteralRelativeSegments`.

Python/Go differences:

- Go maps cached Resource input through concrete SDK-owned Track/Load and Edit/Commit/Refresh handles, context and explicit ID/Name references. Scalar/null helpers distinguish omission, explicit zero/false/empty and null without caller builders or Python truthiness coercion. Views are independent snapshots instead of the same mutable Resource pointer.
- Documented singular envelopes and PATCH202 success are strict. Raw HTTP/header/body evidence and exact JSON numbers are retained separately from merged cached views; malformed accepted responses keep dirty state and are not automatically resent.
- Connection-selected numeric microversion and protected field/query/header extensions are explicit; Commit gates only the pending key, including null/removal. Offline Edit does not negotiate or fetch. Nullable field support remains deployment-defined.
- Pinned AsyncResource does not override Update/commit or convert its Location to an Action. Go additionally requires an accepted action Location at the selected origin/collection and snapshots it as Operation without following, polling, inventing status, or treating dirty=false as completion.
- The direct base_path control is mapped to a concrete UpdateScope with a literal unescaped relative path, rejecting full URLs, templates, percent escapes, dot/empty segments, whitespace and controls. Python applies URI formatting for that commit only; the Go handle additionally fixes collection+ID for Load/Commit/Refresh. Other API operations retain their original routes and selected service action collection.
- Python raw attrs normalization accepts mapped Python/wire aliases and locally mutable cached IDs or readonly-looking fields. Go owns its mutable concrete schema and explicit vendor fields, protects core/readonly aliases and keeps the tracked route independent of cache/input changes. These are deliberate policies; separate inherited commit controls are not exposed by these two direct proxies.

### `python:clustering/v1/nodes`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithListMicroversion`

- Lazy owned query options retain explicit false, sort grammar, vendor query and selected filters across linked and short-page last-wire-ID continuation; constructor snapshots and consumer mutation cannot change raw markers, and local Body subsets/aliases/exact decimal filters do not become server query.
  Tests: `api/clustering_nodes_pagination_test.go:TestClusteringNodesListWireOptionsAndRawMarkerSnapshots`, `api/clustering_nodes_pagination_test.go:TestClusteringNodesListPreflightAndSharedStatus`.
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

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Known local Body filters support recursive object subsets, arrays and exact decimal comparison while keeping JSON boolean and numeric types distinct; arbitrary vendor query keys are explicit server extensions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.

Unresolved boundaries:

- Map inherited per-call base_path and deprecated proxy JMESPath controls; the tested local cap counts raw rows before local filters and paginated=False stops at the first response.
- Compare full inherited Resource field normalization/defaults and query consumption; explicit Go vendor queries currently reach the server rather than Python unknown-key omission.

### `python:clustering/v1/find_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindMicroversion`

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

### `python:clustering/v1/check_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Check`

- Check sends {check:{}} by default without inventing a version requirement; owned plugin fields/header snapshots stay inside the command parameter object.
  Tests: `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsBaseDefaultsAndSubmissionOwnership`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsCheckParameterExtensionsSnapshot`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsNameLookupFreezesBodyHeadersAndGate`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsVersionPreflightAndRecheck`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsPreflightInvalidInput`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsMalformedAcceptedResponsesRetainEvidence`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsHTTPFailuresRemainOriginalErrors`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsLookupMissingAndAmbiguousNeverSubmit`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.
- Connection shares the selected Senlin client, current authentication token, numeric version and reverse-proxy prefix across cluster scale-out, node check and action cancellation, without implicit action fetch or resubmission.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/recover_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Recover`

- Recover sends {recover:{}} by default, preserves omitted/empty/null operation and object/null/omitted operation_params, and freezes any explicit check false/true/null requirement>=1.6 before parent resolution.
  Tests: `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsBaseDefaultsAndSubmissionOwnership`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsRecoverNullableSnapshotAndReuse`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsRecoverOperationOmittedEmptyAndNull`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsNameLookupFreezesBodyHeadersAndGate`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsVersionPreflightAndRecheck`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsPreflightInvalidInput`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsMalformedAcceptedResponsesRetainEvidence`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsHTTPFailuresRemainOriginalErrors`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsLookupMissingAndAmbiguousNeverSubmit`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/perform_operation_on_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.PerformOperation`

- PerformOperation sends {operation:{params}} to nodes/{id}/ops, supports owned plugin parameters without a Resource-field blacklist, and requires numeric microversion>=1.4 before lookup and before POST.
  Tests: `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsPluginParametersDoNotReplaceTargetOrEnvelope`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsNameLookupFreezesBodyHeadersAndGate`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsVersionPreflightAndRecheck`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsPreflightInvalidInput`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsMalformedAcceptedResponsesRetainEvidence`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsHTTPFailuresRemainOriginalErrors`, `api/clustering_nodes_commands_test.go:TestClusteringNodeCommandsLookupMissingAndAmbiguousNeverSubmit`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/adopt_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Adopt`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.AdoptPreview`

- Both pinned adoption branches send flat POST200 requests at selected>=1.7. Adopt forwards owned attributes retaining optional omission/empty/null/false. Preview sends exactly identity/type/overrides/snapshot, including null for absent overrides/snapshot, and returns the whole response with raw inner properties/version. Location is incidental evidence and neither branch fabricates an action or performs another HTTP request.
  Tests: `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptDefaultsFlatBodyAndIncidentalLocation`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptPresenceSnapshotsAndIndependentReuse`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptPreviewFourFieldsNullDefaultsAndExactVersion`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptPreviewSnapshotsFalseNullAndEmptyObject`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptVersionTypeRecheckAndCanceledOptions`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptInputCapabilityAndOwnedFieldPreflight`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptMalformedAcceptedResponsesRetainEvidence`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptHTTPFailuresRetainNativeErrors`, `api/clustering_nodes_adopt_test.go:TestClusteringNodeAdoptTransportAndAcceptedReadErrorsPreserveCauses`.
- Shared flat body serialization snapshots exact JSON, protects concrete fields, owned headers and forbidden query/arguments; wrapped resource mutations still reject an empty body and commands retain empty parameter objects.
  Tests: `internal/senlin/mutation_test.go:TestMutationFlatBodyPreservesRootFieldsAndGuards`, `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`.
- Connection shares its selected1.7, refreshed authentication token and reverse-proxy prefix for both adoption branches and cluster recovery/operations.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAdoptionAndClusterRecoveryShareSelectedSource`.

Python/Go differences:

- Go splits preview=False/True into Adopt/AdoptPreview with owned typed options and context. Physical resource Identity/Type are required body strings with no route-identifier restriction; optional name/null/profile semantics remain server decisions.
- Preview extra fields/query/arguments cause a preflight error rather than Python silently dropping unknown preview attrs. Typed field and authentication/version/transport header replacement is rejected; ordinary Adopt still exposes owned extension attributes.
- The SDK requires the selected numeric>=1.7 without silently upgrading the adapter, snapshots caller JSON/options, and revalidates the source before POST. Pinned methods do not enforce the numeric gate themselves.
- Go enforces published200 and strict node/node_preview object envelopes, returning Node with Operation=nil or an owned PreviewResult preserving exact entire RawBody, outer/inner fields and raw version/properties. Python translates the adopt response through mutable Resource or returns preview JSON directly; Go never assumes an action from incidental Location.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
node = conn.clustering.create_node(
    name="worker_1", profile_id="PROFILE_ID", cluster_id="CLUSTER_ID",
    role="worker", metadata={"team": "infra"},
)
node = conn.clustering.get_node(node.id, details=True)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
node, err := service.Nodes.Create(ctx, nodes.CreateOpts{
    Name: "worker_1", ProfileID: "PROFILE_ID",
}, nodes.WithCreateClusterID("CLUSTER_ID"), nodes.WithCreateRole("worker"),
   nodes.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
node, err = service.Nodes.Get(ctx, node.ID, nodes.WithGetDetails(true))
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, node.Details)
```

```python
node = conn.clustering.update_node(
    "worker_1", name="worker_2", role=None, metadata={}, tainted=False,
)
```

```go
api := nodes.New(client)
node, err := api.Update(ctx, resource.Name("worker_1"), nodes.UpdateOpts{},
    nodes.WithUpdateName("worker_2"), nodes.WithUpdateRoleNull(),
    nodes.WithUpdateTainted(false), nodes.WithUpdateMetadata(map[string]any{}))
if err != nil { return err }
fmt.Println(node.ID, node.Operation.ActionID)
```

```python
nodes = conn.clustering.nodes(cluster_id="CLUSTER_ID", status="ACTIVE", limit=20)
node = conn.clustering.find_node("worker_1", ignore_missing=True)
```

```go
api := nodes.New(client)
for node, err := range api.List(ctx,
    nodes.WithListOptions(nodes.ListOpts{ClusterID: "CLUSTER_ID", Status: "ACTIVE", Limit: 20}),
    nodes.WithListShowDetails(false), nodes.WithListGlobalProject(false),
    nodes.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(node.ID, node.PhysicalID)
}
node, err := api.Find(ctx, resource.Name("worker_1"))
if err != nil { return err }
if node != nil { fmt.Println(node.ID) }
```

```python
action = conn.clustering.delete_node("NODE_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := nodes.New(client)
submission, err := api.Delete(ctx, resource.ID("NODE_ID"), nodes.WithDeleteForce(true))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```python
checked = conn.clustering.check_node("NODE_ID")
recovered = conn.clustering.recover_node(
    "NODE_ID", operation="REBOOT", operation_params={}, check=False,
)
performed = conn.clustering.perform_operation_on_node("NODE_ID", "reboot", type="SOFT")
```

```go
api := nodes.New(client)
checked, err := api.Check(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
recovered, err := api.Recover(ctx, resource.ID("NODE_ID"), nodes.RecoverOpts{},
    nodes.WithRecoverOperation("REBOOT"), nodes.WithRecoverOperationParams(map[string]any{}),
    nodes.WithRecoverCheck(false))
if err != nil { return err }
performed, err := api.PerformOperation(ctx, resource.ID("NODE_ID"), "reboot",
    nodes.WithPerformOperationField("type", "SOFT"))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, performed.ActionID)
```

```python
node = conn.clustering.adopt_node(
    identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
    name="adopted_worker", snapshot=False, metadata={},
)
preview = conn.clustering.adopt_node(
    preview=True, identity="PHYSICAL_RESOURCE_ID", type="os.nova.server-1.0",
)
```

```go
api := nodes.New(client)
node, err := api.Adopt(ctx, nodes.AdoptOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
}, nodes.WithAdoptName("adopted_worker"), nodes.WithAdoptSnapshot(false),
   nodes.WithAdoptMetadata(map[string]any{}))
if err != nil { return err }
preview, err := api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{
    Identity: "PHYSICAL_RESOURCE_ID", Type: "os.nova.server-1.0",
})
if err != nil { return err }
fmt.Println(node.ID, node.PhysicalID, preview.Spec.Type, string(preview.Spec.Version))
```

```go
listingAPI := nodes.New(client)
values, err := listingAPI.All(ctx,
    nodes.WithListOptions(nodes.ListOpts{Limit: 20}),
    nodes.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, nodes.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
)

func FindNodeStrict(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Nodes.FindIdentity(ctx, "worker_1",
        nodes.WithFindIgnoreMissing(false))
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedNode(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Nodes.Load(ctx, resource.ID("NODE_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(nodes.UpdateOpts{},
        nodes.WithUpdateName("renamed")); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [Actions](../actions/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#nodes)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Finding](../finding/README.md)
- [Async](../tracking/async/README.md)
- [Scoping](../scoping/README.md)
