# Volume Snapshots

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/list_volume_snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListVolumeSnapshots`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ListVolumeSnapshots`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeSnapshotListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListDetailed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListPagination`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListExpression`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListAllowUnknownParams`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotListConflictingAttrs`

- Cloud List defaults detailed/paginated true and reads /snapshots/detail to completion; explicit detailed false selects /snapshots and pagination false selects one page. Empty canonical snapshots[] succeeds as [] with a nonnil empty row slice and stops before unused continuation. No default page limit/maximum/version/expression or header beyond JSON Accept is invented.
  Tests: `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListDefaultDetailedBasicAndCanonicalEmptyEOF`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadZeroMaximumIsUnlimitedAndPaginationFalseStopsFirstPage`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPreparePreservesOmissionAndExplicitFalseZeroEmptyPolicies`.
- List separates the v3 QueryMapping server fields from ordered known Body-only local predicates; unknown keys and wire-only Body aliases are source-dropped. Proxy-bound all_projects uses truthiness, server alias precedence stays canonical, and late conflicting all_projects bypasses Proxy so raw false remains a query value.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListCompileSplitsServerQueryFromOrderedBodyPredicates`, `internal/cloudsnapshot/query_test.go:TestSnapshotListAllProjectsIsBoundBeforeLateAliasOverlay`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListSeparatesServerFiltersLocalSubsetsAndUnknownFields`.
- Resource controls use the final raw/conflicting overlay with typed controls taking precedence; Proxy paginated/expression were already bound and late collisions cannot be suppressed. Initial details/base_path/resource_type/self and late paginated/base_path/session/cls collisions fail locally. Late unknown route/expression controls and allow_unknown follow the source no-op domain.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListTypedControlsOverrideFinalRawValuesButKeepProxyStage`, `internal/cloudsnapshot/query_test.go:TestSnapshotListControlCollisionsDependOnActualBindingStage`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotRawListBindingCollisionsFailBeforeServiceSelection`.
- Falsey raw Filters are empty; truthy nonmapping, malformed raw bytes and unsupported dynamic controls reject atomically with ErrInvalidOption. Nil pointer omission differs from explicit invalid nil/empty RawMessage. Captured options/factories own all raw filters/conflicting attributes/headers/pointers and prepared policies are reusable.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListCompileRejectsMalformedRawAndUnsupportedDynamicControlsAtomically`, `internal/cloudsnapshot/query_test.go:TestSnapshotListCompiledPolicyOwnsRawQueryLocalControlsAndHeadersIndependently`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotCompleteListFactoryOwnsRawHeadersPointersAndReturnedPolicies`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotRawFactoriesPreserveExactJSONAndNilVersusEmptyBytes`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotIndividualListFactoriesRemainIndependentAndLaterOptionsOverride`.
- Query encoding preserves raw ordinary Requests two-level iterable expansion, ordered object-key iteration, null omission, booleans/arbitrary integers/parsed floats and deeper Python repr without partial values on malformed input. Typed controls and raw filters serialize once to one physical query; repeated value order remains.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListQueryEncoderUsesTwoIterableLevelsAndFailsWithoutPartialValues`, `internal/cloudfilter/query_test.go:TestRequestQueryValuesMatchesRequestsTwoLevelEncoding`, `internal/cloudfilter/query_test.go:TestRequestQueryValuesKeepsObjectOrderAndUsesReprOnlyAfterDoseqExpansion`.
- Ordered local dictionary predicates preserve lazy missing=None, falsey actual rejection even for expected{}, recursive subset matching and Python boolean/number equality. Only reached keys/shapes are consumed; guards and context interrupts return no partial selection. Eager row descriptor construction happens before local exclusion.
  Tests: `internal/cloudsnapshot/local_filter_test.go:TestSnapshotLocalFiltersUseMissingNoneTruthyDictionaryAndExactLeafEquality`, `internal/cloudsnapshot/local_filter_test.go:TestSnapshotLocalFiltersRetainParsedOrderAndDoNotConsumeUnusedShapeErrors`, `internal/cloudsnapshot/local_filter_test.go:TestSnapshotLocalFiltersOwnInputsBeforeGuardAndJoinContextWithSourceFailureAtomically`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Canonical snapshots envelope is required; a nonarray payload wraps once as one source row. Full response JSON is parsed before row consumption; consumed nonobject/invalidUTF8/descriptor rows fail with current physical proof. self is source-discarded but remains raw evidence; list connection/microversion/_synchronized key presence including null collides, while member response fields do not become runtime controls.
  Tests: `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListWrapsCanonicalSingletonAndRejectsInvalidCollections`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListConstructorControlsArePresenceSensitiveAndSelfIsIgnored`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelListConstructorCollisionsDoNotBecomeMemberResponseControls`.
- max_items counts consumed raw rows including locally excluded rows; false/zero/null is unlimited, bool true counts one, fractional values compare numerically and negative maximum makes first HTTP/full JSON parse before stopping before first row. A max reached before same-page tail avoids its row errors; at page end it still consumes continuation/next response before the next raw-row check.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListMaximumKeepsBooleanFractionNegativeAndHugeExponentCounterSemantics`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadMaxItemsCountsExcludedRowsAndStopsBeforeUnusedTail`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadMaxItemsAtPageBoundaryStillFetchesAndParsesNextPage`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadNegativeMaximumParsesResponseWithoutConsumingRows`.
- Pager selects top links by key presence before snapshots_links, consumes rel/href lazily, and stops at the first exact next with present href even falsey before top next/HTTP fallback. Consumed malformed prefix fails; unused tail is ignored. Snapshot does not add the separate Keystone links.next dictionary repair.
  Tests: `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerBodyLinksPrecedePluralNextAndHeaderWithoutDictionaryRepair`, `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerConsumesMalformedPrefixButStopsAtFirstPresentHrefEvenFalsey`.
- Inherited HTTP Link url support is an explicit Go parser repair: quoted commas and last case-insensitive multi-relation next are supported with lazy validation. Continuation removes old marker/limit, drops advertised blanks, merges remaining keys once and keeps omitted nonpagination raw values. Truthy initial limit synthesizes a next request even on a short page from the last consumed untyped ID.
  Tests: `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerHTTPLinkRepairSupportsQuotedCommasLastNextAndLazyValidation`, `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerContinuationKeepsRawQueryAndDropsOnlyOldPaginationAndAdvertisedBlanks`, `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerTruthyInitialLimitSynthesizesShortPageAndUsesRawLastID`, `blockstorage/snapshot_read_pagination_test.go:TestVolumeSnapshotReadInitialLimitFollowsShortPageExcludedRawDictionaryMarker`.
- Same-marker equality retains parsed query array shape and Python bool/number equality. Complete canonical URL cycle checks preserve repeated-value order and reject origin/escaped path/traversal/userinfo/fragment/nonUTF8 continuation before resend. Pager owns body/query/next inputs and returns atomic failures.
  Tests: `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerAdvertisedMarkerEqualityKeepsParseQSArrayShape`, `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerPageKeyCanonicalizesQueryButRejectsOriginEscapedPathAndTraversal`, `internal/cloudsnapshot/pager_test.go:TestSnapshotPagerOwnsInputsAndReturnsAtomicFailuresWithoutAnInventedResource`, `blockstorage/snapshot_read_pagination_test.go:TestVolumeSnapshotReadCanonicalURLCycleWithoutMarkerEqualityKeepsTwoPagesAtomically`.
- Explicit List expression repairs the pinned Proxy generator path by evaluating the full materialized normalized list after HTTP. Raw falsey legacy expression skips evaluation, but typed empty expression still reaches the engine and errors after completed pages. Arbitrary finite JSON expression results have no invented RawResource row association.
  Tests: `internal/cloudsnapshot/query_test.go:TestSnapshotListFalseyFiltersAndLegacyExpressionKeepExplicitProjectionSeparate`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListExpressionRepairsGeneratorInputAfterCompleteMaterialization`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadTypedEmptyListExpressionEvaluatesAfterCompleteList`.
- Per-call header/version policy owns canonical inputs, rejects invalid/reserved/inconsistent version headers and leaves selected native client/cached defaults unchanged. Explicit empty/symbolic microversions are preserved; no automatic3.65 negotiation or cap is performed by the supplied-client helper.
  Tests: `internal/cloudread/policy_test.go:TestCloudReadPolicyCanonicalOverridesOwnInputsAndKeepSelectedSource`, `internal/cloudread/policy_test.go:TestCloudReadPolicyExplicitMicroversionsKeepEmptyAndSymbolicValues`, `internal/cloudread/policy_test.go:TestCloudReadPolicyRejectsInvalidHeadersAndVersionsAtomically`, `internal/cloudread/policy_test.go:TestCloudReadPolicyVersionOverrideRequiresConsistentCapturedHeaders`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotListControlsArePerCallAndDoNotChangeCachedDefaults`.
- Normalized Snapshot contains all16 actual nullable fields, retaining arbitrary JSON for untyped fields and excluding unknown/case-variant aliases from logical output. No implicit false/time/string/int native projection replaces null. RawResource retains actual unknown fields and number spelling independently.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelHasSixteenNullableFieldsAndPreservesUntypedValues`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelBoolStrIsNullableWithoutAnImplicitFalseDefault`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`.
- Ordinary descriptor conversion is eager at Resource constructor/cache refresh: strict nullable BoolStr, Python bool/arbitrary integer preservation, parsed finite IEEE float truncation and Unicode decimal-string integer conversion, nullable ordinary metadata dictionary. Parsed duplicate keys keep first insertion position with final value; selected alias wins in dictionary traversal order and discarded alias is not converted.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelIntegerAndMetadataDescriptorsMatchOrdinaryResourceConversion`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelAliasesUseParsedDuplicateCollapseAndFirstInsertionOrder`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListSelectedAliasConvertsOnlyFinalParsedDictionaryValue`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Snapshot location is recomputed from the owned configured/authenticated scope and actual project descriptor, including empty/location-only rows. Falsey/equal project retains configured names; truthy foreign project keeps raw ID and clears names/domains. Zone is always null because Snapshot has no zone descriptor. Raw location/availability_zone cannot replace it; caller location errors remain local without fabricated response errors. Owned Location is validated before first HTTP, and Connection validates it before its Cinder getter.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelRecomputesLocationFromProjectAndAlwaysUsesNullZone`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelRejectsPhysicalShapeAndLocalLocationFailuresAtomicallyAndOwnsResult`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotRecordedScopeIsCapturedAfterOptionsBeforeGetterAndRetainedAcrossPages`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotExplicitLocationIsOwnedAndZoneIgnoredWithoutChangingDefaults`.
- Prepare/factories own full raw Location, pointer and collection policy; complete options replace earlier policy. Original callbacks run exactly once in captured slice order with independent snapshots after each callback. Nil context/options, canceled custom cause and callback failures stop without partial prepared policy or HTTP.
  Tests: `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotLocationFactoriesOwnEveryNullableComponentAndAreReusable`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotWholeOptionsReplaceEarlierPolicyRatherThanMerging`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareRejectsNilAndCanceledContextBeforeOriginals`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareNilOptionAndCallbackFailuresStopWithoutPartialPolicy`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and retain sticky source violations; ordinary headers remain snapshots while auth tokens stay live. Connection executes originals once before only cached Cinder v3 selection and captures current/explicit location before getter. Local callback/getter/preflight errors retain outer operation and custom causes without inventing physical proof.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalOptionsOwnEachSnapshotAndSourceOrdinaryHeaders`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalCallbackFailureJoinsSourceAndContextBeforeHTTP`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceChangeStaysStickyBeforeAnyPolicyCommit`, `internal/cloudread/policy_test.go:TestCloudReadPolicyCancellationRetainsCauseAndIsAtomic`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause`.
- Each admitted physical page/member owns body/header/status; normalized Value, raw row and error proof own independent state. Read/Close/source drift/custom cancellation keep the actual current phase and all causes while logical outputs remain unpublished. Rejected later/blocked requests retain only earlier completed pages and never borrow an earlier successful or suppressed member response.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedSourceFactsAreFrozenThroughBodyClose`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadBlockedContinuationCancellationKeepsOnlyCompletedPage`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation`.
- Native reauth/retry remains live and bounded in fixtures while method/fixed route/query/bodyless decoder/body retention/original100..399 policy and source remain guarded. Expanded native400/403/404 is terminal with actual native status/body/header causes, not compatible fallback or admitted ResponseError. Redirect cannot forward credentials to an altered target. No SDK orchestration replay/wait/action/rollback is added.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeRetryCannotReplaceBodyDecoderRetentionOrSource`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadRedirectCannotSendTokenToChangedFixedTarget`, `internal/cloudread/policy_test.go:TestCloudReadPolicyOwnedWirePolicyKeepsLiveTokenAcrossNativeRetry`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceMutationInNativeRetryStopsBeforeResend`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is mapped using selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter/cache declarations retain separate reviews; full SDK is not complete.
- Python default/overloaded/runtime arguments map to explicit Go request types and owned concrete With/Prepare options, not caller-implemented interface builders. Go accepts finite ordinary JSON and typed controls rather than arbitrary mutable Python Resource/custom runtime objects.
- Logical normalized16-field nullable values, physical raw Resource/Observed/Pages and error proof are independently owned Go representations. Unknown raw fields and number spelling are evidence extensions; mutable Resource identity, stale dict cache and session memoization are not reproduced.
- Ordinary source descriptor behavior and parsed dictionary duplicate/alias order are preserved. Go Unicode16 decimal tables and arbitrary integer policy do not reproduce configurable CPython integer digit limits; exact decimal truthiness/equality differs from Python float rounding/underflow. Untyped timestamps/IDs/names stay JSON rather than native typed leaf extraction.
- Connected Snapshot location is recomputed including empty/location-only rows, using explicit/current recorded scope and raw project descriptor; zone remains null. Direct clients recover recorded auth without fabricating configured names. Go owns snapshots and rejects invalid policy locally rather than emulating dynamic Connection/session objects.
- Go captures original source facts and ordinary headers before originals on direct calls; Connection prepares originals before cached Cinder getter. Native auth remains live, sticky source/fixed request/cause guards and atomic logical publication are explicit Go safety/proof extensions; no SDK replay/wait/rollback is introduced.
- Source validHTTP100..399 and UTF8-valid empty/malformed member JSON tolerance are preserved instead of narrowing to native200. Go additionally requires complete valid UTF8 and JSON/object/envelope/descriptor shape on consumed data, preserves native rejected evidence and prevents expanded native>=400 from becoming success or fallback.
- Selected native microversion is preserved, including symbolic/empty settings when supplied. Source Snapshot3.65 limit only caps absent-default server-discovery negotiation; Go does not automatically negotiate/cap/upgrade a supplied client. Explicit Connection version/range negotiation remains independent.
- List retains source known-query/body-local split and inherited binding phases, including falsey filters, unknown drop, late conflicting attrs, all_projects aliases and counter/pagination timing. Dynamic controls are explicit Go bool/number/string/string-header JSON domains; allow_unknown remains a no-op and nil header omission does not claim an explicit nil-clear feature.
- Inherited List expression is a deliberate generator repair: source Proxy passes a Resource generator directly to JMESPath, whereas Go evaluates completed normalized finite JSON after HTTP. Raw falsey legacy expression skips evaluation; typed empty expression reaches engine and can error after complete pages. Timing/results differ from pinned legacy generator path.
- Paging keeps source lazy body links, marker/limit pop, blank-drop, raw ID synthesis and same-marker shape equality. Go repairs HTTP Link uri/url mismatch, resolves against effective URL, serializes canonical query once and guards origin/escaped collection path/complete URL cycles. It does not add Keystone links.next dictionary support to Snapshot.
- Requests two-level JSON query expansion and deeper ordered Python repr are audited compatibility mappings. Requests/urllib/CPython float representation are dependencies not pinned by SDK commit; Go canonical key ordering and no duplicate query-only serialization are declared transport differences.

### `python:cloud/_block_storage/BlockStorageCloudMixin/search_volume_snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchVolumeSnapshots`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SearchVolumeSnapshots`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeSnapshotSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchExpression`

- Cloud Search first completes ordinary detailed queryless List before name/ID or Filters selection, preserving source order/duplicates and all reached row/page/schema failures. A name/filter that excludes a malformed later row cannot hide eager constructor errors. Search status/project filters do not become server query.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchCompletesQuerylessListBeforeSelectionAndReturnsArbitraryExpression`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchDoesNotHideExcludedMalformedLaterRows`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Identifier selection retains per-row exact ID/name or case-sensitive CPython glob matching, slash/newline/rune/classes/reversed ranges/unmatched bracket/literal backslash behavior and ordinary JSON-domain Python str/repr for nonstring fields. Empty input means no identifier restriction; missing/null differs from empty string.
  Tests: `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder`, `internal/cloudfilter/identifier_test.go:TestCloudFilterGlobUsesCPythonRuneDotAllClassesAndLiteralBackslashes`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/identifier_test.go:TestCloudFilterMissingAndNullIdentifiersDifferFromEmptyStringsAndRemainCaseSensitive`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchCompletesQuerylessListBeforeSelectionAndReturnsArbitraryExpression`.
- Filters validation remains after the identifier phase. Ordered mapping keys use duplicate first-position/final-value semantics and lazy nested subset traversal; reached unknown keys fail, unmatched earlier predicates or empty identifier selection can leave unusable mapping fields unconsumed. Falsey Filters select all identifier matches; raw malformed Filters still fail locally.
  Tests: `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNonmappingFilterShapeIsLazyAfterIdentifierAndActualTruthiness`, `internal/cloudfilter/mapping_test.go:TestCloudFilterInvalidJSONIsLocalAtomicAndStillValidatedAfterEmptyIdentifierSelection`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchLocalFilterErrorsDoNotBorrowAdmittedHTTPStatus`.
- Leaf comparisons preserve deep exact JSON shape and Python bool/number equality with arbitrary precision decimal policy. JMESPath nonempty expression runs the full engine on normalized complete list even empty selection, supports arbitrary JSON Value and has no row indices/RawResource projection; parser/runtime errors remain local with completed physical pages.
  Tests: `internal/cloudfilter/mapping_test.go:TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionsStillParseEmptySelectionAndKeepRuntimeFailuresAtomic`, `internal/jmespath/engine_test.go:TestSearchReferenceAndDeclaredBoundaries`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchCompletesQuerylessListBeforeSelectionAndReturnsArbitraryExpression`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchLocalFilterErrorsDoNotBorrowAdmittedHTTPStatus`.
- Search options/factories independently own raw filters, literal expression JSON and full Location; originals are reusable and execute once. A later callback/source/context failure cannot change a retained prepared policy or partially publish selected results.
  Tests: `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotSearchExpressionFactoryEncodesLiteralAndSearchOptionsOwnItsInputs`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy`, `internal/cloudfilter/results_test.go:TestCloudFilterSelectOwnsViewsFiltersIndicesAndResultsBeforeGuardCallbacks`, `internal/cloudfilter/results_test.go:TestCloudFilterGuardInterruptsTraversalAndRetainsCancellationCauseWithoutPartialSelection`.
- Normalized Snapshot contains all16 actual nullable fields, retaining arbitrary JSON for untyped fields and excluding unknown/case-variant aliases from logical output. No implicit false/time/string/int native projection replaces null. RawResource retains actual unknown fields and number spelling independently.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelHasSixteenNullableFieldsAndPreservesUntypedValues`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelBoolStrIsNullableWithoutAnImplicitFalseDefault`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`.
- Ordinary descriptor conversion is eager at Resource constructor/cache refresh: strict nullable BoolStr, Python bool/arbitrary integer preservation, parsed finite IEEE float truncation and Unicode decimal-string integer conversion, nullable ordinary metadata dictionary. Parsed duplicate keys keep first insertion position with final value; selected alias wins in dictionary traversal order and discarded alias is not converted.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelIntegerAndMetadataDescriptorsMatchOrdinaryResourceConversion`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelAliasesUseParsedDuplicateCollapseAndFirstInsertionOrder`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListSelectedAliasConvertsOnlyFinalParsedDictionaryValue`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Snapshot location is recomputed from the owned configured/authenticated scope and actual project descriptor, including empty/location-only rows. Falsey/equal project retains configured names; truthy foreign project keeps raw ID and clears names/domains. Zone is always null because Snapshot has no zone descriptor. Raw location/availability_zone cannot replace it; caller location errors remain local without fabricated response errors. Owned Location is validated before first HTTP, and Connection validates it before its Cinder getter.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelRecomputesLocationFromProjectAndAlwaysUsesNullZone`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelRejectsPhysicalShapeAndLocalLocationFailuresAtomicallyAndOwnsResult`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotRecordedScopeIsCapturedAfterOptionsBeforeGetterAndRetainedAcrossPages`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotExplicitLocationIsOwnedAndZoneIgnoredWithoutChangingDefaults`.
- Prepare/factories own full raw Location, pointer and collection policy; complete options replace earlier policy. Original callbacks run exactly once in captured slice order with independent snapshots after each callback. Nil context/options, canceled custom cause and callback failures stop without partial prepared policy or HTTP.
  Tests: `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotLocationFactoriesOwnEveryNullableComponentAndAreReusable`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotWholeOptionsReplaceEarlierPolicyRatherThanMerging`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareRejectsNilAndCanceledContextBeforeOriginals`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareNilOptionAndCallbackFailuresStopWithoutPartialPolicy`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and retain sticky source violations; ordinary headers remain snapshots while auth tokens stay live. Connection executes originals once before only cached Cinder v3 selection and captures current/explicit location before getter. Local callback/getter/preflight errors retain outer operation and custom causes without inventing physical proof.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalOptionsOwnEachSnapshotAndSourceOrdinaryHeaders`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalCallbackFailureJoinsSourceAndContextBeforeHTTP`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceChangeStaysStickyBeforeAnyPolicyCommit`, `internal/cloudread/policy_test.go:TestCloudReadPolicyCancellationRetainsCauseAndIsAtomic`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause`.
- Each admitted physical page/member owns body/header/status; normalized Value, raw row and error proof own independent state. Read/Close/source drift/custom cancellation keep the actual current phase and all causes while logical outputs remain unpublished. Rejected later/blocked requests retain only earlier completed pages and never borrow an earlier successful or suppressed member response.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedSourceFactsAreFrozenThroughBodyClose`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadBlockedContinuationCancellationKeepsOnlyCompletedPage`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation`.
- Native reauth/retry remains live and bounded in fixtures while method/fixed route/query/bodyless decoder/body retention/original100..399 policy and source remain guarded. Expanded native400/403/404 is terminal with actual native status/body/header causes, not compatible fallback or admitted ResponseError. Redirect cannot forward credentials to an altered target. No SDK orchestration replay/wait/action/rollback is added.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeRetryCannotReplaceBodyDecoderRetentionOrSource`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadRedirectCannotSendTokenToChangedFixedTarget`, `internal/cloudread/policy_test.go:TestCloudReadPolicyOwnedWirePolicyKeepsLiveTokenAcrossNativeRetry`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceMutationInNativeRetryStopsBeforeResend`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is mapped using selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter/cache declarations retain separate reviews; full SDK is not complete.
- Python default/overloaded/runtime arguments map to explicit Go request types and owned concrete With/Prepare options, not caller-implemented interface builders. Go accepts finite ordinary JSON and typed controls rather than arbitrary mutable Python Resource/custom runtime objects.
- Logical normalized16-field nullable values, physical raw Resource/Observed/Pages and error proof are independently owned Go representations. Unknown raw fields and number spelling are evidence extensions; mutable Resource identity, stale dict cache and session memoization are not reproduced.
- Ordinary source descriptor behavior and parsed dictionary duplicate/alias order are preserved. Go Unicode16 decimal tables and arbitrary integer policy do not reproduce configurable CPython integer digit limits; exact decimal truthiness/equality differs from Python float rounding/underflow. Untyped timestamps/IDs/names stay JSON rather than native typed leaf extraction.
- Connected Snapshot location is recomputed including empty/location-only rows, using explicit/current recorded scope and raw project descriptor; zone remains null. Direct clients recover recorded auth without fabricating configured names. Go owns snapshots and rejects invalid policy locally rather than emulating dynamic Connection/session objects.
- Go captures original source facts and ordinary headers before originals on direct calls; Connection prepares originals before cached Cinder getter. Native auth remains live, sticky source/fixed request/cause guards and atomic logical publication are explicit Go safety/proof extensions; no SDK replay/wait/rollback is introduced.
- Source validHTTP100..399 and UTF8-valid empty/malformed member JSON tolerance are preserved instead of narrowing to native200. Go additionally requires complete valid UTF8 and JSON/object/envelope/descriptor shape on consumed data, preserves native rejected evidence and prevents expanded native>=400 from becoming success or fallback.
- Selected native microversion is preserved, including symbolic/empty settings when supplied. Source Snapshot3.65 limit only caps absent-default server-discovery negotiation; Go does not automatically negotiate/cap/upgrade a supplied client. Explicit Connection version/range negotiation remains independent.
- Search preserves complete detailed list-before-selection and the distinct full _filter_list path. NameOrID is a Go string (empty means omitted); nonstring raw row identifiers use ordinary JSON Python str/repr compatibility, full case-sensitive fnmatch translation and source order rather than filesystem matching.
- The full in-repo JMESPath engine accepts UseNumber JSON and returns arbitrary finite JSON, not a promised Snapshot array. Selected Python jmespath1.0.1 reference pin is a validation dependency satisfying SDK>=0.9.0, not a new SDK source pin. Exact decimal, lexical object order, stricter malformed grammar and arbitrary nonJSON/runtime-object boundaries are declared in engine documentation.
- Lazy ordered mapping/shape errors and nested Python bool-number equality are retained without narrowing expression outputs to rows. Go explicit Value/RawResource association distinguishes ordinary selected row indices from arbitrary expression values; local errors do not borrow completed HTTP status.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeSnapshotSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotSearchExpression`

- Omitted or literal null Filters takes ordinary exact Find; every nonnull Filters including{}[]false0 and empty string takes complete cloud Search then the source truthy/len/index0 rule. JSON null is the only explicit null that restores Find; original slice policy is not reapplied during prepared Find.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadGetNonNullFalseyFiltersDispatchFullSearchButNullUsesFind`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy`.
- Safe literal identity first makes one bodyless member GET. Only original clean direct native400/403/404 enables detailed name-hinted fallback, with no trim/glob and exact string ID/name matching. Unsafe/path/space names use literal list query as a declared Go route boundary. Expanded status-policy/wrapped native failures and accepted schema/read errors are terminal.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadFindCleanFallbackUsesExactNameAndActualResponseIdentity`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadFindUnsafeAndWildcardNamesKeepExactLiteralSelection`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberSchemaAndUTF8ErrorsRemainPhysicalAndTerminal`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack`.
- A second exact match raises ambiguity immediately before unused row/link errors; one match must exhaust all pages successfully and cannot hide later descriptor/link/HTTP failure. Completed absence returns nil Value/Snapshot with actual list proof, clearing a suppressed rejected member observation before list processing.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadFindAmbiguityStopsBeforeUnusedBadTailButUniqueMustExhaust`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack`.
- Accepted member100..399 supports canonical snapshot object or flat object; UTF8-valid empty/malformed JSON tolerates response.json ValueError as a nullable logical Snapshot, including physical204. Valid null/scalar/array/wrong envelope, invalidUTF8 and reached descriptor failures are terminal physical errors; they do not trigger fallback.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberAcceptsFlatPartialEmptyAndMalformedUTF8JSON`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberSchemaAndUTF8ErrorsRemainPhysicalAndTerminal`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberUsesSourceSub400PolicyIncludingOpaqueEmpty204`.
- Missing member id retains source input seed only in logical Value and RequestedID/SeededID, without adding raw Body.id. Explicit null/false/zero/empty/container ID and mismatched actual string ID are retained without seed or request-ID equality constraint. List missing id is null with no seed.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberSeparatesSeededLogicalIDFromActualWireBody`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelMemberSeedsOnlyAbsentIDsWithoutChangingRawProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack`.
- Filtered Get allows arbitrary expression results: falsey overall result is absence; length>1 is ambiguity; remaining truthy shape must support integer index0. Selected false/zero/empty value is not retested as absence. Scalar/dictionary length/index errors stay local and no expression value gains a fabricated Snapshot association.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadGetArbitraryExpressionDoesNotInventSnapshotAssociation`, `internal/cloudfilter/results_test.go:TestCloudFilterFirstFollowsTruthyLengthAndIntegerIndexWithoutRetestingSelectedFalsyValue`, `internal/cloudfilter/results_test.go:TestCloudFilterPythonTruthyPreservesExactDecimalZeroPolicyAndNullableResults`.
- Get selection/filter/expression errors retain completed actual pages without borrowing their accepted status. Partial logical values never escape; cancellation after admitted member keeps its member error despite shared Find cancellation override, while fallback clears suppressed member error before subsequent list failure.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadSearchLocalFilterErrorsDoNotBorrowAdmittedHTTPStatus`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation`.
- Normalized Snapshot contains all16 actual nullable fields, retaining arbitrary JSON for untyped fields and excluding unknown/case-variant aliases from logical output. No implicit false/time/string/int native projection replaces null. RawResource retains actual unknown fields and number spelling independently.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelHasSixteenNullableFieldsAndPreservesUntypedValues`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelBoolStrIsNullableWithoutAnImplicitFalseDefault`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`.
- Ordinary descriptor conversion is eager at Resource constructor/cache refresh: strict nullable BoolStr, Python bool/arbitrary integer preservation, parsed finite IEEE float truncation and Unicode decimal-string integer conversion, nullable ordinary metadata dictionary. Parsed duplicate keys keep first insertion position with final value; selected alias wins in dictionary traversal order and discarded alias is not converted.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelIntegerAndMetadataDescriptorsMatchOrdinaryResourceConversion`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelAliasesUseParsedDuplicateCollapseAndFirstInsertionOrder`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListSelectedAliasConvertsOnlyFinalParsedDictionaryValue`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Snapshot location is recomputed from the owned configured/authenticated scope and actual project descriptor, including empty/location-only rows. Falsey/equal project retains configured names; truthy foreign project keeps raw ID and clears names/domains. Zone is always null because Snapshot has no zone descriptor. Raw location/availability_zone cannot replace it; caller location errors remain local without fabricated response errors. Owned Location is validated before first HTTP, and Connection validates it before its Cinder getter.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelRecomputesLocationFromProjectAndAlwaysUsesNullZone`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelRejectsPhysicalShapeAndLocalLocationFailuresAtomicallyAndOwnsResult`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotRecordedScopeIsCapturedAfterOptionsBeforeGetterAndRetainedAcrossPages`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotExplicitLocationIsOwnedAndZoneIgnoredWithoutChangingDefaults`.
- Prepare/factories own full raw Location, pointer and collection policy; complete options replace earlier policy. Original callbacks run exactly once in captured slice order with independent snapshots after each callback. Nil context/options, canceled custom cause and callback failures stop without partial prepared policy or HTTP.
  Tests: `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotLocationFactoriesOwnEveryNullableComponentAndAreReusable`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotWholeOptionsReplaceEarlierPolicyRatherThanMerging`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareRejectsNilAndCanceledContextBeforeOriginals`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareNilOptionAndCallbackFailuresStopWithoutPartialPolicy`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and retain sticky source violations; ordinary headers remain snapshots while auth tokens stay live. Connection executes originals once before only cached Cinder v3 selection and captures current/explicit location before getter. Local callback/getter/preflight errors retain outer operation and custom causes without inventing physical proof.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalOptionsOwnEachSnapshotAndSourceOrdinaryHeaders`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalCallbackFailureJoinsSourceAndContextBeforeHTTP`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceChangeStaysStickyBeforeAnyPolicyCommit`, `internal/cloudread/policy_test.go:TestCloudReadPolicyCancellationRetainsCauseAndIsAtomic`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause`.
- Each admitted physical page/member owns body/header/status; normalized Value, raw row and error proof own independent state. Read/Close/source drift/custom cancellation keep the actual current phase and all causes while logical outputs remain unpublished. Rejected later/blocked requests retain only earlier completed pages and never borrow an earlier successful or suppressed member response.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedSourceFactsAreFrozenThroughBodyClose`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadBlockedContinuationCancellationKeepsOnlyCompletedPage`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation`.
- Native reauth/retry remains live and bounded in fixtures while method/fixed route/query/bodyless decoder/body retention/original100..399 policy and source remain guarded. Expanded native400/403/404 is terminal with actual native status/body/header causes, not compatible fallback or admitted ResponseError. Redirect cannot forward credentials to an altered target. No SDK orchestration replay/wait/action/rollback is added.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeRetryCannotReplaceBodyDecoderRetentionOrSource`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadRedirectCannotSendTokenToChangedFixedTarget`, `internal/cloudread/policy_test.go:TestCloudReadPolicyOwnedWirePolicyKeepsLiveTokenAcrossNativeRetry`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceMutationInNativeRetryStopsBeforeResend`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is mapped using selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter/cache declarations retain separate reviews; full SDK is not complete.
- Python default/overloaded/runtime arguments map to explicit Go request types and owned concrete With/Prepare options, not caller-implemented interface builders. Go accepts finite ordinary JSON and typed controls rather than arbitrary mutable Python Resource/custom runtime objects.
- Logical normalized16-field nullable values, physical raw Resource/Observed/Pages and error proof are independently owned Go representations. Unknown raw fields and number spelling are evidence extensions; mutable Resource identity, stale dict cache and session memoization are not reproduced.
- Ordinary source descriptor behavior and parsed dictionary duplicate/alias order are preserved. Go Unicode16 decimal tables and arbitrary integer policy do not reproduce configurable CPython integer digit limits; exact decimal truthiness/equality differs from Python float rounding/underflow. Untyped timestamps/IDs/names stay JSON rather than native typed leaf extraction.
- Connected Snapshot location is recomputed including empty/location-only rows, using explicit/current recorded scope and raw project descriptor; zone remains null. Direct clients recover recorded auth without fabricating configured names. Go owns snapshots and rejects invalid policy locally rather than emulating dynamic Connection/session objects.
- Go captures original source facts and ordinary headers before originals on direct calls; Connection prepares originals before cached Cinder getter. Native auth remains live, sticky source/fixed request/cause guards and atomic logical publication are explicit Go safety/proof extensions; no SDK replay/wait/rollback is introduced.
- Source validHTTP100..399 and UTF8-valid empty/malformed member JSON tolerance are preserved instead of narrowing to native200. Go additionally requires complete valid UTF8 and JSON/object/envelope/descriptor shape on consumed data, preserves native rejected evidence and prevents expanded native>=400 from becoming success or fallback.
- Selected native microversion is preserved, including symbolic/empty settings when supplied. Source Snapshot3.65 limit only caps absent-default server-discovery negotiation; Go does not automatically negotiate/cap/upgrade a supplied client. Explicit Connection version/range negotiation remains independent.
- Search preserves complete detailed list-before-selection and the distinct full _filter_list path. NameOrID is a Go string (empty means omitted); nonstring raw row identifiers use ordinary JSON Python str/repr compatibility, full case-sensitive fnmatch translation and source order rather than filesystem matching.
- The full in-repo JMESPath engine accepts UseNumber JSON and returns arbitrary finite JSON, not a promised Snapshot array. Selected Python jmespath1.0.1 reference pin is a validation dependency satisfying SDK>=0.9.0, not a new SDK source pin. Exact decimal, lexical object order, stricter malformed grammar and arbitrary nonJSON/runtime-object boundaries are declared in engine documentation.
- Lazy ordered mapping/shape errors and nested Python bool-number equality are retained without narrowing expression outputs to rows. Go explicit Value/RawResource association distinguishes ordinary selected row indices from arbitrary expression values; local errors do not borrow completed HTTP status.
- Omitted/literal null filters use ordinary exact Find; every nonnull falsey JSON uses complete Search and source truthy/len/index0 selection. Python deprecated warning is not emitted as a Go runtime warning. Arbitrary expression values/shape failures are not narrowed to resource-only outputs.
- Go NameOrID string and safe member route domain avoid source arbitrary Resource/object interpolation. Unsafe/path/whitespace names use literal name-hinted list and exact matching without trimming/glob. Only original direct native400/403/404 is compatible absence; wrappers/expanded policy remain terminal with native causes.
- Source missing member ID seed is explicit logical Value plus RequestedID/SeededID, never fabricated physical Body.id. Explicit null/arbitrary IDs and list missing-id null preserve actual values. Accepted member cancellation proof survives shared Find cancellation, and suppressed member proof is cleared before fallback failure.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_snapshot_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeSnapshotByID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeSnapshotByID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeSnapshotReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeSnapshotReadLocation`

- ByID performs only fixed literal member GET with no list/name fallback; source Proxy.get_snapshot raises rejected400/403/404 despite cloud docstring absent=None wording. Accepted actual arbitrary/mismatched ID is not checked against the requested safe path ID. Preflight uses the documented single unescaped UTF8 segment route domain and nil result before execution.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadByIDNeverFallsBackOnCleanNativeRejections`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberSeparatesSeededLogicalIDFromActualWireBody`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack`, `connection_snapshot_id_test.go:TestConnectionVolumeSnapshotByIDValidatesPathBeforeHTTPAndPreservesUnicodeProof`.
- Accepted member100..399 preserves canonical/flat/empty/malformed JSON-ValueError tolerance and source logical missing-ID seed independently of physical Body. Explicit null/false/zero/empty/container ID never seeds; null/scalar/array/envelope/invalidUTF8 and descriptor errors retain current admitted proof and publish no Snapshot/Value.
  Tests: `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberAcceptsFlatPartialEmptyAndMalformedUTF8JSON`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberSchemaAndUTF8ErrorsRemainPhysicalAndTerminal`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadMemberUsesSourceSub400PolicyIncludingOpaqueEmpty204`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelMemberSeedsOnlyAbsentIDsWithoutChangingRawProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack`.
- Normalized Snapshot contains all16 actual nullable fields, retaining arbitrary JSON for untyped fields and excluding unknown/case-variant aliases from logical output. No implicit false/time/string/int native projection replaces null. RawResource retains actual unknown fields and number spelling independently.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelHasSixteenNullableFieldsAndPreservesUntypedValues`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelBoolStrIsNullableWithoutAnImplicitFalseDefault`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`.
- Ordinary descriptor conversion is eager at Resource constructor/cache refresh: strict nullable BoolStr, Python bool/arbitrary integer preservation, parsed finite IEEE float truncation and Unicode decimal-string integer conversion, nullable ordinary metadata dictionary. Parsed duplicate keys keep first insertion position with final value; selected alias wins in dictionary traversal order and discarded alias is not converted.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelIntegerAndMetadataDescriptorsMatchOrdinaryResourceConversion`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelAliasesUseParsedDuplicateCollapseAndFirstInsertionOrder`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListSelectedAliasConvertsOnlyFinalParsedDictionaryValue`, `blockstorage/snapshot_read_contract_test.go:TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation`.
- Snapshot location is recomputed from the owned configured/authenticated scope and actual project descriptor, including empty/location-only rows. Falsey/equal project retains configured names; truthy foreign project keeps raw ID and clears names/domains. Zone is always null because Snapshot has no zone descriptor. Raw location/availability_zone cannot replace it; caller location errors remain local without fabricated response errors. Owned Location is validated before first HTTP, and Connection validates it before its Cinder getter.
  Tests: `internal/cloudsnapshot/model_test.go:TestSnapshotModelRecomputesLocationFromProjectAndAlwaysUsesNullZone`, `internal/cloudsnapshot/model_test.go:TestSnapshotModelRejectsPhysicalShapeAndLocalLocationFailuresAtomicallyAndOwnsResult`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotRecordedScopeIsCapturedAfterOptionsBeforeGetterAndRetainedAcrossPages`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotExplicitLocationIsOwnedAndZoneIgnoredWithoutChangingDefaults`.
- Prepare/factories own full raw Location, pointer and collection policy; complete options replace earlier policy. Original callbacks run exactly once in captured slice order with independent snapshots after each callback. Nil context/options, canceled custom cause and callback failures stop without partial prepared policy or HTTP.
  Tests: `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotLocationFactoriesOwnEveryNullableComponentAndAreReusable`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotWholeOptionsReplaceEarlierPolicyRatherThanMerging`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareRejectsNilAndCanceledContextBeforeOriginals`, `blockstorage/volume_snapshot_options_test.go:TestVolumeSnapshotPrepareNilOptionAndCallbackFailuresStopWithoutPartialPolicy`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy`.
- Direct workflows capture provider/endpoint/ResourceBase/type/microversion before originals and retain sticky source violations; ordinary headers remain snapshots while auth tokens stay live. Connection executes originals once before only cached Cinder v3 selection and captures current/explicit location before getter. Local callback/getter/preflight errors retain outer operation and custom causes without inventing physical proof.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalOptionsOwnEachSnapshotAndSourceOrdinaryHeaders`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadOriginalCallbackFailureJoinsSourceAndContextBeforeHTTP`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceChangeStaysStickyBeforeAnyPolicyCommit`, `internal/cloudread/policy_test.go:TestCloudReadPolicyCancellationRetainsCauseAndIsAtomic`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause`.
- Each admitted physical page/member owns body/header/status; normalized Value, raw row and error proof own independent state. Read/Close/source drift/custom cancellation keep the actual current phase and all causes while logical outputs remain unpublished. Rejected later/blocked requests retain only earlier completed pages and never borrow an earlier successful or suppressed member response.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadAcceptedSourceFactsAreFrozenThroughBodyClose`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadBlockedContinuationCancellationKeepsOnlyCompletedPage`, `blockstorage/snapshot_read_selection_test.go:TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof`, `connection_volume_snapshots_test.go:TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation`.
- Native reauth/retry remains live and bounded in fixtures while method/fixed route/query/bodyless decoder/body retention/original100..399 policy and source remain guarded. Expanded native400/403/404 is terminal with actual native status/body/header causes, not compatible fallback or admitted ResponseError. Redirect cannot forward credentials to an altered target. No SDK orchestration replay/wait/action/rollback is added.
  Tests: `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadNativeRetryCannotReplaceBodyDecoderRetentionOrSource`, `blockstorage/snapshot_read_transport_test.go:TestVolumeSnapshotReadRedirectCannotSendTokenToChangedFixedTarget`, `internal/cloudread/policy_test.go:TestCloudReadPolicyOwnedWirePolicyKeepsLiveTokenAcrossNativeRetry`, `internal/cloudread/policy_test.go:TestCloudReadPolicySourceMutationInNativeRetryStopsBeforeResend`.

Python/Go differences:

- Only this actual direct BlockStorageCloudMixin helper is mapped using selected Cinder v3 and Gophercloud v2.15.0. Independent Proxy/native/v2/Resource/session/adapter/cache declarations retain separate reviews; full SDK is not complete.
- Python default/overloaded/runtime arguments map to explicit Go request types and owned concrete With/Prepare options, not caller-implemented interface builders. Go accepts finite ordinary JSON and typed controls rather than arbitrary mutable Python Resource/custom runtime objects.
- Logical normalized16-field nullable values, physical raw Resource/Observed/Pages and error proof are independently owned Go representations. Unknown raw fields and number spelling are evidence extensions; mutable Resource identity, stale dict cache and session memoization are not reproduced.
- Ordinary source descriptor behavior and parsed dictionary duplicate/alias order are preserved. Go Unicode16 decimal tables and arbitrary integer policy do not reproduce configurable CPython integer digit limits; exact decimal truthiness/equality differs from Python float rounding/underflow. Untyped timestamps/IDs/names stay JSON rather than native typed leaf extraction.
- Connected Snapshot location is recomputed including empty/location-only rows, using explicit/current recorded scope and raw project descriptor; zone remains null. Direct clients recover recorded auth without fabricating configured names. Go owns snapshots and rejects invalid policy locally rather than emulating dynamic Connection/session objects.
- Go captures original source facts and ordinary headers before originals on direct calls; Connection prepares originals before cached Cinder getter. Native auth remains live, sticky source/fixed request/cause guards and atomic logical publication are explicit Go safety/proof extensions; no SDK replay/wait/rollback is introduced.
- Source validHTTP100..399 and UTF8-valid empty/malformed member JSON tolerance are preserved instead of narrowing to native200. Go additionally requires complete valid UTF8 and JSON/object/envelope/descriptor shape on consumed data, preserves native rejected evidence and prevents expanded native>=400 from becoming success or fallback.
- Selected native microversion is preserved, including symbolic/empty settings when supplied. Source Snapshot3.65 limit only caps absent-default server-discovery negotiation; Go does not automatically negotiate/cap/upgrade a supplied client. Explicit Connection version/range negotiation remains independent.
- ByID requires one safe unescaped UTF8 string path segment; arbitrary Python Resource/runtime object and route-changing coercion are excluded. Native rejected400/403/404 is an error with no list fallback, matching actual Proxy behavior rather than the inaccurate cloud absent=None docstring.
- Missing member ID is retained as explicit logical requested seed/RequestedID/SeededID while physical Body remains unchanged. Actual explicit null/false/zero/empty/containers or mismatched IDs remain actual response values; direct lookup does not invent a request-ID equality policy.

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

func Configured(ctx context.Context, cloudName string) (*sdk.Connection, error) {
    return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPages(pages []*blockstorage.VolumeSnapshotsPage) {
    for _, page := range pages {
        fmt.Println("admitted snapshot page:", page.StatusCode, "bytes:", len(page.Body))
    }
}

func Inspect(result *blockstorage.GetVolumeSnapshotResult) {
    if result == nil { return }
    InspectPages(result.Pages)
    if result.Observed != nil {
        fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
    }
    fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
    if result.Snapshot != nil {
        owned := result.Snapshot.Clone()
        fmt.Println("actual response ID JSON:", string(owned.Body["id"]))
        var wire map[string]json.RawMessage
        if err := owned.Decode(&wire); err != nil {
            fmt.Println("optional wire projection:", err)
        } else {
            fmt.Println("wire fields:", len(wire))
        }
    }
}

func ViaConnection(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(
            `{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }
    fmt.Println("complete normalized list:", string(listed.Value), "raw rows:", len(listed.Snapshots))

    searched, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{"status":"available"}`)))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }
    fmt.Println("selected normalized snapshots:", string(searched.Value))

    selected, err := conn.GetVolumeSnapshot(ctx,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"})
    Inspect(selected) // Physical proof remains useful when err is nonnil.
    if err != nil { return err }
    fmt.Println("selected normalized JSON:", string(selected.Value))

    byID, err := conn.GetVolumeSnapshotByID(ctx,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"})
    Inspect(byID)
    return err
}

func ProjectNames(ctx context.Context, conn *sdk.Connection) error {
    result, err := conn.SearchVolumeSnapshots(ctx,
        blockstorage.SearchVolumeSnapshotsRequest{},
        blockstorage.WithVolumeSnapshotSearchExpression("[].name"))
    if result != nil { InspectPages(result.Pages) }
    if err != nil { return err }
    fmt.Println("arbitrary expression JSON:", string(result.Value))
    return nil
}

func FirstPage(ctx context.Context, conn *sdk.Connection) error {
    detailed, paginated, max := false, false, 10
    result, err := conn.ListVolumeSnapshots(ctx,
        blockstorage.WithVolumeSnapshotListOptions(blockstorage.VolumeSnapshotListOpts{
            Detailed: &detailed, Paginated: &paginated, MaxItems: &max,
            Headers: map[string]string{"X-Request-ID": "snapshot-read-example"},
        }))
    if result != nil { InspectPages(result.Pages) }
    return err
}

func Direct(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    location := resource.CloudLocation{}
    listed, err := blockstorage.ListVolumeSnapshots(ctx, cinder,
        blockstorage.WithVolumeSnapshotListLocation(location))
    if listed != nil { InspectPages(listed.Pages) }
    if err != nil { return err }

    searched, err := blockstorage.SearchVolumeSnapshots(ctx, cinder,
        blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
        blockstorage.WithVolumeSnapshotSearchLocation(location))
    if searched != nil { InspectPages(searched.Pages) }
    if err != nil { return err }

    // Explicit {} filters choose full Search, rather than ordinary GET-first find.
    selected, err := blockstorage.GetVolumeSnapshot(ctx, cinder,
        blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"},
        blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{}`)))
    Inspect(selected)
    if err != nil { return err }

    byID, err := blockstorage.GetVolumeSnapshotByID(ctx, cinder,
        blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"},
        blockstorage.WithVolumeSnapshotReadOptions(blockstorage.VolumeSnapshotReadOpts{Location: &location}))
    Inspect(byID)
    return err
}
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L546)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L710)
- [Search helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L864)
- [Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2165)
- [Proxy.get_snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L119)
- [pinned Snapshot descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/snapshot.py#L62)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [Utils](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2256)
- [Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Microversions](../docs/microversions.md)
- [Snapshots](v3/snapshots/README.md)
- [Metadata](metadata/README.md)
