# Create Volume

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/create_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CreateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCreateVolumeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeAttributes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBootable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeWaitPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeDescription`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeType`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSchedulerHints`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeFields`

- CreateVolume maps required Size and optional Image reference with default enabled fresh available waiting, unlimited SDK deadline, two-second interval and exact error failure policy. Cinder POST accepts only202; explicit Size, including0 or negative, always overrides any attribute size rather than being omitted. No initial volume, snapshot/source/backup lookup, Nova server creation or attachment operation is introduced.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsNoWaitSizeAndLiteralSourcesAreOwnedWithoutLookups`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsEffectiveDefaultsAndBootableForcesWaitAfterReplacement`.
- Optional image ID is captured and used without a Glance client or request; image Name requires Glance and resolves once through the existing exact Collection before Cinder creation. Missing/ambiguous/unsafe image identity aborts creation, the selected imageRef overrides normalized attributes and later caller reference mutation cannot change the selection.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsImageNameResolvesOnceAndReferenceWinsAllAttributes`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsNoWaitSizeAndLiteralSourcesAreOwnedWithoutLookups`, `connection_create_volume_test.go:TestConnectionCreateVolumeCachedCinderLiveTokenAndOwnedImageSelection`, `resource/resolve_test.go:TestResolveIDUsesExactNamesAndNeverFetchesExplicitIDs`.
- The owned attribute schema covers all34 pinned direct Cinder v3 Volume Body declarations and inherited id/name/metadata by Python attribute or exact wire key, with source_replica an explicit Go-native extension. Unknown attributes and base_path/microversion/private route controls fail before lookup/POST. Caller request builders are unnecessary.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAllPinnedAttributesAndWireAliasesAreSerialized`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsPreflightValidatesEveryProvidedSourceBeforeCallbacks`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsValidateKnownFieldBoundaryInactiveWaitAndCancellation`.
- Alias mapping preserves source_volume_id/source_volid, image_id/imageRef, consistency_group_id/consistencygroup_id, bool names and host/project/migration/replication extension wires. When both attribute alias and wire key exist the wire key wins deterministically; nonnil concrete common attributes replace corresponding raw canonical values. Literal raw JSON values keep false/zero/null/empty and large numeric precision rather than applying Python descriptor coercion.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAllPinnedAttributesAndWireAliasesAreSerialized`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsEveryTypedCommonAttributeOverridesRawWireValues`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAliasesPresenceConcreteOverridesMetadataAndHints`.
- Canonical name and description presence wins over display_name/display_description; selected null,false,numeric-zero,empty string/array/object is omitted and aliases never leak into the body. Required input Size and selected Image still override the resulting volume envelope independently of attribute order.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAliasesPresenceConcreteOverridesMetadataAndHints`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsNoWaitSizeAndLiteralSourcesAreOwnedWithoutLookups`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsImageNameResolvesOnceAndReferenceWinsAllAttributes`.
- String Metadata nil omits, nonnil empty emits{}, and raw Fields metadata:null is retained. Scheduler hints accept an object or null; null/empty object is omitted while a nonempty object moves to the top-level OS-SCH-HNT:scheduler_hints key outside volume without losing raw values, duplicates within arrays or unknown numeric precision.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAliasesPresenceConcreteOverridesMetadataAndHints`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsEveryTypedCommonAttributeOverridesRawWireValues`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsFactoriesOwnPointersMapsRawValuesAndReusableSnapshots`.
- Whole/attribute/wait replacement factories and scalar/map helpers own pointers, maps, slices and raw JSON at factory capture and after each ordered callback. Retained callback configuration or reusable caller data cannot change the prepared execution. All inactive wait, JSON, attribute and selected source policies are validated before lookup or mutation, preserving nil versus empty behavior.
  Tests: `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsEffectiveDefaultsAndBootableForcesWaitAfterReplacement`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsFactoriesOwnPointersMapsRawValuesAndReusableSnapshots`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsCallbacksRunOnceAndPreparedStateDoesNotAliasRetainedHandles`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsValidateKnownFieldBoundaryInactiveWaitAndCancellation`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsPreflightValidatesEveryProvidedSourceBeforeCallbacks`.
- PrepareCreateVolumeOptions applies original callbacks once locally and returns an independently owned complete effective policy. Connection validates context/receiver/image reference, prepares before cached Cinder/required Glance discovery, and delegates with that policy without rerunning originals or holding a mutex through callback phases. Explicit image ID or absent Image selects no Glance.
  Tests: `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsCallbacksRunOnceAndPreparedStateDoesNotAliasRetainedHandles`, `connection_create_volume_test.go:TestConnectionCreateVolumeCachedCinderLiveTokenAndOwnedImageSelection`, `connection_create_volume_test.go:TestConnectionCreateVolumeOptionsPrecedeDiscoveryAndReuseMicroversion`, `connection_create_volume_test.go:TestConnectionCreateVolumePreflightBeforeCallbacksAndServiceSelection`.
- Only the actual lower-case canonical creation id, validated as a nonempty safe segment, binds VolumeID and future GET/action paths. Canonical nonnull status is required; actual literal lowercase initial error fails immediately even with waiting disabled, while differently cased ERROR follows the separate no-wait or waiter semantics. Response aliases, request attribute id, foreign Location or later response id cannot retarget the workflow.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAllPinnedAttributesAndWireAliasesAreSerialized`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsInitialLiteralErrorAndExactFreshWaitStates`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAcceptedCreateAndActionFailuresPreservePhaseEvidence`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitCancellationAndLaterFailuresRetainAcceptedCreation`.
- Any nonnil Bootable forces effective waiting regardless of Wait option order. False waits but sends no false bootable action; true sends fixed POST200 os-set_bootable true only after a fresh successful available observation. Omitted Bootable with Wait(false) returns after valid202 creation without polling or an action.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsBootableTriStateAlwaysWaitsButOnlyTrueSendsAction`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsEffectiveDefaultsAndBootableForcesWaitAfterReplacement`, `connection_create_volume_test.go:TestConnectionCreateVolumeOptionsPrecedeDiscoveryAndReuseMicroversion`.
- Waiting begins after creation response reading, Close, canonical identity binding and source validation. Nil/zero Timeout is unlimited and negative/inactive-invalid policy fails before mutation. Positive context deadline cancels Cinder poll HTTP and timers; parent context bounds every phase. Target available is tested before exact case-insensitive failures, no error-prefix inference occurs and nonnil empty failure states disable matching.
  Tests: `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitBudgetExcludesCreateAndBootableUsesParentContext`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitCancellationAndLaterFailuresRetainAcceptedCreation`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsEffectiveDefaultsAndBootableForcesWaitAfterReplacement`, `blockstorage/create_volume_options_test.go:TestCreateVolumeOptionsValidateKnownFieldBoundaryInactiveWaitAndCancellation`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsInitialLiteralErrorAndExactFreshWaitStates`.
- Nonterminal ProgressCallback receives zero rather than arbitrary response progress extensions. Its error, source change or caller cancellation preserves inspectable callback/context/custom causes and prior Created plus LastAccepted proof. Target/failure observations suppress callback and source/context violations prevent another poll or action.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsInitialLiteralErrorAndExactFreshWaitStates`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitCancellationAndLaterFailuresRetainAcceptedCreation`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportAcceptedPhaseReadCloseContextAndSourceFailuresKeepProof`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportEntrySnapshotsLiveImageTokenAndSourceMutationGuards`.
- Created and LastAccepted capture independent actual accepted202/200 raw bytes, cloned headers and status before read/Close/decode/post-source failures. Ready independently owns the successful available VolumeInfo. Later rejected HTTP/transport failures preserve native physical status/body/header/method/URL or original causes without overwriting prior LastAccepted or fabricating accepted evidence. Accepted processing failures never start a new logical creation or cleanup.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsDefaultFreshWaitAndIndependentOwnedResults`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAcceptedCreateAndActionFailuresPreservePhaseEvidence`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportAcceptedPhaseReadCloseContextAndSourceFailuresKeepProof`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitCancellationAndLaterFailuresRetainAcceptedCreation`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportUnacceptedCauseAndCanceledContextDoNotInventProof`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportNativeRetryCannotRewritePhaseBodyOrPromoteStatus`.
- VolumeInfo atomically decodes exact canonical nullable string/int/flag fields, literal timestamps, inherited links, raw MetadataFields and reused attachment records. Bootable/encrypted accept bool or exact case-insensitive true/false strings; other typed flags require JSON bool. Absent/null/empty metadata or attachment containers retain distinct raw evidence, wrong known types/UTF8/null rows fail without partial assignment, and remaining response fields keep raw JSON precision rather than falsely gaining typed Python parity.
  Tests: `blockstorage/create_volume_models_test.go:TestCreateVolumeModelsCanonicalNullableTypesAndLiteralMetadataPrecision`, `blockstorage/create_volume_models_test.go:TestCreateVolumeModelsKnownFieldFailuresAreAtomicAndNullEmptyRemainDistinct`, `blockstorage/attach_volume_models_test.go:TestAttachVolumeModelsRejectInvalidFieldsAtomically`.
- BootableSet retains actual opaque POST200 body/header/status on action acknowledgement or accepted handling errors without requiring JSON decoding, rewriting Ready.IsBootable, fetching again or certifying a later bootable observation. The action uses the original caller context after the wait-only budget has ended; action failure leaves Created/LastAccepted/Ready and its own native or accepted proof intact.
  Tests: `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsBootableTriStateAlwaysWaitsButOnlyTrueSendsAction`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsAcceptedCreateAndActionFailuresPreservePhaseEvidence`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportWaitBudgetExcludesCreateAndBootableUsesParentContext`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportAcceptedPhaseReadCloseContextAndSourceFailuresKeepProof`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportNativeAuthBackoffRetryOwnsCreateAndBootableJSON`.
- Provided Cinder/Glance sources are captured before original package options, keep effective prefixed bases, canonical ordinary entry headers and consistent service role/microversion, and use original live providers for authentication. Provider/Endpoint/ResourceBase/Type/Microversion replacements are terminal after callbacks, lookup or phase handling. Configured native reauthentication/backoff/retry remain separate from SDK method/URL/serialized-body/accepted-policy ownership guards.
  Tests: `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportEntrySnapshotsLiveImageTokenAndSourceMutationGuards`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportNativeAuthBackoffRetryOwnsCreateAndBootableJSON`, `blockstorage/create_volume_transport_test.go:TestCreateVolumeTransportNativeRetryCannotRewritePhaseBodyOrPromoteStatus`, `blockstorage/create_volume_contracts_test.go:TestCreateVolumeContractsPreflightValidatesEveryProvidedSourceBeforeCallbacks`.
- Connection cache reuse, optional Glance selection, local preparation failures before discovery and service-initialization cancellation/custom causes are exposed through CreateVolume/volume operation errors. Existing native Cinder Create/SetBootable APIs, generated inventory, attachment workflows and Collection error-prefix waiter remain independent; no dependency declaration is promoted by this direct cloud-helper review.
  Tests: `connection_create_volume_test.go:TestConnectionCreateVolumeCachedCinderLiveTokenAndOwnedImageSelection`, `connection_create_volume_test.go:TestConnectionCreateVolumeOptionsPrecedeDiscoveryAndReuseMicroversion`, `connection_create_volume_test.go:TestConnectionCreateVolumePreflightBeforeCallbacksAndServiceSelection`, `connection_create_volume_test.go:TestConnectionCreateVolumeInitializationRetainsCancellationCause`.

Python/Go differences:

- Go uses an explicit optional resource.Ref Image instead of Python image dict/Resource/string identity polymorphism; ID requires no Glance request/client and Name resolves exactly once. Snapshot/source volume/backup attrs remain literal IDs with no extra lookup in either helper. SourceReplica is a declared native Go extension, not a missing pinned Python Body field.
- Go covers known Cinder attribute/wire names with concrete common controls and owned raw JSON. It deliberately preserves chosen JSON values without Python int/bool/BoolStr/dict/list descriptor coercion and rejects unknown/private/base_path/microversion controls instead of changing the selected route/session. Wire keys win duplicate alias representations deterministically because Go maps have no Python insertion-order contract; canonical name/description presence and falsey omission still match the cloud helper.
- Go fixes every follow-up route to a validated canonical creation id and requires usable canonical status, rejecting unsafe/missing/contradictory response identity without adopting request id, alternate identity or Location. Rich nullable models and raw precision are independent from mutable Python Resource hydration/descriptors, and fields without public typed projection remain in Metadata.Body.
- Go always fetches a fresh available observation after creation when effective waiting is enabled; Python may short-circuit a created Resource already in the target cached state. Both apply the literal initial lowercase error check even with wait disabled. Both nonnull bootable values force waiting, false sends no action and true sends a true action after readiness; Go preserves opaque action proof without synthesizing observed bootable state.
- Go intentionally uses nil/numeric-zero Timeout as unlimited waiting and rejects negative/inactive-invalid policies before mutation. Python forwards timeout and its iterate_timeout treats numeric zero/negative as immediate post-creation timeout unless an already-target Resource short-circuits. Positive Go wait context interrupts HTTP/timers while Python checks between fetches. The bootable action uses the original parent context after the wait-only phase, and caller context bounds the full Go workflow.
- Go returns Created/LastAccepted/Ready/BootableSet evidence and original errors after accepted side effects rather than Python returning one mutable Volume or raising without phase results. Error-returning progress callbacks, explicit failure controls and local preparation are Go extensions; no automatic delete or second logical create is added. Accepted phases are not a complete physical-attempt trace.
- Connection prepares original callbacks once before cached service selection and chooses Glance only for image Name; package calls capture every supplied source before originals. Configured current authentication and native retry/reauthentication remain in place while source and request ownership are guarded. This is not a packet-level exactly-once promise or arbitrary mutable provider configuration contract.
- This mapping closes only the directly declared cloud create_volume size/wait/timeout/image/bootable/known-kwargs workflow under documented Go policies. Separate Python Cinder create/wait/set_bootable, Glance find, inherited Resource/session/cache/API-version declarations retain independent reviews. Connection selects Cinder v3 and this row does not assert full Cinder v2 Resource object semantics or whole-SDK completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func createDefault(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 10},
        blockstorage.WithCreateVolumeName("data"),
        blockstorage.WithCreateVolumeMetadata(map[string]string{"owner": "worker"}))
}

func createWithoutWaiting(ctx context.Context, cinder *gophercloud.ServiceClient) (*blockstorage.CreateVolumeResult, error) {
    return blockstorage.CreateVolume(ctx, cinder, nil,
        blockstorage.CreateVolumeRequest{Size: 10},
        blockstorage.WithCreateVolumeWait(false))
}

func createBootableFromImage(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
    image := resource.Name("ubuntu")
    return conn.CreateVolume(ctx,
        blockstorage.CreateVolumeRequest{Size: 20, Image: &image},
        blockstorage.WithCreateVolumeBootable(true))
}

func createWithPolicy(ctx context.Context, conn *sdk.Connection, progress func(int) error) (*blockstorage.CreateVolumeResult, error) {
    timeout, interval := 5*time.Minute, time.Second
    return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 30},
        blockstorage.WithCreateVolumeSnapshot("snapshot-id"),
        blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{
            "volume_type_id": json.RawMessage(`"type-id"`),
            "is_multiattach": json.RawMessage(`false`),
        }),
        blockstorage.WithCreateVolumeSchedulerHints(map[string]json.RawMessage{
            "same_host": json.RawMessage(`["source-volume-id"]`),
        }),
        blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{
            Timeout: &timeout, PollInterval: &interval,
            ProgressCallback: progress,
        }))
}
```

```python
volume = conn.create_volume(10, name="data", metadata={"owner": "worker"})
boot_volume = conn.create_volume(20, image="ubuntu", bootable=True)
```

## References

- [..](../README.md)
- [Volume SetBootable API](v3/README.md)
- [BlockStorageCloudMixin.create_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L175-L232)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494-L503)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54-L144)
- [hint request preparation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446-L464)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L102)
- [Attach Volume](attach-volume.md)
- [Detach Volume](detach-volume.md)
