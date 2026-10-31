# Detach Volume

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/detach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DetachVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DetachVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareDetachVolumeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDetachVolumeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDetachVolumeWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDetachVolumeWaitPolicy`

- DetachVolume maps the directly declared cloud helper with fixed Server and Volume references, enabled wait by default, an unlimited SDK wait deadline, two-second polling and exact case-insensitive error failure state. Explicit IDs perform bodyless Nova DELETE accepted only at202/204 followed by an immediate fresh fixed-ID Cinder GET200; no pre-delete server, attachment or Cinder state GET is added.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsNoWaitIDsNeedOnlyNovaAndOpaqueAcknowledgement`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsPreparationOwnsEffectiveDefaultsAndReplacementOrder`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsAvailableBeforeExactFailuresAndStatusOnlyReadiness`.
- Wait(false) with a volume ID supports a Nova-only package call with nil Cinder and returns only the accepted Deleted proof. A server Name resolves through Nova without requiring Cinder; a volume Name still requires Cinder. All selected source, reference, context and concrete option policies are validated before name lookup or deletion, and invalid inactive wait policy is rejected even when waiting is false.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsNoWaitIDsNeedOnlyNovaAndOpaqueAcknowledgement`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsPreflightAndRequiredCinderBeforeNameLookup`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsNamesResolveOnceServerThenVolumeWithoutSwap`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsValidateInactivePolicyAndContextCauses`.
- Whole and wait-subtree factories snapshot reusable pointers and slices, preserve nil versus nonnil empty failures and apply replacement in order. PrepareDetachVolumeOptions applies original external callbacks once, validates the complete effective policy and returns an independently owned DetachVolumeOpts without SDK HTTP or service initialization. Reapplying its concrete result never reruns the original callbacks.
  Tests: `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsPreparationOwnsEffectiveDefaultsAndReplacementOrder`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsFactoriesAreReusableOwnedSnapshots`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsCallbacksRunOnceAndCannotRetainPreparedPolicy`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsValidateInactivePolicyAndContextCauses`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeSharesCachedClientsAndLiveToken`.
- Direct package preparation captures the required selected client sources before original option callbacks, retaining live provider authentication and entry headers. Connection validates context, receiver and references, prepares the original options before service discovery, selects cached Nova and only necessary Cinder, then delegates with the concrete prepared policy without a mutex around caller callbacks.
  Tests: `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportSnapshotsHeadersLiveDistinctProvidersAndSourceGuards`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeOptionsPrecedeDiscoveryAndAreCached`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeSharesCachedClientsAndLiveToken`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeValidatesBeforeClientDiscovery`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeNoWaitNeedsOnlyNova`.
- Server and volume Name references resolve once in their existing exact native collections, server then volume, with missing/duplicate failures before mutation. Resolved IDs stay fixed, the final Nova route segment is the volume ID and the workflow does not apply Python server-volume argument swapping, attachment-ID substitution, Location retargeting or fixed-ID fallback searches.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsNamesResolveOnceServerThenVolumeWithoutSwap`, `resource/resolve_test.go:TestResolveIDUsesExactNamesAndNeverFetchesExplicitIDs`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`.
- Only actual Nova202/204 responses populate Deleted with opaque original body bytes, cloned headers and actual status. Empty, binary and non-JSON accepted bodies require no JSON decoding. Accepted read, Close, cancellation or post-source-check errors retain partial or complete Deleted proof and original causes without starting waiting or resending the completed phase.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsNoWaitIDsNeedOnlyNovaAndOpaqueAcknowledgement`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportAcceptedReadCloseContextAndSourceFailuresRetainOpaqueProof`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportSnapshotsHeadersLiveDistinctProvidersAndSourceGuards`.
- Nova404 is a failure rather than an ignored missing attachment. Other rejected status codes preserve native method, URL, expected202204, physical status/body/header and original cause, leave Deleted nil and never poll. The workflow does not call Cinder os-detach, delete the volume, force cleanup or attach again as compensation.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsMissingAndUnexpectedDeleteCodesKeepNativeFailure`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportRejectedIOAndContextDoNotInventAcceptedAcknowledgement`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`.
- Waiting always fetches the selected fixed Cinder volume ID and requires matching canonical nonnull id and status. It tests exact case-insensitive available before configured failures, uses error rather than error-prefix matching, permits absent/null attachments and rejects malformed known fields or null array rows. A multiattach volume remaining in-use is nonterminal even when the target association is absent; Cinder404 or disappearance fails rather than completing the operation.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsAvailableBeforeExactFailuresAndStatusOnlyReadiness`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsAcceptedInvalidPollsAnd404PreserveAcknowledgement`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportSnapshotsHeadersLiveDistinctProvidersAndSourceGuards`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsCanonicalNullableFieldsAndRawExtensions`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsDistinguishAbsentNullAndEmptyAttachments`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsRejectInvalidFieldsAtomically`.
- Nil or zero Timeout is unlimited, positive Timeout begins after accepted DELETE body reading/Close and source validation, and negative Timeout fails before lookup or deletion even with wait disabled. Positive deadline and caller context cancel poll HTTP and timer waits. This implements a functional Go wait policy while pinned Python declares timeout but never forwards it to its post-delete lookup or waiter.
  Tests: `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportWaitBudgetStartsAfterDeleteAndZeroRemainsUnlimited`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsWaitTimeoutCancellationAndLaterRejectionKeepLastAccepted`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportBlockedPollCancellationKeepsPriorAcceptedObservation`, `blockstorage/detach_volume_options_test.go:TestDetachVolumeOptionsValidateInactivePolicyAndContextCauses`.
- Only nonterminal observations invoke ProgressCallback with zero. Its error or custom context cancellation preserves original inspectable causes alongside accepted Deleted and LastAccepted proof; target and failure observations do not invoke it. Nonnil empty FailureStates disable exact failure matching and an available target wins even when also configured as a failure.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsAvailableBeforeExactFailuresAndStatusOnlyReadiness`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsWaitTimeoutCancellationAndLaterRejectionKeepLastAccepted`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportWaitBudgetStartsAfterDeleteAndZeroRemainsUnlimited`.
- LastAccepted is captured from each actual accepted Cinder response before read, Close, canonical decode or later guard failures; later rejected HTTP/transport errors keep their own native evidence without overwriting it. Ready is independently decoded from successful available proof so model pointers, headers, raw metadata and nested association records do not alias LastAccepted.Volume or earlier phases.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsAcceptedInvalidPollsAnd404PreserveAcknowledgement`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportAcceptedReadCloseContextAndSourceFailuresRetainOpaqueProof`, `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsWaitTimeoutCancellationAndLaterRejectionKeepLastAccepted`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportBlockedPollCancellationKeepsPriorAcceptedObservation`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportLaterFailureKeepsPriorAcceptedProof`.
- Captured effective ResourceBase prefixes, consistent service type/microversion and canonical ordinary headers remain fixed while original live tokens, configured reauthentication, backoff and RetryFunc remain provider-owned. Required source Provider/Endpoint/ResourceBase/Type/Microversion replacement is terminal. SDK fixed-request ownership prevents native callback replacement of logical method, URL, bodyless request or accepted response policy without claiming packet-level exactly-once.
  Tests: `blockstorage/detach_volume_contracts_test.go:TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportSnapshotsHeadersLiveDistinctProvidersAndSourceGuards`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportNativeAuthBackoffAndRetryPreserveBodylessFixedDelete`, `blockstorage/detach_volume_transport_test.go:TestDetachVolumeTransportNativeRetryCannotAddBodyOrExpandAcceptedProof`.
- Connection no-wait volume-ID calls initialize only cached Nova, while default wait and volume Name require cached Cinder. Preparation errors and invalid receiver/context/references precede service discovery; client-initialization cancellation retains context.Canceled and a custom cause. Reuse of attachment source/model/wait helpers preserves existing AttachVolume POST200, fresh available/association guards, options, proof ownership and the older Cinder Collection error-prefix waiter.
  Tests: `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeNoWaitNeedsOnlyNova`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeOptionsPrecedeDiscoveryAndAreCached`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeValidatesBeforeClientDiscovery`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeServiceInitializationCancellationCause`, `connectiontest/connection_detach_volume_test.go:TestConnectionDetachVolumeSharesCachedClientsAndLiveToken`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsDefaultSequenceAndIndependentProof`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsFreshPreconditionForbidsUnsafeMutation`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsExactWaitStatesTargetFirstAndCallback`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsWaitTimeoutAndCancellationKeepLastAccepted`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsAcceptedBodyFailuresRetainPhaseProofWithoutResend`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsSourceSnapshotsLiveTokenAndConsistentVersions`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsNativeRetryCannotReplaceOwnedRequestOrStatusPolicy`, `blockstorage/attach_volume_options_test.go:TestAttachVolumeOptionsConcreteDefaultsAndReplacementOrder`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsRejectInvalidFieldsAtomically`, `blockstorage/blockstorage_test.go:TestCinderFailurePrefixAndMicroversion`.

Python/Go differences:

- Go accepts explicit Server/Volume resource.Ref values and resolves names once. Pinned cloud helper indexes caller Resource/dict values for IDs and passes strings into the Compute proxy; that proxy may find_server and reverse server/volume arguments through a compatibility heuristic. Go never guesses or swaps the caller roles and fixes deletion to the chosen volume ID, not attachment identity.
- Go defaults to the same enabled available wait with two-second polling, exact error failure detection and no SDK wait deadline, but every post-delete observation is a fresh fixed-ID GET with canonical identity/status proof. Pinned cloud helper calls cloud get_volume, which delegates to find_volume and may obtain a Resource that immediately short-circuits on cached available state. Missing post-delete Cinder observations remain failures in both workflows.
- Pinned Python declares timeout but ignores it: it is never passed to get_volume or wait_for_status. Go intentionally honors a positive wait-only Timeout after the Nova acknowledgement has been read and closed, uses nil/zero for unlimited waiting and rejects negative/invalid inactive policies before mutation. Caller context additionally controls every phase; positive context deadlines interrupt polling HTTP and timers rather than only checking elapsed time between Python fetches.
- Python success returns None and an exception exposes no accepted delete or last observed volume through the helper result. Go returns Deleted, LastAccepted and independently owned Ready alongside processing/native/callback/context errors, retains opaque202/204 body bytes without JSON parsing and leaves accepted side effects for caller follow-up. The accepted-phase result is not a complete physical-attempt trace.
- Go provides a concrete owned policy and a local PrepareDetachVolumeOptions seam. Connection applies original option callbacks before service discovery and selects Cinder only for waiting or volume Name; package callers may use only Nova with wait false and a volume ID. Direct package calls capture required sources before original callbacks; both use frozen selected client settings and current authentication from original providers.
- Go retains raw JSON precision, nullable canonical fields and literal timestamps in its reused volume observation models rather than reproducing mutable Python Resource descriptors, alternate identity, dirty state, cache and injected session/adapter objects. ProgressCallback and caller-selectable failure states are explicit Go extensions; this helper does not expose Cinder force/connector detach action controls.
- This go_mapping closes only the directly declared cloud detach_volume parameter/control-flow contract and its documented Go adaptation. Separately inventoried Compute delete_volume_attachment, cloud get_volume, Cinder find/wait and inherited Resource declarations retain independent reviews; native leaves and other volume workflows are not promoted.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func detachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func detachNovaOnly(ctx context.Context, nova *gophercloud.ServiceClient, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, nil,
        blockstorage.DetachVolumeRequest{
            Server: resource.ID(serverID), Volume: resource.ID(volumeID),
        }, blockstorage.WithDetachVolumeWait(false))
}

func detachWithDeadline(ctx context.Context, conn *sdk.Connection, input blockstorage.DetachVolumeRequest, progress func(int) error) (*blockstorage.DetachVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.DetachVolume(ctx, input,
        blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{
            Timeout: &timeout, PollInterval: &interval,
            ProgressCallback: progress,
        }))
}

func detachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient) (*blockstorage.DetachVolumeResult, error) {
    return blockstorage.DetachVolume(ctx, nova, cinder,
        blockstorage.DetachVolumeRequest{
            Server: resource.Name("worker"), Volume: resource.Name("data"),
        })
}
```

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
conn.detach_volume(server, volume)  # wait=True; success returns None
```

## References

- [..](../README.md)
- [BlockStorageCloudMixin.detach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L398-L429)
- [Compute delete_volume_attachment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2542-L2578)
- [Cinder waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource waiter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2502-L2540)
- [cloud get_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Attach Volume](attach-volume.md)
