# Clusters

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/create_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Create`

- POST201 published and POST202 controller creation preserve omitted capacity/size/timeout defaults, explicit zero/-1/null and raw config/metadata, require name/profile_id, and preserve an optional201 or mandatory202 action Location.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersCreateGetExactModelAndSubmission`, `api/clustering_clusters_test.go:TestClusteringClustersCreateDefaultsOmitOptionalFields`, `api/clustering_clusters_response_test.go:TestClusteringClustersAcceptedMalformedObjectRetainsEvidenceWithoutResend`, `api/clustering_clusters_test.go:TestClusteringClustersMutationSchemasProtectCoreAndResponseFields`, `api/clustering_async_location_case_test.go:TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
- The verified16.0.0 router accepts creation with202; both201 and202 preserve the cluster model and response evidence, while a202 without one valid selected-service action Location fails without a second POST. Undeclared200 and malformed accepted bodies retain their distinct HTTP/decode causes.
  Tests: `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreatePublishedAndControllerSuccessCodes`, `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreateAcceptedRequiresValidActionLocation`, `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreateUndeclaredStatusAndMalformedAcceptedBody`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Accepted Location must identify one action at the selected origin and collection, including reverse-proxy prefixes; missing required, empty, ambiguous, foreign or malformed references return ResponseError without following or resending. Submission holds action identity and original response, not an invented action status or completed result.
- Go accepts published201 and verified16.0.0 controller202;202 is an asynchronous submission and requires action evidence. This release-source audit and local HTTP test do not claim live-cloud acceptance.

### `python:clustering/v1/get_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Get`

- GET200 routes the controller identity directly and retains nullable cluster fields, exact json.Number capacity/sizes and raw config/metadata/data.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersCreateGetExactModelAndSubmission`, `api/clustering_clusters_response_test.go:TestClusteringClustersExplicitSuccessCodesAndNativeErrors`, `api/clustering_clusters_response_test.go:TestClusteringClustersContextAndSourceValidationBeforeHTTP`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.

### `python:clustering/v1/delete_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Delete`

- DELETE202 returns accepted action evidence without fetching or polling; ordinary deletion has no body and ignores404 by default, force sends {"force":true} with strict404 by default, explicit ignore overrides both, and403/409 or invalid resolved IDs remain errors.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersDeleteNormalForceBodiesAndSubmission`, `api/clustering_clusters_test.go:TestClusteringClustersDeleteDefaultsIgnoreOnlyMissingAndForceIsStrict`, `api/clustering_clusters_response_test.go:TestClusteringClustersDeleteAcceptsEmptyBodyButRequiresAction`, `api/clustering_clusters_response_test.go:TestClusteringClustersNamedDeleteFreezesHeadersForceAndRechecksSource`, `api/clustering_async_location_case_test.go:TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Accepted Location must identify one action at the selected origin and collection, including reverse-proxy prefixes; missing required, empty, ambiguous, foreign or malformed references return ResponseError without following or resending. Submission holds action identity and original response, not an invented action status or completed result.
- Go WithDeleteIgnoreMissing deliberately overrides either ordinary or force defaults. Shared Resources.Delete returns ErrUnsupported because its error-only signature would discard the asynchronous Submission.

### `python:clustering/v1/update_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Track`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.TrackedCluster.Edit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.TrackedCluster.Commit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.TrackedCluster.Refresh`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.Submission.Snapshot`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithUpdateProfileOnlyNull`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.AtBasePath`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.UpdateScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.UpdateScope.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.UpdateScope.Track`

- Object PATCH202 prepares nullable name/profile_id/timeout/config/metadata and headers before one exact-name lookup; explicit profile_only true or false requires numeric>=1.6 before lookup and again before mutation.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersUpdateNullableFieldsFalseAndAcceptedAction`, `api/clustering_clusters_test.go:TestClusteringClustersProfileOnlyAnyExplicitValueRequires16BeforeLookup`, `api/clustering_clusters_test.go:TestClusteringClustersNameUpdateFreezesBodyHeadersAndVersionGate`, `api/clustering_clusters_response_test.go:TestClusteringClustersOptionalPresentValuesAndRawNullRemainDistinct`, `api/clustering_async_location_case_test.go:TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
- Owned tracked handles snapshot models and accepted Operation evidence, compare current values with sticky dirty state, preserve zero/false/empty/null and deletion tombstones, validate Edit atomically offline, and PATCH pending fields only. Clean Commit validates source/context/header without HTTP; accepted responses merge whole fields while actual Response stays separate. Fixed explicit ID or one canonical-name resolution owns the route; pending-only numeric field gates, failures and concurrent edits preserve state; successful Refresh resets the older revision and accepted Operation.
  Tests: `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNoOpStickyDirtyShallowMergeAndActualResponse`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedScalarPresenceAndRemovalTombstones`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedInvalidEditsAndHeaderOnlyCommitAreAtomic`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedModelAndOperationDeepOwnership`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedFixedIDAndNameLookupOnce`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedFailuresKeepDirtyAndAcceptedEvidence`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedRefreshResetsOperationAndFailureKeepsChanges`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNewEditsSurviveCommitAndRefreshInFlight`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedPendingPresenceVersionGates`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedOfflineEditAndCleanSourceContextValidation`, `api/clustering_async_lifecycle_test.go:TestClusteringAsyncTrackedNameLoadAbsenceDuplicatesAndNilHandles`, `clustering/v1/actions/submission_test.go:TestSubmissionSnapshotOwnsAcceptedResponseEvidence`, `internal/senlin/tracked_test.go:TestTrackedCurrentComparisonStickyDirtyAndRemoval`, `internal/senlin/tracked_test.go:TestTrackedOwnsAllSnapshotsAndRejectedEditIsAtomic`, `internal/senlin/tracked_test.go:TestTrackedAcceptShallowMergeAndPreservesNewerEdits`.
- Explicit profile_only null is preserved through JSON decode/encode and WithUpdateOptions snapshots, including absent-to-null tracked edits and case-folded aliases. Bool/null helper last-wins and old public nonnil bool pointers stay authoritative. Null, false and removal all require numeric>=1.6 only when transmitted; clean Commit rejects body options and failed source gates retain dirty state.
  Tests: `api/clustering_cluster_profile_only_null_test.go:TestClusteringClusterProfileOnlyNullStatelessWireSnapshotsAndLastWins`, `api/clustering_cluster_profile_only_null_test.go:TestClusteringClusterProfileOnlyNullTrackedPresenceDirtyAndVersionGate`, `api/clustering_cluster_profile_only_null_test.go:TestClusteringClusterProfileOnlyNullBoolCompatibilityAndLookupGate`.
- AtBasePath owns a validated literal relative collection and escapes segments once without altering the shared client, default API routes, ResourceBase, headers or original action collection. Scoped Update freezes inputs before all-page exact-name lookup and PATCH; explicit ID is direct. Scoped Load/Track captures both collection and canonical/fixed resource identity for Commit/Refresh. Nil/zero scopes and unsafe paths fail without HTTP; distinct concurrent scopes, live token, escaped reverse prefix, foreign continuation, pending-null gates/source rechecks, native and accepted-response failures preserve routing and dirty/evidence ownership.
  Tests: `api/clustering_update_scope_test.go:TestClusteringUpdateScopeRejectsUnsafePathsAndNilSourcesWithoutHTTP`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeLiteralRoutingLiveTokenAndConcurrentIsolation`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeNameLookupUsesAllScopedPagesAndFrozenInputs`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeTrackedLoadAndTrackKeepScopedIdentity`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopePendingNullGatesAndSourceRecheck`, `api/clustering_update_scope_test.go:TestClusteringUpdateScopeFailuresRetainAcceptedEvidenceDirtyAndFixedRoute`, `internal/senlin/collection_path_test.go:TestCollectionPathOwnsLiteralRelativeSegments`.

Python/Go differences:

- Go maps cached Resource input through concrete SDK-owned Track/Load and Edit/Commit/Refresh handles, context and explicit ID/Name references. Scalar/null helpers distinguish omission, explicit zero/false/empty and null without caller builders or Python truthiness coercion. Views are independent snapshots instead of the same mutable Resource pointer.
- Documented singular envelopes and PATCH202 success are strict. Raw HTTP/header/body evidence and exact JSON numbers are retained separately from merged cached views; malformed accepted responses keep dirty state and are not automatically resent.
- Connection-selected numeric microversion and protected field/query/header extensions are explicit; Commit gates only the pending key, including null/removal. Offline Edit does not negotiate or fetch. Nullable field support remains deployment-defined.
- Pinned AsyncResource does not override Update/commit or convert its Location to an Action. Go additionally requires an accepted action Location at the selected origin/collection and snapshots it as Operation without following, polling, inventing status, or treating dirty=false as completion.
- The direct base_path control is mapped to a concrete UpdateScope with a literal unescaped relative path, rejecting full URLs, templates, percent escapes, dot/empty segments, whitespace and controls. Python applies URI formatting for that commit only; the Go handle additionally fixes collection+ID for Load/Commit/Refresh. Other API operations retain their original routes and selected service action collection.
- Python raw attrs normalization accepts mapped Python/wire aliases and locally mutable cached IDs or readonly-looking fields. Go owns its mutable concrete schema and explicit vendor fields, protects core/readonly aliases and keeps the tracked route independent of cache/input changes. These are deliberate policies; separate inherited commit controls are not exposed by these two direct proxies.

### `python:clustering/v1/clusters`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithListMicroversion`

- Lazy owned query options retain explicit false, sort grammar, vendor query and selected filters across linked and short-page last-wire-ID continuation; constructor snapshots and consumer mutation cannot change raw markers, and local Body subsets/aliases/exact decimal filters do not become server query.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersListShortPageFiltersAndSnapshots`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersLocalFilterJSONTypesAndNull`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersContinuationGuardsBreakAndCancellation`, `api/clustering_clusters_pagination_test.go:TestClusteringClustersSharedStatusAndSortUseActualWireQuery`.
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
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python cluster query keys are name, status, sort and global_project.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

### `python:clustering/v1/find_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindMicroversion`

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

### `python:clustering/v1/scale_in_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.ScaleIn`

- ScaleIn sends count:null by default like the pinned proxy, preserves explicit zero/null and owned parameter extensions, and returns accepted action evidence.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/scale_out_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.ScaleOut`

- ScaleOut sends count:null by default, preserves explicit count/zero/null and parameter extensions, and leaves scaling policy decisions to the controller.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.
- Connection shares the selected Senlin client, current authentication token, numeric version and reverse-proxy prefix across cluster scale-out, node check and action cancellation, without implicit action fetch or resubmission.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/resize_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Resize`

- Resize sends only specified adjustment_type, precise decimal number, min_size/max_size/min_step and strict fields, retaining explicit zero/false/null and server-defined size/policy constraints.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/add_nodes_to_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.AddNodes`

- AddNodes sends a required nonempty node identity array in add_nodes; input and extension JSON are frozen before one parent name resolution, without fetching each child.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/remove_nodes_from_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.RemoveNodes`

- RemoveNodes sends del_nodes with required identities and omitted or explicit destroy_after_deletion false/true/null; any explicit destruction field requires numeric microversion>=1.4 before lookup and before mutation.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/replace_nodes_in_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.ReplaceNodes`

- ReplaceNodes sends the required old-to-new node identity map, owns mutable input and requires numeric microversion>=1.3 before lookup and again before mutation.
  Tests: `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsWireDefaultsAndPresence`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsVersionPreflight`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsPreflightErrorsAndCancellation`, `api/clustering_clusters_commands_test.go:TestClusteringClusterCommandsCustomTransportSubmissionOwnership`.
- Accepted command response decoding requires matching body/Location identities while retaining independent HTTP evidence on malformed body, missing/foreign/multiple/mismatched Location and header case variants. Empty command parameters remain a JSON object.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses context, concrete owned options and explicit ID/Name references; IDs route without a GET, while Names use one exact paginated lookup and reject ambiguity. Python mutable Resource inputs and combined-string identity selection are represented by an explicit Go caller choice.
- Known parameter collisions and authentication/version/transport header replacement fail before HTTP. Plugin/vendor parameters remain nested command inputs rather than mutable Resource fields; the controller/profile owns their schema.
- Selected numeric microversions replace implicit adapter negotiation. Body/header/version requirements are prepared before lookup and rechecked before mutation; default base commands do not invent an explicit-version requirement.
- All six pinned cluster string proxies call strict _find with combined-string ID-first fallback. Go explicit ID/Name separates routing from exact-name resolution; node identities within membership payloads are passed directly to the controller.
- Published POST202 requires a top-level action string and one Location at the selected action collection identifying the same action. Go returns Submission with original Body/Header/StatusCode, never invents completion, follows the Location, fetches state or resends an accepted command. This is stricter than the pinned Python raw JSON dictionary return.

### `python:clustering/v1/check_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Check`

- POST202 health check sends an empty check object by default and preserves nested owned plugin parameters without inventing a version gate.
  Tests: `api/clustering_cluster_health_test.go:TestClusteringClusterHealthWirePresenceAndRoutes`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthSnapshotsBeforeNameLookup`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthVersionPreflightAndRecheck`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthInvalidInputsNeverRequest`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthAcceptedEvidenceAndNativeErrors`.
- Owned bodies, extension headers and minimum version are frozen before exact-name lookup and checked again before mutation. Accepted action and Location must identify the same action; original response evidence is retained without implicit follow, state fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.

Python/Go differences:

- Go uses explicit ID/Name/context and concrete options. IDs route directly without GET like pinned _get_resource string input; an exact paginated Name lookup with ambiguity rejection is an explicit Go convenience instead of Python mutable Resource input.
- Selected numeric minimums and typed input/header ownership are checked locally. Optional fields preserve wire presence; published object inputs accept omitted/object/null shapes, while actual null meaning, profile operation names and plugin schema remain controller decisions.
- Go returns Submission with action identity, Location, original Body/Header/StatusCode and requires published202 plus matching body/Location. Pinned Python returns an unvalidated response JSON dictionary. This strict accepted-response policy does not infer completion or automatically fetch the action.

### `python:clustering/v1/recover_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Recover`

- POST202 recovery preserves omitted/empty/null operation, object/null/omitted operation_params and bool omission/false/null; explicit check requires>=1.6 and check_capacity>=1.7, including successful check-only use at exact1.6.
  Tests: `api/clustering_cluster_health_test.go:TestClusteringClusterHealthWirePresenceAndRoutes`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthSnapshotsBeforeNameLookup`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthVersionPreflightAndRecheck`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthInvalidInputsNeverRequest`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthAcceptedEvidenceAndNativeErrors`.
- Owned bodies, extension headers and minimum version are frozen before exact-name lookup and checked again before mutation. Accepted action and Location must identify the same action; original response evidence is retained without implicit follow, state fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.
- Connection shares the selected version1.7, refreshed token and reverse-proxy service prefix across adoption, preview, cluster recovery and profile operation, without extra resource lookup or Location follow.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAdoptionAndClusterRecoveryShareSelectedSource`.

Python/Go differences:

- Go uses explicit ID/Name/context and concrete options. IDs route directly without GET like pinned _get_resource string input; an exact paginated Name lookup with ambiguity rejection is an explicit Go convenience instead of Python mutable Resource input.
- Selected numeric minimums and typed input/header ownership are checked locally. Optional fields preserve wire presence; published object inputs accept omitted/object/null shapes, while actual null meaning, profile operation names and plugin schema remain controller decisions.
- Go returns Submission with action identity, Location, original Body/Header/StatusCode and requires published202 plus matching body/Location. Pinned Python returns an unvalidated response JSON dictionary. This strict accepted-response policy does not infer completion or automatically fetch the action.

### `python:clustering/v1/perform_operation_on_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.PerformOperation`

- POST202 clusters/{identity}/ops sends {operation:{filters,params}} at>=1.4, preserving missing/object/null input and exact numbers; filters select operation nodes on the server rather than filtering a local resource iterator.
  Tests: `api/clustering_cluster_health_test.go:TestClusteringClusterHealthWirePresenceAndRoutes`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthSnapshotsBeforeNameLookup`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthVersionPreflightAndRecheck`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthInvalidInputsNeverRequest`, `api/clustering_cluster_health_test.go:TestClusteringClusterHealthAcceptedEvidenceAndNativeErrors`.
- Owned bodies, extension headers and minimum version are frozen before exact-name lookup and checked again before mutation. Accepted action and Location must identify the same action; original response evidence is retained without implicit follow, state fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandBodyAllowsEmptyParamsAndRetainsOwnedInputPolicy`, `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`.
- Connection shares the selected version1.7, refreshed token and reverse-proxy service prefix across adoption, preview, cluster recovery and profile operation, without extra resource lookup or Location follow.
  Tests: `connection_senlin_async_test.go:TestConnectionSenlinAdoptionAndClusterRecoveryShareSelectedSource`.

Python/Go differences:

- Go uses explicit ID/Name/context and concrete options. IDs route directly without GET like pinned _get_resource string input; an exact paginated Name lookup with ambiguity rejection is an explicit Go convenience instead of Python mutable Resource input.
- Selected numeric minimums and typed input/header ownership are checked locally. Optional fields preserve wire presence; published object inputs accept omitted/object/null shapes, while actual null meaning, profile operation names and plugin schema remain controller decisions.
- Go returns Submission with action identity, Location, original Body/Header/StatusCode and requires published202 plus matching body/Location. Pinned Python returns an unvalidated response JSON dictionary. This strict accepted-response policy does not infer completion or automatically fetch the action.

### `python:clustering/v1/attach_policy_to_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.AttachPolicy`

- POST202 sends the pinned policy_attach object with required raw policy_id, without a child policy GET or invented version gate. Attach/update preserve enabled omission/false/true/null; detach defaults to policy_id alone. Owned body/header snapshots precede one Name lookup and source/context are rechecked before submission.
  Tests: `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- The response action must match one mandatory Location at the selected action origin/path. Submission retains independent exact body/header/status; malformed accepted evidence, native HTTP/context/option errors remain observable with no automatic action fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Pinned proxies perform _find on the cluster string first. Go explicit ID submits directly and explicit Name performs one exact paginated lookup with ambiguity rejection; policy names/UUIDs/short IDs are controller body identities rather than URL identities.
- Go returns a strict published202 Submission with matching body/Location rather than an unvalidated JSON dictionary and never infers completed action status.
- Library-owned concrete options distinguish enabled omission/false/null and snapshot reusable extensions; typed fields and authentication/version/transport headers cannot be overridden. Pinned attach/update kwargs can override policy_id, while Go rejects that collision.
- Detach WithField is a documented Go extension beyond pinned Python policy_id-only input. policy_update follows the pinned implementation and official contributor contract despite the API-ref example spelling update_policy.

### `python:clustering/v1/detach_policy_from_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.DetachPolicy`

- POST202 sends the pinned policy_detach object with required raw policy_id, without a child policy GET or invented version gate. Attach/update preserve enabled omission/false/true/null; detach defaults to policy_id alone. Owned body/header snapshots precede one Name lookup and source/context are rechecked before submission.
  Tests: `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- The response action must match one mandatory Location at the selected action origin/path. Submission retains independent exact body/header/status; malformed accepted evidence, native HTTP/context/option errors remain observable with no automatic action fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Pinned proxies perform _find on the cluster string first. Go explicit ID submits directly and explicit Name performs one exact paginated lookup with ambiguity rejection; policy names/UUIDs/short IDs are controller body identities rather than URL identities.
- Go returns a strict published202 Submission with matching body/Location rather than an unvalidated JSON dictionary and never infers completed action status.
- Library-owned concrete options distinguish enabled omission/false/null and snapshot reusable extensions; typed fields and authentication/version/transport headers cannot be overridden. Pinned attach/update kwargs can override policy_id, while Go rejects that collision.
- Detach WithField is a documented Go extension beyond pinned Python policy_id-only input. policy_update follows the pinned implementation and official contributor contract despite the API-ref example spelling update_policy.

### `python:clustering/v1/update_cluster_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.UpdatePolicy`

- POST202 sends the pinned policy_update object with required raw policy_id, without a child policy GET or invented version gate. Attach/update preserve enabled omission/false/true/null; detach defaults to policy_id alone. Owned body/header snapshots precede one Name lookup and source/context are rechecked before submission.
  Tests: `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- The response action must match one mandatory Location at the selected action origin/path. Submission retains independent exact body/header/status; malformed accepted evidence, native HTTP/context/option errors remain observable with no automatic action fetch or accepted mutation resend.
  Tests: `internal/senlin/command_test.go:TestCommandResponseRequiresMatchingBodyAndLocationAndRetainsEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsInputPreflight`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative`, `api/clustering_cluster_policy_commands_test.go:TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal`.
- Connection exposes synchronous Receivers and asynchronous cluster policy commands with the same selected1.4, refreshed authentication token and reverse-proxy prefix. user reaches its wire query, global_project=false survives and no incidental Location/channel URL or accepted action is followed automatically.
  Tests: `connection_senlin_receivers_policy_test.go:TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource`.

Python/Go differences:

- Pinned proxies perform _find on the cluster string first. Go explicit ID submits directly and explicit Name performs one exact paginated lookup with ambiguity rejection; policy names/UUIDs/short IDs are controller body identities rather than URL identities.
- Go returns a strict published202 Submission with matching body/Location rather than an unvalidated JSON dictionary and never infers completed action status.
- Library-owned concrete options distinguish enabled omission/false/null and snapshot reusable extensions; typed fields and authentication/version/transport headers cannot be overridden. Pinned attach/update kwargs can override policy_id, while Go rejects that collision.
- Detach WithField is a documented Go extension beyond pinned Python policy_id-only input. policy_update follows the pinned implementation and official contributor contract despite the API-ref example spelling update_policy.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.clustering.create_cluster(
    name="workers", profile_id="PROFILE_ID",
    min_size=0, max_size=-1, desired_capacity=1, timeout=None,
    metadata={"team": "infra"},
)
cluster = conn.clustering.get_cluster(cluster.id)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
cluster, err := service.Clusters.Create(ctx, clusters.CreateOpts{
    Name: "workers", ProfileID: "PROFILE_ID",
}, clusters.WithCreateMinSize(0), clusters.WithCreateMaxSize(-1),
   clusters.WithCreateDesiredCapacity(1), clusters.WithCreateTimeoutNull(),
   clusters.WithCreateMetadata(map[string]any{"team": "infra"}))
if err != nil { return err }
cluster, err = service.Clusters.Get(ctx, cluster.ID)
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.NodeIDs)
```

```python
cluster = conn.clustering.update_cluster(
    "workers", name="workers_v2", timeout=None, metadata={}, profile_only=False,
)
```

```go
api := clusters.New(client)
cluster, err := api.Update(ctx, resource.Name("workers"), clusters.UpdateOpts{},
    clusters.WithUpdateName("workers_v2"), clusters.WithUpdateTimeoutNull(),
    clusters.WithUpdateMetadata(map[string]any{}), clusters.WithUpdateProfileOnly(false))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Operation.ActionID)
```

```python
clusters = conn.clustering.clusters(status="ACTIVE", limit=20)
cluster = conn.clustering.find_cluster("workers", ignore_missing=True)
```

```go
api := clusters.New(client)
for cluster, err := range api.List(ctx,
    clusters.WithListOptions(clusters.ListOpts{Status: "ACTIVE", Limit: 20}),
    clusters.WithListGlobalProject(false),
    clusters.WithListFilter("metadata", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(cluster.ID, cluster.Name)
}
cluster, err := api.Find(ctx, resource.Name("workers"))
if err != nil { return err }
if cluster != nil { fmt.Println(cluster.ID) }
```

```python
action = conn.clustering.delete_cluster("CLUSTER_ID", force_delete=True)
action = conn.clustering.get_action(action.id)
```

```go
api := clusters.New(client)
submission, err := api.Delete(ctx, resource.ID("CLUSTER_ID"), clusters.WithDeleteForce(true))
if err != nil { return err }
if submission == nil { return nil }
action, err := actions.New(client).Get(ctx, submission.ActionID)
if err != nil { return err }
fmt.Println(action.ID, action.Status)
```

```python
action = conn.clustering.scale_out_cluster("CLUSTER_ID", count=1)
action = conn.clustering.resize_cluster(
    "CLUSTER_ID", adjustment_type="EXACT_CAPACITY", number=3, min_size=0, strict=False,
)
```

```go
api := clusters.New(client)
scaled, err := api.ScaleOut(ctx, resource.ID("CLUSTER_ID"), clusters.ScaleOutOpts{},
    clusters.WithScaleOutCount(1))
if err != nil { return err }
resized, err := api.Resize(ctx, resource.ID("CLUSTER_ID"), clusters.ResizeOpts{},
    clusters.WithResizeAdjustmentType(clusters.ExactCapacity),
    clusters.WithResizeNumber(json.Number("3")), clusters.WithResizeMinSize(0),
    clusters.WithResizeStrict(false))
if err != nil { return err }
fmt.Println(scaled.ActionID, resized.ActionID)
```

```python
action = conn.clustering.add_nodes_to_cluster("CLUSTER_ID", ["NODE_ID"])
action = conn.clustering.remove_nodes_from_cluster(
    "CLUSTER_ID", ["NODE_ID"], destroy_after_deletion=False,
)
action = conn.clustering.replace_nodes_in_cluster("CLUSTER_ID", {"OLD_NODE": "NEW_NODE"})
```

```go
api := clusters.New(client)
added, err := api.AddNodes(ctx, resource.ID("CLUSTER_ID"), clusters.AddNodesOpts{
    Nodes: []string{"NODE_ID"},
})
if err != nil { return err }
removed, err := api.RemoveNodes(ctx, resource.ID("CLUSTER_ID"), clusters.RemoveNodesOpts{
    Nodes: []string{"NODE_ID"},
}, clusters.WithRemoveNodesDestroyAfterDeletion(false))
if err != nil { return err }
replaced, err := api.ReplaceNodes(ctx, resource.ID("CLUSTER_ID"), clusters.ReplaceNodesOpts{
    Nodes: map[string]string{"OLD_NODE": "NEW_NODE"},
})
if err != nil { return err }
fmt.Println(added.ActionID, removed.ActionID, replaced.ActionID)
```

```python
checked = conn.clustering.check_cluster("CLUSTER_ID")
recovered = conn.clustering.recover_cluster("CLUSTER_ID", check=False, check_capacity=False)
operated = conn.clustering.perform_operation_on_cluster(
    "CLUSTER_ID", "reboot", filters={"role": "worker"}, params={"type": "SOFT"},
)
```

```go
api := clusters.New(client)
checked, err := api.Check(ctx, resource.ID("CLUSTER_ID"))
if err != nil { return err }
recovered, err := api.Recover(ctx, resource.ID("CLUSTER_ID"), clusters.RecoverOpts{},
    clusters.WithRecoverCheck(false), clusters.WithRecoverCheckCapacity(false))
if err != nil { return err }
operated, err := api.PerformOperation(ctx, resource.ID("CLUSTER_ID"), "reboot",
    clusters.PerformOperationOpts{},
    clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}),
    clusters.WithPerformOperationParams(map[string]string{"type": "SOFT"}))
if err != nil { return err }
fmt.Println(checked.ActionID, recovered.ActionID, operated.ActionID)
```

```python
attached = conn.clustering.attach_policy_to_cluster("CLUSTER_ID", "POLICY_ID")
updated = conn.clustering.update_cluster_policy("CLUSTER_ID", "POLICY_ID", enabled=False)
detached = conn.clustering.detach_policy_from_cluster("CLUSTER_ID", "POLICY_ID")
```

```go
api := clusters.New(client)
attached, err := api.AttachPolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.AttachPolicyOpts{PolicyID: "POLICY_ID"})
if err != nil { return err }
updated, err := api.UpdatePolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.UpdatePolicyOpts{PolicyID: "POLICY_ID"}, clusters.WithUpdatePolicyEnabled(false))
if err != nil { return err }
detached, err := api.DetachPolicy(ctx, resource.ID("CLUSTER_ID"),
    clusters.DetachPolicyOpts{PolicyID: "POLICY_ID"})
if err != nil { return err }
fmt.Println(attached.ActionID, updated.ActionID, detached.ActionID)
```

```go
listingAPI := clusters.New(client)
values, err := listingAPI.All(ctx,
    clusters.WithListOptions(clusters.ListOpts{Limit: 20}),
    clusters.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, clusters.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Clusters.FindIdentity(ctx, "workers",
        clusters.WithFindIgnoreMissing(false))
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.ID("CLUSTER_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        clusters.WithUpdateName("renamed")); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [16.0.0 source audit](../../../docs/senlin-server-contracts.md)
- [Finding](../finding/README.md)
- [Actions](../actions/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#clusters)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#update-a-policy-on-a-cluster)
- [Cluster](https://docs.openstack.org/senlin/ocata/developer/cluster.html#cluster-policy-bindings)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Async](../tracking/async/README.md)
- [Scoping](../scoping/README.md)
