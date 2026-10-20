# Volume Snapshot Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/create_volume_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CreateVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCreateVolumeSnapshotOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotWaitPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotAttributes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotFields`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotDisplayName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotDescription`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotDisplayDescription`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeSnapshotLocation`

- Concrete With/Prepare policy preserves omitted versus explicit false/zero, complete replacement and all raw nullable attributes/location/wait subtrees. Original callbacks run exactly once in captured slice order with an owned snapshot after each callback; factories support concurrent reuse. Preparation is pure and defers reached-value validation. Nil options/context, cancellation including custom causes and callback errors return no partial policy or HTTP.
  Tests: `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreserveOmissionAndExplicitFalseZero`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCompleteFactoriesOwnEverySubtreeAcrossReuses`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsFactoriesSupportConcurrentIndependentPreparedValues`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsAttributeFactoriesRetainAllAliasesWithoutNormalizing`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsRawFactoriesPreserveNilEmptyMapsAndExactByteShapes`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsIndividualFactoriesOverrideOnlySelectedSubtrees`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCompleteReplacementClearsPriorPolicyAndSubtrees`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCaptureOriginalSliceAndIsolateRetainedCallbacks`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsRejectMissingCanceledOrExpiredContextBeforeCallbacks`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsStopOnNilErrorOrCancellationWithoutPartialPolicy`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreparationIsPureAndDefersReachedValueValidation`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and ordinary headers as owned snapshots while auth tokens remain live. Sticky source changes and original failures join context causes before HTTP. Connection prepares originals once before only cached Cinder v3 selection without readiness probing; recorded/explicit scope is owned after originals and before getter, retaining snapshot location across phases.
  Tests: `blockstorage/snapshot_mutation_source_test.go:TestVolumeSnapshotMutationOriginalFailuresJoinSourceAndContextBeforeHTTP`, `blockstorage/snapshot_mutation_source_test.go:TestVolumeSnapshotMutationDirectCaptureOwnsHeadersOptionsAndEveryPublishedStage`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsCaptureRecordedScopeAfterOriginalsBeforeGetterAcrossWait`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsExplicitLocationAndRetainedOriginalsStayOwned`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsNonemptyPreflightAndGetterFailuresKeepOperationCauses`.
- Snapshot logical state merges only known descriptors while preserving raw descriptor values, duplicate-key insertion/alias order and eager nullable16-field conversion. Location is recomputed from captured current/foreign-project scope and zone is null. Partial/flat responses merge prior state; valid UTF8 empty/malformed JSON has no actual object, parsed {} is an actual empty resource, and valid wrong shape/envelope/null or invalid UTF8 fails with current physical proof.
  Tests: `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateMergesKnownPartialFieldsWithoutChangingRawDescriptorValues`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateAliasesUseParsedDuplicateInsertionOrder`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateConversionIsEagerAndErrorsDoNotRewriteRawState`, `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationObjectDistinguishesToleranceFromActualEmptyAndRejectsWrongShape`, `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationObjectOwnsActualFieldsAndMetadataSeparatelyFromLogicalView`.
- Accepted physical body/header/status, raw resources, logical stages and errors own independent copies. Admitted read/Close/source/context failures keep the actual current phase and all causes while completed logical outputs remain unpublished. Later blocked/canceled/rejected requests keep prior phase proof without borrowing it as the new failure response. Opaque mutation acknowledgement bytes remain actual proof.
  Tests: `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationProofKeepsOpaqueBytesAndEveryPhaseOwnsItsCopy`, `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationAcceptedFailuresRetainCurrentPhaseAndCauses`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsDescriptorAndDeleteRejectionRetainTheirOwnPhase`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Create forwards literal volume_id including empty string with explicit force=false by default, without generated ID/name/description. Only name/display_name/description/display_description kwargs are accepted. Canonical key presence suppresses aliases even when falsey; selected falsey JSON is omitted, typed fields overlay the same key, and invalid unused aliases remain unused. Reached input validation occurs before POST and Connection getter; body data is not routed.
  Tests: `internal/cloudsnapshot/mutation_body_test.go:TestSnapshotMutationCreateBodyPreservesOnlyFourCloudAttributesAndDefaultFalse`, `internal/cloudsnapshot/mutation_body_test.go:TestSnapshotMutationCreateBodyCanonicalPresenceSuppressesAliasesByJSONTruthiness`, `internal/cloudsnapshot/mutation_body_test.go:TestSnapshotMutationCreateBodyTypedOverridesAndUnusedAliasValidation`, `internal/cloudsnapshot/mutation_body_test.go:TestSnapshotMutationCreateBodyInputValidationIsAtomicAndBodyDataDoesNotUseRoutes`, `internal/cloudsnapshot/mutation_body_test.go:TestSnapshotMutationCreateBodyOwnsSuppliedFieldsAndIndependentState`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotDefaultsForwardLiteralBodyAndPreserveRequestSeed`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotAliasesUseCanonicalPresenceAndLiteralJSONValues`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotValidatesConsumedInputBeforePOSTButIgnoresShadowedAlias`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotOriginalOptionsAndFactoriesCannotChangeCapturedRequestAfterBoundary`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationCreateInvalidLiteralAndUnknownAttributesFailBeforeGetter`.
- POST accepts original HTTP100..399 and merges the submitted known seed with actual response fields. Created physical proof and successfully normalized CreatedValue remain separate from final Value/Ready; malformed responses retain submitted values without manufacturing a raw object. Eager descriptor/shape failure keeps parsed current raw resource when available and never publishes final Snapshot/Value.
  Tests: `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotMergesPartialFlatEmptyAndMalformedCreationResponses`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotAcceptedWrongShapesAndDescriptorsKeepOnlyCreationProof`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotOwnsEveryLogicalAndPhysicalPhaseIndependently`.
- Create defaults wait=true. Initial case-insensitive available is terminal before unused ID/timeout/interval validation; no-wait returns merged values even arbitrary/null IDs or statuses. Waiting consumes initial nonstring status locally; initial error never shortcuts the fresh GET. Fresh available succeeds before fresh case-insensitive error failure, while null status is nonterminal. Every poll merges prior fields and uses the latest logical ID only when another safe member request is reached, so terminal unsafe IDs need no route.
  Tests: `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotCachedReadySkipsUnusedWaitAndRouteControls`, `blockstorage/snapshot_mutation_create_contract_test.go:TestCreateVolumeSnapshotNoWaitKeepsArbitraryNullableIdentityAndStatus`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotFreshPollOverridesInitialErrorBeforeFailure`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotFreshErrorOrInvalidStatusKeepsCreatedAndLastAccepted`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotInitialNonstringStatusIsConsumedOnlyWhenWaiting`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotWaitRebindsLatestLogicalIDButDoesNotRouteTerminalUnsafeID`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotNonterminalUnsafeIDFailsBeforeAnotherPhysicalPoll`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotToleratedPollBodiesMergePriorStateBeforeTerminalCheck`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateRawIDsAndNullableStatusesValidateOnlyAtReachedPhases`.
- Create SDK wait budget starts after POST/initial readiness check. Nil is unlimited, explicit nonpositive timeout fails before polling, and positive expiry is checked at loop boundaries without deriving an HTTP context deadline; a slow target GET succeeds. Interval defaults2s, zero normalizes100ms or a smaller positive timeout, and negative interval fails only at reached nonterminal sleep. Parent context cancellation still governs transport/sleep.
  Tests: `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotExplicitNonpositiveTimeoutIsPostCreateAndPrePoll`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotPositiveWaitBudgetBeginsAfterCreationAndDoesNotAbortSlowTargetGET`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotPositiveTimeoutIsNoticedAtNextLoopBoundary`, `blockstorage/snapshot_mutation_wait_contract_test.go:TestCreateVolumeSnapshotPollIntervalsAreConsumedOnlyAtNonterminalSleep`.
- Native retry cannot replace owned POST body, fixed route, method, decoder, retention, original status policy or captured source. Native auth is live but orchestration does not independently replay create or roll back accepted resources.
  Tests: `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationCreateRetryCannotReplaceOwnedBodyOrDecoder`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin cloud helper is mapped with selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter declarations retain separate reviews; full SDK remains incomplete.
- Python defaults/kwargs and mutable runtime objects map to explicit Go request types and library-owned concrete With/Prepare options. Go controls use finite JSON, typed pointers and time.Duration rather than Python runtime objects or arbitrary Resource identity.
- Owned raw HTTP/resource proof and normalized nullable logical stages are separate Go representations. Unknown raw fields and number spelling remain physical evidence; source stale dict cache, mutable Resource identity and session memoization are not reproduced.
- Ordinary eager descriptor and parsed dictionary duplicate/alias semantics are preserved. Exact decimal truthiness/equality, Unicode16 decimal tables and arbitrary integers do not reproduce Python float rounding/underflow or configurable CPython integer digit limits.
- Go requires complete valid UTF8 for consumed JSON and safe unescaped UTF8 member route segments; arbitrary runtime/route-changing coercion is excluded. Request VolumeID is literal body data and does not require route validation. Only reached fields/status/route/wait controls are validated.
- Direct source capture, owned ordinary headers, sticky fixed-request/source/cause guards and atomic logical publication are Go safety/proof extensions. Native token/reauth/retry stays live; SDK orchestration adds no replay, rollback or cache invalidation.
- Configured/authenticated location is captured and recomputed from project descriptors with null snapshot zone; raw location/availability_zone cannot replace it. Explicit selected Cinder microversion is preserved without automatic3.65 discovery/cap/upgrade or v2 selection.
- Wait timeout is a nullable Go duration checked at source loop boundaries without an HTTP-derived context deadline. Parent contexts supply transport/sleep cancellation. Configurable poll interval is a Go extension; default2s and source zero100ms/smaller-timeout behavior are preserved.
- Source creation default force=false/wait=true, four kwargs, merged request seed and initial available shortcut are preserved. No automatic native projection, generated names or metadata kwargs are added.

### `python:cloud/_block_storage/BlockStorageCloudMixin/delete_volume_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DeleteVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareDeleteVolumeSnapshotOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeSnapshotOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeSnapshotWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeSnapshotWaitPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeSnapshotLocation`

- Concrete With/Prepare policy preserves omitted versus explicit false/zero, complete replacement and all raw nullable attributes/location/wait subtrees. Original callbacks run exactly once in captured slice order with an owned snapshot after each callback; factories support concurrent reuse. Preparation is pure and defers reached-value validation. Nil options/context, cancellation including custom causes and callback errors return no partial policy or HTTP.
  Tests: `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreserveOmissionAndExplicitFalseZero`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCompleteFactoriesOwnEverySubtreeAcrossReuses`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsFactoriesSupportConcurrentIndependentPreparedValues`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsAttributeFactoriesRetainAllAliasesWithoutNormalizing`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsRawFactoriesPreserveNilEmptyMapsAndExactByteShapes`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsIndividualFactoriesOverrideOnlySelectedSubtrees`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCompleteReplacementClearsPriorPolicyAndSubtrees`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsCaptureOriginalSliceAndIsolateRetainedCallbacks`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsRejectMissingCanceledOrExpiredContextBeforeCallbacks`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsStopOnNilErrorOrCancellationWithoutPartialPolicy`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreparationIsPureAndDefersReachedValueValidation`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and ordinary headers as owned snapshots while auth tokens remain live. Sticky source changes and original failures join context causes before HTTP. Connection prepares originals once before only cached Cinder v3 selection without readiness probing; recorded/explicit scope is owned after originals and before getter, retaining snapshot location across phases.
  Tests: `blockstorage/snapshot_mutation_source_test.go:TestVolumeSnapshotMutationOriginalFailuresJoinSourceAndContextBeforeHTTP`, `blockstorage/snapshot_mutation_source_test.go:TestVolumeSnapshotMutationDirectCaptureOwnsHeadersOptionsAndEveryPublishedStage`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsCaptureRecordedScopeAfterOriginalsBeforeGetterAcrossWait`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsExplicitLocationAndRetainedOriginalsStayOwned`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsNonemptyPreflightAndGetterFailuresKeepOperationCauses`.
- Snapshot logical state merges only known descriptors while preserving raw descriptor values, duplicate-key insertion/alias order and eager nullable16-field conversion. Location is recomputed from captured current/foreign-project scope and zone is null. Partial/flat responses merge prior state; valid UTF8 empty/malformed JSON has no actual object, parsed {} is an actual empty resource, and valid wrong shape/envelope/null or invalid UTF8 fails with current physical proof.
  Tests: `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateMergesKnownPartialFieldsWithoutChangingRawDescriptorValues`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateAliasesUseParsedDuplicateInsertionOrder`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateConversionIsEagerAndErrorsDoNotRewriteRawState`, `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationObjectDistinguishesToleranceFromActualEmptyAndRejectsWrongShape`, `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationObjectOwnsActualFieldsAndMetadataSeparatelyFromLogicalView`.
- Accepted physical body/header/status, raw resources, logical stages and errors own independent copies. Admitted read/Close/source/context failures keep the actual current phase and all causes while completed logical outputs remain unpublished. Later blocked/canceled/rejected requests keep prior phase proof without borrowing it as the new failure response. Opaque mutation acknowledgement bytes remain actual proof.
  Tests: `internal/cloudsnapshot/mutation_response_test.go:TestSnapshotMutationProofKeepsOpaqueBytesAndEveryPhaseOwnsItsCopy`, `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationAcceptedFailuresRetainCurrentPhaseAndCauses`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsDescriptorAndDeleteRejectionRetainTheirOwnPhase`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Delete defaults wait=false. Empty NameOrID runs original policy/context checks once then returns false without consuming service/location, including nil Connection/client. Nonempty uses exact member lookup with only clean original concrete400/403/404 fallback, preserves normal absence and ambiguity, and requires exhausted unique fallback before mutation. Actual selected logical ID, including the missing-member-ID seed, determines the safe DELETE route; explicit null cannot route and raw fields are not manufactured.
  Tests: `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotDefaultWaitOffDeletesActualResolvedIDWithOpaqueAcknowledgement`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotEmptyInputRunsOriginalsOnceWithoutConsumingServiceOrLocation`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotFallbackRequiresCompletedExactLookupAndPreservesAbsenceOrAmbiguity`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotMemberMissingIDSeedsLookupButExplicitNullCannotRoute`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationEmptyDeleteIsServiceFreeEvenForNilConnection`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateDeleteLogicalSeedDoesNotManufacturePhysicalFields`.
- DELETE is bodyless without force/cascade and original HTTP100..399 acknowledgement is opaque, including nonJSON or binary bytes. No-wait returns true only after acknowledgement and guards. Original DELETE404 is terminal with mutation evidence and can never be treated as completed polling absence.
  Tests: `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotDefaultWaitOffDeletesActualResolvedIDWithOpaqueAcknowledgement`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotMutation404IsTerminalAndNeverPollAbsence`, `connection_volume_snapshot_mutations_test.go:TestConnectionSnapshotMutationsDescriptorAndDeleteRejectionRetainTheirOwnPhase`.
- With wait=true, cached deleted alone cannot complete: after reached timeout/context/route guards, delete performs a fresh GET. Fresh case-insensitive deleted or a clean original concrete GET404 completes; error/error_deleting are nonterminal, fresh null/nonstring status is local failure, and omitted/empty/malformed fresh body merges prior state before terminal checks. The latest merged ID routes only a subsequent needed request. Deleted stays nil on error, false on initial absence/empty input, true only after completion.
  Tests: `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotWaitAlwaysFetchesFreshAndToleratesMalformedBodyByMergingPriorDeleted`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotFreshNullOrNonstringStatusIsAnErrorAfterAcknowledgement`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotErrorsAreNonterminalAndLatestMergedIDControlsNextPoll`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotCleanPolling404CompletesWithSeparateAbsenceProof`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateRawIDsAndNullableStatusesValidateOnlyAtReachedPhases`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateDeleteLogicalSeedDoesNotManufacturePhysicalFields`.
- Delete wait budget starts after acknowledgement, overrides the standalone Proxy120-second default with cloud nil-unlimited behavior, and explicit nonpositive timeout fails before even a cached-deleted fresh GET. Positive SDK expiry is loop-only and does not abort a slow terminal GET; parent contexts still cancel HTTP/sleep. Nil-unlimited routing and multi-poll behavior are runtime fixtures; Python120-second default contrast is pinned source evidence, not a120-second live Go run.
  Tests: `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotErrorsAreNonterminalAndLatestMergedIDControlsNextPoll`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotNonpositiveWaitTimeoutIsCheckedAfterDELETEBeforeEvenCachedDeletedGET`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotPositiveBudgetStartsAfterDELETEAndAcceptsSlowFreshDeleted`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreserveOmissionAndExplicitFalseZero`, `blockstorage/snapshot_mutation_options_test.go:TestSnapshotMutationOptionsPreparationIsPureAndDefersReachedValueValidation`.
- Polling404 needs completed clean body Read/Close and source/context guards. Gophercloud v2.15.0 ignores unexpected-code read/Close errors, so the private observer preserves original rejection policy and joins all faults, including joined EOF/custom Read and Close EOF, preventing false absence. Wrapped native errors or expanded >=400 policy never complete. Native404 retry-to200 and401 reauth remain live with fixed request guards; clean404 has separate Absent proof rather than LastAccepted proof.
  Tests: `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationDelete404RequiresCleanReadCloseAndGuard`, `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationDeletePollingPreservesLiveAuthAndNative404Retry`, `blockstorage/snapshot_mutation_delete_contract_test.go:TestDeleteVolumeSnapshotCleanPolling404CompletesWithSeparateAbsenceProof`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin cloud helper is mapped with selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter declarations retain separate reviews; full SDK remains incomplete.
- Python defaults/kwargs and mutable runtime objects map to explicit Go request types and library-owned concrete With/Prepare options. Go controls use finite JSON, typed pointers and time.Duration rather than Python runtime objects or arbitrary Resource identity.
- Owned raw HTTP/resource proof and normalized nullable logical stages are separate Go representations. Unknown raw fields and number spelling remain physical evidence; source stale dict cache, mutable Resource identity and session memoization are not reproduced.
- Ordinary eager descriptor and parsed dictionary duplicate/alias semantics are preserved. Exact decimal truthiness/equality, Unicode16 decimal tables and arbitrary integers do not reproduce Python float rounding/underflow or configurable CPython integer digit limits.
- Go requires complete valid UTF8 for consumed JSON and safe unescaped UTF8 member route segments; arbitrary runtime/route-changing coercion is excluded. Request VolumeID is literal body data and does not require route validation. Only reached fields/status/route/wait controls are validated.
- Direct source capture, owned ordinary headers, sticky fixed-request/source/cause guards and atomic logical publication are Go safety/proof extensions. Native token/reauth/retry stays live; SDK orchestration adds no replay, rollback or cache invalidation.
- Configured/authenticated location is captured and recomputed from project descriptors with null snapshot zone; raw location/availability_zone cannot replace it. Explicit selected Cinder microversion is preserved without automatic3.65 discovery/cap/upgrade or v2 selection.
- Wait timeout is a nullable Go duration checked at source loop boundaries without an HTTP-derived context deadline. Parent contexts supply transport/sleep cancellation. Configurable poll interval is a Go extension; default2s and source zero100ms/smaller-timeout behavior are preserved.
- Source deletion default wait=false and empty input false are preserved with nullable bool Deleted. Mutation404 stays terminal; only a clean original concrete polling404 completes. Extra body-fault observation compensates for native ignored rejection Read/Close errors without bypassing native retry.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectCreate(result *blockstorage.CreateVolumeSnapshotResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedSnapshot != nil {
        actual := result.CreatedSnapshot.Clone()
        fmt.Println("actual POST ID field:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeSnapshotResult) {
    if result == nil { return }
    if result.Deleted != nil { fmt.Println("requested deletion completed:", *result.Deleted) }
    if result.Resolved != nil {
        fmt.Println("lookup pages:", len(result.Resolved.Pages), "seeded ID:", result.Resolved.SeededID)
        if result.Resolved.Observed != nil {
            fmt.Println("lookup member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil { fmt.Println("DELETE acknowledged:", result.Applied.StatusCode) }
    if result.Absent != nil { fmt.Println("clean polling absence:", result.Absent.StatusCode) }
    fmt.Println("latest logical ID JSON:", string(result.SnapshotID))
}

func InspectError(err error) {
    if err == nil { return }
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Println("accepted response failure:", response.StatusCode, len(response.Body))
    }
    var sdkTimeout *blockstorage.SnapshotWaitTimeoutError
    if errors.As(err, &sdkTimeout) {
        fmt.Println("SDK wait budget:", sdkTimeout.Timeout)
    }
    fmt.Println("deadline cause:", errors.Is(err, context.DeadlineExceeded))
}

func ViaConnection(ctx context.Context, cloudName, volumeID string) error {
    conn, err := Configured(ctx, cloudName)
    if err != nil { return err }
    created, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotName("nightly"),
        blockstorage.WithCreateVolumeSnapshotDescription("nightly volume snapshot"))
    InspectCreate(created) // Inspect completed physical stages even when err is nonnil.
    if err != nil { InspectError(err); return err }

    // Default deletion resolves exactly one snapshot and returns after acknowledgement.
    deleted, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "nightly"})
    InspectDelete(deleted)
    InspectError(err)
    return err
}

func DirectNoWait(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    result, err := blockstorage.CreateVolumeSnapshot(ctx, cinder,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotForce(false),
        blockstorage.WithCreateVolumeSnapshotWait(false),
        blockstorage.WithCreateVolumeSnapshotFields(map[string]json.RawMessage{
            "display_name": json.RawMessage(`"nightly"`),
            "description": json.RawMessage(`"created without polling"`),
        }))
    InspectCreate(result)
    InspectError(err)
    return err
}

func WaitForDeletion(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    budget, interval := 10*time.Minute, time.Second
    result, err := conn.DeleteVolumeSnapshot(ctx,
        blockstorage.DeleteVolumeSnapshotRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeSnapshotWait(true),
        blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{
            Timeout: &budget, PollInterval: &interval,
        }))
    InspectDelete(result)
    InspectError(err)
    return err
}

func ZeroWaitBudget(ctx context.Context, conn *sdk.Connection, volumeID string) (*blockstorage.CreateVolumeSnapshotResult, error) {
    zero := time.Duration(0)
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &zero}))
    InspectCreate(result) // POST occurs first; an initially available result may already succeed.
    InspectError(err)
    return result, err
}

func WithCallerDeadline(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    parent, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    result, err := conn.CreateVolumeSnapshot(parent,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID})
    InspectCreate(result)
    InspectError(err)
    return err
}

func OwnedLocation(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Project: resource.CloudProject{
            ID: json.RawMessage(`"token-scope-id"`), Name: &project,
        },
    }
    wait := false
    result, err := conn.CreateVolumeSnapshot(ctx,
        blockstorage.CreateVolumeSnapshotRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{
            Wait: &wait, Location: &location,
        }))
    InspectCreate(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeSnapshot(ctx, blockstorage.DeleteVolumeSnapshotRequest{})
    InspectDelete(result) // Valid context, no service selection, successful false.
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="configured-cloud")
snapshot = conn.create_volume_snapshot(
    "volume-id", name="nightly", description="nightly volume snapshot"
)  # force=False, wait=True, timeout=None
accepted = conn.create_volume_snapshot("volume-id", display_name="nightly", wait=False)
deleted = conn.delete_volume_snapshot("nightly")  # wait=False
deleted_after_wait = conn.delete_volume_snapshot("nightly", wait=True, timeout=600)
absent_input = conn.delete_volume_snapshot()  # False, no lookup
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L505)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L799)
- [pinned v3 Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py)
- [Volume Snapshots](volume-snapshots.md)
