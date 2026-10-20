# Volume Id

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeID`

- GetVolumeID always performs the ordinary unfiltered exact name/ID lookup before extracting the actual response ID. Safe member success sends one fixed bodyless200 GET; only compatible rejected400/403/404 fall back to detailed exact-name list. Input UUID/string IDs never become a return-value shortcut, and unsafe names use the existing Go list-only safety policy without glob or JMESPath selection.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDRetainsFoundResourceAndArbitraryActualResponseID`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDResolvesActualListIDAfterCompatibleMemberRejection`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDExactNamesAndCompletedAbsenceNeverBecomeIDShortcutsOrGlob`, `connection_volume_id_test.go:TestConnectionGetVolumeIDAbsenceUnsafeNamesAndLookupFailuresRemainDistinct`.
- A found ordinary Volume is present independently of raw-body or ID truthiness. ID preserves the actual untyped JSON string/null/bool/number/list/object, including false, zero and empty string, rather than coercing to string or returning the supplied identity. A found missing canonical id yields literal JSON null with Value/Volume retained; a present different response ID remains unchanged.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDRetainsFoundResourceAndArbitraryActualResponseID`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDResolvesActualListIDAfterCompatibleMemberRejection`, `connection_volume_id_test.go:TestConnectionGetVolumeIDCachedCinderOriginalOptionsAndUntypedResponseIDs`.
- Completed no-match lookup returns nil ID/Value/Volume with actual admitted Pages; every failed lookup leaves ID nil and logical values uncommitted. Preparation failures return no result and send no HTTP, whereas executed identity/physical/local failures return the result with only actual available evidence and exact GetVolumeID operation context. Error is checked before treating nil ID as absence.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDExactNamesAndCompletedAbsenceNeverBecomeIDShortcutsOrGlob`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDDistinguishesPreparationFailureFromExecutedEmptyProofFailure`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDTerminalNativeErrorsKeepCurrentRejectionWithoutAcceptedEvidence`, `connection_volume_id_test.go:TestConnectionGetVolumeIDPreflightOptionAndGetterFailuresKeepOperationContext`, `connection_volume_id_test.go:TestConnectionGetVolumeIDAbsenceUnsafeNamesAndLookupFailuresRemainDistinct`.
- Every visited member/list row consumes the full normalized v3 descriptor/location view before identity selection. A malformed visited canonical object or descriptor fails without fallback or ID output; the second exact list match raises ambiguity before an unused bad row or continuation. A unique early match still requires successful completion of the remaining fallback search before its ID is committed.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDAmbiguityPrecedesUnusedBadTailAndContinuation`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDAcceptedMemberSchemaAndDescriptorErrorsNeverReturnIDOrFallback`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDLaterFallbackFailuresDoNotBorrowSuppressedMemberOrCommitEarlierID`.
- ID, complete normalized38-field Value, physical RawResource.Volume and actual Observed/Pages body/header/status are independently owned. Original wire unknown fields and numeric text remain available, the ID slice never aliases raw/normalized/proof data, and an admitted decoder failure owns its originating ResponseError separately from returned proof.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDRetainsFoundResourceAndArbitraryActualResponseID`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDResolvesActualListIDAfterCompatibleMemberRejection`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDAcceptedMemberSchemaAndDescriptorErrorsNeverReturnIDOrFallback`, `connection_volume_id_test.go:TestConnectionGetVolumeIDCachedCinderOriginalOptionsAndUntypedResponseIDs`.
- The direct entry captures and validates Cinder/service/header facts before original read callbacks. Original callbacks and the option slice run once with owned location snapshots; ordinary captured headers stay fixed, live provider authentication remains available, and selected provider/endpoint/resource base/type/microversion changes stop execution before a later restoring callback or another exchange. Native authentication/backoff/retry repeats only the protected bodyless exact request and does not rerun originals.
  Tests: `blockstorage/volume_id_contract_test.go:TestGetVolumeIDOwnsOriginalOptionSliceLocationAndHeaders`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDOriginalSourceGuardStopsRestoringOptionBeforeHTTP`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDPostResponseSourceDriftKeepsActualMemberAndNoID`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDNativeAuthenticationRetriesKeepOwnedBodylessMemberAndSingleID`.
- Admitted member/list evidence survives read, Close, schema, source and cancellation failures without establishing an ID or causing orchestration replay. Joined read/Close/context cancellation and context.Cause remain inspectable; a later fallback transport/rejected/admitted failure retains its own evidence and never borrows the earlier suppressed member400/404 as the current error.
  Tests: `blockstorage/volume_id_transport_test.go:TestGetVolumeIDAcceptedReadCloseCancellationPreservesAllCausesAndProof`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDTerminalNativeErrorsKeepCurrentRejectionWithoutAcceptedEvidence`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDLaterFallbackFailuresDoNotBorrowSuppressedMemberOrCommitEarlierID`, `blockstorage/volume_id_transport_test.go:TestGetVolumeIDPostResponseSourceDriftKeepsActualMemberAndNoID`, `connection_volume_id_test.go:TestConnectionGetVolumeIDAcceptedProofSurvivesReadCloseCancellationAndSourceDrift`.
- Connection applies original read options once before recorded CurrentLocation and cached Cinder v3 selection, then delegates an owned policy without selecting Nova or adding scope/network enrichment. Location factories, retained callbacks and per-call scope are independent snapshots; caller-invalid Location is consumed by actual rows only and remains local ErrInvalidOption without a borrowed200 ResponseError, while independently admitted proof remains available.
  Tests: `connection_volume_id_test.go:TestConnectionGetVolumeIDCachedCinderOriginalOptionsAndUntypedResponseIDs`, `connection_volume_id_test.go:TestConnectionGetVolumeIDLocationSnapshotsAndOwnedOverrides`, `connection_volume_id_test.go:TestConnectionGetVolumeIDPreflightOptionAndGetterFailuresKeepOperationContext`, `connection_volume_id_test.go:TestConnectionGetVolumeIDAbsenceUnsafeNamesAndLookupFailuresRemainDistinct`, `blockstorage/volume_id_contract_test.go:TestGetVolumeIDOwnsOriginalOptionSliceLocationAndHeaders`.

Python/Go differences:

- Go selects Cinder v3 under Gophercloud v2.15.0 and reuses the already audited ordinary Volume find/normalized38-field view. Pinned cloud can select v2; this direct helper row promotes no independent v2, native, Proxy, Resource, mutable session/cache or volume-type declarations.
- Pinned get_volume_id calls default get_volume and tests the returned Volume Resource, whose underlying dict is filled with default descriptor values and is ordinarily truthy even for empty wire body. The inherited id is untyped and returned unchanged; Go maps ordinary resource presence, not raw-body or ID truthiness. Dynamic monkeypatched get_volume objects are not new typed direct APIs.
- Go returns ID as original json.RawMessage, preserving numeric spelling/precision and nested JSON encoding. Python returns decoded runtime values; exact raw JSON and finite canonical decoder boundaries are deliberate rather than string coercion or a complete Python object-runtime claim.
- Go found missing/null id is literal JSON null plus retained Volume/Value, distinct from successful absence with nil ID/Volume/Value. Python returns None for both. Pinned seeded member Resource can preserve requested ID when response omits id, while Go deliberately retains only actual response identity rather than fabricating that seed.
- The existing safe-route Go FindIdentity policy skips member GET for identities containing slash/backslash/query/fragment/percent or Unicode whitespace and searches exact list identity directly. Pinned Resource.find unconditionally attempts GET first and catches only400/403/404; wildcard characters remain exact ordinary identities in this helper.
- Strict canonical UTF-8 object envelopes and accepted200 plus native volumes_links same path/origin/cycle policy are inherited Go reader boundaries. Generic Resource permissive translation, warnings, other continuation sources and lifecycle profiles remain separately reviewed.
- Go exposes richer owned ID/resource/proof results with errors and joined cancellation causes. Actual admitted response proof can survive failure without becoming successful output; rejected responses remain in their original native errors, with no stale proof borrowing or SDK orchestration replay.
- Computed location follows the existing owned Volume/recorded-scope mapping. Local caller Location errors remain separate from physical response/descriptor errors. Disconnected mutable Resource and Type constructor raw computed-location edge cases remain independent profiles, with no current connected Volume helper defect indicated.

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

func ReadVolumeID(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil {
        if result != nil {
            fmt.Println("admitted fallback pages:", len(result.Pages))
            if result.Observed != nil {
                fmt.Println("admitted member status:", result.Observed.StatusCode)
            }
        }
        return err
    }
    if result.ID == nil {
        fmt.Println("volume absent")
        return nil
    }
    fmt.Println("ID JSON:", string(result.ID))
    fmt.Println("normalized volume:", string(result.Value))
    copy := result.Volume.Clone()
    fmt.Println("original ID field:", string(copy.Body["id"]))
    return nil
}

func ReadVolumeIDWithLocation(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    project := "configured-project"
    location := resource.CloudLocation{
        Project: resource.CloudProject{
            ID: json.RawMessage(`"scope-id"`), Name: &project,
        },
    }
    result, err := conn.GetVolumeID(ctx,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID},
        blockstorage.WithVolumeReadLocation(location))
    if err != nil { return err }
    fmt.Println(string(result.ID))
    return nil
}

func ReadVolumeIDDirect(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.GetVolumeID(ctx, cinder,
        blockstorage.GetVolumeIDRequest{NameOrID: nameOrID})
    if err != nil { return err }
    if result.ID != nil { fmt.Println(string(result.ID)) }
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="example")
volume_id = conn.get_volume_id("data")
if volume_id is None:
    print("no ID returned")
else:
    print(volume_id)
```

## References

- [Volume Reads](volume-reads.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L362-L371)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L82-L122)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
