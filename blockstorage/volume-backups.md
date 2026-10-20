# Volume Backups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/list_volume_backups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListVolumeBackups`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ListVolumeBackups`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeBackupListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListDetailed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListPagination`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListExpression`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListAllowUnknownParams`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListConflictingAttrs`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupListLocation`

- Library-owned list policy preserves omission and source defaults detailed=true and paginated=true. Default detailed GET/backups/detail and basic GET/backups return complete owned lists. Preparation is pure; original callbacks run once from a captured slice and each snapshot owns raw filters, headers, pointer controls and location. Connection validates reached binding policy before selecting only cached Cinder v3 without readiness I/O.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListDefaultDetailedAndBasicKeepCanonicalEmptyEOF`, `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadOriginalOptionsOwnEveryPreparedValueAndSourceHeaders`, `connection_volume_backups_test.go:TestConnectionVolumeBackupOriginalOptionsRunOnceBeforeCachedCinderForEveryEntry`, `connection_volume_backups_test.go:TestConnectionVolumeBackupPreflightAndGetterFailuresKeepBackupOperationAndJoinedCauses`, `connection_volume_backups_test.go:TestConnectionVolumeBackupListBindingValidationPrecedesServiceSelection`, `connection_volume_backups_test.go:TestConnectionVolumeBackupCapturedOptionSliceAndRetainedPointersCannotReplacePreparedPolicy`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListPolicyOwnsQueryLocalHeadersAndResourceSelectionAcrossReuse`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListFalseyAndInvalidRawPoliciesHaveAtomicResourceSpecificCompilation`.
- Logical backups expose exactly nullable24 descriptors, distinct from typed native backups or Snapshot. force/is_incremental/has_dependent_backups use ordinary Python JSON-domain bool, links wraps a nonlist singleton, metadata uses nullable dict and size/object_count apply eager nullable integer conversion. Known conversions precede local exclusion; unknown raw fields remain only in physical evidence. Parsed duplicate insertion/alias order is preserved.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListOwnsAllTwentyFourFieldsAndRawIndependentProofs`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadInvalidIntegersAreEagerBeforeLocalExclusion`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadCollectionEnvelopeAndListConstructorControlsStayResourceSpecific`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelListConstructorControlsDoNotBecomeMemberResponseControls`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListCallerPredicatesAreNeverCoercedLikeBackupResponseDescriptors`.
- Current/explicit scope is owned after originals and before the Connection getter, then retained across live-auth pages. Backup project wire alias and literal availability_zone recompute location; current/falsey projects retain configured names, foreign projects clear them. Caller zone/wire location cannot replace it. Physical descriptor failures and caller-local invalid location remain distinct and atomic.
  Tests: `connection_volume_backups_test.go:TestConnectionVolumeBackupLocationIsCapturedAfterOriginalsBeforeGetterAndKeepsRawZoneAcrossLiveAuthPages`, `connection_volume_backups_test.go:TestConnectionVolumeBackupExplicitLocationOwnsEveryFieldAndUsesBackupProjectAndZone`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelPhysicalAndLocalLocationFailuresAreAtomicAndOutputIsOwned`.
- List separates actual server query mappings from ordered local Body predicates. Raw all_projects is literal all_tenants data and canonical presence beats its wire alias even falsey; late conflicting attrs bypass earlier Proxy bindings. Typed controls override the final applicable raw controls while preserving stage collisions. Unknown, Snapshot-only, native-only and creation-only names are ignored; caller predicates are uncoerced and use nested Python bool/numeric equality.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadAllProjectsIsLiteralWhileSnapshotProxyBindsTruthiness`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadLocalAttributesDifferFromIgnoredSnapshotAndCreationNames`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadLocalFilterOrderIsLazyAfterEagerBoolNormalization`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListAllProjectsPreservesRawValuesInsteadOfSnapshotProxyTruthiness`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListCanonicalAllProjectsPresenceWinsWireAliasEvenWhenFalsey`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListLateAliasOverlayDoesNotPassThroughEitherProxyAgain`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListQueryUsesItsOwnBodyPredicatesAndPreservesServerOnlyParameters`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListCallerPredicatesAreNeverCoercedLikeBackupResponseDescriptors`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListControlsKeepBindingStageAndTypedOverridesIndependentOfResourceModel`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListFalseyAndInvalidRawPoliciesHaveAtomicResourceSpecificCompilation`.
- Raw canonical backups pages preserve actual rows, duplicate identities and plural backups_links continuation. Empty canonical collection ends before unused links. Source body-link precedence/falsey fallback, advertised query merge and repeated/null Requests-domain values are preserved within safe fixed-origin/path URLs; source HTTP uri/url mismatch is explicitly repaired. A truthy initial limit may follow the last consumed arbitrary raw ID even from an excluded short-page row into a second empty EOF.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadShortLimitUsesExcludedArbitraryMarkerThenEmptyEOF`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadBackupsLinksMaterializeDistinctRowsAcrossPages`.
- Maximum counts consumed raw rows including local exclusions; source zero is unlimited and negative maximum still fetches/parses the first canonical response before stopping. Unconsumed row constructors remain lazy. At an exact page-end maximum, source continuation and another canonical HTTP response may occur before the next raw-count check. Detailed/pagination/max/header/microversion policies remain per call and cannot alter cached source or Snapshot schema.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadMaximumCountsExcludedRowsAndDefersUnusedConstruction`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadMaximumAtPageEndStillFetchesAndDecodesNextResponse`, `connection_volume_backups_test.go:TestConnectionVolumeBackupListControlsStayPerCallAndDoNotLeakSnapshotQueryOrModel`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListControlsKeepBindingStageAndTypedOverridesIndependentOfResourceModel`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListPolicyOwnsQueryLocalHeadersAndResourceSelectionAcrossReuse`.
- Explicit list expression evaluates the completed normalized backup array, including empty syntax errors, and returns arbitrary owned JSON without fabricated raw-row association. Raw falsey expression controls remain omitted. This whole-value behavior deliberately repairs the source expression-on-generator defect, while ordinary successful empty selection exposes an owned nonnil empty Backups slice.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListExpressionUsesCompletedBackupValueAndNoRowAssociation`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListFalseyAndInvalidRawPoliciesHaveAtomicResourceSpecificCompilation`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListDefaultDetailedAndBasicKeepCanonicalEmptyEOF`.
- Logical Value/Backups publish only after requested completion. Admitted Pages own original100..399 body/header/status, independent from row values/errors. Accepted Read/Close/source/context faults retain current proof and joined cancellation causes; later rejected/unreceived phases retain earlier pages without borrowing their response. Fixed bodyless requests keep live native reauth/retry and captured headers; concurrent Snapshot/Backup calls cannot change either closed schema.
  Tests: `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `connection_volume_backups_test.go:TestConnectionVolumeBackupAcceptedIOJoinsReadCloseCancellationAndSourceDriftWithCurrentProof`, `connection_volume_backups_test.go:TestConnectionVolumeBackupLaterFailureRetainsPagesWithoutBorrowingEarlierMemberEvidence`, `blockstorage/volume_backup_read_isolation_test.go:TestVolumeBackupReadConcurrentSnapshotAndBackupCallsKeepSchemaPolicyIsolated`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListPolicyOwnsQueryLocalHeadersAndResourceSelectionAcrossReuse`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin cloud helper is mapped with selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy, native typed leaf, v2 Backup, generic Resource/session/adapter and backup mutation/export declarations retain separate reviews; the full SDK remains incomplete.
- Python mutable kwargs/runtime objects map to owned Go request strings, nullable concrete With/Prepare policy, finite UTF8 JSON, raw-resource evidence and a separate normalized nullable24-field view. Go does not reproduce Resource identity, stale dict caches, session memoization, deprecated warning emission or arbitrary Python object coercion.
- Parsed JSON dictionary insertion order and duplicate-key first-position/last-value alias traversal are preserved. Exact decimal truthiness/equality and arbitrary integers do not reproduce Python float rounding/underflow or configurable CPython integer digit limits; the Go Unicode16 decimal table is an explicit version boundary.
- The selected Cinder microversion is retained, including explicit list overrides. Source Backup max3.64 limits absent-default negotiation only; Go performs no automatic negotiation, cap, upgrade or v2 selection. Selected and operation-owned headers are captured and validated while provider auth/reauth/retry tokens remain live.
- Captured current or explicit location is recomputed from the actual backup project alias and literal availability_zone. Raw location and caller zone cannot replace the owned scope. This connected cloud-helper model is distinct from standalone/disconnected Resource constructor behavior.
- Go validates consumed UTF8 JSON and safe literal member/continuation URLs, same origin/escaped path and canonical full-URL cycles. Source relative /v joining, query-only physical duplication and same-marker cycle rules are bounded by those deliberate safety rules. Robust HTTP Link URL support repairs the source uri/url mismatch.
- Direct source snapshots precede original callbacks; Connection originals run once before only cached Cinder v3 selection, with location captured after originals before the getter. Fixed bodyless requests, sticky source/cause guards, independently owned phase proof and atomic logical publication are Go extensions; caller-configured native retry remains live and the SDK adds no orchestration replay.
- Accepted response Read/Close/context/source faults retain their own physical evidence. Ordinary member native400/403/404 has a scoped body-fault observer so only fault-free original concrete rejections permit lookup fallback, without admitting those statuses or fabricating Observed. Rejected list responses and other rejected member statuses retain Gophercloud v2.15.0 discarded unexpected-body Read/Close behavior; this is not a universal rejection-I/O observer.
- Local descriptor/predicate/selection errors keep completed independently admitted proof and do not borrow a successful response as a new local failure. Source probes use explicit AST/session/response/location stand-ins and are source-only, not authenticated Python/OpenStack runtime evidence.
- Backup Proxy has no named all_projects argument: raw all_projects is a literal Resource alias to all_tenants. Snapshot named-argument truthiness/drop is intentionally isolated. Server-query versus local descriptor predicates, late conflicting-attribute remapping and inherited pagination/max/header/microversion controls have explicit owned Go equivalents; unsupported unknown keys remain ignored.
- Explicit list expression evaluates the completed normalized list, including empty-expression syntax failure. Raw falsey jmespath_filters skips evaluation. This repairs the source generator passed to selected Python JMESPath reference, whose projection can return None before iteration; it is a whole-value Go mapping, not a claim to reproduce that generator defect.

### `python:cloud/_block_storage/BlockStorageCloudMixin/search_volume_backups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchVolumeBackups`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SearchVolumeBackups`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeBackupSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchLocation`

- Search always completes a queryless default-detailed backup list and every reached nullable descriptor before applying local identifier/filter selection. A matching first row cannot hide a later excluded malformed descriptor or later page failure; no partial Value/Backups is published. Search does not send a name hint or caller filter as list query.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadSearchCompletesQuerylessPagesBeforeBackupExpression`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadSearchDoesNotHideExcludedLaterDescriptorFailure`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadInvalidIntegersAreEagerBeforeLocalExclusion`.
- Within finite JSON, optional NameOrID uses source per-row exact ID/name OR case-sensitive fnmatch over Python-style value stringification. It preserves source order, duplicate match occurrences, Unicode/rune and dot-all glob/class behavior. Empty identifier skips that stage rather than becoming an ordinary member lookup.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadSearchCompletesQuerylessPagesBeforeBackupExpression`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder`, `internal/cloudfilter/identifier_test.go:TestCloudFilterGlobUsesCPythonRuneDotAllClassesAndLiteralBackslashes`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/identifier_test.go:TestCloudFilterMissingAndNullIdentifiersDifferFromEmptyStringsAndRemainCaseSensitive`.
- Mapping filters evaluate normalized backups as ordered recursive subsets. Reached unknown keys and invalid filter shape are local errors; preceding identifier or predicate mismatch can leave later keys unused. Actual truthiness precedes empty-object matching, nested arrays keep exact shape and bool/number equality follows Python within the precise JSON numeric domain.
  Tests: `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadLocalAttributesDifferFromIgnoredSnapshotAndCreationNames`, `internal/cloudsnapshot/backup_query_test.go:TestBackupListCallerPredicatesAreNeverCoercedLikeBackupResponseDescriptors`, `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`, `internal/cloudfilter/mapping_test.go:TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNonmappingFilterShapeIsLazyAfterIdentifierAndActualTruthiness`, `internal/cloudfilter/mapping_test.go:TestCloudFilterInvalidJSONIsLocalAtomicAndStillValidatedAfterEmptyIdentifierSelection`.
- String/explicit expression filters evaluate the complete selected normalized JSON through the full owned JMESPath engine and may return any JSON value. Projection has nil raw Backups association; ordinary selection retains its real rows. Expression/selection errors are local, preserve completed Pages, and do not invent an HTTP response or partial output.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadSearchCompletesQuerylessPagesBeforeBackupExpression`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetArbitraryExpressionRetainsNoBackupAssociation`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionsStillParseEmptySelectionAndKeepRuntimeFailuresAtomic`.
- Search consumes the same closed Backup nullable24 descriptor policy: ordinary Boolean values, singleton links, nullable dict/integers, duplicate alias order and project plus literal zone location. Unknown raw fields stay in owned RawResource proof; mutable descriptor globals and concurrent Snapshot calls cannot leak Snapshot BoolStr, query all-projects or zoneless policy into Backup.
  Tests: `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`, `blockstorage/volume_backup_read_isolation_test.go:TestVolumeBackupReadConcurrentSnapshotAndBackupCallsKeepSchemaPolicyIsolated`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListOwnsAllTwentyFourFieldsAndRawIndependentProofs`.
- Concrete With/Prepare search options own raw filters/expression/location across reusable factories and callbacks. Direct service capture precedes originals; Connection originals run once before only cached Cinder v3 selection, then explicit/current location is captured before getter/auth changes. Preflight/callback/getter errors retain operation and custom context causes with no invented physical stage.
  Tests: `connection_volume_backups_test.go:TestConnectionVolumeBackupOriginalOptionsRunOnceBeforeCachedCinderForEveryEntry`, `connection_volume_backups_test.go:TestConnectionVolumeBackupLocationIsCapturedAfterOriginalsBeforeGetterAndKeepsRawZoneAcrossLiveAuthPages`, `connection_volume_backups_test.go:TestConnectionVolumeBackupExplicitLocationOwnsEveryFieldAndUsesBackupProjectAndZone`, `connection_volume_backups_test.go:TestConnectionVolumeBackupPreflightAndGetterFailuresKeepBackupOperationAndJoinedCauses`, `connection_volume_backups_test.go:TestConnectionVolumeBackupCapturedOptionSliceAndRetainedPointersCannotReplacePreparedPolicy`, `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadOriginalOptionsOwnEveryPreparedValueAndSourceHeaders`.
- Search Value/Backups is atomic after completion; admitted Pages and raw rows own independent bytes/header/status. Accepted Read/Close/cancel/source failures preserve their current stage and all joined causes, while a later rejected or blocked phase keeps only actual earlier pages. Physical errors are not borrowed from a suppressed member or an earlier successful response.
  Tests: `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `connection_volume_backups_test.go:TestConnectionVolumeBackupAcceptedIOJoinsReadCloseCancellationAndSourceDriftWithCurrentProof`, `connection_volume_backups_test.go:TestConnectionVolumeBackupLaterFailureRetainsPagesWithoutBorrowingEarlierMemberEvidence`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFallbackLaterNativeFailureNeverBorrowsSuppressedMemberProof`.
- Original source HTTP100..399 policy, fixed GET URL/query/body/decoder ownership and live native authentication/retry remain in force. Per-call or captured microversion is preserved without automatic max3.64 negotiation/cap. Safe continuation and source guards are shared closed reader rules; cached-client header/microversion state remains unchanged by prior Backup list controls.
  Tests: `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `connection_volume_backups_test.go:TestConnectionVolumeBackupListControlsStayPerCallAndDoNotLeakSnapshotQueryOrModel`, `connection_volume_backups_test.go:TestConnectionVolumeBackupLaterFailureRetainsPagesWithoutBorrowingEarlierMemberEvidence`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadSearchCompletesQuerylessPagesBeforeBackupExpression`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin cloud helper is mapped with selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy, native typed leaf, v2 Backup, generic Resource/session/adapter and backup mutation/export declarations retain separate reviews; the full SDK remains incomplete.
- Python mutable kwargs/runtime objects map to owned Go request strings, nullable concrete With/Prepare policy, finite UTF8 JSON, raw-resource evidence and a separate normalized nullable24-field view. Go does not reproduce Resource identity, stale dict caches, session memoization, deprecated warning emission or arbitrary Python object coercion.
- Parsed JSON dictionary insertion order and duplicate-key first-position/last-value alias traversal are preserved. Exact decimal truthiness/equality and arbitrary integers do not reproduce Python float rounding/underflow or configurable CPython integer digit limits; the Go Unicode16 decimal table is an explicit version boundary.
- The selected Cinder microversion is retained, including explicit list overrides. Source Backup max3.64 limits absent-default negotiation only; Go performs no automatic negotiation, cap, upgrade or v2 selection. Selected and operation-owned headers are captured and validated while provider auth/reauth/retry tokens remain live.
- Captured current or explicit location is recomputed from the actual backup project alias and literal availability_zone. Raw location and caller zone cannot replace the owned scope. This connected cloud-helper model is distinct from standalone/disconnected Resource constructor behavior.
- Go validates consumed UTF8 JSON and safe literal member/continuation URLs, same origin/escaped path and canonical full-URL cycles. Source relative /v joining, query-only physical duplication and same-marker cycle rules are bounded by those deliberate safety rules. Robust HTTP Link URL support repairs the source uri/url mismatch.
- Direct source snapshots precede original callbacks; Connection originals run once before only cached Cinder v3 selection, with location captured after originals before the getter. Fixed bodyless requests, sticky source/cause guards, independently owned phase proof and atomic logical publication are Go extensions; caller-configured native retry remains live and the SDK adds no orchestration replay.
- Accepted response Read/Close/context/source faults retain their own physical evidence. Ordinary member native400/403/404 has a scoped body-fault observer so only fault-free original concrete rejections permit lookup fallback, without admitting those statuses or fabricating Observed. Rejected list responses and other rejected member statuses retain Gophercloud v2.15.0 discarded unexpected-body Read/Close behavior; this is not a universal rejection-I/O observer.
- Local descriptor/predicate/selection errors keep completed independently admitted proof and do not borrow a successful response as a new local failure. Source probes use explicit AST/session/response/location stand-ins and are source-only, not authenticated Python/OpenStack runtime evidence.
- Search completes a queryless detailed list before identifier and recursive mapping/expression filters. Finite JSON filter values replace arbitrary Python filters; JMESPath output is returned as owned arbitrary JSON with no invented association to physical backup rows. Object key traversal is deterministic where JMESPath objects are unordered.
- Source identifier stringification/fnmatch and ordered mapping lazy errors are preserved within the selected JSON domain. List server-query kwargs, inherited list controls and the source generator-expression defect are not silently reapplied to cloud Search.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeBackupSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeBackupSearchLocation`

- Omitted or JSONnull filter uses ordinary exact member-first find. A physically absent member ID seeds only normalized logical id with RequestedID/SeededID evidence; raw Backup fields stay actual. Explicit null, false, zero, empty/container or a different returned ID is preserved passively. An actual empty object Resource remains present, distinct from completed lookup absence, which returns nil Value and Backup.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetKeepsActualIDsAndSeparatesSeedFromPhysicalBody`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetAllowsPartialFlatMalformedAndSub400MemberResponses`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelMemberSeedsOnlyAbsentInheritedIDAndKeepsPhysicalBytes`, `connection_volume_backups_test.go:TestConnectionVolumeBackupMemberSeedsOnlyLogicalMissingIDAndCleanFallbackKeepsListProof`, `blockstorage/volume_backup_read_isolation_test.go:TestVolumeBackupReadSuccessfulAbsenceIsDistinctFromPresentEmptyMember`.
- Only fault-free original direct concrete native400/403/404 permits detailed name-hinted fallback. Fallback treats ID/name literally, including wildcard or unsafe-member names, and preserves actual response identity. A second exact match raises ambiguity before unused bad tail/link construction; a unique result must exhaust all pages, and later errors never publish it or borrow suppressed member proof.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindCleanNativeFallbackIsExactAndUsesActualResponseID`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindUnsafeLiteralAndWildcardNamesNeverBecomeSearchPatterns`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFindAmbiguityPrecedesUnusedBadTailAndUniqueMustExhaust`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadFallbackLaterNativeFailureNeverBorrowsSuppressedMemberProof`, `connection_volume_backups_test.go:TestConnectionVolumeBackupMemberSeedsOnlyLogicalMissingIDAndCleanFallbackKeepsListProof`, `blockstorage/cinder_member_rejection_test.go:TestCinderMemberCleanCompatibleRejectionsRetainFallbackAndByIDContrast`.
- Every explicitly nonnull filter, including false, zero, empty string/object/array, follows complete queryless Search plus first-selection behavior; only omitted/null follows ordinary find. This dispatch preserves full-list errors and source deprecation path semantics without warning emission, shortcut IDs, glob ordinary names or fabricated member observation.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetEveryNonNullFalseyFilterUsesFullSearchWhileNullFinds`, `connection_volume_backups_test.go:TestConnectionVolumeBackupSelectionErrorRetainsBackupIdentityWithoutChangingSnapshotErrorShape`.
- Arbitrary expression output uses source JSON-domain truthiness, length and integer-index first-selection without retesting a selected falsey value. Multiple output produces distinct VolumeBackupSelectionError carrying NameOrID and Length; projection never gains invented raw Backup association, and the existing Snapshot selection-error shape is unchanged.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetArbitraryExpressionRetainsNoBackupAssociation`, `connection_volume_backups_test.go:TestConnectionVolumeBackupSelectionErrorRetainsBackupIdentityWithoutChangingSnapshotErrorShape`, `internal/cloudfilter/results_test.go:TestCloudFilterFirstFollowsTruthyLengthAndIntegerIndexWithoutRetestingSelectedFalsyValue`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`.
- Original admitted100..399 member responses allow canonical backup object, flat object and validUTF8 empty/malformed body tolerance. Parsed wrong shape/null envelope, nonobject valid JSON or invalidUTF8 is terminal with the actual current response. List-constructor controls are not applied to member response fields; absent-ID seeding stays separate from physical proof.
  Tests: `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetAllowsPartialFlatMalformedAndSub400MemberResponses`, `blockstorage/volume_backup_read_selection_test.go:TestVolumeBackupReadGetValidShapeAndUTF8FailuresRemainTerminalPhysicalErrors`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelListConstructorControlsDoNotBecomeMemberResponseControls`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelPhysicalAndLocalLocationFailuresAreAtomicAndOutputIsOwned`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelMemberSeedsOnlyAbsentInheritedIDAndKeepsPhysicalBytes`.
- Selected logical member/fallback values use the same eager nullable24 Backup descriptors, ordinary Boolean/list/int/dict conversions and parsed alias order as List/Search. Captured project scope and literal availability_zone determine location independently of caller/raw location. Physical raw resource, normalized value and admitted response proof own distinct storage.
  Tests: `internal/cloudsnapshot/backup_model_test.go:TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder`, `internal/cloudsnapshot/backup_model_test.go:TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone`, `connection_volume_backups_test.go:TestConnectionVolumeBackupExplicitLocationOwnsEveryFieldAndUsesBackupProjectAndZone`, `blockstorage/volume_backup_read_contract_test.go:TestVolumeBackupReadListOwnsAllTwentyFourFieldsAndRawIndependentProofs`.
- Direct source facts and ordinary headers are captured before original options; owned slices/retained pointers cannot replace prepared policy. Connection originals run once before cached Cinder v3 and captured location before the getter. Invalid/context/callback/getter and reached local location failures retain operation/custom causes; accepted member I/O keeps current proof while failed selection remains atomic.
  Tests: `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadOriginalOptionsOwnEveryPreparedValueAndSourceHeaders`, `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `connection_volume_backups_test.go:TestConnectionVolumeBackupOriginalOptionsRunOnceBeforeCachedCinderForEveryEntry`, `connection_volume_backups_test.go:TestConnectionVolumeBackupPreflightAndGetterFailuresKeepBackupOperationAndJoinedCauses`, `connection_volume_backups_test.go:TestConnectionVolumeBackupCapturedOptionSliceAndRetainedPointersCannotReplacePreparedPolicy`, `connection_volume_backups_test.go:TestConnectionVolumeBackupAcceptedIOJoinsReadCloseCancellationAndSourceDriftWithCurrentProof`, `connection_volume_backups_test.go:TestConnectionVolumeBackupLaterFailureRetainsPagesWithoutBorrowingEarlierMemberEvidence`.
- Fixed original bodyless member GET, native live reauth/caller-configured retry and original status policy remain owned. Expanded native OkCodes400/403/404 or callback/transport wrappers are terminal and never compatible absence. The scoped native400/403/404 body observer preserves rejected Read/Close faults and native causes without fabricated admitted Observed/Pages; clean native retries may still continue under guarded source/auth policy.
  Tests: `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `blockstorage/backup_read_transport_test.go:TestVolumeBackupReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `blockstorage/cinder_member_rejection_test.go:TestCinderMemberCompatibleRejectionsKeepPureReadAndCloseFaultsTerminal`, `blockstorage/cinder_member_rejection_test.go:TestCinderMemberNativeRetryCannotSwallowRejectedIOOrReplaceOriginalStatus`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin cloud helper is mapped with selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy, native typed leaf, v2 Backup, generic Resource/session/adapter and backup mutation/export declarations retain separate reviews; the full SDK remains incomplete.
- Python mutable kwargs/runtime objects map to owned Go request strings, nullable concrete With/Prepare policy, finite UTF8 JSON, raw-resource evidence and a separate normalized nullable24-field view. Go does not reproduce Resource identity, stale dict caches, session memoization, deprecated warning emission or arbitrary Python object coercion.
- Parsed JSON dictionary insertion order and duplicate-key first-position/last-value alias traversal are preserved. Exact decimal truthiness/equality and arbitrary integers do not reproduce Python float rounding/underflow or configurable CPython integer digit limits; the Go Unicode16 decimal table is an explicit version boundary.
- The selected Cinder microversion is retained, including explicit list overrides. Source Backup max3.64 limits absent-default negotiation only; Go performs no automatic negotiation, cap, upgrade or v2 selection. Selected and operation-owned headers are captured and validated while provider auth/reauth/retry tokens remain live.
- Captured current or explicit location is recomputed from the actual backup project alias and literal availability_zone. Raw location and caller zone cannot replace the owned scope. This connected cloud-helper model is distinct from standalone/disconnected Resource constructor behavior.
- Go validates consumed UTF8 JSON and safe literal member/continuation URLs, same origin/escaped path and canonical full-URL cycles. Source relative /v joining, query-only physical duplication and same-marker cycle rules are bounded by those deliberate safety rules. Robust HTTP Link URL support repairs the source uri/url mismatch.
- Direct source snapshots precede original callbacks; Connection originals run once before only cached Cinder v3 selection, with location captured after originals before the getter. Fixed bodyless requests, sticky source/cause guards, independently owned phase proof and atomic logical publication are Go extensions; caller-configured native retry remains live and the SDK adds no orchestration replay.
- Accepted response Read/Close/context/source faults retain their own physical evidence. Ordinary member native400/403/404 has a scoped body-fault observer so only fault-free original concrete rejections permit lookup fallback, without admitting those statuses or fabricating Observed. Rejected list responses and other rejected member statuses retain Gophercloud v2.15.0 discarded unexpected-body Read/Close behavior; this is not a universal rejection-I/O observer.
- Local descriptor/predicate/selection errors keep completed independently admitted proof and do not borrow a successful response as a new local failure. Source probes use explicit AST/session/response/location stand-ins and are source-only, not authenticated Python/OpenStack runtime evidence.
- Omitted or JSONnull filter uses exact ordinary find, with safe member-first lookup and a literal name-hinted fallback. Every nonnull filter, even falsey JSON, follows full Search plus source truthy/length/index selection; arbitrary expression output is not invented into a Resource list or raw-row association.
- Logical member lookup seeds only a physically absent ID; RequestedID and SeededID make that state explicit while raw Backup evidence remains actual. Explicit null, false, zero, empty/container or different returned IDs remain passive response values; Go does not require equality with the request or silently manufacture a physical ID.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"

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

func InspectPages(pages []*blockstorage.VolumeBackupsPage) {
    for _, page := range pages {
        fmt.Println("admitted backup page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Explain(err error) {
    if err == nil { return }
    var ambiguous *blockstorage.VolumeBackupSelectionError
    if errors.As(err, &ambiguous) {
        fmt.Println("filtered selection length:", ambiguous.Length)
    }
    fmt.Println("ambiguous:", errors.Is(err, resource.ErrAmbiguous), "error:", err)
}

func Inspect(result *blockstorage.GetVolumeBackupResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Backup != nil {
        owned := result.Backup.Clone()
        fmt.Println("actual wire ID JSON:", string(owned.Body["id"]))
        var projection struct {
            ID json.RawMessage `json:"id"`
            Name *string `json:"name"`
        }
        if err := owned.Decode(&projection); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("projected wire ID:", string(projection.ID))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw backups:", len(listed.Backups))

    searched, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized backups:", string(searched.Value))

    selected, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"})
    Inspect(selected) // Inspect actual proof before deciding what to do with err.
    Explain(err)
    if err != nil { return err }
    if selected.Value == nil {
        fmt.Println("completed lookup: no backup found")
    } else {
        fmt.Println("selected normalized JSON:", string(selected.Value))
    }
    return nil
}

func PreparedFirstPage(ctx context.Context, conn *sdk.Connection) error {
    calls := 0
    policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx,
        func(value *blockstorage.VolumeBackupListOpts) error {
            calls++
            return blockstorage.WithVolumeBackupListFilters(
                json.RawMessage(`{"status":"available"}`))(value)
        },
        blockstorage.WithVolumeBackupListDetailed(false),
        blockstorage.WithVolumeBackupListPagination(false),
        blockstorage.WithVolumeBackupListMaxItems(10),
        blockstorage.WithVolumeBackupListHeaders(map[string]string{
            "X-Request-ID": "backup-read-example",
        }))
    if err != nil { return err }
    fmt.Println("original callback calls:", calls)
    result, err := conn.ListVolumeBackups(ctx,
        blockstorage.WithVolumeBackupListOptions(policy))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeBackups(ctx,
        blockstorage.SearchVolumeBackupsRequest{},
        blockstorage.WithVolumeBackupSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value), "raw associations:", len(result.Backups))
    return nil
}

func FilteredGet(ctx context.Context, conn *sdk.Connection, name string) error {
    // Explicit {} chooses complete Search instead of ordinary GET-first find.
    result, err := conn.GetVolumeBackup(ctx,
        blockstorage.GetVolumeBackupRequest{NameOrID: name},
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{}`)))
    Inspect(result)
    Explain(err)
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    cloud, region, projectName := "owned-cloud", "RegionOne", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, RegionName: &region,
        Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`), Name: &projectName},
    }
    listed, err := blockstorage.ListVolumeBackups(ctx, cinder,
        blockstorage.WithVolumeBackupListLocation(location))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx,
        blockstorage.WithVolumeBackupSearchLocation(location),
        blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if err != nil { return err }
    searched, err := blockstorage.SearchVolumeBackups(ctx, cinder,
        blockstorage.SearchVolumeBackupsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeBackupSearchOptions(policy))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    selected, err := blockstorage.GetVolumeBackup(ctx, cinder,
        blockstorage.GetVolumeBackupRequest{NameOrID: "nightly"},
        blockstorage.WithVolumeBackupSearchLocation(location))
    Inspect(selected)
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="my-cloud")
backups = conn.list_volume_backups(
    detailed=True,
    filters={"status": "available", "description": "nightly"},
)
matches = conn.search_volume_backups("nightly*", {"status": "available"})
backup = conn.get_volume_backup("nightly")
filtered_backup = conn.get_volume_backup("nightly", filters={})
names = conn.search_volume_backups(filters="[].name")
```

## References

- [List](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L733)
- [Search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L890)
- [Get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L668)
- [Proxy.backups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1631)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L540)
- [v3 Backup descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L26)
- [Requests parser](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Microversions](../docs/microversions.md)
- [V3](v3/README.md)
