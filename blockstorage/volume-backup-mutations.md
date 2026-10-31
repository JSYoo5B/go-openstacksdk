# Volume Backup Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/create_volume_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CreateVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCreateVolumeBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupDescription`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupIncremental`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupSnapshotID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupWaitPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCreateVolumeBackupLocation`

- All fixed six body fields are present: nullable name/description/snapshot_id plus literal volume_id, explicit forcefalse/incrementalfalse defaults. No generated name/description, Volume/Snapshot resolver or extra raw kwargs/native builder; empty/slashed body IDs are forwarded after stringUTF8 validation.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupDefaultsSendSixLiteralFieldsWithoutResolvers`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupConcreteOptionsPreserveEmptyFalseAndLiteralSnapshot`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationCreateTextValidationPrecedesCinderSelection`.
- Incremental bool serializes wire incremental while request-seeded normalized state uses is_incremental. All six constructor fields including null/false are retained as dirty input; POST is queryless/backups and does not seed Backup.id from VolumeID.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupDefaultsSendSixLiteralFieldsWithoutResolvers`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupConcreteOptionsPreserveEmptyFalseAndLiteralSnapshot`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`.
- Original admittedHTTP100..399 create responses are consumed independently of native Create202 leaf. Canonical backup or flat object merges only known fields; missing preserves prior values and explicit null replaces. Parsed duplicate-key first-position/last-value alias order is retained without dirty-equality suppression.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupMergesKnownResponseFieldsWithoutInventingPhysicalInputs`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupInitialErrorIsCheckedOnlyAfterFreshFetchAndMerge`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupNullablePollContinuesAndLatestIDIsUsedOnlyForNeededRoute`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateAliasesUseParsedDuplicateInsertionOrder`.
- Create and each fresh poll eagerly normalize the closed Backup23Body+computedlocation24 view, ordinary Boolean/list/dict/integer descriptors and untyped remaining values. Physical raw object keeps unknown fields and original numeric/types; snapshot descriptors/state indices do not leak.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupDefaultsSendSixLiteralFieldsWithoutResolvers`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupMergesKnownResponseFieldsWithoutInventingPhysicalInputs`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupRejectedShapesAndDescriptorsRetainCreationProof`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`.
- Current or supplied owned location is captured after originals once, then recomputed for every merged Backup from actual project alias and literal availability_zone. Foreign project clears configured names/domain; rowzone overrides caller/raw location; live auth does not recapture current call scope.
  Tests: `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsCaptureRecordedScopeAfterOriginalsBeforeGetterAcrossWait`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsExplicitLocationAndRetainedOriginalsStayOwned`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelPhysicalAndLocalLocationFailuresAreAtomicAndOutputIsOwned`.
- Empty or malformed validUTF8 JSON preserves prior logical state and actual proof without fabricating RawResource. Parsed{} is actual nonnil object; valid wrongshape/null envelope/invalidUTF8 or descriptor failure keeps current physical evidence and leaves final success publication atomic.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupMergesKnownResponseFieldsWithoutInventingPhysicalInputs`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupRejectedShapesAndDescriptorsRetainCreationProof`.
- Wait defaults true. Initial merged case-insensitive available completes with no GET and no unused route/timeout/interval checks; noWait also avoids status/route waiting while complete descriptor normalization remains eager.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupNoWaitPreservesArbitraryIdentityAndStatus`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupCachedAvailableSkipsTimeoutIntervalAndUnusableIdentity`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupConcreteOptionsPreserveEmptyFalseAndLiteralSnapshot`.
- Otherwise a fresh GET begins after loop boundary and safe currentID validation. Initialerror is not failed until fresh observation; available wins before exactcaselesserror failure. Explicitnullstatus remains nonterminal, nonstringstatus errors only when status wait is reached.
  Tests: `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupInitialErrorIsCheckedOnlyAfterFreshFetchAndMerge`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupNullablePollContinuesAndLatestIDIsUsedOnlyForNeededRoute`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupPollIntervalIsConsumedOnlyAfterNonterminalState`.
- Fresh canonical/flat/empty/malformed polls merge prior state and may change logicalID. A latest arbitrary or unusable ID is preserved passively on terminal/noWait output; it is validated only if another fixed member request is needed.
  Tests: `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupInitialErrorIsCheckedOnlyAfterFreshFetchAndMerge`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupNullablePollContinuesAndLatestIDIsUsedOnlyForNeededRoute`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupNoWaitPreservesArbitraryIdentityAndStatus`.
- Nil timeout is unlimited; explicitzero/negative fails before first usedpoll unless initialready/noWait. Positive budget begins after initial mutation/merge and checks loop boundaries without deriving HTTP deadline. Pollnil2s; zero100ms capped by a smaller explicit timeout, negative only errors on reachedsleep; parentctx interrupts HTTP/sleep and retains customcause.
  Tests: `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationNonpositiveTimeoutRunsMutationButPreventsFirstPoll`, `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationPositiveBudgetStartsAfterMutationAndAllowsSlowTerminalHTTP`, `blockstorage/backup_mutation_wait_test.go:TestCreateVolumeBackupPollIntervalIsConsumedOnlyAfterNonterminalState`, `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationPositiveTimeoutIsObservedAtLoopBoundaryWithoutBorrowingProof`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Created/CreatedValue/CreatedBackup, latest raw BackupID, LastAccepted and Ready/ReadyBackup are independently owned actual/logical phases. Final Value/Backup publishes only after requested policy and final guard; acceptedRead/Close/decode/source/context errors retain prior/current actual evidence without rollback.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupOwnsCreatedFinalReadyAndRawIdentityIndependently`, `blockstorage/backup_mutation_contract_test.go:TestCreateVolumeBackupRejectedShapesAndDescriptorsRetainCreationProof`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationAcceptedFailuresRetainOnlyCurrentProofAndAllCauses`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Concrete With/Prepare owns pointers/location/waitpolicy and snapshots original options slice. Direct capture precedes originals; Connection originals run once before ownedCurrentLocation and only cached Cinderv3 getter. Source/method/URL/body/status ownership holds through native liveauth/reauth/retry without SDK orchestration replay.
  Tests: `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsPreserveOmissionAndExplicitEmptyFalseZero`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsOwnFactoryInputsAndEveryPreparedTree`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsRunOriginalsOnceAndFreezeMembershipAndWrites`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsJoinCancellationAndOriginalErrors`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsReuseFactoriesConcurrentlyWithOwnedLocations`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsExplicitLocationAndRetainedOriginalsStayOwned`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsNonemptyPreflightAndGetterFailuresKeepOperationCauses`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationCreateTextValidationPrecedesCinderSelection`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationNativeAuthAndRetryUseLiveTokenWithOwnedHeadersAndVersion`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationRetryCannotReplaceOwnedBodyDecoderOrFixedScope`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationExpandedNativeStatusNeverCreatesSuccessOrAbsenceProof`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationOriginalSourceChangesAreTerminalBeforeLaterRestoration`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is the candidate target with selected Cinder v3 and Gophercloudv2.15.0. Independent Proxy/native/v2/generic Resource/session/adapter and other backup helpers retain separate reviews; full SDK remains incomplete.
- Python mutable/arbitrary runtime inputs map to owned Go UTF8 request strings, concrete nullable bool/string options, finite JSON raw/logical values and phase results. Python Resource object identity, stale dict caches, arbitrary Python object coercion and warning emission are not reproduced.
- Normal selected microversion is retained without automatic absent-default discovery/cap/upgrade. Python Backupmax3.64 constrains absent-default negotiation only; explicit configured defaults take precedence. Forceaction is separately the exact source3.64 behavior.
- Go safe consumed member IDs and fixed same-origin method/URL/body/status ownership replace source urljoin falsey/container coercion/leading slash behavior. Unsafe names use exact list lookup; passive resultIDs remain arbitrary JSON until a needed route.
- Nullable23Body+computedlocation model is independent of actual RawResource and admitted/rejected HTTP proof. Unknown actual fields/numeric spelling are preserved as Go evidence extension while only known fields overlay logical state; parsed JSON alias/duplicate insertion traversal is retained.
- Exact decimal truthiness/equality, arbitrary integer JSON, Go Unicode decimal runtime table and strict UTF8 are explicit boundaries from Python parsedfloat rounding/underflow/configurableintegerlimits. Nonnumeric integer containers normalize0; bool remainsbool; digit-but-nondecimal strings can fail.
- Captured current/explicit configured scope recomputes from Backup project alias and literal rowzone, separate from live provider authentication. Standalone/disconnected Resource location/runtime construction are independent capabilities.
- Positive SDK timeout is loop-boundary only; parentcontext cancellation/deadline adds a Go interruption/cause contract. Nil alone unlimited; zero/negative remain used-loop timeout. PollInterval is a concrete Go convenience absent from the direct Python helper.
- Accepted Read/Close/decode/context/source failures retain their actual phase evidence. Only ordinary member400403404 and deletionpoll404 have bounded rejection-body fault observation; rejected Lists, other member statuses and mutations retain native discarded-bodyIO policy. This is not universal rejection-IO preservation.
- Go errors retain operation/resource/custom context causes and independently owned partial phase results; Source probes use explicit AST/session/response/location stand-ins and are not authenticated Python/OpenStack runtime evidence. No orchestration replay or rollback is added.
- The fixed cloud signature has no Fields/display aliases/native extraCreate attributes. Omitted nullabletext serializesnull, explicitempty retained, force/incremental defaults serializefalse; Go typed bool/string domain replaces arbitrary Python arguments.
- VolumeID/SnapshotID are literal bodydata, not resource.Ref resolution or generated Backup.id/name. Create response partial merge preserves request-known state while CreatedBackup is actual only.

### `python:cloud/_block_storage/BlockStorageCloudMixin/delete_volume_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DeleteVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareDeleteVolumeBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeBackupForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeBackupWait`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeBackupWaitPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithDeleteVolumeBackupLocation`

- Empty string input returns completed Deletedfalse after valid context/original callbacks, selecting no client/location/service/waitpolicy and allowing nil direct client/nilConnection. Successful completed lookup absence is false; any lookup/operation error leaves Deletednil.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupEmptyInputRunsOriginalsWithoutServiceOrLocation`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupLookupCompletionAndLogicalSeedControlMutation`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationEmptyDeleteIsServiceFreeEvenForNilConnection`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsJoinCancellationAndOriginalErrors`.
- Nonempty delete runs one ordinary exact Backup find on the same captured reader, with member-first safe identity and detailed literal-name fallback only from clean original native400403404. Unsafe names use exact list-only route domain, duplicates/later exhaustion errors prevent mutation; no extra confirmGET or Volume/Snapshot/Identity resolution.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupLookupCompletionAndLogicalSeedControlMutation`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindCleanNativeFallbackIsExactAndUsesActualResponseID`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindUnsafeLiteralAndWildcardNamesNeverBecomeSearchPatterns`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindAmbiguityPrecedesUnusedBadTailAndUniqueMustExhaust`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFallbackLaterNativeFailureNeverBorrowsSuppressedMemberProof`.
- Selected actual changedID is used; only a physically absent memberID seeds requested identity logically with Resolved RequestedID/SeededID, never physical raw fields. Explicitnull/false/zero/empty/container/unusable mutationID is a reached local route error with completed Resolved retained.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupLookupCompletionAndLogicalSeedControlMutation`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelMemberSeedsOnlyAbsentInheritedIDAndKeepsPhysicalBytes`.
- Normal branch sends fixed bodyless DELETE/backups/{selectedID}, no force/cascadequery. Force branch sends fixed POST/backups/{selectedID}/action with literal os-force_deleteNull; no native{} body substitution.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupForceActionUsesNullAnd364WithoutChangingPollVersion`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupNormalAndForceMutation404NeverBecomeAbsence`.
- Force action explicitly uses3.64 independently of empty/3.60/3.80 normal selection. Only copied operation client/canonical generic+legacy version headers override; original service/headers and ordinary lookup/poll microversion remain unchanged and original source facts remain guarded.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupForceActionUsesNullAnd364WithoutChangingPollVersion`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsForceKeepsCachedSelectionAndNormalWaitPolicyIndependent`.
- Force nativeRetry cannot alter/remove required version aliases via MoreHeaders/OmitHeaders, and actual physical request requires exactly one exact generic/legacy3.64header each. Callback/source/policy faults keep original native evidence and block physical resend; fixed redirects/liveauth/native retry remain guarded.
  Tests: `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationNativeAuthAndRetryUseLiveTokenWithOwnedHeadersAndVersion`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationRetryCannotReplaceOwnedBodyDecoderOrFixedScope`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsForceKeepsCachedSelectionAndNormalWaitPolicyIndependent`.
- Both mutations use original finiteHTTP100..399 opaque acknowledgement without JSON/UTF8/status merge. Native or expanded-policy rejected normal/force404 is terminal ignore_missingFalse and cannot become Deletedfalse/Absent or admitted Applied.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupForceActionUsesNullAnd364WithoutChangingPollVersion`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupNormalAndForceMutation404NeverBecomeAbsence`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationExpandedNativeStatusNeverCreatesSuccessOrAbsenceProof`.
- DeleteWait defaults false, so admitted acknowledgement plus final guard yields true without later route/status/wait checks. WithWaittrue always performs firstfreshGET even cacheddeleted after used loop-boundary/currentID checks.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement`, `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupForceActionUsesNullAnd364WithoutChangingPollVersion`, `blockstorage/backup_mutation_wait_test.go:TestDeleteVolumeBackupFreshFetchMergesCachedDeletedButRejectsExplicitNullStatus`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`.
- Fresh merged case-insensitivedeleted or clean original nativeGET404 completes. Error/error_deleting are nonterminal; explicitnull/nonstringstatus is error when reached, unlike create nullstatus. Missing/partial/flat/malformed poll fields retain semantic prior state and changedID controls needed laterpoll only.
  Tests: `blockstorage/backup_mutation_wait_test.go:TestDeleteVolumeBackupFreshFetchMergesCachedDeletedButRejectsExplicitNullStatus`, `blockstorage/backup_mutation_wait_test.go:TestDeleteVolumeBackupErrorStatesContinueAndLatestIDDoesNotRouteTerminalFalse`.
- Poll404 keeps native rejection/retry/reauth and bounded Read/Close observation. Only direct fault-free native404 can become Absent; wrapped/callback/expandedOkCodes/IO/source/context404 remains terminal with Deletednil. Rejected proof never overwrites admitted LastAccepted or earlier Applied.
  Tests: `blockstorage/backup_mutation_transport_test.go:TestDeleteVolumeBackupPolling404RequiresCleanBodyAndOriginalNativePolicy`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationExpandedNativeStatusNeverCreatesSuccessOrAbsenceProof`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationAcceptedFailuresRetainOnlyCurrentProofAndAllCauses`.
- Nil cloud timeout overrides Proxy standalone120 with unlimited. Explicitzero/negative timeout fails after acknowledgement before firstfreshpoll. Positive loop budget does not derive HTTP deadline and may accept slowterminal response; parentctx/customcause separately interrupts request/sleep.
  Tests: `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationNonpositiveTimeoutRunsMutationButPreventsFirstPoll`, `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationPositiveBudgetStartsAfterMutationAndAllowsSlowTerminalHTTP`, `blockstorage/backup_mutation_wait_test.go:TestVolumeBackupMutationPositiveTimeoutIsObservedAtLoopBoundaryWithoutBorrowingProof`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Resolved, raw BackupID, opaque Applied, admitted LastAccepted, Ready/ReadyBackup from deleted GET and clean404Absent are independently owned phase evidence. Deleted is nil on error and true only after requested policy/finalguard; no partial remote-completion bool or rollback is invented.
  Tests: `blockstorage/backup_mutation_contract_test.go:TestDeleteVolumeBackupDefaultWaitOffUsesActualIDAndOpaqueAcknowledgement`, `blockstorage/backup_mutation_wait_test.go:TestDeleteVolumeBackupFreshFetchMergesCachedDeletedButRejectsExplicitNullStatus`, `blockstorage/backup_mutation_wait_test.go:TestDeleteVolumeBackupErrorStatesContinueAndLatestIDDoesNotRouteTerminalFalse`, `blockstorage/backup_mutation_transport_test.go:TestDeleteVolumeBackupPolling404RequiresCleanBodyAndOriginalNativePolicy`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationAcceptedFailuresRetainOnlyCurrentProofAndAllCauses`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause`.
- Known24 Backup descriptor/projectzone/alias semantics are preserved from selected logical prior values without claiming original wire provenance; fresh physicalRawResource remains actual. Caller options/source/headers/scope remain owned and concurrent snapshot/backup tables independent.
  Tests: `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelPhysicalAndLocalLocationFailuresAreAtomicAndOutputIsOwned`, `internal/cloudsnapshot/mutation_state_test.go:TestSnapshotMutationStateAliasesUseParsedDuplicateInsertionOrder`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsOwnFactoryInputsAndEveryPreparedTree`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsReuseFactoriesConcurrentlyWithOwnedLocations`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsCaptureRecordedScopeAfterOriginalsBeforeGetterAcrossWait`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsExplicitLocationAndRetainedOriginalsStayOwned`.
- Original callbacks execute once before ConnectionCurrentLocation/cachedCinderv3 getter; direct source capture precedes callbacks. Fixed method/URL/body/originalstatus, sticky source and parentcause guards preserve native liveauth/reauth/caller retry; no SDK workflow replay or automatic cleanup.
  Tests: `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsRunOriginalsOnceAndFreezeMembershipAndWrites`, `blockstorage/backup_mutation_options_test.go:TestBackupMutationOptionsJoinCancellationAndOriginalErrors`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsNonemptyPreflightAndGetterFailuresKeepOperationCauses`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationNativeAuthAndRetryUseLiveTokenWithOwnedHeadersAndVersion`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationRetryCannotReplaceOwnedBodyDecoderOrFixedScope`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationExpandedNativeStatusNeverCreatesSuccessOrAbsenceProof`, `blockstorage/backup_mutation_transport_test.go:TestVolumeBackupMutationOriginalSourceChangesAreTerminalBeforeLaterRestoration`, `connectiontest/connection_volume_backup_mutations_test.go:TestConnectionBackupMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is the candidate target with selected Cinder v3 and Gophercloudv2.15.0. Independent Proxy/native/v2/generic Resource/session/adapter and other backup helpers retain separate reviews; full SDK remains incomplete.
- Python mutable/arbitrary runtime inputs map to owned Go UTF8 request strings, concrete nullable bool/string options, finite JSON raw/logical values and phase results. Python Resource object identity, stale dict caches, arbitrary Python object coercion and warning emission are not reproduced.
- Normal selected microversion is retained without automatic absent-default discovery/cap/upgrade. Python Backupmax3.64 constrains absent-default negotiation only; explicit configured defaults take precedence. Forceaction is separately the exact source3.64 behavior.
- Go safe consumed member IDs and fixed same-origin method/URL/body/status ownership replace source urljoin falsey/container coercion/leading slash behavior. Unsafe names use exact list lookup; passive resultIDs remain arbitrary JSON until a needed route.
- Nullable23Body+computedlocation model is independent of actual RawResource and admitted/rejected HTTP proof. Unknown actual fields/numeric spelling are preserved as Go evidence extension while only known fields overlay logical state; parsed JSON alias/duplicate insertion traversal is retained.
- Exact decimal truthiness/equality, arbitrary integer JSON, Go Unicode decimal runtime table and strict UTF8 are explicit boundaries from Python parsedfloat rounding/underflow/configurableintegerlimits. Nonnumeric integer containers normalize0; bool remainsbool; digit-but-nondecimal strings can fail.
- Captured current/explicit configured scope recomputes from Backup project alias and literal rowzone, separate from live provider authentication. Standalone/disconnected Resource location/runtime construction are independent capabilities.
- Positive SDK timeout is loop-boundary only; parentcontext cancellation/deadline adds a Go interruption/cause contract. Nil alone unlimited; zero/negative remain used-loop timeout. PollInterval is a concrete Go convenience absent from the direct Python helper.
- Accepted Read/Close/decode/context/source failures retain their actual phase evidence. Only ordinary member400403404 and deletionpoll404 have bounded rejection-body fault observation; rejected Lists, other member statuses and mutations retain native discarded-bodyIO policy. This is not universal rejection-IO preservation.
- Go errors retain operation/resource/custom context causes and independently owned partial phase results; Source probes use explicit AST/session/response/location stand-ins and are not authenticated Python/OpenStack runtime evidence. No orchestration replay or rollback is added.
- Empty typed identity is a service-free Go no-op but still context/originalcallback checked. Actual Resource presence is not ID truthiness; member missingID seed is logical only, while changed/unusable responseID drives reached safe route policy.
- Forced deletion deliberately uses sourceNull action at explicit3.64, even when normal selected version differs; native ForceDelete202/{} remains independently reviewed. Mutation404 is terminal, cleanpoll404 completes only after proof/source/cause guards.

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

func Explain(err error) {
    if err == nil { return }
    fmt.Println("workflow error:", err)
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("SDK wait timeout or parent context deadline")
    }
    if errors.Is(err, context.Canceled) {
        fmt.Println("parent context canceled")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("local input, location, status or route error")
    }
}

func InspectCreate(result *blockstorage.CreateVolumeBackupResult) {
    if result == nil { return }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Created != nil {
        fmt.Println("POST:", result.Created.StatusCode, "bytes:", len(result.Created.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    fmt.Println("created-stage logical JSON:", string(result.CreatedValue))
    if result.CreatedBackup != nil {
        actual := result.CreatedBackup.Clone()
        fmt.Println("actual POST ID:", string(actual.Body["id"]))
        var fields map[string]json.RawMessage
        if err := actual.Decode(&fields); err != nil {
            fmt.Println("optional projection:", err)
        } else {
            fmt.Println("actual POST field count:", len(fields))
        }
    }
    fmt.Println("final logical JSON:", string(result.Value), "ready:", len(result.Ready) != 0)
}

func InspectDelete(result *blockstorage.DeleteVolumeBackupResult) {
    if result == nil { return }
    if result.Deleted == nil {
        fmt.Println("requested delete policy did not complete")
    } else {
        fmt.Println("deleted:", *result.Deleted)
    }
    fmt.Println("latest logical ID JSON:", string(result.BackupID))
    if result.Resolved != nil {
        fmt.Println("resolved logical JSON:", string(result.Resolved.Value))
        fmt.Println("resolved list pages:", len(result.Resolved.Pages))
        if result.Resolved.Observed != nil {
            fmt.Println("resolved member:", result.Resolved.Observed.StatusCode)
        }
    }
    if result.Applied != nil {
        fmt.Println("DELETE or force POST:", result.Applied.StatusCode, "bytes:", len(result.Applied.Body))
    }
    if result.LastAccepted != nil {
        fmt.Println("last admitted response:", result.LastAccepted.StatusCode)
    }
    if result.Absent != nil {
        fmt.Println("clean polling GET absence:", result.Absent.StatusCode)
    }
    fmt.Println("deleted-status logical JSON:", string(result.Ready))
}

func CreateDefault(ctx context.Context, conn *sdk.Connection, volumeID string) error {
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID})
    InspectCreate(result)
    Explain(err)
    return err
}

func CreateWithoutWait(ctx context.Context, conn *sdk.Connection, volumeID, snapshotID string) error {
    policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx,
        blockstorage.WithCreateVolumeBackupName(""),
        blockstorage.WithCreateVolumeBackupDescription(""),
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupIncremental(true),
        blockstorage.WithCreateVolumeBackupSnapshotID(snapshotID),
        blockstorage.WithCreateVolumeBackupWait(false))
    if err != nil { return err }
    result, err := conn.CreateVolumeBackup(ctx,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeBackupOptions(policy))
    InspectCreate(result)
    return err
}

func RemoveDefault(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID})
    InspectDelete(result)
    Explain(err)
    return err
}

func ForceAndWait(ctx context.Context, conn *sdk.Connection, nameOrID string) error {
    timeout := 10 * time.Minute
    policy, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx,
        blockstorage.WithDeleteVolumeBackupForce(true),
        blockstorage.WithDeleteVolumeBackupWait(true),
        blockstorage.WithDeleteVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{
            Timeout: &timeout,
        }))
    if err != nil { return err }
    result, err := conn.DeleteVolumeBackup(ctx,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupOptions(policy))
    InspectDelete(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient, volumeID string) error {
    cloud, region := "owned-cloud", "RegionOne"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`)},
    }
    result, err := blockstorage.CreateVolumeBackup(ctx, cinder,
        blockstorage.CreateVolumeBackupRequest{VolumeID: volumeID},
        blockstorage.WithCreateVolumeBackupLocation(location),
        blockstorage.WithCreateVolumeBackupForce(false),
        blockstorage.WithCreateVolumeBackupWait(false))
    InspectCreate(result)
    return err
}

func DirectDelete(ctx context.Context, cinder *gophercloud.ServiceClient, nameOrID string) error {
    result, err := blockstorage.DeleteVolumeBackup(ctx, cinder,
        blockstorage.DeleteVolumeBackupRequest{NameOrID: nameOrID},
        blockstorage.WithDeleteVolumeBackupForce(false),
        blockstorage.WithDeleteVolumeBackupWait(false))
    InspectDelete(result)
    return err
}

func EmptyDelete(ctx context.Context) error {
    var conn *sdk.Connection
    result, err := conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{})
    InspectDelete(result)
    if err != nil { return err }
    if result == nil || result.Deleted == nil || *result.Deleted {
        return errors.New("empty delete must return a completed false result")
    }
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
created = conn.create_volume_backup(volume_id)
queued = conn.create_volume_backup(
    volume_id, name="", description="", force=False, wait=False,
    incremental=True, snapshot_id=snapshot_id,
)
deleted = conn.delete_volume_backup("nightly")
forced = conn.delete_volume_backup("nightly", force=True, wait=True, timeout=600)
empty = conn.delete_volume_backup()
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L600)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L758)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L116)
- [V3](v3/README.md)
- [v3 Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Volume Backups](volume-backups.md)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L132)
- [native Backup Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L60)
- [ForceDelete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L357)
- [Microversions](../docs/microversions.md)
