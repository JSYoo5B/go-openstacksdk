# Volume Types

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/list_volume_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListVolumeTypes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ListVolumeTypes`

- ListVolumeTypes completes the raw v3 /types sequence with no initial visibility/name/detail/limit query or per-type extra-spec request. Every row view is constructed before continuation, response order and duplicate occurrences survive, and Value/Types commit only after complete success with an owned empty array for an empty list.
  Tests: `blockstorage/volume_types_contract_test.go:TestListVolumeTypesCompletesQuerylessPagesAndOwnsSixFieldViewsAndWireProof`, `blockstorage/volume_types_contract_test.go:TestVolumeTypesEmptyCanonicalListIgnoresUnusedLinksAndLocation`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection`.
- Every visited row is eagerly normalized to exactly id, name, description, extra_specs, is_public and location. Untyped attributes and original unknown wire fields stay owned; dictionary null is retained and nonobjects become an empty object; public aliases use ordinary JSON truthiness. RawResource, normalized Value and physical proof do not alias.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewHasOnlySixDescriptorsAndOwnsAllValues`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewBooleanAndDictionaryDescriptorsRetainDeclaredJSONPolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewAliasOrderAndDeclaredDuplicatePolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewRejectsNonobjectsMalformedJSONAndUTF8Atomically`, `blockstorage/volume_types_contract_test.go:TestListVolumeTypesCompletesQuerylessPagesAndOwnsSixFieldViewsAndWireProof`.
- List construction preserves arbitrary canonical wire location only when no recognized Body attribute was consumed; a known null also counts as presence. Member translation always uses computed location. Unknown metadata/project/zone fields do not become descriptors or inferred scope. Caller-invalid owned Location is consumed lazily and stays a local ErrInvalidOption without borrowing200 ResponseError; admitted proof remains.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewListWireLocationIsLazyAndMemberAlwaysUsesOwnedLocation`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewKnownPresenceIncludingNullDiscardsRawLocation`, `blockstorage/volume_types_options_test.go:TestVolumeTypeInvalidOwnedLocationIsLazyAndNeverFabricatesResponseError`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`.
- After a nonempty canonical page, source top-level links presence suppresses plural volume_types_links. The first exact next relation with present href is consumed; falsey href ends that scan and falls to body next, then usable HTTP and finally native singular compatibility. Consumed errors stop fallback. Next nonblank query values replace matching current keys after old marker/limit removal, while omitted nonpagination keys persist. Reader validates origin, escaped path and canonical URL cycles before another request; empty resource array stops without consuming unused continuation.
  Tests: `blockstorage/volume_types_next_test.go:TestVolumeTypesNextSourcePrecedenceAndFirstConsumedHref`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextConsumedShapesAndFalseyValues`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextBodyThenHTTPThenNativeCompatibility`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextHTTPQuotedLinksUseLastNextAcrossHeaders`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextQueryMergeDropsBlankValuesAndOwnsInputs`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextChecksOriginalHrefBeforeURLSerialization`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextRequiresReaderOriginPathAndCycleValidation`, `internal/rest/header_next_test.go:TestHeaderNextLinksPreservesQuotedDelimitersAndRelationOrder`, `blockstorage/volume_types_contract_test.go:TestVolumeTypesEmptyCanonicalListIgnoresUnusedLinksAndLocation`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRespectsSourcePrecedenceAndLazyFirstMatchingHref`, `blockstorage/volume_types_pagination_test.go:TestGetVolumeTypeContinuationMergesVisibilityDropsPaginationAndBlankAdvertisedValues`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRejectsConsumedLinksOriginPathFragmentsAndQueryCycles`.
- Direct calls capture Cinder/service/header facts before originals. Original callbacks and retained factory values run once from owned snapshots; source provider/endpoint/resource base/type/microversion drift is sticky and stops a restoring callback or later exchange. Captured ordinary headers remain fixed and live authentication is available. Connection executes originals before CurrentLocation and cached Cinder v3 selection, then delegates an owned policy.
  Tests: `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalOptionsOwnFactoriesRetainedStateHeadersAndCallbackSlice`, `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalSourceMutationStopsBeforeRestoringOptionAndHTTP`, `blockstorage/volume_types_options_test.go:TestVolumeTypePreflightContextRoleAndOriginalCallbacksFailBeforeHTTP`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesPreflightAndGetterFailuresKeepExactOperationContext`.
- Fixed bodyless accepted200 exchanges retain native rejected status/header/body and authentication behavior without SDK orchestration replay. Admitted read, Close, canonical-schema, source and cancellation failures preserve actual independent proof and joined context.Cause while logical values remain uncommitted. A later failed page never borrows an earlier accepted response as its physical error.
  Tests: `blockstorage/volume_types_transport_test.go:TestVolumeTypesAcceptedReadCloseCancellationRetainsAllCausesAndIndependentCurrentProof`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesStrictCanonicalAcceptedEnvelopesNeverProduceLogicalValues`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesPostResponseSourceChangesKeepCurrentAdmittedProofAndStop`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesBlockedContinuationCancellationKeepsPriorPageAndCallerCause`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesAcceptedProofSurvivesReadCloseCancellationAndSourceDrift`.
- Connection snapshots current location after originals for each operation and refreshes it on the next call. Complete read/search overrides and callback output own their pointers/raw JSON, keep an explicit Zone and leave defaults unchanged. Invalid recorded scope prevents cached service selection unless a complete override supplies the facts; direct recorded project scope is a snapshot without enrichment I/O.
  Tests: `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesInvalidRecordedScopePreventsSelectionButOwnedLocationWorks`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`.

Python/Go differences:

- Deprecated get_extra is a warning-only no-op in pinned list_volume_types and is omitted from Go; no extra-spec enrichment is lost. Go does not reproduce warning delivery.
- Go selects Cinder v3 under Gophercloud v2.15.0. Only this actual cloud helper is mapped; independent v2 Type, native/Proxy declarations, generic Resource/session/cache, access and mutation profiles are not promoted by this row.
- The six-field view follows v3 Type descriptors, with unknown original wire fields retained only in RawResource. Go owns raw JSON and exact numeric text; this is not a claim that a plain Go map reproduces the complete mutable Python Resource runtime.
- Go public aliases and repeated wire keys consume textual order and the last value wins. Python json.loads collapses repeated keys at their first insertion position before alias descriptor consumption; interleaved duplicate/alias inputs can differ. Existing exact mathematical JSON truthiness also keeps 1e-9999 nonzero whereas ordinary Python float decoding can underflow to false.
- List rows with no recognized Body key preserve canonical wire location, including false/null/arbitrary JSON; known-key presence discards it and member translation always ignores wire computed location. Go supplies owned current scope or a complete override, retaining override Zone and performing no unknown owner/zone inference. Local caller Location failures remain separate from physical response failures.
- Usable HTTP Link fallback is a declared Go repair: pinned Resource._get_next_link reads response.links[next][uri] but ordinary Requests returns url. Existing quoted Link parsing handles multiple angle targets, quoted delimiters, case-insensitive/multi-relation next and last-next replacement across headers. Native canonical singular volume_type_links is an explicit lower-priority compatibility extension; Python derives plural volume_types_links.
- Next query uses source-style nonblank parse_qs merge, with old marker/limit removed and omitted nonpagination keys retained; explicit advertised keys replace current keys. Go encodes one merged query and preserves effective advertised paths instead of copying source relative /v prefix stripping or query-only URL-plus-params duplication. No initial caller limit or invented numeric marker fallback is exposed.
- Strict canonical UTF-8 envelopes and accepted200, fragment/userinfo/opaque/query/origin/escaped-path checks and complete-URL cycle detection are Go safety boundaries. Source Resource.list permits other shapes and has separate same-marker bookkeeping, so these rules are intentional mappings rather than generic pagination equivalence.
- Go exposes independently owned normalized values, raw resources and actual physical response proof with errors and joined cancellation causes. Native protected request retry is retained; SDK workflow replay, mutable session negotiation and cached object lifecycle are separate scope.

### `python:cloud/_block_storage/BlockStorageCloudMixin/search_volume_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchVolumeTypes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SearchVolumeTypes`

- SearchVolumeTypes first completes queryless ListVolumeTypes and all eager descriptor/location views, then performs row-by-row exact or fnmatch-style name/ID selection followed by filters. Source order and duplicate occurrences remain. A later page/descriptor failure wins before filter interpretation; local filter errors do not borrow a successful response as ResponseError.
  Tests: `blockstorage/volume_types_contract_test.go:TestSearchVolumeTypesCompletesAllPagesAndPreservesExactGlobDuplicateSelection`, `blockstorage/volume_types_contract_test.go:TestVolumeTypeLocalFiltersFollowCompleteEagerNormalizationAndKeepOnlyOriginPages`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection`.
- Owned JSON filters preserve omission and arbitrary explicit values. Falsey filters retain identifier-selected rows; dictionaries compare normalized nested fields with selected Python JSON equality and lazy error consumption. Nonempty strings evaluate the shared full JMESPath engine and can return any JSON type. Expressions never invent a RawResource association.
  Tests: `blockstorage/volume_types_contract_test.go:TestVolumeTypeExpressionsKeepDynamicSearchAndGetterFirstValueSemantics`, `blockstorage/volume_types_contract_test.go:TestVolumeTypeLocalFiltersFollowCompleteEagerNormalizationAndKeepOnlyOriginPages`, `blockstorage/volume_types_contract_test.go:TestVolumeTypeSearchMappingUsesNormalizedSubsetsAndOrderedLazyErrors`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesExpressionFactoriesOwnFiltersAndDoNotInventRawRows`.
- Every visited row is eagerly normalized to exactly id, name, description, extra_specs, is_public and location. Untyped attributes and original unknown wire fields stay owned; dictionary null is retained and nonobjects become an empty object; public aliases use ordinary JSON truthiness. RawResource, normalized Value and physical proof do not alias.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewHasOnlySixDescriptorsAndOwnsAllValues`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewBooleanAndDictionaryDescriptorsRetainDeclaredJSONPolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewAliasOrderAndDeclaredDuplicatePolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewRejectsNonobjectsMalformedJSONAndUTF8Atomically`, `blockstorage/volume_types_contract_test.go:TestListVolumeTypesCompletesQuerylessPagesAndOwnsSixFieldViewsAndWireProof`.
- List construction preserves arbitrary canonical wire location only when no recognized Body attribute was consumed; a known null also counts as presence. Member translation always uses computed location. Unknown metadata/project/zone fields do not become descriptors or inferred scope. Caller-invalid owned Location is consumed lazily and stays a local ErrInvalidOption without borrowing200 ResponseError; admitted proof remains.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewListWireLocationIsLazyAndMemberAlwaysUsesOwnedLocation`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewKnownPresenceIncludingNullDiscardsRawLocation`, `blockstorage/volume_types_options_test.go:TestVolumeTypeInvalidOwnedLocationIsLazyAndNeverFabricatesResponseError`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`.
- After a nonempty canonical page, source top-level links presence suppresses plural volume_types_links. The first exact next relation with present href is consumed; falsey href ends that scan and falls to body next, then usable HTTP and finally native singular compatibility. Consumed errors stop fallback. Next nonblank query values replace matching current keys after old marker/limit removal, while omitted nonpagination keys persist. Reader validates origin, escaped path and canonical URL cycles before another request; empty resource array stops without consuming unused continuation.
  Tests: `blockstorage/volume_types_next_test.go:TestVolumeTypesNextSourcePrecedenceAndFirstConsumedHref`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextConsumedShapesAndFalseyValues`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextBodyThenHTTPThenNativeCompatibility`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextHTTPQuotedLinksUseLastNextAcrossHeaders`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextQueryMergeDropsBlankValuesAndOwnsInputs`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextChecksOriginalHrefBeforeURLSerialization`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextRequiresReaderOriginPathAndCycleValidation`, `internal/rest/header_next_test.go:TestHeaderNextLinksPreservesQuotedDelimitersAndRelationOrder`, `blockstorage/volume_types_contract_test.go:TestVolumeTypesEmptyCanonicalListIgnoresUnusedLinksAndLocation`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRespectsSourcePrecedenceAndLazyFirstMatchingHref`, `blockstorage/volume_types_pagination_test.go:TestGetVolumeTypeContinuationMergesVisibilityDropsPaginationAndBlankAdvertisedValues`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRejectsConsumedLinksOriginPathFragmentsAndQueryCycles`.
- Direct calls capture Cinder/service/header facts before originals. Original callbacks and retained factory values run once from owned snapshots; source provider/endpoint/resource base/type/microversion drift is sticky and stops a restoring callback or later exchange. Captured ordinary headers remain fixed and live authentication is available. Connection executes originals before CurrentLocation and cached Cinder v3 selection, then delegates an owned policy.
  Tests: `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalOptionsOwnFactoriesRetainedStateHeadersAndCallbackSlice`, `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalSourceMutationStopsBeforeRestoringOptionAndHTTP`, `blockstorage/volume_types_options_test.go:TestVolumeTypePreflightContextRoleAndOriginalCallbacksFailBeforeHTTP`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesPreflightAndGetterFailuresKeepExactOperationContext`.
- Fixed bodyless accepted200 exchanges retain native rejected status/header/body and authentication behavior without SDK orchestration replay. Admitted read, Close, canonical-schema, source and cancellation failures preserve actual independent proof and joined context.Cause while logical values remain uncommitted. A later failed page never borrows an earlier accepted response as its physical error.
  Tests: `blockstorage/volume_types_transport_test.go:TestVolumeTypesAcceptedReadCloseCancellationRetainsAllCausesAndIndependentCurrentProof`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesStrictCanonicalAcceptedEnvelopesNeverProduceLogicalValues`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesPostResponseSourceChangesKeepCurrentAdmittedProofAndStop`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesBlockedContinuationCancellationKeepsPriorPageAndCallerCause`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesAcceptedProofSurvivesReadCloseCancellationAndSourceDrift`.
- Connection snapshots current location after originals for each operation and refreshes it on the next call. Complete read/search overrides and callback output own their pointers/raw JSON, keep an explicit Zone and leave defaults unchanged. Invalid recorded scope prevents cached service selection unless a complete override supplies the facts; direct recorded project scope is a snapshot without enrichment I/O.
  Tests: `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesInvalidRecordedScopePreventsSelectionButOwnedLocationWorks`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`.

Python/Go differences:

- Pinned search_volume_types warns for non-None get_extra then calls list_volume_types without that value; get_extra produces no enrichment and is omitted. Go also omits the deprecated JMESPath/logging warning channel while retaining functional filter behavior.
- Go models name_or_id as a string and JSON filters as owned bytes. Arbitrary Python object coercions are outside the ordinary JSON direct-helper contract. The reused shared JMESPath engine declares exact-number, finite arithmetic/allocation and deterministic unordered-object boundaries in its independent guide; this row does not promote that dependency declaration.
- Go selects Cinder v3 under Gophercloud v2.15.0. Only this actual cloud helper is mapped; independent v2 Type, native/Proxy declarations, generic Resource/session/cache, access and mutation profiles are not promoted by this row.
- The six-field view follows v3 Type descriptors, with unknown original wire fields retained only in RawResource. Go owns raw JSON and exact numeric text; this is not a claim that a plain Go map reproduces the complete mutable Python Resource runtime.
- Go public aliases and repeated wire keys consume textual order and the last value wins. Python json.loads collapses repeated keys at their first insertion position before alias descriptor consumption; interleaved duplicate/alias inputs can differ. Existing exact mathematical JSON truthiness also keeps 1e-9999 nonzero whereas ordinary Python float decoding can underflow to false.
- List rows with no recognized Body key preserve canonical wire location, including false/null/arbitrary JSON; known-key presence discards it and member translation always ignores wire computed location. Go supplies owned current scope or a complete override, retaining override Zone and performing no unknown owner/zone inference. Local caller Location failures remain separate from physical response failures.
- Usable HTTP Link fallback is a declared Go repair: pinned Resource._get_next_link reads response.links[next][uri] but ordinary Requests returns url. Existing quoted Link parsing handles multiple angle targets, quoted delimiters, case-insensitive/multi-relation next and last-next replacement across headers. Native canonical singular volume_type_links is an explicit lower-priority compatibility extension; Python derives plural volume_types_links.
- Next query uses source-style nonblank parse_qs merge, with old marker/limit removed and omitted nonpagination keys retained; explicit advertised keys replace current keys. Go encodes one merged query and preserves effective advertised paths instead of copying source relative /v prefix stripping or query-only URL-plus-params duplication. No initial caller limit or invented numeric marker fallback is exposed.
- Strict canonical UTF-8 envelopes and accepted200, fragment/userinfo/opaque/query/origin/escaped-path checks and complete-URL cycle detection are Go safety boundaries. Source Resource.list permits other shapes and has separate same-marker bookkeeping, so these rules are intentional mappings rather than generic pagination equivalence.
- Go exposes independently owned normalized values, raw resources and actual physical response proof with errors and joined cancellation causes. Native protected request retry is retained; SDK workflow replay, mutable session negotiation and cached object lifecycle are separate scope.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeType`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeType`

- Omitted or literal-null filters use exact identity lookup. Safe identities send a fixed bodyless member GET with lowercase is_public=none; compatible clean rejected400/403/404 fall back to a /types list retaining is_public=none but no name hint. Successful member values keep actual untyped response identity rather than an input-ID shortcut. Unsafe literal identities use existing exact list-only routing.
  Tests: `blockstorage/volume_types_contract_test.go:TestGetVolumeTypeOmittedAndNullFiltersKeepMemberQueryAndFreshCanonicalIdentity`, `blockstorage/volume_types_contract_test.go:TestGetVolumeTypeDefaultFallbackUsesNoneWithoutNameHintAndWaitsForExactAbsence`, `connectiontest/connection_volume_types_test.go:TestConnectionGetVolumeTypeDefaultAndExplicitFalseyFilterRoutes`.
- Default exact fallback confirms absence only after successful complete search and stops on its second exact match before unused bad rows or links. Visited descriptors remain eager. Later native/transport/admitted failures retain their own evidence without borrowing the suppressed member400/404; no incomplete Type/Value is committed.
  Tests: `blockstorage/volume_types_contract_test.go:TestGetVolumeTypeDefaultFallbackUsesNoneWithoutNameHintAndWaitsForExactAbsence`, `blockstorage/volume_types_contract_test.go:TestVolumeTypeExactFindStopsAtSecondMatchWhileSearchConsumesUnusedBadRowAndLink`, `blockstorage/volume_types_transport_test.go:TestGetVolumeTypeFallbackFailuresKeepOnlyActualCurrentPhaseEvidence`.
- Every explicit nonnull filter, including {}, [], false, zero and empty string, selects complete queryless SearchVolumeTypes. Getter then applies source truthiness, length and integer-index-zero semantics to any JSON expression output: successful absence is nil Value, selected false/zero/empty string remain values, multiple matches wrap ErrAmbiguous, and invalid length/index operations remain errors. Expression output never invents a Type pointer.
  Tests: `blockstorage/volume_types_contract_test.go:TestGetVolumeTypeEveryNonnullFalseyFilterUsesFullQuerylessSearch`, `blockstorage/volume_types_contract_test.go:TestVolumeTypeExpressionsKeepDynamicSearchAndGetterFirstValueSemantics`, `connectiontest/connection_volume_types_test.go:TestConnectionGetVolumeTypeDefaultAndExplicitFalseyFilterRoutes`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesExpressionFactoriesOwnFiltersAndDoNotInventRawRows`.
- Every visited row is eagerly normalized to exactly id, name, description, extra_specs, is_public and location. Untyped attributes and original unknown wire fields stay owned; dictionary null is retained and nonobjects become an empty object; public aliases use ordinary JSON truthiness. RawResource, normalized Value and physical proof do not alias.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewHasOnlySixDescriptorsAndOwnsAllValues`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewBooleanAndDictionaryDescriptorsRetainDeclaredJSONPolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewAliasOrderAndDeclaredDuplicatePolicy`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewRejectsNonobjectsMalformedJSONAndUTF8Atomically`, `blockstorage/volume_types_contract_test.go:TestListVolumeTypesCompletesQuerylessPagesAndOwnsSixFieldViewsAndWireProof`.
- List construction preserves arbitrary canonical wire location only when no recognized Body attribute was consumed; a known null also counts as presence. Member translation always uses computed location. Unknown metadata/project/zone fields do not become descriptors or inferred scope. Caller-invalid owned Location is consumed lazily and stays a local ErrInvalidOption without borrowing200 ResponseError; admitted proof remains.
  Tests: `blockstorage/volume_type_view_test.go:TestVolumeTypeViewListWireLocationIsLazyAndMemberAlwaysUsesOwnedLocation`, `blockstorage/volume_type_view_test.go:TestVolumeTypeViewKnownPresenceIncludingNullDiscardsRawLocation`, `blockstorage/volume_types_options_test.go:TestVolumeTypeInvalidOwnedLocationIsLazyAndNeverFabricatesResponseError`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`.
- After a nonempty canonical page, source top-level links presence suppresses plural volume_types_links. The first exact next relation with present href is consumed; falsey href ends that scan and falls to body next, then usable HTTP and finally native singular compatibility. Consumed errors stop fallback. Next nonblank query values replace matching current keys after old marker/limit removal, while omitted nonpagination keys persist. Reader validates origin, escaped path and canonical URL cycles before another request; empty resource array stops without consuming unused continuation.
  Tests: `blockstorage/volume_types_next_test.go:TestVolumeTypesNextSourcePrecedenceAndFirstConsumedHref`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextConsumedShapesAndFalseyValues`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextBodyThenHTTPThenNativeCompatibility`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextHTTPQuotedLinksUseLastNextAcrossHeaders`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextQueryMergeDropsBlankValuesAndOwnsInputs`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextChecksOriginalHrefBeforeURLSerialization`, `blockstorage/volume_types_next_test.go:TestVolumeTypesNextRequiresReaderOriginPathAndCycleValidation`, `internal/rest/header_next_test.go:TestHeaderNextLinksPreservesQuotedDelimitersAndRelationOrder`, `blockstorage/volume_types_contract_test.go:TestVolumeTypesEmptyCanonicalListIgnoresUnusedLinksAndLocation`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRespectsSourcePrecedenceAndLazyFirstMatchingHref`, `blockstorage/volume_types_pagination_test.go:TestGetVolumeTypeContinuationMergesVisibilityDropsPaginationAndBlankAdvertisedValues`, `blockstorage/volume_types_pagination_test.go:TestVolumeTypesPaginationRejectsConsumedLinksOriginPathFragmentsAndQueryCycles`.
- Direct calls capture Cinder/service/header facts before originals. Original callbacks and retained factory values run once from owned snapshots; source provider/endpoint/resource base/type/microversion drift is sticky and stops a restoring callback or later exchange. Captured ordinary headers remain fixed and live authentication is available. Connection executes originals before CurrentLocation and cached Cinder v3 selection, then delegates an owned policy.
  Tests: `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalOptionsOwnFactoriesRetainedStateHeadersAndCallbackSlice`, `blockstorage/volume_types_options_test.go:TestVolumeTypeOriginalSourceMutationStopsBeforeRestoringOptionAndHTTP`, `blockstorage/volume_types_options_test.go:TestVolumeTypePreflightContextRoleAndOriginalCallbacksFailBeforeHTTP`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesPreflightAndGetterFailuresKeepExactOperationContext`.
- Fixed bodyless accepted200 exchanges retain native rejected status/header/body and authentication behavior without SDK orchestration replay. Admitted read, Close, canonical-schema, source and cancellation failures preserve actual independent proof and joined context.Cause while logical values remain uncommitted. A later failed page never borrows an earlier accepted response as its physical error. Native auth retry preserves exact is_public=none request ownership; caller retry body/response-decoder mutation rejects before resend; ordinary native retry headers and OkCodes expansion remain supported, while an actual unrequested status is rejected afterward with native response evidence and no admitted proof.
  Tests: `blockstorage/volume_types_transport_test.go:TestVolumeTypesAcceptedReadCloseCancellationRetainsAllCausesAndIndependentCurrentProof`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesStrictCanonicalAcceptedEnvelopesNeverProduceLogicalValues`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesPostResponseSourceChangesKeepCurrentAdmittedProofAndStop`, `blockstorage/volume_types_transport_test.go:TestVolumeTypesBlockedContinuationCancellationKeepsPriorPageAndCallerCause`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesAcceptedProofSurvivesReadCloseCancellationAndSourceDrift`, `blockstorage/volume_types_transport_test.go:TestGetVolumeTypeNativeAuthenticationRetriesKeepNoneQueryAndOneAcceptedProof`, `blockstorage/volume_types_transport_test.go:TestGetVolumeTypeNativeRetryPreservesBodyOwnershipAndOriginalAcceptedStatus`, `blockstorage/volume_types_transport_test.go:TestGetVolumeTypeAcceptedReadContainingNative404IsTerminalWithoutFallback`.
- Connection snapshots current location after originals for each operation and refreshes it on the next call. Complete read/search overrides and callback output own their pointers/raw JSON, keep an explicit Zone and leave defaults unchanged. Invalid recorded scope prevents cached service selection unless a complete override supplies the facts; direct recorded project scope is a snapshot without enrichment I/O.
  Tests: `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots`, `connectiontest/connection_volume_types_test.go:TestConnectionVolumeTypesInvalidRecordedScopePreventsSelectionButOwnedLocationWorks`, `blockstorage/volume_types_options_test.go:TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed`.

Python/Go differences:

- Pinned get_volume_type treats every filters is not None value as the deprecated Search branch, independently of filter truthiness, and Go preserves that dispatch. Go omits warning delivery.
- Pinned Resource.find attempts member GET for all identities and carries mutable seeded request ID/is_public state. Go uses safe single-segment exact-routing policy for slash/backslash/query/fragment/percent/Unicode whitespace names and retains only fresh canonical response ID/public fields, producing null when absent rather than fabricating the seed.
- Go returns nullable Value plus optional actual Type and Observed/Pages proof. Python returns Resource/None or arbitrary expression first values; Go distinguishes absence from a selected false/zero/empty string and exposes a library-owned VolumeTypeSelectionError wrapping ErrAmbiguous instead of an SDKException class.
- Go selects Cinder v3 under Gophercloud v2.15.0. Only this actual cloud helper is mapped; independent v2 Type, native/Proxy declarations, generic Resource/session/cache, access and mutation profiles are not promoted by this row.
- The six-field view follows v3 Type descriptors, with unknown original wire fields retained only in RawResource. Go owns raw JSON and exact numeric text; this is not a claim that a plain Go map reproduces the complete mutable Python Resource runtime.
- Go public aliases and repeated wire keys consume textual order and the last value wins. Python json.loads collapses repeated keys at their first insertion position before alias descriptor consumption; interleaved duplicate/alias inputs can differ. Existing exact mathematical JSON truthiness also keeps 1e-9999 nonzero whereas ordinary Python float decoding can underflow to false.
- List rows with no recognized Body key preserve canonical wire location, including false/null/arbitrary JSON; known-key presence discards it and member translation always ignores wire computed location. Go supplies owned current scope or a complete override, retaining override Zone and performing no unknown owner/zone inference. Local caller Location failures remain separate from physical response failures.
- Usable HTTP Link fallback is a declared Go repair: pinned Resource._get_next_link reads response.links[next][uri] but ordinary Requests returns url. Existing quoted Link parsing handles multiple angle targets, quoted delimiters, case-insensitive/multi-relation next and last-next replacement across headers. Native canonical singular volume_type_links is an explicit lower-priority compatibility extension; Python derives plural volume_types_links.
- Next query uses source-style nonblank parse_qs merge, with old marker/limit removed and omitted nonpagination keys retained; explicit advertised keys replace current keys. Go encodes one merged query and preserves effective advertised paths instead of copying source relative /v prefix stripping or query-only URL-plus-params duplication. No initial caller limit or invented numeric marker fallback is exposed.
- Strict canonical UTF-8 envelopes and accepted200, fragment/userinfo/opaque/query/origin/escaped-path checks and complete-URL cycle detection are Go safety boundaries. Source Resource.list permits other shapes and has separate same-marker bookkeeping, so these rules are intentional mappings rather than generic pagination equivalence.
- Go exposes independently owned normalized values, raw resources and actual physical response proof with errors and joined cancellation causes. Native protected request retry is retained; SDK workflow replay, mutable session negotiation and cached object lifecycle are separate scope.

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
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ReadTypes(ctx context.Context, conn *sdk.Connection) error {
    listed, err := conn.ListVolumeTypes(ctx)
    if err != nil {
        if listed != nil { fmt.Println("admitted list pages:", len(listed.Pages)) }
        return err
    }
    fmt.Println("normalized types:", string(listed.Value))
    for _, item := range listed.Types {
        owned := item.Clone()
        fmt.Println("original extra_specs:", string(owned.Body["extra_specs"]))
        var projection struct {
            ID *string `json:"id"`
            Name *string `json:"name"`
            ExtraSpecs map[string]json.RawMessage `json:"extra_specs"`
        }
        if err := owned.Decode(&projection); err != nil { return err }
    }

    searched, err := conn.SearchVolumeTypes(ctx,
        blockstorage.SearchVolumeTypesRequest{NameOrID: "fast*"},
        blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{"is_public":true}`)))
    if err != nil {
        if searched != nil { fmt.Println("admitted search pages:", len(searched.Pages)) }
        return err
    }
    fmt.Println("matched types:", string(searched.Value))

    selected, err := conn.GetVolumeType(ctx,
        blockstorage.GetVolumeTypeRequest{NameOrID: "fast"})
    if err != nil {
        if selected != nil {
            fmt.Println("admitted fallback pages:", len(selected.Pages))
            if selected.Observed != nil {
                fmt.Println("admitted member status:", selected.Observed.StatusCode)
            }
        }
        return err
    }
    if selected.Value == nil { fmt.Println("type absent") } else {
        fmt.Println("selected type:", string(selected.Value))
    }

    expression, err := conn.SearchVolumeTypes(ctx,
        blockstorage.SearchVolumeTypesRequest{},
        blockstorage.WithVolumeTypeSearchExpression("[].name"))
    if err != nil { return err }
    fmt.Println("expression JSON:", string(expression.Value))
    return nil
}

func ReadTypesWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud, Zone: json.RawMessage(`"owned-zone"`),
        Project: resource.CloudProject{ID: json.RawMessage(`"scope-id"`), Name: &project},
    }
    result, err := conn.ListVolumeTypes(ctx,
        blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location}))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}

func ReadTypesDirect(ctx context.Context, cinder *gophercloud.ServiceClient) error {
    result, err := blockstorage.GetVolumeType(ctx, cinder,
        blockstorage.GetVolumeTypeRequest{NameOrID: "fast"},
        blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`{}`)))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="example")
types = conn.list_volume_types()
matches = conn.search_volume_types("fast*", filters={"is_public": True})
selected = conn.get_volume_type("fast")
names = conn.search_volume_types(filters="[].name")
```

## References

- [Requests Response.links](https://requests.readthedocs.io/en/latest/_modules/requests/models/#Response.links)
- [parse_header_links](https://requests.readthedocs.io/en/latest/_modules/requests/utils/#parse_header_links)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L66-L79)
- [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L916-L947)
- [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L133-L173)
- [v3 Type descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/type.py)
- [v3 find query](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L474-L496)
