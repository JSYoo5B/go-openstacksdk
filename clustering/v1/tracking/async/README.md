# Async

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersUpdateNullableFieldsFalseAndAcceptedAction`, `api/clustering_clusters_test.go:TestClusteringClustersProfileOnlyAnyExplicitValueRequires16BeforeLookup`, `api/clustering_clusters_test.go:TestClusteringClustersNameUpdateFreezesBodyHeadersAndVersionGate`, `api/clustering_clusters_response_test.go:TestClusteringClustersOptionalPresentValuesAndRawNullRemainDistinct`, `api/clustering_async_location_case_test.go:TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity`, `connectiontest/connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
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
  Tests: `api/clustering_nodes_test.go:TestClusteringNodeUpdateNameResolutionFreezesNullableInputsAndHeaders`, `api/clustering_nodes_test.go:TestClusteringNodeUpdateIDNoLookupAndOptionalNullEmptyObject`, `api/clustering_nodes_test.go:TestClusteringNodeTaintedFalseAndNullRequireExplicitVersion`, `api/clustering_nodes_test.go:TestClusteringNodeVersionRevalidatedAfterNameLookup`, `api/clustering_async_location_case_test.go:TestClusteringNodeMutationLocationHeaderCaseAndAmbiguity`, `connectiontest/connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
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
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster.name = "workers_renamed"
cluster.metadata = {"team": "platform"}
cluster = conn.clustering.update_cluster(cluster)

node = conn.clustering.get_node("NODE_ID")
node.role = ""
node.metadata = {}
node = conn.clustering.update_node(node)
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateTrackedCluster(ctx context.Context, conn *sdk.Connection) (*clusters.Cluster, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Clusters.Load(ctx, resource.Name("workers"))
    if err != nil { return nil, err }
    if err := tracked.Edit(clusters.UpdateOpts{},
        clusters.WithUpdateName("workers_renamed"),
        clusters.WithUpdateTimeout(0),
        clusters.WithUpdateMetadata(map[string]any{"team": "platform"}),
        clusters.WithUpdateProfileOnlyNull()); err != nil {
        return nil, err
    }
    accepted, err := tracked.Commit(ctx, clusters.WithUpdateHeader("X-Audit-Tag", "rename"))
    if err != nil { return nil, err }
    if accepted.Operation != nil {
        fmt.Println(accepted.Operation.ActionID, accepted.Operation.Location)
    }
    fmt.Println(tracked.Dirty(), tracked.Response().Body, accepted.UserMetadata)
    return accepted, nil
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

func UpdateTrackedNode(ctx context.Context, conn *sdk.Connection) (*nodes.Node, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Nodes.Load(ctx, resource.ID("NODE_ID"))
    if err != nil { return nil, err }
    if err := tracked.Edit(nodes.UpdateOpts{},
        nodes.WithUpdateRole(""),
        nodes.WithUpdateMetadata(map[string]any{}),
        nodes.WithUpdateTainted(false)); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [..](../README.md)
- [Microversions](../../../../docs/microversions.md)
- [update_cluster](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L368)
- [update_node](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L826)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [AtBasePath scope](../../scoping/README.md)
- [TrackedCluster](../../clusters/lifecycle.go)
- [TrackedNode](../../nodes/lifecycle.go)
- [Tracked](../../../../internal/senlin/tracked.go)
- [accepted snapshot](../../actions/submission.go)
- [Clustering Async Lifecycle Test](../../../../api/clustering_async_lifecycle_test.go)
- [Submission Test](../../actions/submission_test.go)
- [Tracked Test](../../../../internal/senlin/tracked_test.go)
- [Clustering Cluster Profile Only Null Test](../../../../api/clustering_cluster_profile_only_null_test.go)
