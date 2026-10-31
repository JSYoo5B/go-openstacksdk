# Volume Type Access

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeTypeAccess`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeTypeAccess`

- Resolve the ordinary unfiltered exact Type once through the captured reader. Safe identities use member GET with lowercase is_public=none; clean compatible member400/403/404 can fall back to exact list without a name hint. Lookup errors and completed absence retain Resolved; no access/action request follows.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessCompletedAbsenceAndEarlyAmbiguityStopBeforeSecondStage`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessFallbackFindUsesNoneThenReturnedIDWithoutVisibilityQuery`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Construct the queryless second route only from the actual returned canonical wire id. Missing, null, nonstring, empty or unsafe id is a local ErrInvalidOption with Resolved retained and no invented ResponseError; no input-ID seeding, coercion, refetch or Identity lookup occurs.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessRejectsInvalidCanonicalResponseIDLocallyAfterLookup`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Read one bodyless queryless /types/{returnedID}/os-volume-type-access response. Exact volume_type_access absence gives raw[]; present null/scalar/object/arbitrary arrays are preserved without typed field coercion. Unused links never paginate.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestGetVolumeTypeAccessPreservesMissingAndEveryArbitraryFieldShape`, `blockstorage/volume_type_access_contract_test.go:TestGetVolumeTypeAccessUsesReturnedIDAndOwnsRawAccessRowsWithoutProjectionOrPaging`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`.
- A complete all-object array, including empty[], may additionally project into owned RawResource rows with every unknown field and response metadata. Any nonobject item disables the entire convenience projection while the full successful Value remains. Value, Accesses, Observed and Resolved do not alias.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestGetVolumeTypeAccessUsesReturnedIDAndOwnsRawAccessRowsWithoutProjectionOrPaging`, `blockstorage/volume_type_access_contract_test.go:TestGetVolumeTypeAccessPreservesMissingAndEveryArbitraryFieldShape`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`.
- Admit every final valid HTTP100..399 under the direct source exception policy, independently from ordinary lookup200. Then require a UTF8 JSON object: empty204, malformed JSON or nonobject envelopes fail with actual admitted proof and no logical Value/Accesses; rejected>=400 remains native failure.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessAdmitsSub400StatusWhileAccessJSONRemainsRequired`, `blockstorage/volume_type_access_transport_test.go:TestGetVolumeTypeAccessInvalidJSONAndTopLevelShapeRetainOnlyActualAccessProof`.
- Direct calls capture selected Cinder facts and canonical header copies before original options. Connection prepares originals once before recorded location and cached Cinder selection. Owned option/location snapshots and the same reader persist through both phases; source drift is sticky and blocks subsequent work.
  Tests: `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalOptionsOwnLocationAndSliceAcrossBothPhysicalStages`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalSourceViolationIsTerminalBeforeRestoringOption`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessPreflightContextRoleAndOptionsStopBeforeLookup`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessMalformedCallerLocationKeepsLookupProofWithoutPhysicalError`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOwnsLocationFactoriesAndOriginalCallbackSlice`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessRecordedScopeSnapshotsBetweenStagesAndRefreshesNextCall`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessPreflightAndGetterFailuresKeepOuterOperationAndCause`.
- Physical lookup and second-phase evidence remain separate. Admitted body/header/status survive read, Close, cancellation and post-response source errors; native rejected responses preserve their own native evidence. Wrappers retain errors.Is/As, ctx.Err and context.Cause without replaying the SDK workflow.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessAcceptedReadCloseCancellationRetainsCorrectStageAndAllCauses`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessSourceFactsRemainFrozenAfterLookupAndAcceptedSecondResponse`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessBlockedSecondStageCancellationKeepsLookupWithoutInventedPhaseProof`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection`.
- Native authentication and retry preserve live token behavior and fixed method/URL/body/response ownership. Retry changes to body/decoder/KeepResponseBody are rejected before altered resend. Ordinary native header and OkCodes updates remain allowed, while an actual status>=400 cannot become SDK success outside the original owned policy.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeReauthenticationAndBoundedRetryKeepPayloadTargetAndLiveHeaders`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeRetryCannotReplaceOwnedBodiesDecoderOrBroadenOriginalStatus`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessRedirectCannotChangeFixedSecondStageTargetOrMethod`.

Python/Go differences:

- Only this actual cloud helper is mapped to Cinder v3 under Gophercloud v2.15.0. Independent native/Proxy/v2 declarations, inherited Resource descriptors/session negotiation/cache and other Type workflows are not promoted by this row.
- Python consumes a mutable returned Type and utils.urljoin coercion of its id. Go uses a fresh canonical response id that must be a safe nonempty UTF8 string URL segment; input identity is never inherited or used as a fallback route. Local target validation retains Resolved without borrowing HTTP200.
- Lookup reuses the existing Go exact identity, six-descriptor view, owned location and canonical accepted200 reader policy. Its safe input routing, complete-URL pagination safety and fresh response-field rules are declared mappings; this access row does not claim full Resource runtime equivalence.
- The direct access/action status gate preserves valid HTTP 100..399 rather than narrowing to native GET200/204/300 or action202. Fixed request origin, escaped path, exact query and method guards can block source-session redirect behavior before another exchange.
- Go exposes independently owned physical response evidence and joined native/read/Close/context causes; Python raises its own SDK exception classes and returns its ordinary helper value. Accepted evidence is not proof of subsequent server completion.
- Original caller options execute once from owned snapshots. Direct calls capture sources before originals; Connection runs originals before recorded location and service selection. Live native authentication/retry remains distinct from SDK-level replay, which is absent.
- Python cast is a runtime no-op and present access fields may be arbitrary values. Go preserves finite valid UTF8 raw JSON, requiring a top-level object; its optional all-object RawResource projection adds no schema coercion, descriptors, location or pagination.
- Python returns the selected untyped field or absent[]. Go returns Value plus optional Accesses and lookup/access proof. Nonarray or mixed arrays remain successful Value and do not fabricate partial row resources.
- Raw access Value and row field JSON retain original numeric spelling and precision, while Python response.json decodes integers/floats before returning them; finite valid JSON and Go string semantics do not emulate every Python runtime value.

### `python:cloud/_block_storage/BlockStorageCloudMixin/add_volume_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.AddVolumeTypeAccess`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AddVolumeTypeAccess`

- Resolve the ordinary unfiltered exact Type once through the captured reader. Safe identities use member GET with lowercase is_public=none; clean compatible member400/403/404 can fall back to exact list without a name hint. Lookup errors and completed absence retain Resolved; no access/action request follows.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessCompletedAbsenceAndEarlyAmbiguityStopBeforeSecondStage`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessFallbackFindUsesNoneThenReturnedIDWithoutVisibilityQuery`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Construct the queryless second route only from the actual returned canonical wire id. Missing, null, nonstring, empty or unsafe id is a local ErrInvalidOption with Resolved retained and no invented ResponseError; no input-ID seeding, coercion, refetch or Identity lookup occurs.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessRejectsInvalidCanonicalResponseIDLocallyAfterLookup`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Post one queryless /types/{returnedID}/action body with the exact addProjectAccess nested project string. ProjectID is literal, including empty, nonexistent, control, Unicode, slash, query and percent characters; no project existence or Identity resolution. InvalidUTF8 project is a local error after Type lookup and before POST.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessInvalidUTF8ProjectIsLocalAfterResolvedType`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Admit every final valid HTTP100..399 under the direct source exception policy, independently from ordinary lookup200. Accepted action bodies are opaque, including empty or nonJSON/nonUTF8 bytes; no JSON decode or membership refresh occurs. Rejected>=400 is a native terminal failure with its own evidence.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessAdmitsSub400StatusWhileAccessJSONRemainsRequired`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`.
- Direct calls capture selected Cinder facts and canonical header copies before original options. Connection prepares originals once before recorded location and cached Cinder selection. Owned option/location snapshots and the same reader persist through both phases; source drift is sticky and blocks subsequent work.
  Tests: `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalOptionsOwnLocationAndSliceAcrossBothPhysicalStages`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalSourceViolationIsTerminalBeforeRestoringOption`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessPreflightContextRoleAndOptionsStopBeforeLookup`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessMalformedCallerLocationKeepsLookupProofWithoutPhysicalError`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOwnsLocationFactoriesAndOriginalCallbackSlice`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessRecordedScopeSnapshotsBetweenStagesAndRefreshesNextCall`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessPreflightAndGetterFailuresKeepOuterOperationAndCause`.
- Physical lookup and second-phase evidence remain separate. Admitted body/header/status survive read, Close, cancellation and post-response source errors; native rejected responses preserve their own native evidence. Wrappers retain errors.Is/As, ctx.Err and context.Cause without replaying the SDK workflow.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessAcceptedReadCloseCancellationRetainsCorrectStageAndAllCauses`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessSourceFactsRemainFrozenAfterLookupAndAcceptedSecondResponse`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessBlockedSecondStageCancellationKeepsLookupWithoutInventedPhaseProof`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection`.
- Native authentication and retry preserve live token behavior and fixed method/URL/body/response ownership. Retry changes to body/decoder/KeepResponseBody are rejected before altered resend. Ordinary native header and OkCodes updates remain allowed, while an actual status>=400 cannot become SDK success outside the original owned policy.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeReauthenticationAndBoundedRetryKeepPayloadTargetAndLiveHeaders`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeRetryCannotReplaceOwnedBodiesDecoderOrBroadenOriginalStatus`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessRedirectCannotChangeFixedSecondStageTargetOrMethod`.
- Return the actual accepted Applied acknowledgement proof without claiming membership change, completion or idempotency. Applied remains independently owned from Resolved; no verification, wait, rollback or SDK action replay occurs, including after accepted response I/O failures.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection`.

Python/Go differences:

- Only this actual cloud helper is mapped to Cinder v3 under Gophercloud v2.15.0. Independent native/Proxy/v2 declarations, inherited Resource descriptors/session negotiation/cache and other Type workflows are not promoted by this row.
- Python consumes a mutable returned Type and utils.urljoin coercion of its id. Go uses a fresh canonical response id that must be a safe nonempty UTF8 string URL segment; input identity is never inherited or used as a fallback route. Local target validation retains Resolved without borrowing HTTP200.
- Lookup reuses the existing Go exact identity, six-descriptor view, owned location and canonical accepted200 reader policy. Its safe input routing, complete-URL pagination safety and fresh response-field rules are declared mappings; this access row does not claim full Resource runtime equivalence.
- The direct access/action status gate preserves valid HTTP 100..399 rather than narrowing to native GET200/204/300 or action202. Fixed request origin, escaped path, exact query and method guards can block source-session redirect behavior before another exchange.
- Go exposes independently owned physical response evidence and joined native/read/Close/context causes; Python raises its own SDK exception classes and returns its ordinary helper value. Accepted evidence is not proof of subsequent server completion.
- Original caller options execute once from owned snapshots. Direct calls capture sources before originals; Connection runs originals before recorded location and service selection. Live native authentication/retry remains distinct from SDK-level replay, which is absent.
- The actual cloud project_id parameter maps to a literal Go string, including empty/nonexistent values. Independent Proxy Project Resource overloads are outside this row; Go rejects invalidUTF8 project locally after lookup to avoid JSON serializer replacement.
- Python returns None after an accepted response and does not parse its body. Go adds Applied as opaque acknowledgement evidence; it does not expose a Changed/completion boolean or infer membership, verify, wait, rollback or retry the workflow.

### `python:cloud/_block_storage/BlockStorageCloudMixin/remove_volume_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.RemoveVolumeTypeAccess`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RemoveVolumeTypeAccess`

- Resolve the ordinary unfiltered exact Type once through the captured reader. Safe identities use member GET with lowercase is_public=none; clean compatible member400/403/404 can fall back to exact list without a name hint. Lookup errors and completed absence retain Resolved; no access/action request follows.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessCompletedAbsenceAndEarlyAmbiguityStopBeforeSecondStage`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessFallbackFindUsesNoneThenReturnedIDWithoutVisibilityQuery`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Construct the queryless second route only from the actual returned canonical wire id. Missing, null, nonstring, empty or unsafe id is a local ErrInvalidOption with Resolved retained and no invented ResponseError; no input-ID seeding, coercion, refetch or Identity lookup occurs.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessRejectsInvalidCanonicalResponseIDLocallyAfterLookup`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Post one queryless /types/{returnedID}/action body with the exact removeProjectAccess nested project string. ProjectID is literal, including empty, nonexistent, control, Unicode, slash, query and percent characters; no project existence or Identity resolution. InvalidUTF8 project is a local error after Type lookup and before POST.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessInvalidUTF8ProjectIsLocalAfterResolvedType`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage`.
- Admit every final valid HTTP100..399 under the direct source exception policy, independently from ordinary lookup200. Accepted action bodies are opaque, including empty or nonJSON/nonUTF8 bytes; no JSON decode or membership refresh occurs. Rejected>=400 is a native terminal failure with its own evidence.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessAdmitsSub400StatusWhileAccessJSONRemainsRequired`, `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`.
- Direct calls capture selected Cinder facts and canonical header copies before original options. Connection prepares originals once before recorded location and cached Cinder selection. Owned option/location snapshots and the same reader persist through both phases; source drift is sticky and blocks subsequent work.
  Tests: `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalOptionsOwnLocationAndSliceAcrossBothPhysicalStages`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessOriginalSourceViolationIsTerminalBeforeRestoringOption`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessPreflightContextRoleAndOptionsStopBeforeLookup`, `blockstorage/volume_type_access_options_test.go:TestVolumeTypeAccessMalformedCallerLocationKeepsLookupProofWithoutPhysicalError`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOwnsLocationFactoriesAndOriginalCallbackSlice`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessRecordedScopeSnapshotsBetweenStagesAndRefreshesNextCall`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessPreflightAndGetterFailuresKeepOuterOperationAndCause`.
- Physical lookup and second-phase evidence remain separate. Admitted body/header/status survive read, Close, cancellation and post-response source errors; native rejected responses preserve their own native evidence. Wrappers retain errors.Is/As, ctx.Err and context.Cause without replaying the SDK workflow.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessAcceptedReadCloseCancellationRetainsCorrectStageAndAllCauses`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessCurrentNativeRejectionNeverBorrowsSuppressedMemberOrLookupProof`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessSourceFactsRemainFrozenAfterLookupAndAcceptedSecondResponse`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessBlockedSecondStageCancellationKeepsLookupWithoutInventedPhaseProof`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection`.
- Native authentication and retry preserve live token behavior and fixed method/URL/body/response ownership. Retry changes to body/decoder/KeepResponseBody are rejected before altered resend. Ordinary native header and OkCodes updates remain allowed, while an actual status>=400 cannot become SDK success outside the original owned policy.
  Tests: `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeReauthenticationAndBoundedRetryKeepPayloadTargetAndLiveHeaders`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessNativeRetryCannotReplaceOwnedBodiesDecoderOrBroadenOriginalStatus`, `blockstorage/volume_type_access_transport_test.go:TestVolumeTypeAccessRedirectCannotChangeFixedSecondStageTargetOrMethod`.
- Return the actual accepted Applied acknowledgement proof without claiming membership change, completion or idempotency. Applied remains independently owned from Resolved; no verification, wait, rollback or SDK action replay occurs, including after accepted response I/O failures.
  Tests: `blockstorage/volume_type_access_contract_test.go:TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements`, `connectiontest/connection_volume_type_access_test.go:TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection`.

Python/Go differences:

- Only this actual cloud helper is mapped to Cinder v3 under Gophercloud v2.15.0. Independent native/Proxy/v2 declarations, inherited Resource descriptors/session negotiation/cache and other Type workflows are not promoted by this row.
- Python consumes a mutable returned Type and utils.urljoin coercion of its id. Go uses a fresh canonical response id that must be a safe nonempty UTF8 string URL segment; input identity is never inherited or used as a fallback route. Local target validation retains Resolved without borrowing HTTP200.
- Lookup reuses the existing Go exact identity, six-descriptor view, owned location and canonical accepted200 reader policy. Its safe input routing, complete-URL pagination safety and fresh response-field rules are declared mappings; this access row does not claim full Resource runtime equivalence.
- The direct access/action status gate preserves valid HTTP 100..399 rather than narrowing to native GET200/204/300 or action202. Fixed request origin, escaped path, exact query and method guards can block source-session redirect behavior before another exchange.
- Go exposes independently owned physical response evidence and joined native/read/Close/context causes; Python raises its own SDK exception classes and returns its ordinary helper value. Accepted evidence is not proof of subsequent server completion.
- Original caller options execute once from owned snapshots. Direct calls capture sources before originals; Connection runs originals before recorded location and service selection. Live native authentication/retry remains distinct from SDK-level replay, which is absent.
- The actual cloud project_id parameter maps to a literal Go string, including empty/nonexistent values. Independent Proxy Project Resource overloads are outside this row; Go rejects invalidUTF8 project locally after lookup to avoid JSON serializer replacement.
- Python returns None after an accepted response and does not parse its body. Go adds Applied as opaque acknowledgement evidence; it does not expose a Changed/completion boolean or infer membership, verify, wait, rollback or retry the workflow.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func InspectAccess(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.GetVolumeTypeAccess(ctx,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
    if result != nil {
        if result.Resolved != nil {
            fmt.Println("lookup pages:", len(result.Resolved.Pages))
            if result.Resolved.Observed != nil {
                fmt.Println("lookup status:", result.Resolved.Observed.StatusCode)
            }
        }
        if result.Observed != nil {
            fmt.Println("access status:", result.Observed.StatusCode)
        }
    }
    if err != nil { return err }
    fmt.Println("actual type ID:", result.TypeID)
    fmt.Println("raw access JSON:", string(result.Value))
    // Accesses exists only when the entire value is an array of objects.
    for _, row := range result.Accesses {
        owned := row.Clone()
        fmt.Println("original project_id:", string(owned.Body["project_id"]))
        // This optional native projection enforces two string fields.
        // A projection error does not change the successful raw access result.
        var projection volumetypes.VolumeTypeAccess
        if err := owned.Decode(&projection); err != nil { return err }
        fmt.Println("project/type:", projection.ProjectID, projection.VolumeTypeID)
    }
    return nil
}

func ChangeAccess(ctx context.Context, conn *sdk.Connection, projectID string) error {
    input := blockstorage.VolumeTypeAccessRequest{NameOrID: "fast", ProjectID: projectID}
    added, err := conn.AddVolumeTypeAccess(ctx, input)
    if added != nil {
        fmt.Println("resolved target:", added.TypeID)
        if added.Applied != nil {
            fmt.Println("add acknowledgement:", added.Applied.StatusCode,
                "raw bytes:", len(added.Applied.Body))
        }
    }
    if err != nil { return err }
    removed, err := conn.RemoveVolumeTypeAccess(ctx, input)
    if removed != nil && removed.Applied != nil {
        fmt.Println("remove acknowledgement:", removed.Applied.StatusCode)
    }
    return err
}

func AccessWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.GetVolumeTypeAccess(ctx,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"},
        blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
    if err != nil { return err }
    // Location belongs to the resolved Type view, not to access rows.
    fmt.Println("resolved Type JSON:", string(result.Resolved.Value))
    return nil
}

func DirectAccess(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.GetVolumeTypeAccess(ctx, cinder,
        blockstorage.GetVolumeTypeAccessRequest{NameOrID: "fast"})
    if err != nil { return err }
    fmt.Println("raw access JSON:", string(result.Value))
    return nil
}
```

```python
access = conn.get_volume_type_access("fast")
conn.add_volume_type_access("fast", "project-id")
conn.remove_volume_type_access("fast", "project-id")
```

## References

- [Volume Types](volume-types.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L949)
- [pinned Type.get_private_access](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L119)
- [Type](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py#L136)
- [Exceptions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/exceptions.py#L214)
- [native pager status](https://github.com/gophercloud/gophercloud/blob/v2.15.0/pagination/http.go#L59)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/results.go#L159)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go#L287)
