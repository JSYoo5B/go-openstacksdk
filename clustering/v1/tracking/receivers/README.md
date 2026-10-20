# Receivers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
receiver = conn.clustering.get_receiver("RECEIVER_ID")
receiver.name = "scale_in"
receiver.action = "CLUSTER_SCALE_IN"
receiver.params = {"team": "platform"}
receiver = conn.clustering.update_receiver(receiver)
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedReceiver(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Receivers.Load(ctx, resource.Name("scale_out"))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{},
        receivers.WithUpdateName("scale_in"),
        receivers.WithUpdateAction("CLUSTER_SCALE_IN"),
        receivers.WithUpdateParams(map[string]any{"team": "platform"})); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx, receivers.WithUpdateHeader("X-Audit-Tag", "rename"))
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateReceiverAt(ctx context.Context, conn *sdk.Connection, collectionPath, receiverID string) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Receivers.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    tracked, err := scope.Load(ctx, resource.ID(receiverID))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{}, receivers.WithUpdateActionNull()); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [update_receiver](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1114)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [TrackedReceiver](../../receivers/lifecycle.go)
- [UpdateScope](../../receivers/scope.go)
- [Tracked](../../../../internal/senlin/tracked.go)
- [Collection Path](../../../../internal/senlin/collection_path.go)
- [Clustering Receivers Lifecycle Test](../../../../api/clustering_receivers_lifecycle_test.go)
- [Tracked Test](../../../../internal/senlin/tracked_test.go)
- [Receivers](../../receivers/README.md)
