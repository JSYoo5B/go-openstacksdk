# Tracking

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/update_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.Track`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.Load`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Edit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.RemoveName`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.RemoveMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Commit`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Refresh`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Value`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Response`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.TrackedProfile.Dirty`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithUpdateName`

- PATCH200 sends a resource object rather than JSON Patch operations; optional mutable fields and headers are prepared before exact-name resolution, explicit IDs skip lookup, and empty updates fail before HTTP.
  Tests: `api/clustering_profiles_test.go:TestClusteringProfilesCRUDExactPathsBodiesAndResponseEvidence`, `api/clustering_profiles_test.go:TestClusteringProfilesNameUpdateFindDefaultsAndAmbiguity`, `api/clustering_profiles_test.go:TestClusteringProfilesMetadataPresenceAndOptionsReuse`, `api/clustering_profiles_test.go:TestClusteringProfilesDefaultsInvalidOptionsAndEmptyUpdatePreflight`.
- Name resolution cannot mutate the prepared body or cloned headers; an invalid selected version introduced during lookup prevents the subsequent PATCH.
  Tests: `api/clustering_profiles_update_snapshot_test.go:TestClusteringProfilesUpdateSnapshotsHeadersBeforeLookupAndRechecksVersion`.
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
- The official mutable core is name and metadata. Go rejects known readonly fields and case-variant typed collisions; explicit RemoveName and profile RemoveMetadata send null tombstones, whose server acceptance is not assumed. Python permissive descriptors/attrs and implicit Body coercion are replaced by concrete owned options; explicit unknown vendor fields are a documented Go extension.
- Track uses raw cached Body and its decoded ID when present, ignoring caller mutations to exported typed fields; typed-only models are local seeds rather than evidence of an HTTP response. Explicit Load preserves the request ID, exact Name lookup uses one list resolution, and response IDs never retarget subsequent requests.
- JSON numbers compare exactly and booleans remain distinct from numbers, unlike Python True-equals-1 assignment comparison. Unknown response fields are preserved, Value.Body is the merged cache, and Response.Body contains last successful response fields; a manual Track Response remains the local seed until HTTP succeeds.
- Clean Commit validates context, selected service/version and options and returns without HTTP even with valid header-only options. Header dirtiness and allow_empty_commit do not force an empty Go PATCH; newer concurrent edits are revision-protected rather than cleared by an older response.
- The documented object envelope and success200 are strict. Python flat/empty fallback and ignored non-parseable responses are replaced by ResponseError with accepted HTTP evidence; failure keeps dirty state and the SDK does not automatically retry, so callers can Refresh before an explicit resend.
- Endpoint, authentication and numeric microversion belong to the selected client. Arbitrary Python Resource subclasses and inherited per-call base_path, prepend_key, has_body, retry_on_conflict and microversion controls are not options on this concrete Senlin mapping; implicit adapter negotiation and version upgrades are not performed.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed-profile"
profile.metadata = {"team": "platform"}
profile = conn.clustering.update_profile(profile)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.Edit(profiles.UpdateOpts{},
    profiles.WithUpdateName("renamed-profile"),
    profiles.WithUpdateMetadata(map[string]any{"team": "platform"})); err != nil {
    return err
}
profile, err := tracked.Commit(ctx, profiles.WithUpdateHeader("X-Audit-Tag", "rename"))
if err != nil { return err }
fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.Name("existing-policy"))
if err != nil { return err }
current := tracked.Value().Name
if err := tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName(current)); err != nil {
    return err
}
_, err = tracked.Commit(ctx)
return err
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.RemoveMetadata(); err != nil { return err }
_, err = tracked.Commit(ctx)
return err
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L214)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1914)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1380)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231)
- [generic update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Profile update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L226)
- [Policy update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1030)
- [Tracked Test](../../../internal/senlin/tracked_test.go)
- [Clustering Lifecycle Test](../../../api/clustering_lifecycle_test.go)
- [Async](async/README.md)
- [Receivers](receivers/README.md)
