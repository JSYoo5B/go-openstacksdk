# Volume Reads

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/list_volumes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListVolumes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ListVolumes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadLocation`

- ListVolumes is the direct cloud complete detailed list. Its initial bodyless GET200 is /volumes/detail without identity/status/all-project/page/filter controls; complete materialization returns all source rows in order and never performs member GETs or name resolution. The deprecated Python cache argument is warning-only/no-op and is omitted in Go.
  Tests: `blockstorage/volume_reads_contract_test.go:TestListVolumesMaterializesAllPagesAndOwnsNormalizedAndWireRows`, `connection_volume_reads_test.go:TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection`.
- Every raw row is converted through the complete eager normalized v3 descriptor/location view while consuming that row, before any continuation is inspected or requested. Malformed visited descriptor/location errors stop the collector; a later page error never exposes earlier partial Value/Volumes.
  Tests: `blockstorage/volume_reads_contract_test.go:TestListVolumesEagerDescriptorFailureStopsBeforeNextAndDiscardsPartialValues`, `blockstorage/volume_reads_contract_test.go:TestVolumeExistsAmbiguityStopsBeforeUnusedTailThatFullListMustConsume`, `blockstorage/volume_reads_transport_test.go:TestListVolumesLaterNativeOrTransportErrorKeepsOnlyActualPriorPages`, `blockstorage/volume_search_test.go:TestSearchVolumesNormalizesEveryRowBeforeNextAndConsumesFiltersAfterFullList`.
- Value is an owned JSON array of normalized resources and Volumes is the corresponding owned raw source-row slice. Empty success is nonnull [] plus a nonnil empty slice. Duplicate IDs remain separate original rows/pointers in wire order; unrelated unknown fields and exact numeric text remain in RawResource.Body rather than being fabricated into the normalized view.
  Tests: `blockstorage/volume_reads_contract_test.go:TestListVolumesMaterializesAllPagesAndOwnsNormalizedAndWireRows`, `blockstorage/volume_reads_contract_test.go:TestListVolumesEmptyPageStopsWithoutConsumingUnusedContinuation`.
- Canonical volumes [] ends the list before unused next-link consumption. Nonempty pages use exact native volumes_links continuation extraction with same collection escaped path/origin and cycle guards. Source-advertised queries are retained; no HTTP Link/data.next/marker fallback, local pagination restart or page cap is invented.
  Tests: `blockstorage/volume_reads_contract_test.go:TestListVolumesMaterializesAllPagesAndOwnsNormalizedAndWireRows`, `blockstorage/volume_reads_contract_test.go:TestListVolumesEmptyPageStopsWithoutConsumingUnusedContinuation`, `blockstorage/get_volumes_test.go:TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof`.
- Each physically admitted200 list response contributes independently cloned Pages body/header/status even when its read/Close/decoder/source/context phase fails. Rejected HTTP/native/transport failures add no admitted page and keep their own error evidence, never an old page proof. Every error leaves Value/Volumes nil while actual Pages remain inspectable.
  Tests: `blockstorage/volume_reads_transport_test.go:TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues`, `blockstorage/volume_reads_transport_test.go:TestListVolumesLaterNativeOrTransportErrorKeepsOnlyActualPriorPages`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadsPostResponseSourceChangesStopWithCurrentAdmittedProof`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadBlockedContinuationCancellationRetainsActualPagesAndCause`, `connection_volume_reads_test.go:TestConnectionVolumeReadsAcceptedProofSurvivesReadCloseCancellationAndSourceFailure`.
- Owned VolumeReadOpts.Location is the only read policy. Original callbacks apply once in order; direct package captures Cinder/header facts before originals and guards sticky context/source through every phase. Connection prepares originals once before its cached Cinder v3 getter and delegates an owned policy, supplying recorded CurrentLocation unless overridden. Captured ordinary headers are independent while provider authentication stays live; empty source Type is normalized only on the execution copy.
  Tests: `blockstorage/volume_reads_options_test.go:TestPrepareVolumeReadOptionsOwnsFactoriesCallbacksAndOriginalSlice`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsApplyOriginalOptionsOnceWithCapturedHeadersAndOwnedLocation`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsOriginalSourceMutationIsTerminalBeforeRestoringOption`, `blockstorage/volume_reads_options_test.go:TestVolumeReadPreflightStopsOptionsAndHTTPOnInvalidSourceOrContext`, `connection_volume_reads_test.go:TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection`, `connection_volume_reads_test.go:TestConnectionVolumeReadsLocationFactoriesOwnInputsAndLeaveConnectionDefault`, `connection_volume_reads_test.go:TestConnectionVolumeReadsOriginalCallbackSliceAndRetainedLocationAreSnapshots`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeIsCapturedAfterOptionsAndRefreshedPerCall`, `connection_volume_reads_test.go:TestConnectionVolumeReadsPreflightAndOptionFailuresSelectNoService`, `connection_volume_reads_test.go:TestConnectionVolumeReadsGetterFailureRetainsOperationAndCancellationCause`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeFailurePrecedesGetterAndOwnedOverrideBypassesIt`.
- Caller-owned Location JSON is validated only when an actual resource is normalized; an empty ListVolumes list does not consume an unused invalid location. Invalid caller location remains a local ErrInvalidOption without borrowing a200 ResponseError, while any independently admitted Pages/Observed evidence remains available and logical outputs stay uncommitted. Physical canonical row/descriptor failures retain their own originating accepted response error evidence.
  Tests: `blockstorage/volume_reads_options_test.go:TestVolumeReadLocationIsConsumedOnlyByActualRowsWithoutBorrowingResponseFailure`, `blockstorage/volume_reads_contract_test.go:TestListVolumesEagerDescriptorFailureStopsBeforeNextAndDiscardsPartialValues`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues`.

Python/Go differences:

- Go selects Cinder v3 under Gophercloud v2.15.0 and uses the audited 38-field normalized Volume view plus an independent RawResource physical row and response proof. Pinned cloud can select v2; this row does not implement a v2 projection or promote native/Proxy/Resource/session/cache declarations.
- Complete canonical UTF-8 volume(s) envelopes and accepted200 responses are the deliberate Go decoder boundary. Python generic Resource translation can tolerate invalid JSON on member fetch, unwrap absent envelopes more permissively, wrap nonarray resource lists, strip self, and preserve seeded omitted IDs. Go raw evidence preserves unknown/self fields while normalized known fields exclude them.
- Computed location follows existing owned CloudLocation/recorded provider-scope mapping. Exact decimal raw precision, Unicode/version and configurable Python numeric conversion boundaries are inherited from the already audited normalized view; no Python object identity, mutable Resource lifecycle or warning subsystem is recreated.
- Go returns richer owned result/evidence structures with an error; rejected native status/body/header remain in their original error. Admitted phase proof survives later failures without establishing successful resource/boolean output, and no SDK orchestration replay or swallowed cancellation is introduced.
- Pinned list_volumes(cache=None) warns for every non-None cache value, including False, without changing list behavior. The Go API deliberately has no obsolete cache/warning parameter and exposes none of the lower Proxy query/details/all_projects controls.
- Go continuation policy maps native canonical volumes_links only. Pinned generic Resource.list can consult links/data.next/HTTP Link and limited marker fallback; this direct workflow does not promote those generic capabilities.
- Local caller Location errors are separate from physical response/descriptor errors: actual admitted proof remains inspectable, but a local error is not reclassified as a200 response failure. This preserves the existing shared GetVolume/SearchVolumes normalizer behavior.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumeByID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeByID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadLocation`

- GetVolumeByIDRequest.ID is a literal direct member identifier, never a name or a Name Ref. Required UTF-8/control/Unicode-whitespace/single unescaped segment validation protects the fixed bodyless GET200 /volumes/{escaped-id}; no query/list/search/ref lookup is performed.
  Tests: `blockstorage/volume_reads_contract_test.go:TestGetVolumeByIDUsesOnlyLiteralMemberAndKeepsResponseIdentity`, `blockstorage/volume_reads_contract_test.go:TestGetVolumeByIDAcceptsUnicodeLiteralSingleSegment`, `blockstorage/volume_reads_options_test.go:TestGetVolumeByIDRejectsUnsafeIDsAfterOptionsWithoutHTTP`.
- All rejected member HTTP codes, including404/400/403, remain errors with original native status/body/header/cause and never trigger name fallback or a successful absent value. The misleading Python docstring else None is not treated as source behavior; actual proxy _get fetch propagates NotFoundException.
  Tests: `blockstorage/volume_reads_contract_test.go:TestGetVolumeByIDPropagatesNativeClientErrorsWithoutFindFallback`, `connection_volume_reads_test.go:TestConnectionVolumeReadsMember404IsErrorButCompletedExistenceAbsenceIsFalse`.
- A successful canonical volume object produces the eager complete normalized v3/location Value and the actual independently owned RawResource Volume. Omitted response ID remains absent raw and null normalized; a present different/untyped ID is preserved rather than silently replaced with the request ID.
  Tests: `blockstorage/volume_reads_contract_test.go:TestGetVolumeByIDUsesOnlyLiteralMemberAndKeepsResponseIdentity`, `connection_volume_reads_test.go:TestConnectionVolumeReadsLocationFactoriesOwnInputsAndLeaveConnectionDefault`.
- Observed clones the actual admitted member response before the workflow returns read/Close, post-source/context or decoding errors. On any failure Value/Volume remain nil, Observed remains available if200 was admitted, and Pages stays empty. Read+Close+context.Cause failures retain all causes without inventing a successful completion or replaying a list.
  Tests: `blockstorage/volume_reads_transport_test.go:TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadsPostResponseSourceChangesStopWithCurrentAdmittedProof`, `connection_volume_reads_test.go:TestConnectionVolumeReadsAcceptedProofSurvivesReadCloseCancellationAndSourceFailure`.
- Response headers/raw body and normalized Value are independent from Volume.Body/Header and ResponseError snapshots. The member getter shares the existing guarded transport/normalizer with default GetVolume without changing existing find fallback/selection behavior.
  Tests: `blockstorage/volume_reads_contract_test.go:TestGetVolumeByIDUsesOnlyLiteralMemberAndKeepsResponseIdentity`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof`, `blockstorage/volume_search_test.go:TestGetVolumeOmittedNullGETFirstAndCompatibleListFallback`, `blockstorage/volume_search_test.go:TestGetVolumeEveryFalseyNonnullFilterUsesFullListOnly`, `blockstorage/volume_search_test.go:TestSearchAndGetVolumeExpressionsKeepDynamicValuesAndSingleFalseyElement`, `blockstorage/volume_search_test.go:TestGetVolumeDefaultExactListMissingAndAmbiguityKeepSharedIteratorPolicy`.
- Owned VolumeReadOpts.Location is the only read policy. Original callbacks apply once in order; direct package captures Cinder/header facts before originals and guards sticky context/source through every phase. Connection prepares originals once before its cached Cinder v3 getter and delegates an owned policy, supplying recorded CurrentLocation unless overridden. Captured ordinary headers are independent while provider authentication stays live; empty source Type is normalized only on the execution copy.
  Tests: `blockstorage/volume_reads_options_test.go:TestPrepareVolumeReadOptionsOwnsFactoriesCallbacksAndOriginalSlice`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsApplyOriginalOptionsOnceWithCapturedHeadersAndOwnedLocation`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsOriginalSourceMutationIsTerminalBeforeRestoringOption`, `blockstorage/volume_reads_options_test.go:TestVolumeReadPreflightStopsOptionsAndHTTPOnInvalidSourceOrContext`, `connection_volume_reads_test.go:TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection`, `connection_volume_reads_test.go:TestConnectionVolumeReadsLocationFactoriesOwnInputsAndLeaveConnectionDefault`, `connection_volume_reads_test.go:TestConnectionVolumeReadsOriginalCallbackSliceAndRetainedLocationAreSnapshots`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeIsCapturedAfterOptionsAndRefreshedPerCall`, `connection_volume_reads_test.go:TestConnectionVolumeReadsPreflightAndOptionFailuresSelectNoService`, `connection_volume_reads_test.go:TestConnectionVolumeReadsGetterFailureRetainsOperationAndCancellationCause`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeFailurePrecedesGetterAndOwnedOverrideBypassesIt`, `blockstorage/volume_reads_transport_test.go:TestGetVolumeByIDNativeAuthenticationAndRetryKeepFixedOwnedRequest`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadNativeRetryCannotChangeBodyDecoderOrAcceptedStatus`.
- Caller-owned Location JSON is validated only when an actual resource is normalized; an empty ListVolumes list does not consume an unused invalid location. Invalid caller location remains a local ErrInvalidOption without borrowing a200 ResponseError, while any independently admitted Pages/Observed evidence remains available and logical outputs stay uncommitted. Physical canonical row/descriptor failures retain their own originating accepted response error evidence.
  Tests: `blockstorage/volume_reads_options_test.go:TestVolumeReadLocationIsConsumedOnlyByActualRowsWithoutBorrowingResponseFailure`, `blockstorage/volume_reads_contract_test.go:TestListVolumesEagerDescriptorFailureStopsBeforeNextAndDiscardsPartialValues`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues`.

Python/Go differences:

- Go selects Cinder v3 under Gophercloud v2.15.0 and uses the audited 38-field normalized Volume view plus an independent RawResource physical row and response proof. Pinned cloud can select v2; this row does not implement a v2 projection or promote native/Proxy/Resource/session/cache declarations.
- Complete canonical UTF-8 volume(s) envelopes and accepted200 responses are the deliberate Go decoder boundary. Python generic Resource translation can tolerate invalid JSON on member fetch, unwrap absent envelopes more permissively, wrap nonarray resource lists, strip self, and preserve seeded omitted IDs. Go raw evidence preserves unknown/self fields while normalized known fields exclude them.
- Computed location follows existing owned CloudLocation/recorded provider-scope mapping. Exact decimal raw precision, Unicode/version and configurable Python numeric conversion boundaries are inherited from the already audited normalized view; no Python object identity, mutable Resource lifecycle or warning subsystem is recreated.
- Go returns richer owned result/evidence structures with an error; rejected native status/body/header remain in their original error. Admitted phase proof survives later failures without establishing successful resource/boolean output, and no SDK orchestration replay or swallowed cancellation is introduced.
- Go direct identifier validation is stronger than dynamic Python Resource/URL interpolation. Existing Python Volume instances or arbitrary proxy values are not new direct cloud id APIs; this typed helper accepts a literal string ID only.
- Go direct getter errors on404 instead of returning an absence flag. The Python get_volume_by_id declaration delegates proxy get_volume/_get/fetch directly and also raises despite its inaccurate else-None docstring.
- Local caller Location errors are separate from physical response/descriptor errors: actual admitted proof remains inspectable, but a local error is not reclassified as a200 response failure. This preserves the existing shared GetVolume/SearchVolumes normalizer behavior.

### `python:cloud/_block_storage/BlockStorageCloudMixin/volume_exists`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.VolumeExists`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.VolumeExists`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadLocation`

- VolumeExists performs default unfiltered exact GetVolume lookup and tests the returned resource identity against nil. Any successfully returned raw resource, including canonical empty volume {}, completes Exists=true independently of body truthiness or normalized ID presence. Successful completed absence completes Exists=false; every error leaves Exists=nil and no successful Value/Volume.
  Tests: `blockstorage/volume_reads_contract_test.go:TestVolumeExistsDistinguishesPresentResourceFromSuccessfulAbsence`, `connection_volume_reads_test.go:TestConnectionVolumeReadsMember404IsErrorButCompletedExistenceAbsenceIsFalse`.
- Safe identity segments attempt the exact bodyless member GET first. Only rejected compatible400/403/404 allow the detailed name-hinted list fallback; accepted read/Close/schema/source/context errors and native/transport/authentication errors never establish absence or trigger that fallback. Lookup Value/Volume/Observed/Pages evidence is retained consistently without borrowed stale native errors.
  Tests: `blockstorage/volume_reads_contract_test.go:TestVolumeExistsDistinguishesPresentResourceFromSuccessfulAbsence`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof`, `blockstorage/volume_reads_transport_test.go:TestVolumeExistsFallbackFailureNeverBorrowsEarlierRejectedMember`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadsPostResponseSourceChangesStopWithCurrentAdmittedProof`.
- Names unsafe for a direct ID route use the exact name-hinted list directly under the existing Go safety policy. Wildcard characters remain literal exact ID/name matches; no SearchVolumes glob/dictionary/JMESPath branch is introduced and no filter or all-project/details controls are exposed.
  Tests: `blockstorage/volume_reads_contract_test.go:TestVolumeExistsUnsafeNamesAndWildcardTextUseExactNameSemantics`.
- The fallback iterator consumes full visited normalized descriptors while searching exact ID/name matches and reports ambiguity on the second match. It stops before unused later rows or continuation; any rejected list response or later read/decoder/source/context failure propagates with Exists=nil rather than being treated as false.
  Tests: `blockstorage/volume_reads_contract_test.go:TestVolumeExistsAmbiguityStopsBeforeUnusedTailThatFullListMustConsume`, `blockstorage/volume_reads_transport_test.go:TestVolumeExistsFallbackFailureNeverBorrowsEarlierRejectedMember`, `blockstorage/volume_search_test.go:TestGetVolumeDefaultExactListMissingAndAmbiguityKeepSharedIteratorPolicy`.
- Existing GetVolume cancellation precedence and original memberFailure joining survive shared getter/default-find factoring. Connection and package original callbacks are not applied again by an outer exists wrapper; prepared state and response sinks are reused.
  Tests: `blockstorage/volume_reads_transport_test.go:TestVolumeReadsAcceptedReadCloseAndCancellationKeepAllCausesAndCurrentProof`, `blockstorage/volume_reads_transport_test.go:TestVolumeExistsFallbackFailureNeverBorrowsEarlierRejectedMember`, `connection_volume_reads_test.go:TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection`, `connection_volume_reads_test.go:TestConnectionVolumeReadsAcceptedProofSurvivesReadCloseCancellationAndSourceFailure`, `blockstorage/volume_search_test.go:TestGetVolumeOmittedNullGETFirstAndCompatibleListFallback`.
- Owned VolumeReadOpts.Location is the only read policy. Original callbacks apply once in order; direct package captures Cinder/header facts before originals and guards sticky context/source through every phase. Connection prepares originals once before its cached Cinder v3 getter and delegates an owned policy, supplying recorded CurrentLocation unless overridden. Captured ordinary headers are independent while provider authentication stays live; empty source Type is normalized only on the execution copy.
  Tests: `blockstorage/volume_reads_options_test.go:TestPrepareVolumeReadOptionsOwnsFactoriesCallbacksAndOriginalSlice`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsApplyOriginalOptionsOnceWithCapturedHeadersAndOwnedLocation`, `blockstorage/volume_reads_options_test.go:TestVolumeReadsOriginalSourceMutationIsTerminalBeforeRestoringOption`, `blockstorage/volume_reads_options_test.go:TestVolumeReadPreflightStopsOptionsAndHTTPOnInvalidSourceOrContext`, `connection_volume_reads_test.go:TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection`, `connection_volume_reads_test.go:TestConnectionVolumeReadsLocationFactoriesOwnInputsAndLeaveConnectionDefault`, `connection_volume_reads_test.go:TestConnectionVolumeReadsOriginalCallbackSliceAndRetainedLocationAreSnapshots`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeIsCapturedAfterOptionsAndRefreshedPerCall`, `connection_volume_reads_test.go:TestConnectionVolumeReadsPreflightAndOptionFailuresSelectNoService`, `connection_volume_reads_test.go:TestConnectionVolumeReadsGetterFailureRetainsOperationAndCancellationCause`, `connection_volume_reads_test.go:TestConnectionVolumeReadsRecordedScopeFailurePrecedesGetterAndOwnedOverrideBypassesIt`.
- Caller-owned Location JSON is validated only when an actual resource is normalized; an empty ListVolumes list does not consume an unused invalid location. Invalid caller location remains a local ErrInvalidOption without borrowing a200 ResponseError, while any independently admitted Pages/Observed evidence remains available and logical outputs stay uncommitted. Physical canonical row/descriptor failures retain their own originating accepted response error evidence.
  Tests: `blockstorage/volume_reads_options_test.go:TestVolumeReadLocationIsConsumedOnlyByActualRowsWithoutBorrowingResponseFailure`, `blockstorage/volume_reads_contract_test.go:TestListVolumesEagerDescriptorFailureStopsBeforeNextAndDiscardsPartialValues`, `blockstorage/volume_reads_transport_test.go:TestVolumeReadStrictAcceptedEnvelopesKeepPhysicalEvidenceWithoutLogicalValues`.

Python/Go differences:

- Go selects Cinder v3 under Gophercloud v2.15.0 and uses the audited 38-field normalized Volume view plus an independent RawResource physical row and response proof. Pinned cloud can select v2; this row does not implement a v2 projection or promote native/Proxy/Resource/session/cache declarations.
- Complete canonical UTF-8 volume(s) envelopes and accepted200 responses are the deliberate Go decoder boundary. Python generic Resource translation can tolerate invalid JSON on member fetch, unwrap absent envelopes more permissively, wrap nonarray resource lists, strip self, and preserve seeded omitted IDs. Go raw evidence preserves unknown/self fields while normalized known fields exclude them.
- Computed location follows existing owned CloudLocation/recorded provider-scope mapping. Exact decimal raw precision, Unicode/version and configurable Python numeric conversion boundaries are inherited from the already audited normalized view; no Python object identity, mutable Resource lifecycle or warning subsystem is recreated.
- Go returns richer owned result/evidence structures with an error; rejected native status/body/header remain in their original error. Admitted phase proof survives later failures without establishing successful resource/boolean output, and no SDK orchestration replay or swallowed cancellation is introduced.
- Pinned volume_exists returns self.get_volume(name_or_id) is not None with no filters argument. Go uses a nullable *bool to distinguish completed true/false from errors and retains owned lookup proof; it does not call filtered SearchVolumes or test JSON truthiness.
- The existing Go safe-route FindIdentity policy skips member GET for identities containing slash/backslash/query/fragment/percent or whitespace and searches exact list identity directly. Pinned Resource.find unconditionally attempts GET first and catches400/403/404. This routing safety boundary is explicit, rather than claimed as unchanged Python dispatch.
- The exact find branch keeps its own v3 canonical/native pagination and strict normalized-view boundaries; generic Resource.find/list/fetch, warnings, proxy lifecycle and cache declarations remain independently reviewed.
- Local caller Location errors are separate from physical response/descriptor errors: actual admitted proof remains inspectable, but a local error is not reclassified as a200 response failure. This preserves the existing shared GetVolume/SearchVolumes normalizer behavior.

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

func ReadVolumes(ctx context.Context, conn *sdk.Connection, id, name string) error {
    listed, err := conn.ListVolumes(ctx)
    if err != nil {
        if listed != nil { fmt.Println("admitted list pages:", len(listed.Pages)) }
        return err
    }
    fmt.Println(string(listed.Value), len(listed.Volumes))

    found, err := conn.GetVolumeByID(ctx,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil {
        if found != nil && found.Observed != nil {
            fmt.Println("admitted member response:", found.Observed.StatusCode)
        }
        return err
    }
    fmt.Println(string(found.Value), string(found.Volume.Body["id"]))
    copy := found.Volume.Clone()
    var projection struct { ID *string `json:"id"` }
    if err := copy.Decode(&projection); err != nil { return err }

    exists, err := conn.VolumeExists(ctx,
        blockstorage.VolumeExistsRequest{NameOrID: name})
    if err != nil {
        if exists != nil { fmt.Println("admitted fallback pages:", len(exists.Pages)) }
        return err
    }
    if exists.Exists == nil { return fmt.Errorf("missing completed existence result") }
    fmt.Println("exists:", *exists.Exists)
    return nil
}

func ListWithLocation(ctx context.Context, conn *sdk.Connection) error {
    cloud, project := "configured-cloud", "configured-project"
    location := resource.CloudLocation{
        Cloud: &cloud,
        Project: resource.CloudProject{
            ID: json.RawMessage(`"project-id"`), Name: &project,
        },
    }
    result, err := conn.ListVolumes(ctx, blockstorage.WithVolumeReadLocation(location))
    if err != nil { return err }
    fmt.Println(string(result.Value))
    return nil
}

func ReadDirect(ctx context.Context, cinder *gophercloud.ServiceClient, id string) error {
    listed, err := blockstorage.ListVolumes(ctx, cinder)
    if err != nil { return err }
    fmt.Println(len(listed.Volumes), len(listed.Pages))
    volume, err := blockstorage.GetVolumeByID(ctx, cinder,
        blockstorage.GetVolumeByIDRequest{ID: id})
    if err != nil { return err }
    exists, err := blockstorage.VolumeExists(ctx, cinder,
        blockstorage.VolumeExistsRequest{NameOrID: id})
    if err != nil { return err }
    fmt.Println(string(volume.Value), *exists.Exists)
    return nil
}
```

```python
import openstack

conn = openstack.connect(cloud="example")
volumes = conn.list_volumes()
volume = conn.get_volume_by_id("volume-id")
exists = conn.volume_exists("data")

volumes = conn.list_volumes(cache=False)
```

## References

- [Volume Id](volume-id.md)
- [Search Volumes](search-volumes.md)
- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py)
- [v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
