# Volume Limits

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeLimits`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeLimits`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareGetVolumeLimitsOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithGetVolumeLimitsOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithGetVolumeLimitsLocation`

- Empty NameOrID skips Identity entirely, including unused invalid clients, and makes one bodyless unfiltered Cinder limits GET. Authenticated scope is captured only for location; it never becomes an implicit project_id query.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsDefaultSkipsIdentityAndDoesNotInferScopeQuery`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsEmptyInputSkipsIdentityAndImplicitProjectFilter`.
- Every nonempty literal string, including an ID-looking or whitespace-only input, resolves the project before limits. No trim or inferred domain query occurs. Safe names/IDs try a member GET; unsafe path/space names use a name-hinted exact list as the documented Go routing boundary. Connection completes cached Identity v3 resolution before choosing cached Cinder v3.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsTruthyInputResolvesIdentityBeforeLimitsWithoutTrimming`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOriginalsPrecedeIdentityResolutionAndCachedCinder`.
- Only clean direct native member400/403/404 can fall back. Accepted project decode/read/schema failures, wrapped retry-policy failures and other rejected statuses are terminal; no Cinder request follows. Rejected fallback member evidence is not relabeled as accepted Observed.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsAcceptedProjectSchemaErrorsAreTerminalBeforeFallbackOrLimits`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsNativeRetryProtectsBodyDecoderRetentionAndOriginalStatusPolicy`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsStrictProjectErrorsNeverSelectCinder`.
- Actual project id remains arbitrary JSON, including explicit null/false/zero/empty/containers. Missing id after an accepted member, including tolerated malformed/empty JSON, preserves the source input seed in ID and marks SeededID without adding a raw server field. Missing list id is literal null with no seed. RequestedProjectID is an independent owned query snapshot.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsMemberSeedIsExplicitAndNeverFabricatesRawProjectID`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsRawProjectIDsUseRequestsTwoLevelQueryExpansion`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsResultProofProjectAndNormalizedValueAreIndependent`.
- Project collection requires the canonical projects key and consumes ordinary object rows, wrapping a nonlist payload once. String id/name equality is exact. A second match raises ambiguity immediately before unused invalid rows/links; exhausted absence is ErrNotFound, and an empty page stops before unused continuation. No limits stage is published after failed find.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsProjectAmbiguityStopsBeforeUnusedInvalidRowAndLink`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsProjectCompletedAbsenceStopsBeforeLimitsAndUnusedEmptyPageLink`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsAcceptedProjectSchemaErrorsAreTerminalBeforeFallbackOrLimits`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsStrictProjectErrorsNeverSelectCinder`.
- Source Project list construction ignores raw self but rejects key-presence collisions with connection/microversion/_synchronized even for null/false and even in a nonmatching consumed row. Member response translation accepts the same ordinary fields. Each accepted failure retains only its actual current project page proof.
  Tests: `blockstorage/volume_limits_source_edges_test.go:TestGetVolumeLimitsProjectListConstructorCollisionsDifferFromMemberResponseFields`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging`.
- Project paging consumes top links by key presence before plural links, rel/href items lazily until a first exact next with present href, then falsey selected href falls through to body next/HTTP Link. Consumed invalid prefix fails; selected href leaves unused invalid tail unconsumed. Keystone links.next dictionary priority and RFC Link support are explicit Go compatibility repairs.
  Tests: `blockstorage/volume_limits_source_edges_test.go:TestGetVolumeLimitsProjectLinksAreConsumedInOrderAndStopAtSelectedHref`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsProjectCompletedAbsenceStopsBeforeLimitsAndUnusedEmptyPageLink`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging`.
- Continuation preserves the source pop of old marker/limit and drop of advertised blank values, merges remaining advertised keys once while retaining omitted nonpagination fields, and adds no page limit or implicit marker request. Relative paging remains on the same origin/escaped collection path; canonical complete-URL cycles fail before duplicate physical requests and retain the actual admitted page.
  Tests: `blockstorage/volume_limits_source_edges_test.go:TestGetVolumeLimitsProjectPagingRemovesPriorMarkerLimitAndDropsAdvertisedBlankValues`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsProjectPagingRejectsUnsafeContinuationWithoutBorrowedProof`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging`.
- Forward the completed raw project id as query data without action-path validation. Preserve Requests JSON-domain two-level iterable expansion: outer null omission, object-key iteration, nested list/object expansion, nested None spelling and deeper ordered Python repr. Literal strings/bools/arbitrary integers and parsed-float representation are covered; invalid documents return no partial query values.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsRawProjectIDsUseRequestsTwoLevelQueryExpansion`, `internal/cloudfilter/query_test.go:TestRequestQueryValuesMatchesRequestsTwoLevelEncoding`, `internal/cloudfilter/query_test.go:TestRequestQueryValuesKeepsObjectOrderAndUsesReprOnlyAfterDoseqExpansion`, `internal/cloudfilter/query_test.go:TestRequestQueryValuesRetainsScalarJSONNumberAndStringDomain`, `internal/cloudfilter/query_test.go:TestObjectMembersAndRequestQueryValuesRejectInvalidDocumentsAtomically`.
- Factories and PrepareGetVolumeLimitsOptions own complete pointer/raw Location policy and are reusable. Original callbacks run once in the captured order from an owned option slice, with snapshots after each callback; caller and retained callback mutations cannot alter later policy or result. Preparation itself performs no HTTP.
  Tests: `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsFactoriesAndPreparedOptionsOwnCompleteLocation`, `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsOriginalsRunOnceInCapturedOrderAndRetainedConfigIsIsolated`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOwnsFactoriesOriginalSliceAndRetainedLocation`.
- Direct calls capture required Cinder/Identity provider, endpoint, ResourceBase, role and microversion before original callbacks; observed source violation is sticky and prevents a restoring callback. Ordinary headers stay captured while auth tokens remain live. Context/input/role/reserved-header preflight and invalid originals stop before HTTP and retain the explicit outer GetVolumeLimits operation and causes.
  Tests: `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsSelectedSourcesAreCapturedBeforeOriginalsAndViolationIsTerminal`, `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsPreflightCancellationAndInvalidSelectionsNeverSendHTTP`, `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsHeaderSnapshotsStayOwnedWhileProviderTokensRemainLive`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsPreflightAndGetterErrorsKeepOuterOperationAndCause`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedRejectsInitialSourceBeforeMarshalOrHTTP`.
- Connection prepares originals once before service getters, snapshots explicit/current location after originals before project lookup, and uses only required cached Identity/Cinder v3. Completed project and requested-ID proof survive Cinder getter error/custom cancellation without borrowed ResponseError. A later recorded authentication scope changes the next call, while this call keeps its own snapshot and live token.
  Tests: `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOriginalsPrecedeIdentityResolutionAndCachedCinder`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsProjectCompletesBeforeCinderGetterFailure`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsCapturesRecordedScopeAfterOptionsBeforeProjectResolution`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOwnsFactoriesOriginalSliceAndRetainedLocation`.
- Preserve the already-selected Cinder microversion and consistent native version headers, including empty/3.0/3.38/3.39/3.60/symbolic latest, without auto-upgrade, project3.39 preflight or cap. Connection exact-version options remain numeric; an already-selected native symbolic version is preserved. Old singleton scope version policy remains independent.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsPreservesSelectedMicroversionWithoutProjectUpgradeOrCap`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsEmptyInputSkipsIdentityAndImplicitProjectFilter`.
- Final source GET admits valid HTTP100..399 and reads exactly once with raw owned query. Canonical limits object or flat object succeeds. Source JSON-ValueError tolerance accepts UTF8-valid empty/malformed JSON as bare nullable Limits, including physical204/304; valid wrong shapes/envelope/null, invalidUTF8 or descriptor failures reject atomically with current Observed proof.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsAcceptedFlatAndMalformedJSONProduceOwnedNullableResource`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsInvalidAcceptedShapesAndDescriptorsFailAtomicallyWithCurrentProof`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsSeparatesLocalRejectedAndAcceptedFinalProof`.
- Normalized Value exposes the complete actual5 top/13 absolute/6 rate-group/8 rule nullable descriptor schema. Absent/null stays null; actual nested object{} yields complete nullable fields, but nonobject Resource values and null/scalar/list list rows become plain{}. Nonlist list descriptors wrap once. Untyped ID/name/next_available/unit/verb/regex/URI retain arbitrary JSON; unknown fields stay only in raw evidence.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsNestedDescriptorsPreserveNullableShapesAndUntypedPrecision`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsAcceptedFlatAndMalformedJSONProduceOwnedNullableResource`.
- Integer descriptors preserve Python bool and arbitrary precision integer, truncate the parsed finite IEEE float, accept unsigned Unicode16 decimal digit strings, map ordinary signed/space/container inputs to zero, and reject nondecimal digit or nonfinite float conversions. The shared audited descriptor helper is reused by the actual Volume size path; legacy strict IntegerJSON is unchanged.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsNestedDescriptorsPreserveNullableShapesAndUntypedPrecision`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsInvalidAcceptedShapesAndDescriptorsFailAtomicallyWithCurrentProof`, `blockstorage/volume_search_view_test.go:TestVolumeSearchViewDescriptorConversionsAndSourceSize`, `blockstorage/volume_search_view_test.go:TestVolumeSearchViewEagerErrorsAndDiscardedAlias`.
- Normalized/wire aliases follow parsed Python dictionary order after duplicate-key collapse preserves first insertion position with final value. Last consumed alias wins rather than last textual member; selected discarded aliases are not converted. Every parsed raw value owns its bytes independently.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsAliasTraversalUsesCollapsedDictionaryInsertionOrder`, `internal/cloudfilter/query_test.go:TestObjectMembersKeepsDuplicateInsertionPositionAndOwnsEveryRawValue`.
- Top location keeps complete current/authenticated or explicit owned location, including Zone, and never infers foreign query or HTTP project/location/zone. Nested constructor location is null unless raw location survives with no recognized Body field, including recognized null. Nested self and truthy JSON connection fail; falsey connection and synchronized/microversion runtime values are consumed without becoming normalized fields. Invalid caller location after project is local with Project retained and no final response borrowed.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsNestedLocationPresenceAndTopLocationHaveDifferentOwnershipRules`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsFalseyNestedConstructorControlsDoNotBecomeBodyFields`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsInvalidAcceptedShapesAndDescriptorsFailAtomicallyWithCurrentProof`, `blockstorage/volume_limits_options_test.go:TestGetVolumeLimitsInvalidOwnedLocationFailsLocallyAfterProjectWithoutFinalProof`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOwnsFactoriesOriginalSliceAndRetainedLocation`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsCapturesRecordedScopeAfterOptionsBeforeProjectResolution`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsSeparatesLocalRejectedAndAcceptedFinalProof`.
- Project raw fields/ID/requested ID, actual project member/pages, final Observed, raw Limits and normalized Value own independent raw bytes/headers/status. Logical Limits/Value are published together only after complete normalization and guard checks; accepted errors keep physical proof without partially publishing logical outputs.
  Tests: `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsResultProofProjectAndNormalizedValueAreIndependent`, `blockstorage/volume_limits_contract_test.go:TestGetVolumeLimitsInvalidAcceptedShapesAndDescriptorsFailAtomicallyWithCurrentProof`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsOriginalsPrecedeIdentityResolutionAndCachedCinder`.
- Accepted member/list/final read, Close, source drift and custom context cancellation retain the correct current physical response and join all causes without SDK replay. Native rejected/transport/blocked-HTTP/getter/local errors do not borrow earlier accepted status or a suppressed fallback member. Error wrappers preserve errors.Is/As and the GetVolumeLimits operation.
  Tests: `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsAcceptedReadCloseCancellationOwnsCurrentStageAndJoinsEveryCause`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsNativeRejectionAndTransportErrorsNeverBorrowEarlierProjectProof`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsSelectedSourcesRemainFrozenAcrossBothAcceptedStages`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsBlockedHTTPStageCancellationRetainsOnlyCompletedProject`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsProjectCompletesBeforeCinderGetterFailure`, `connection_volume_limits_test.go:TestConnectionGetVolumeLimitsSeparatesLocalRejectedAndAcceptedFinalProof`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedAcceptedEvidenceJoinsReadCloseSourceAndContextWithoutReplay`.
- Native authentication/retry remains live while exact method/target/query/bodyless decoder/retention and original accepted status policy remain owned. Source change during retry/reauth/backoff/shared-auth wait prevents physical resend, while permitted retry retains captured headers/live token. Native status expansion cannot turn actual>=400 into success or compatible fallback. Redirect target changes fail before credentials reach a changed route; no SDK wait/action/rollback/replay is introduced.
  Tests: `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsBothStagesHonorNativeReauthAndBoundedRetryWithLiveTokens`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsNativeRetryProtectsBodyDecoderRetentionAndOriginalStatusPolicy`, `blockstorage/volume_limits_transport_test.go:TestGetVolumeLimitsRedirectCannotChangeFixedStageTargetOrForwardCredentials`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedStopsSourceChangingNativeRetryBeforeResend`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedStopsSourceChangingNativeReauthenticationBeforeResend`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedChecksSourceAfterWaitingForLiveAuthentication`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedExpandedNativeStatusKeepsActualRejectionAndSourceCause`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedKeepsLiveAuthenticationAndOwnedRequestOnAllowedRetry`, `internal/rest/guarded_response_test.go:TestNewGuardedStopsSourceChangingNativeBackoffBeforePhysicalResend`.

Python/Go differences:

- Only the actual direct cloud get_volume_limits helper is mapped using selected Cinder/Identity v3 and Gophercloud v2.15.0. Independent Proxy/native/v2 declarations and generic Resource/session/adapter/cache responsibilities keep their separate review statuses; this row does not complete the full SDK.
- Python nullable name_or_id maps to the explicit Go string request: empty skips lookup, every nonempty value resolves even if ID-looking. Go exact safe routing avoids path-like/whitespace member interpolation, rejects input invalidUTF8/controls locally, and introduces no caller builder/getter responsibility.
- The source seeded Project member ID is retained explicitly as ID/SeededID when the response omits id, without inventing an actual raw response field. Explicit null/false/zero/container IDs and absent list ID stay raw; query identity is evidence of the requested filter, not server-confirmed limits ownership.
- Raw project query encoding records ordinary Requests two-level JSON iterable expansion and deeper Python repr. Requests/urllib and Unicode/float representation are external runtime dependencies not pinned by the SDK commit; Go audited float repr/Unicode16 and arbitrary integer conversion boundaries are explicit.
- Source project rel/href/top-next and marker/limit-pop/blank-drop behavior is retained. Keystone links.next dictionary support with priority over top next/HTTP Link and RFC Link url support repair actual pinned Resource bugs. Same-origin/escaped-path and complete-URL cycle guards, effective relative URL handling and canonical query merge are deliberate Go safety/transport mappings.
- The selected native microversion is preserved without project3.39 preflight, upgrade or cap. Pinned Limits._max_microversion=3.39 caps server-discovery negotiation only when session default_microversion is absent; explicit Python session default is uncapped. Go does not emulate automatic Resource server discovery. Existing singleton project scopes and their numeric3.39+ requirement are independent.
- Go adds independently owned physical Project/Observed/Pages, RequestedProjectID, raw Limits fields and normalized Value, with explicit context and joined error causes. Unknown raw fields and JSON number spelling are proof extensions; Python returns a mutable Resource and SDK exception classes.
- Normalized nested Limits/Absolute/Rate/RateLimit preserves the nullable schema and ordinary JSON descriptor/constructor behavior, including duplicate-collapse insertion ordering and reserved-key distinctions. Go does not reproduce mutable Resource object identity, dirty caches or arbitrary nonJSON Python objects; configurable CPython integer digit limits and Unicode runtime are explicitly distinct.
- Owned top location snapshots current configured/recorded authentication after original options and before lookup, with complete explicit override including Zone. Foreign query/raw response does not replace it. Direct ServiceClient defaults recover recorded scope without fabricating configured cloud/project names; Connection owns its existing CloudLocation model.
- Connection originals run once before snapshot/cached service selection, and project resolution completes before Cinder getter. Direct capture occurs before originals. Ordinary headers are copied, provider auth stays live, and sticky source/fixed request guards prevent altered retries/redirects. Native caller retry remains separate from SDK workflow replay, wait or rollback, which is absent.
- Valid HTTP100..399 acceptance and UTF8-valid empty/malformed JSON tolerance follow the direct source instead of narrowing to old native200 typed projection. Strict completeUTF8/object/envelope/descriptor checks and atomic logical publication retain actual physical errors; caller/getter/query failures do not borrow prior HTTP200.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    nativeLimits "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/limits"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Inspect(result *blockstorage.GetVolumeLimitsResult) {
    if result == nil { return }
    if result.Project != nil {
        fmt.Println("project ID JSON:", string(result.Project.ID),
            "seeded:", result.Project.SeededID,
            "lookup pages:", len(result.Project.Pages))
        if result.Project.Observed != nil {
            fmt.Println("project member status:", result.Project.Observed.StatusCode)
        }
    }
    fmt.Println("requested project ID JSON:", string(result.RequestedProjectID))
    if result.Observed != nil {
        fmt.Println("limits status:", result.Observed.StatusCode,
            "raw bytes:", len(result.Observed.Body))
    }
}

func Current(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.GetVolumeLimits(ctx, blockstorage.GetVolumeLimitsRequest{})
    Inspect(result) // Completed physical proof also remains available on error.
    if err != nil { return err }
    fmt.Println("normalized nullable JSON:", string(result.Value))
    owned := result.Limits.Clone()
    fmt.Println("original absolute JSON:", string(owned.Body["absolute"]))

    // This optional native projection enforces the native int/string schema.
    // A projection error does not invalidate the successful raw workflow.
    var projection nativeLimits.Limits
    if projectionErr := owned.Decode(&projection); projectionErr != nil {
        fmt.Println("native projection:", projectionErr)
    } else {
        fmt.Println("native maximum volumes:", projection.Absolute.MaxTotalVolumes)
    }
    return nil
}

func OtherProject(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: nameOrID})
    Inspect(result)
    if err != nil { return err }
    fmt.Println("normalized limits:", string(result.Value))
    return nil
}

func WithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "authenticated-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.GetVolumeLimits(ctx,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "other-project"},
        blockstorage.WithGetVolumeLimitsOptions(blockstorage.GetVolumeLimitsOpts{
            Location: &location,
        }))
    Inspect(result)
    if err != nil { return err }
    fmt.Println("limits with owned authenticated location:", string(result.Value))
    return nil
}

func Direct(ctx context.Context, cinder, identity *gophercloud.ServiceClient) error {
    // Empty input needs no Identity client and sends no project filter.
    current, err := blockstorage.GetVolumeLimits(ctx, cinder, nil,
        blockstorage.GetVolumeLimitsRequest{})
    Inspect(current)
    if err != nil { return err }
    selected, err := blockstorage.GetVolumeLimits(ctx, cinder, identity,
        blockstorage.GetVolumeLimitsRequest{NameOrID: "project-name-or-id"},
        blockstorage.WithGetVolumeLimitsLocation(resource.CloudLocation{}))
    Inspect(selected)
    return err
}
```

## References

- [pinned cloud get_volume_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L348)
- [pinned Cinder Proxy.get_limits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1918)
- [Resource paging](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2345)
- [Utils](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Models](https://requests.readthedocs.io/en/latest/_modules/requests/models/#RequestEncodingMixin._encode_params)
- [Urllib.Parse](https://docs.python.org/3/library/urllib.parse.html#urllib.parse.urlencode)
- [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [Cinder v3 limits singleton](v3/limits/README.md)
- [pinned Limits hierarchy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/limits.py)
- [pinned field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [Resource constructor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538)
