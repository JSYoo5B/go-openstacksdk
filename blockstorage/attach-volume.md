# Attach Volume

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/attach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.AttachVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AttachVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithAttachVolumeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithAttachVolumeDevice`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithAttachVolumeWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithAttachVolumeWaitPolicy`

- AttachVolume defaults to omitted device, enabled waiting, no SDK wait deadline, two-second polling and exact case-insensitive error failure state. Explicit IDs perform fresh Cinder GET200, Nova POST200 with volumeAttachment.volumeId and optional nonempty device, then fresh fixed-ID Cinder waiting GET200; Wait(false) keeps the precondition GET and omits polling.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsDefaultSequenceAndIndependentProof`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsExplicitNoWaitAndLiteralDevice`, `blockstorage/attach_volume_options_test.go:TestAttachVolumeOptionsConcreteDefaultsAndReplacementOrder`.
- Nil or canceled contexts, invalid request references, either invalid source, malformed device or wait policy and nil options stop before name resolution and workflow requests. Disabled wait fields still receive complete validation. Ordered callbacks are applied once and pointer/slice values are copied after callbacks; whole and subtree replacement helpers preserve nil versus nonnil empty failures.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsPreflightBeforeCallbacksOrHTTP`, `blockstorage/attach_volume_options_test.go:TestAttachVolumeOptionsFactoriesOwnReusablePointersAndSlices`, `blockstorage/attach_volume_options_test.go:TestAttachVolumeOptionsExternalCallbacksRunOnceAndRetainedConfigIsolated`, `blockstorage/attach_volume_options_test.go:TestAttachVolumeOptionsValidateInactiveWaitPolicyAndStopOnGuardFailure`.
- Server and volume names use their existing ordinary native Resources.ResolveID once in order with exact names, duplicate and missing handling; explicit IDs do not fetch the server. Resolved IDs become fixed routes, and response IDs, Location and mutable source configuration cannot select another logical target.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsNamesResolveOnceBeforeFreshCheck`.
- The fresh Cinder precondition requires canonical id equal to the fixed volume ID, exact available status and an explicit nonnull attachments array. It rejects any association with the target server even if device is omitted or empty, rejects null rows and preserves the accepted Checked evidence without a Nova POST on failure.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsFreshPreconditionForbidsUnsafeMutation`.
- Nova POST uses the actual server os-volume_attachments route and the minimal volumeAttachment envelope rather than Cinder attachments. Canonical volumeId must match the fixed volume ID; present nonnull serverId must match the server. Attachment id, attachment_id and bdm_uuid remain passive response fields and a mismatch preserves actual Created proof before stopping.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsNovaStatusAndCreatedDecodeFailuresAreTerminal`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsDistinctProvidersAndNullableOptionalServer`.
- Owned Cinder observation, nested association and Nova attachment models decode exact canonical keys atomically. Known optional fields distinguish nullable pointers while raw metadata retains omission versus null, unknown fields, exact large numeric literals and literal timestamps; malformed UTF-8, nonobject envelopes, wrong known field types and malformed attachment rows fail without partial typed assignment.
  Tests: `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsCanonicalNullableFieldsAndRawExtensions`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsDistinguishAbsentNullAndEmptyAttachments`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsRejectInvalidFieldsAtomically`.
- Checked, Created and LastAccepted retain actual accepted response bytes, cloned headers and status before read, Close, decode or later guard failures. Responses, raw fields, typed models and Ready from separate phases own independent data. A nonnil result alone does not establish a mutation; Created HTTP200 records acknowledgement and only successful waiting populates Ready.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsDefaultSequenceAndIndependentProof`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsAcceptedBodyFailuresRetainPhaseProofWithoutResend`.
- Waiting begins after the Nova response is decoded and verified. Nil or zero timeout is unlimited; positive timeout is wait-only and caller context bounds all phases. Each poll requires matching canonical Cinder id and known status, checks case-insensitive in-use before exact failures and does not infer error-prefix states. Nonnil empty failure states disable failure matching. Negative timeout fails before lookup or mutation even when waiting is disabled. A positive wait deadline cancels polling HTTP and timer waits through context; this deliberately differs from Python zero/negative and between-fetch timer semantics.
  Tests: `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsWaitBudgetStartsAfterCreatedResponse`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsExactWaitStatesTargetFirstAndCallback`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsWaitTimeoutAndCancellationKeepLastAccepted`.
- Nonterminal waiting responses invoke a synchronous ProgressCallback with zero rather than reading an unmodeled extension progress field. Callback errors and cancellation retain their original inspectable causes and already accepted evidence; target and failure observations do not invoke progress and timer/context waiting remains cancellable.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsExactWaitStatesTargetFirstAndCallback`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsWaitTimeoutAndCancellationKeepLastAccepted`.
- Rejected HTTP errors preserve native method, URL, accepted-code policy, physical status/body/header and original causes without synthesizing accepted phase wrappers. LastAccepted means the last accepted waiting Cinder response and remains intact when a later rejected request or context error occurs. Later failure never triggers automatic detach, delete or a new logical attach operation.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsCreatedProofSurvivesWaitFailures`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsWaitTimeoutAndCancellationKeepLastAccepted`, `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsNovaStatusAndCreatedDecodeFailuresAreTerminal`.
- Both selected clients retain their effective prefixed ResourceBase, validated service type and microversion with entry-snapshotted canonical ordinary source headers and original live provider authentication. Provider/Endpoint/ResourceBase/Type/Microversion changes are terminal before another phase. Native configured reauthentication and retry remain distinct from SDK guards for method, URL, serialized JSON and accepted-response ownership.
  Tests: `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsSourceSnapshotsLiveTokenAndConsistentVersions`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsRejectSourceChangesAtOptionsAndAfterMutation`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsNativeReauthenticationBackoffAndOwnedRetry`, `blockstorage/attach_volume_transport_test.go:TestAttachVolumeContractsNativeRetryCannotReplaceOwnedRequestOrStatusPolicy`.
- Connection.AttachVolume validates context, receiver and references before selecting cached ComputeV2 and BlockStorageV3 clients, then delegates to the package workflow without holding a Connection mutex through callbacks. Lazy service initialization or microversion discovery can precede package option validation, while options still complete before name/precondition/mutation/poll requests.
  Tests: `connection_attach_volume_test.go:TestConnectionAttachVolumeSharesCachedClientsAndLiveToken`, `connection_attach_volume_test.go:TestConnectionAttachVolumeDiscoveryPrecedesOptionsAndIsCached`, `connection_attach_volume_test.go:TestConnectionAttachVolumeValidatesBeforeClientDiscovery`, `connection_attach_volume_test.go:TestConnectionAttachVolumeServiceInitializationCancellationCause`.
- The workflow owns exact case-insensitive error matching without changing the existing high-level Cinder Collection waiter, which retains its error-prefix failure policy.
  Tests: `blockstorage/attach_volume_contracts_test.go:TestAttachVolumeContractsExactWaitStatesTargetFirstAndCallback`, `blockstorage/blockstorage_test.go:TestCinderFailurePrefixAndMicroversion`.

Python/Go differences:

- Go accepts resource.Ref inputs and performs fresh Cinder checks after optional name resolution. Pinned Python accepts caller-owned mutable server and volume Resource/dict values, reads their supplied available status and attachment records and only rejects a matching server when get_volume_attach_device returns a truthy device. Go deliberately rejects a matching association independent of device and requires explicit canonical identity/status/attachments proof.
- Go waiting always fetches fresh fixed-ID Cinder responses and uses its owned exact-error policy. Python may immediately return a supplied Resource whose cached status already equals in-use; dict inputs are refetched before passing a Resource into the waiter. Both default to enabled wait, two-second polls and an unlimited wait-only SDK timeout, while caller context and error-returning progress callbacks are Go controls.
- Go returns completed accepted phase proof and last accepted waiting evidence alongside read/decode/callback/context/native errors and leaves existing side effects to the caller. Python raises on waiting failure without returning the Nova attachment. Raw evidence is separate from successfully decoded typed results and is not a complete physical-attempt log.
- Go uses strict owned canonical JSON fields, nullable response pointers, raw extension precision, literal timestamps and fixed route identities. Absent Nova id stays nil rather than adopting volumeId as Python alternate identity; serverId reads only actual response data, while selected server identity remains in the workflow result. It does not reproduce Python descriptor coercion or Resource object mutation to choose status or route. Optional Nova response attributes remain passive and no arbitrary attachment attrs are added to this device-only cloud helper.
- Go captures selected source clients and normal headers while using their original live provider authentication. Source configuration changes between phases are terminal; Connection may initialize or discover services before package callback application. Configured native retry/reauthentication remain in place and packet-level exactly-once is not claimed.
- Go intentionally uses nil or numeric-zero Timeout as no SDK wait deadline and rejects negative Timeout before any name lookup or workflow mutation, including when Wait is false. Pinned Python forwards timeout to Resource waiting after the Nova mutation; utils.iterate_timeout treats numeric zero or negative timeout as immediate ResourceTimeout before its first poll unless an already-target Resource short-circuits first. Positive Go wait deadlines cancel HTTP and timers via context, whereas Python checks elapsed time between fetches and does not interrupt an in-flight fetch or sleep with that loop deadline.
- This mapping closes the directly declared cloud attach_volume parameter/control-flow contract using the documented Go-native representation. It does not promote the separately inventoried Compute create_volume_attachment, Cinder get_volume or generic wait_for_status declarations or their inherited Resource/session/cache behavior; their full SDK reviews remain independent.

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

func attachDefault(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    })
}

func attachWithoutWaiting(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.ID(serverID), Volume: resource.ID(volumeID),
    }, blockstorage.WithAttachVolumeDevice("/dev/vdb"),
        blockstorage.WithAttachVolumeWait(false))
}

func attachWithDeadline(ctx context.Context, conn *sdk.Connection) (*blockstorage.AttachVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
        Server: resource.Name("worker"), Volume: resource.Name("data"),
    }, blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{
        Timeout: &timeout, PollInterval: &interval,
    }))
}

func attachWithClients(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input blockstorage.AttachVolumeRequest) (*blockstorage.AttachVolumeResult, error) {
    return blockstorage.AttachVolume(ctx, nova, cinder, input)
}
```

```python
server = conn.compute.get_server("server-id")
volume = conn.block_storage.get_volume("volume-id")
attachment = conn.attach_volume(server, volume)  # wait=True, timeout=None
```

## References

- [..](../README.md)
- [Nova VolumeAttachment API](../compute/v2/README.md)
- [BlockStorageCloudMixin.attach_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L431-L492)
- [Cinder wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3815)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591-L2667)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
