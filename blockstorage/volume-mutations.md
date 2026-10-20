# Volume Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/update_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UpdateVolume`

- Resolve one ordinary unfiltered exact volume through the captured reader, retaining the selected original raw JSON row. Compatible clean member400/403/404 can fall back to name-hinted exact list; completed absence, duplicate second match and lookup schema/status errors stop before mutation. Bind only the actual safe canonical returned string ID and retain Resolved on local target errors.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsOrdinaryFindFallbackUsesNameQueryAndActualReturnedID`, `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsCompletedAbsenceAndSecondMatchAmbiguityStopBeforeMutation`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationLookupSchemaAndUnexpectedNativeStatusStopBeforeMutation`, `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsRejectInvalidActualCanonicalIDLocallyWithResolvedProof`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsLocalTargetAndLocationFailuresKeepLookupWithoutMutation`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`.
- Supply every writable field from the actual37 Volume Body descriptors through SDK-owned typed/raw options. Concrete fields override raw Fields; Go map wire keys win normalized aliases. Canonical name/description presence wins display aliases and selected falsey values are dropped. Preserve nullable/empty raw fields, ignore unknown inputs and source_replica, reject id/runtime route controls, and inject no creation defaults.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeForwardsAllKnownBodyAttributesWithoutCreationDefaults`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeWireAliasesAndTypedAttributesHaveDeterministicPrecedence`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeDropsFalseyCloudNamesAndIgnoresUnknownAttributes`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationPreflightContextRoleCallbacksAndProtectedFieldsStopBeforeHTTP`.
- Compare requested canonical raw values with selected wire-order prior state, using recursive Python boolean/number equality, object order independence, ordered lists and the declared exact-decimal boundary. Equal assignments retain original raw values; absent differs explicit null. Empty/equal/unknown-only updates skip PUT and leave Applied nil with independently owned logical lookup-derived results.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeNoOpPreservesOriginalRawRecursivePythonEquality`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeSendsOnlyDirtyFieldsForStringNumberListOrderAndExactDecimals`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeAbsentKnownFieldVersusExplicitNullIsDirty`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeDirtyComparisonConsumesLookupAliasesInActualWireOrder`, `connection_volume_mutations_test.go:TestConnectionUpdateVolumeNoOpKeepsLookupAndSkipsMutation`.
- Build the complete proposed descriptor/location view before PUT so invalid known conversions fail locally with Resolved and no Applied. Serialize dirty raw values without descriptor coercion. Move dirty scheduler hints outside volume only when truthy; dirty falsey hints still send volume{} and a nonobject raw hint may have an empty dictionary descriptor view.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeProposedDescriptorFailureIsLocalBeforePUT`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeSendsOnlyDirtyFieldsForStringNumberListOrderAndExactDecimals`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeSchedulerHintsDirtyFalseyValuesStillCommitEmptyVolume`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsLocalTargetAndLocationFailuresKeepLookupWithoutMutation`.
- Merge recognized fields from canonical volume or flat object responses into owned prior/request state; omitted fields remain and returned ID is only a result attribute. Accepted empty/malformed UTF8 JSON is tolerated; valid wrong shapes, invalid UTF8 or returned descriptor failures keep Applied and reject atomic Volume/Value. Unknown response fields are ignored while prior unknown raw evidence remains; all results and lookup/response proofs own their bytes.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumePartialAndFlatResponsesMergeRequestedPriorStateAtomically`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeAcceptedEmptyAndMalformedJSONKeepRequestedMergedState`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeValidWrongShapesAndInvalidUTF8RetainAppliedWithoutLogicalResult`, `blockstorage/volume_mutations_contract_test.go:TestUpdateVolumeReturnedDescriptorFailureIsPhysicalAndAtomic`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`.
- Run original option callbacks once from owned slices/pointers/maps/rawJSON, with direct source capture before originals and Connection originals before cached Cinder v3 selection. Snapshot complete configured/recorded scope after originals and before lookup; later authentication scope changes do not alter this call location or defaults. Factories/preparers own reusable policies independently of caller or retained callback mutations.
  Tests: `blockstorage/volume_mutations_options_test.go:TestVolumeMutationPrepareDefaultsAndFactoriesOwnCompleteReusablePolicies`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationOriginalOptionsOwnSlicesPointersFieldsAndHeadersAcrossBothStages`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOwnFactoryAndRetainedCallbackState`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationRecordedScopeSnapshotsAfterOriginalsBeforeLookup`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsRecordedScopeIsCapturedAfterOriginalsThroughMutation`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`.
- Validate context/client role/options before execution and preserve outer operation, original failures, context.Err and custom cancellation cause. Provider/endpoint/resource base/type/microversion drift is sticky across callbacks and accepted physical stages. Local source/location/target/proposed-field errors never borrow prior HTTP200; cancellation of a blocked mutation retains lookup without invented Applied.
  Tests: `blockstorage/volume_mutations_options_test.go:TestVolumeMutationPreflightContextRoleCallbacksAndProtectedFieldsStopBeforeHTTP`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsPreflightAndGetterFailuresKeepOuterOperationAndCause`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationOriginalSourceChangeIsTerminalBeforeRestoringCallback`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationSourceFactsStayFrozenAfterAcceptedLookupOrMutation`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationBlockedHTTPStageCancellationRetainsLookupWithoutInventedApplied`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsLocalTargetAndLocationFailuresKeepLookupWithoutMutation`.
- Send only the bound queryless PUT with original admitted100..399 policy and fixed method/URL/body/response ownership. Preserve live native authentication and caller-configured native retry without workflow replay, refresh or rollback; native hook code expansion never admits actual>=400. Accepted read/Close/cancellation/source errors keep current Applied and joined causes; native/transport rejection keeps its own evidence and never borrows suppressed member or prior lookup proof. Redirect guard blocks a changed target before credentials are sent.
  Tests: `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationNativeReauthAndBoundedRetryPreservePayloadTargetAndLiveHeaders`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationNativeRetryProtectsOwnedBodyDecoderAndOriginalAcceptedStatus`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationRedirectCannotChangeFixedTargetMethodOrSendSharedCredentials`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationAcceptedReadCloseCancellationRetainsCurrentStageAndAllCauses`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsAcceptedAndRejectedSecondStageProofStaysDistinct`.

Python/Go differences:

- Only this actual cloud declaration is mapped to selected Cinder v3 under pinned Gophercloud v2.15.0. Independent native/Proxy/v2 operations, inherited Resource state/session/cache/microversion negotiation and other workflows retain separate support status; this row is not a full SDK completion claim.
- Go uses one fresh canonical lookup result and validates an actual safe nonempty string ID. Python can retain a seeded mutable Resource ID and utils.urljoin coercion, including missing/falsey or route-changing values; local Go target validation retains Resolved without borrowing HTTP200.
- Ordinary lookup reuses existing Go exact identity/name, safe list-only input routing, nullable descriptor/location view, fixed canonical GET200 and continuation policies. These declared mappings do not establish inherited generic Resource equivalence.
- Source v3 mutation raise_from_response status<400 maps to valid HTTP100..399, independently of lookup200 and native mutation200. Informational exposure follows net/http. Fixed origin/escaped-path/query/method guards intentionally block redirects that alter the bound request.
- Go adds independently owned lookup and mutation body/header/status evidence with joined native/read/Close/context causes and OperationError identity. Applied is acknowledgement evidence, not proof of changed/completed cloud state; rejected/local failures do not borrow prior success proof.
- Original SDK options are owned and applied once. Direct calls capture source before originals; Connection applies originals before CurrentLocation/getter and uses cached Cinder v3. Scope is frozen for the call while authentication/native retry stays live; no SDK workflow replay occurs.
- Python update checks configured proxy v3 after lookup and bootable can follow configured v2/v3. Go exposes the selected v3 contract, captures the chosen microversion, and does not perform generic session negotiation; a direct ServiceClient alone cannot prove configured API version.
- The actual cloud update has no valid_kwargs decorator and recognizes all37 Volume Body descriptors. Go accepts the36 nonidentity writable descriptors/aliases with raw nullable JSON; id/runtime route controls are rejected rather than mutating Resource identity/request policy. Unknown inputs are ignored, including creation-only source_replica.
- Input Go maps use deterministic wire-name precedence and concrete-field precedence; Python kwargs use insertion order. Selected lookup/response JSON uses last textual alias order, which differs from interleaved duplicate keys collapsed by Python JSON loading. Known fields compare and serialize raw, rather than normalized descriptor values.
- Go exact-decimal recursive boolean/number equality and truthiness differ from Python JSON float rounding/underflow and arbitrary Python objects. No-op equal assignment retains original raw fields. Ordinary JSON descriptor conversion reuses the declared search view rather than native typed Volume extraction.
- Source partial/flat response merge and empty/malformed JSON ValueError tolerance are preserved with independently owned prior/request/server state. Go rejects invalidUTF8 explicitly, retains prior unknown raw fields as proof extension, ignores unknown response fields, and commits logical Volume/Value atomically rather than exposing a partially mutated shared Resource.

### `python:cloud/_block_storage/BlockStorageCloudMixin/set_volume_bootable`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeBootable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeBootable`

- Resolve one ordinary unfiltered exact volume using the same captured member/list reader. Compatible member400/403/404 fallback uses the ordinary name hint; duplicate second match stops before unused tail/continuation. Lookup schema/unexpected status failures do not proceed to action or trigger a new lookup.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsOrdinaryFindFallbackUsesNameQueryAndActualReturnedID`, `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsCompletedAbsenceAndSecondMatchAmbiguityStopBeforeMutation`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationLookupSchemaAndUnexpectedNativeStatusStopBeforeMutation`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`.
- Require the actual canonical response ID to be a safe nonempty UTF8 string URL segment; no input-ID seed or shortcut route. Completed absence is ErrNotFound and unsafe/missing/untyped IDs are local ErrInvalidOption with Resolved retained and no Applied. Owned location errors remain local and do not borrow admitted lookup proof.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestVolumeMutationsRejectInvalidActualCanonicalIDLocallyWithResolvedProof`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsLocalTargetAndLocationFailuresKeepLookupWithoutMutation`.
- Nil bootable selects true and explicit false serializes unchanged in os-set_bootable.bootable. Accept the selected source100..399 acknowledgement as opaque bytes, including empty, nonJSON and binary bodies, irrespective of current volume status. Expose Applied evidence only; do not parse responseJSON, change lookup fields, refresh, wait, verify, rollback or infer a Changed/completed flag.
  Tests: `blockstorage/volume_mutations_contract_test.go:TestSetVolumeBootableDefaultTrueExplicitFalseAndOpaqueSub400Acknowledgement`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationNativeReauthAndBoundedRetryPreservePayloadTargetAndLiveHeaders`.
- Own bool pointers, location and original callback slice across reusable factories/preparers and retained callback mutation. Run each original once; direct source capture precedes originals while Connection applies originals before selecting cached Cinder v3. A complete replacement nil flag reapplies default true and explicit false remains exact.
  Tests: `blockstorage/volume_mutations_options_test.go:TestVolumeMutationPrepareDefaultsAndFactoriesOwnCompleteReusablePolicies`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationOriginalOptionsOwnSlicesPointersFieldsAndHeadersAcrossBothStages`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOwnFactoryAndRetainedCallbackState`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID`.
- Snapshot complete configured/recorded current scope after original options and before lookup, using an owned per-call location without changing Connection defaults. Authentication scope changes between lookup and action do not recompute this call location; actual volume project/zone still follow owned view rules.
  Tests: `blockstorage/volume_mutations_options_test.go:TestVolumeMutationRecordedScopeSnapshotsAfterOriginalsBeforeLookup`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsRecordedScopeIsCapturedAfterOriginalsThroughMutation`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsOwnFactoryAndRetainedCallbackState`.
- Reject invalid context/service role/nil or failed callbacks before HTTP and preserve outer operation plus custom cancellation/original/getter causes. Provider/endpoint/resource base/type/microversion source drift is terminal before any restoring callback or later action; accepted source drift retains its current physical stage.
  Tests: `blockstorage/volume_mutations_options_test.go:TestVolumeMutationPreflightContextRoleCallbacksAndProtectedFieldsStopBeforeHTTP`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsPreflightAndGetterFailuresKeepOuterOperationAndCause`, `blockstorage/volume_mutations_options_test.go:TestVolumeMutationOriginalSourceChangeIsTerminalBeforeRestoringCallback`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationSourceFactsStayFrozenAfterAcceptedLookupOrMutation`.
- Retain completed lookup and admitted opaque action proof independently on read/Close/context.Cause/source errors. Native/transport action rejection never borrows earlier suppressed member400/403/404 or lookup200 as successful action evidence. A blocked action canceled before response retains lookup and custom cause without inventing Applied.
  Tests: `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationAcceptedReadCloseCancellationRetainsCurrentStageAndAllCauses`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof`, `connection_volume_mutations_test.go:TestConnectionVolumeMutationsAcceptedAndRejectedSecondStageProofStaysDistinct`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationBlockedHTTPStageCancellationRetainsLookupWithoutInventedApplied`.
- Preserve the bound queryless POST /volumes/id/action through native reauthentication and caller-native retries with live headers/tokens. Request body and response decoder/retention stay library-owned; actual status>=400 is rejected even after native OkCodes expansion. Guard fixed origin/path/query/method against redirects, and do not replay the SDK lookup/workflow.
  Tests: `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationNativeReauthAndBoundedRetryPreservePayloadTargetAndLiveHeaders`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationNativeRetryProtectsOwnedBodyDecoderAndOriginalAcceptedStatus`, `blockstorage/volume_mutations_transport_test.go:TestVolumeMutationRedirectCannotChangeFixedTargetMethodOrSendSharedCredentials`.

Python/Go differences:

- Only this actual cloud declaration is mapped to selected Cinder v3 under pinned Gophercloud v2.15.0. Independent native/Proxy/v2 operations, inherited Resource state/session/cache/microversion negotiation and other workflows retain separate support status; this row is not a full SDK completion claim.
- Go uses one fresh canonical lookup result and validates an actual safe nonempty string ID. Python can retain a seeded mutable Resource ID and utils.urljoin coercion, including missing/falsey or route-changing values; local Go target validation retains Resolved without borrowing HTTP200.
- Ordinary lookup reuses existing Go exact identity/name, safe list-only input routing, nullable descriptor/location view, fixed canonical GET200 and continuation policies. These declared mappings do not establish inherited generic Resource equivalence.
- Source v3 mutation raise_from_response status<400 maps to valid HTTP100..399, independently of lookup200 and native mutation200. Informational exposure follows net/http. Fixed origin/escaped-path/query/method guards intentionally block redirects that alter the bound request.
- Go adds independently owned lookup and mutation body/header/status evidence with joined native/read/Close/context causes and OperationError identity. Applied is acknowledgement evidence, not proof of changed/completed cloud state; rejected/local failures do not borrow prior success proof.
- Original SDK options are owned and applied once. Direct calls capture source before originals; Connection applies originals before CurrentLocation/getter and uses cached Cinder v3. Scope is frozen for the call while authentication/native retry stays live; no SDK workflow replay occurs.
- Python update checks configured proxy v3 after lookup and bootable can follow configured v2/v3. Go exposes the selected v3 contract, captures the chosen microversion, and does not perform generic session negotiation; a direct ServiceClient alone cannot prove configured API version.
- Go publicly accepts bool values through an optional owned pointer with nil/defaulttrue and explicitfalse. Python annotations do not restrict runtime string/null/number/container action values; that dynamic input domain is outside the bool mapping.
- Python returns None without parsing or verifying its acknowledgement. Go exposes Applied as opaque evidence without modifying resolved bootable or adding wait/timeout/refresh/completion semantics. Existing CreateVolume true-only readiness action and native zero-defaultfalse/status200 leaf remain separate unchanged APIs.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func InspectUpdate(result *blockstorage.UpdateVolumeResult) {
    if result == nil { return }
    fmt.Println("fixed mutation ID:", result.VolumeID)
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("actual PUT response:", result.Applied.StatusCode,
            "raw bytes:", len(result.Applied.Body))
    }
}

func Update(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeName("renamed"),
        blockstorage.WithUpdateVolumeMetadata(map[string]string{}),
        blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{
            "image_id": json.RawMessage(`"image-id"`),
            "scheduler_hints": json.RawMessage(`{"same_host":["peer-volume"]}`),
        }))
    InspectUpdate(result)
    if err != nil { return err }
    fmt.Println("merged normalized JSON:", string(result.Value))
    owned := result.Volume.Clone()
    fmt.Println("merged raw metadata:", string(owned.Body["metadata"]))

    // Empty kwargs still resolve the volume but do not send a PUT.
    unchanged, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"})
    InspectUpdate(unchanged)
    if err != nil { return err }
    fmt.Println("no PUT:", unchanged.Applied == nil)
    return nil
}

func Bootable(ctx context.Context, conn *sdk.Connection) error {
    input := blockstorage.SetVolumeBootableRequest{NameOrID: "data"}
    enabled, err := conn.SetVolumeBootable(ctx, input) // Default: true.
    if enabled != nil && enabled.Applied != nil {
        fmt.Println("enable acknowledgement:", enabled.Applied.StatusCode)
    }
    if err != nil { return err }
    disabled, err := conn.SetVolumeBootable(ctx, input,
        blockstorage.WithSetVolumeBootable(false))
    if disabled != nil && disabled.Applied != nil {
        fmt.Println("disable acknowledgement:", disabled.Applied.StatusCode)
    }
    return err
}

func UpdateWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    name := "renamed"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.UpdateVolume(ctx,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{
            Attributes: blockstorage.UpdateVolumeAttributes{Name: &name},
            Location: &location,
        }))
    InspectUpdate(result)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.UpdateVolume(ctx, cinder,
        blockstorage.UpdateVolumeRequest{NameOrID: "data"},
        blockstorage.WithUpdateVolumeDescription("updated"))
    InspectUpdate(result)
    if err != nil { return err }
    _, err = blockstorage.SetVolumeBootable(ctx, cinder,
        blockstorage.SetVolumeBootableRequest{NameOrID: "data"},
        blockstorage.WithSetVolumeBootable(false))
    return err
}
```

```python
updated = conn.update_volume("data", name="renamed", metadata={})
conn.set_volume_bootable("data")
conn.set_volume_bootable("data", False)
```

## References

- [pinned cloud update_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L234)
- [set_volume_bootable](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L254)
- [Volume Reads](volume-reads.md)
- [pinned _get_volume_kwargs](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L494)
- [pinned Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [Search Volumes](search-volumes.md)
- [pinned Volume request body](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L446)
- [pinned response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338)
- [pinned Volume bootable action](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L177)
