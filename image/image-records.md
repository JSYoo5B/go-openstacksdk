# Image Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/find_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageRecordListOptions`

- The pinned direct proxy accepts a string name_or_id and ignore_missing=True. Safe strings use native GET first; only compatible GET400/403/404 fall back to the normal list, with explicit strict, 404-only and GET-only Go options. Root-object GET200 extraction preserves the native Image model, additional Properties and import/store response headers.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesGETContractsAndNativeModel`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesMissingDefaultsAndFallbackBoundaries`.
- Only a complete successful normal-list absence triggers a second native LIST. It starts from the frozen caller query, overlays mandatory os_hidden=true, omits the automatic name hint and never repeats GET. Explicit name filters and repeated/nil/empty query-key precedence survive; caller hidden=false is overridden only for that second search.
  Tests: `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListWaitsForCompleteSuccessfulAbsence`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListPreservesCallerQueryAndMandatoryOverlay`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesHiddenPassUsesOriginalCallerQuery`.
- Both searches consume all native ImagePage continuations and compare exact ID OR name. Repeated-ID duplicates, malformed whole-page native extraction, invalid result IDs, cycle, late HTTP/decode errors and cancellation remain terminal. Normal errors never start the hidden pass; list403/404 are never ignored.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesNormalSearchTerminalNeverRetriesHidden`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesHiddenPagesAndTerminalFailures`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListErrorsAndCancellationAreTerminal`.
- IgnoreMissing is applied only after both lists complete successfully without a match; strict absence has no suppressed GET400/403 cause. GET-only never performs either list. Unsafe names retain their literal list-only policy; SDK control keys, unsupported details/all_projects and canceled/nil context fail before HTTP.
  Tests: `api/find_identity_images_test.go:TestNativeFindIdentityImagesMissingDefaultsAndFallbackBoundaries`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesPreflightUnsafeNamesAndCancellation`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListStrictAndFallbackPolicies`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListPolicyPreflightAndOtherOperationsStayUnchanged`.
- SDK policy metadata and caller options are deep snapshots, including callback mutation boundaries and concurrent reuse. Generated leaf and handwritten Image collections preserve reverse ResourceBase, original live provider token, headers, HTTP middleware and caller context across GET and both searches. Ordinary Get/List/explicit Ref lookup are unchanged.
  Tests: `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListSnapshotsAndConcurrentReuse`, `resource/find_identity_missing_list_test.go:TestCollectionFindIdentityMissingListFreezesOriginalQueryAcrossCallbacksAndUnsafeNames`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesQuerySnapshotsAndConcurrentReuse`, `api/find_identity_images_test.go:TestNativeFindIdentityImagesKeepsLiveProviderHeadersAndCallerContext`, `connectiontest/connection_find_identity_image_test.go:TestConnectionImageFindIdentityHiddenSearch`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- FindImageRecord performs a literal escaped ID GET with the owned Image projector; only clean final native400/403/404 enter exact ID-or-name listing. A complete successful normal-list absence triggers a hidden list using the frozen original caller query plus mandatory os_hidden=True and without the auto-name hint. Visible duplicates and normal-list failures never enter hidden lookup.
  Tests: `image/image_record_find_test.go:TestFindImageRecordDefaultHiddenPhases`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- Actual200..399 GET returns the owned seeded record, including tolerated invalid JSON; accepted malformed shapes/conversions and nonaccepted nonselected errors remain terminal. Default missing returns nil only after both successful empty searches; strict IgnoreMissing=false returns ErrNotFound without a suppressed GET cause.
  Tests: `image/image_record_find_test.go:TestFindImageRecordAcceptedGetAndAbsence`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- Both fallback searches consume the shared owned list pager, compare exact passive id/name, continue after the first candidate to prove uniqueness, and discard it after duplicates or later page/row/transport failures. Invalid passive row IDs may be compared but never form a new URI. The selected row is not re-fetched.
  Tests: `image/image_record_find_test.go:TestFindImageRecordDefaultHiddenPhases`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`, `resource/find_identity_response_id_test.go:TestCollectionFindIdentityPassiveIDsAreOnlyCompared`.
- Rejected selected400/403/404 body Read/Close/transport/cancellation/native-hook/source failures are terminal rather than hidden absence. PreserveCleanRetry is applied only to these selected Glance lookup rejections; actual rejected evidence and underlying causes are retained.
  Tests: `image/image_record_find_test.go:TestFindImageRecordRejectedOwnershipFaultsAreTerminal`, `image/image_record_find_test.go:TestFindImageRecordErrorsNeverBecomeHiddenSearch`.
- One prepared source/location snapshot and one concrete option evaluation span GET, normal and hidden pages. Nested List options and filters affect lists only; List.Headers apply to all phases and Find.Headers override them. Option slices/maps/pointers are copied; source checks between nested callbacks stop mutation before HTTP.
  Tests: `image/image_record_find_test.go:TestFindImageRecordOptionsAndLocationOwnAllPhases`, `image/image_record_find_test.go:TestFindImageRecordNestedCallbackSourceChangeStopsBeforeHTTP`.
- Root Connection executes owned literal get, two-page list and hidden find through the same source and current location, retaining independent declared/wire channels and live original auth. Existing native FindIdentity behavior and direct typed Image APIs are preserved as separate profiles.
  Tests: `connectiontest/connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`, `image/image_record_find_test.go:TestFindImageRecordOptionsAndLocationOwnAllPhases`.

Python/Go differences:

- Go exposes SDK-owned concrete options and additional raw wire query on FindIdentity; the pinned direct find_image declaration (openstack/image/v2/_proxy.py:1013-1033) accepts only name_or_id and ignore_missing. Python query alias is_hidden is not automatically transposed; Go uses os_hidden. details/all_projects are not capabilities of this Image binding.
- Unsafe route strings use literal list-only lookup and invalid identities/result IDs fail before further HTTP. Go keeps the native Image extractor/pager and full-page native decode policy; this does not claim Python descriptor coercion, arbitrary-precision unknown numeric fields or every inherited continuation form.
- Earlier contracts and differences above describe the preserved native/typed API profiles. The additional owned ImageRecord default profile supplies the missing declared Image projector, constructor packing, descriptor conversion, aliases, generic paging and lookup behavior described by the new contracts below; existing strict typed/native behavior is unchanged.
- The added ImageRecord APIs are fresh immutable Go snapshots for the named default profile. They do not mutate a supplied Python-like Resource or implement its dirty state, same-instance fetch, configured Adapter cache/conditional GET, dynamic session/microversion/base-path replacement, or arbitrary Python descriptor objects. These full SDK contracts remain explicitly tracked as SDK-R1, SDK-C1, SDK-J1 and SDK-S1 in docs/implementation-plan.md; API mapping here does not mean complete Python runtime equivalence or live cloud validation.
- FindImageRecord List options are a Go extension; the pinned public find_image signature accepts only name_or_id and ignore_missing. Find uses the owned JSON-domain projector/pager rather than the preserved native Image extractor, allows safe escaped slash/space IDs, and keeps unsafe control/dot strings preflight errors. Arbitrary Python Resource identity and session/cache behavior remain common SDK work.

### `python:image/v2/get_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetImageHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordAttributes`

- GetImage requires validated Ref; safe literal directID escapesonce and does no metadata/name lookup; exact queryfree/bodyless GET200, no IgnoreMissing/find fallback/UUID/readmaxlength gate.
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCompletePreflight`, `image/images_contracts_test.go:TestImageContractsGetRouteAndReference`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Name is an explicit Go extension: captured existing exact native Images.ResolveID with duplicate/missing/model/pager boundaries, source and resolved safeID recheck, followed by fresh raw GET; Python get_image usesliteralID or Resource.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_contracts_test.go:TestImageContractsGetRouteAndReference`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- Concrete GetImageOpts Headers/full replacement/lastwins; immediate mapsnapshots and copiedbeforeafter initializedcallbacks; nil/custom error preflight; no builder or borrowed source/header mutation.
  Tests: `image/images_core_test.go:TestImageCoreCompletePreflight`, `image/images_options_test.go:TestImageOptionsImmediateSnapshotsAndPresence`, `image/images_options_test.go:TestImageOptionsReplacementExtensionsAndLastWins`, `image/images_options_test.go:TestImageOptionsLazyCallbacksAndRetainedOwnership`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- ImageInfo explicitly decodes exact25 nullable canonicalfields atomically; exactint64/bool/stringpresence/tags and existing ImageLocation decoder; rawProperties projects noncanonical root values independently, no forced wirepropertiesobject/alias/ID/headerpseudoJSON seeding/outputschema gate.
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCanonicalPresenceAndAtomicDecoder`, `image/images_contracts_test.go:TestImageContractsCanonicalMetadataAndRawOwnership`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Source/provider/Type/Endpoint/base/Microversion and ordinaryheaders capturedbeforecallbacks throughName+firstGET, fixed optionheaders override/liveoriginalproviderauth; source/context checks beforework/afterName/accepted/afterdecode prevent retargeting.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_options_test.go:TestImageOptionsLazyCallbacksAndRetainedOwnership`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Strictactual200, direct404 remains error; acceptedread/Close/context/customcause/source/JSON/model failures niltypedresult with wholeactualResponseError bytes/Header/status, closeonce/no acceptedreplay; native unexpectedstatus/prebodypolicy remains.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_contracts_test.go:TestImageContractsResponseOwnershipAndNativePolicy`.
- Existing Service.Images collection/generated/native Get/List/Image aliases/Find/Wait and previous workflows preserved. APIv2.0/sourcepolicy404mask/version and responseenum/date/name255 schema are server facts, no discovery/cache/Resource mutation/header normalization parity claim.
  Tests: `image/images_contracts_test.go:TestImageContractsGetRouteAndReference`, `image/images_contracts_test.go:TestImageContractsResponseOwnershipAndNativePolicy`, `internal/cmd/sdkgen/glance_image_read_test.go:TestGlanceImageReadFacadePreservesNativeAndRegistry`.
- GetImageRecord accepts a literal ID or a captured RawResource/attribute seed, rejects conflicting literal-ID seeds and reserved constructor controls, validates the final string before HTTP, and performs a bodyless query-free single-escaped-segment GET with actual200..399 acceptance. Request/option attributes overlay raw names before one constructor packing pass; slash/backslash/space are passive ID bytes.
  Tests: `image/image_record_get_test.go:TestImageRecordGetPackingAliasesAndSeedOverlay`, `image/image_record_get_test.go:TestImageRecordGetCompletePreflightAndLegacyBoundary`.
- The pinned Image Body62 fields plus inherited id/tags produce 64 declared Body mappings and computed location, preserving canonical/wire owner aliases, default/null distinctions, self stripping and unknown properties packing. Ordered JSON aliases use first-position/last-value semantics, and scalar explicit properties wrap only when packing unknown values. Resource, actual Wire, raw Envelope and response metadata are independent.
  Tests: `image/image_record_get_test.go:TestImageRecordGetProjectionAndIndependentReceipts`, `image/image_record_get_test.go:TestImageRecordGetPackingAliasesAndSeedOverlay`.
- Image descriptor projection integrates Python truthiness, dictionary/list wrapping, int/float conversion, exact BoolStr, JSON-domain Python string representation and raw untyped values. Missing tags become an empty list while explicit null is retained. Exact case-insensitive actual import header values are folded then split without trimming; computed location comes from captured cloud facts.
  Tests: `image/image_record_get_test.go:TestImageRecordGetDescriptorIntegration`, `image/image_record_get_test.go:TestImageRecordGetLocationAndConcreteSnapshots`, `internal/cloudfilter/python_string_test.go:TestPythonStringKeepsWholeDescriptorValue`.
- Accepted empty/syntax-invalid JSON returns the seeded/default declared view with nil Wire; parsed nonobject, invalid whole UTF-8 and descriptor failures return no record and retain actual accepted receipt and causes. Clean nonaccepted errors preserve native Gophercloud error semantics.
  Tests: `image/image_record_get_test.go:TestImageRecordGetAcceptedToleranceAndNativeErrors`, `image/image_record_get_test.go:TestImageRecordGetDescriptorIntegration`.
- Options, the top-level seed attributes and RawResource are captured before caller marshaling/location/option callbacks. Concrete option replacement and immediate helper snapshots preserve caller ownership. One cloud-location snapshot and sticky endpoint/provider/base/type/microversion/service bindings govern the whole operation while original provider auth remains live.
  Tests: `image/image_record_get_test.go:TestImageRecordGetCapturesOptionsAndTopLevelAttributesBeforeMarshal`, `image/image_record_get_test.go:TestImageRecordGetLocationAndConcreteSnapshots`, `image/image_record_get_test.go:TestImageRecordGetCompletePreflightAndLegacyBoundary`.
- Accepted Read/Close/native-hook/context/source failures are terminal with physical response evidence, including a source change during Read restored by Close. Root Connection uses the same owned Image service and current location. Existing typed GetImage/native/generated APIs retain their separate contracts.
  Tests: `image/image_record_get_test.go:TestImageRecordGetReadCloseAndStickyGuardEvidence`, `image/image_record_get_test.go:TestImageRecordGetCompletePreflightAndLegacyBoundary`, `connectiontest/connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`.

Python/Go differences:

- Pinned Python mutableResource reuse/Adapter configuredcache/session/microversion, typeddescriptor coercion/commonimageproperties/owner aliases/self stripping/unknownproperties packing/headerimport parsing are broader anddifferent from owned canonical ImageInfo/rawProperties/header evidence.
- Canonical nullable Go types/exactint64/tags/locations are deliberateGo shapevalidation, not complete Python descriptor coercion or output JSONSchema/range/enum/date/UUID/name-length parity; URLs and aliases remainpassive.
- Go fixedsource/body/status/preflight/context ownership andacceptedResponseError are newcontracts; native originalprebody/advancedMoreHeaders/reauthwrapperfield behavior remains and no livecloud/sourcepermission guarantee follows.
- Go explicitName Ref extension performs existing native exactName lookup then freshGET. Python get_image is literalID/Resource fetch, no find/name fallback or Resource same-instance/cache semantics equivalence.
- Earlier contracts and differences above describe the preserved native/typed API profiles. The additional owned ImageRecord default profile supplies the missing declared Image projector, constructor packing, descriptor conversion, aliases, generic paging and lookup behavior described by the new contracts below; existing strict typed/native behavior is unchanged.
- The added ImageRecord APIs are fresh immutable Go snapshots for the named default profile. They do not mutate a supplied Python-like Resource or implement its dirty state, same-instance fetch, configured Adapter cache/conditional GET, dynamic session/microversion/base-path replacement, or arbitrary Python descriptor objects. These full SDK contracts remain explicitly tracked as SDK-R1, SDK-C1, SDK-J1 and SDK-S1 in docs/implementation-plan.md; API mapping here does not mean complete Python runtime equivalence or live cloud validation.
- Raw-name seed overlay is a deliberate Go input policy rather than Python mutable Resource._update component replacement. Go map alias processing uses deterministic sorted keys. Literal GetImageRecord does not perform implicit name lookup; the preserved GetImage typed Ref name extension remains separate.

### `python:image/v2/images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImages`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImages`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesLimit`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesMarker`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesName`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesMemberStatus`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesOwner`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesStatus`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSizeMin`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSizeMax`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesProtected`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesHidden`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSort`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSortKeys`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSortDirs`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesTags`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesCreatedAt`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesUpdatedAt`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesFilter`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImagesSinglePage`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordListPaginated`

- Concrete pointer query preserves nilomission versus explicitempty/0/false; emptyMarkeromitted, nonnegativeLimit/Size bounds withoutminmax inference; queryVALUEUTF8-only with escaped controls/no clientenum/date/UUID/length normalization/defaultserver injection.
  Tests: `image/images_core_test.go:TestImageCoreCompletePreflight`, `image/images_options_test.go:TestImageOptionsImmediateSnapshotsAndPresence`, `image/images_options_test.go:TestImageOptionsReplacementExtensionsAndLastWins`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- Filters ownedurl.Values unknownkeyextension, nonemptyUTF8/noncontrolkeys andnonemptyvalueslices; exact reservedknown/local/deleted keys fail. PluralFilters replacesentiremap; singularFilter replacescopiedvalues/zeroargsdeleteskey; no Pythonaliases/clientBodyfilters/builders.
  Tests: `image/images_options_test.go:TestImageOptionsImmediateSnapshotsAndPresence`, `image/images_options_test.go:TestImageOptionsReplacementExtensionsAndLastWins`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- Tags/SortKeys/SortDirs preserve ordered repeats andemptyvalues. Sortpointerpresence excludesnonemptyclassicstyles; multipleSortDirs count matchesSortKeys, remainingservergrammar preserved. Immediate helpers/fullopts own maps/slices/pointers before reusableparallel execution.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_options_test.go:TestImageOptionsImmediateSnapshotsAndPresence`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- Lazy ListImages copies optionslice atconstruction andappliesinitializedcallbacks perexecution; fulloptsreplacement/lastwins/customnil/error semantics, sourcerecheckaroundcallbacks; no GET when merelyconstructingiterator.
  Tests: `image/images_options_test.go:TestImageOptionsLazyCallbacksAndRetainedOwnership`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- WholeUTF8/syntax/canonicalnonnullroot/imagesarray; decodeconsumedrows only/localMaxItems0unlimited/positivecap/SinglePage/earlybreak skipunusedbadrows+next. No localfilter/order/dedup; Allnonnilempty ornilerror discards partials.
  Tests: `image/images_core_test.go:TestImageCorePagingGuardsAndLocalConsumption`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`.
- Every consumed ImageInfo row has exactnullablecanonical25/rawProperties+Body/header/status owned independently; IDs/Names/URLs/passive dates/unknown JSON retainrawprecision with no outputrange/alias/schema/enforcedID equality or extra detailGET.
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCanonicalPresenceAndAtomicDecoder`, `image/images_contracts_test.go:TestImageContractsCanonicalMetadataAndRawOwnership`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Onlycanonicalbody next, including short/emptyadvertisedpages; missing/null/emptyends, consumednextmuststring. Fixedsameorigin/exactescapedcapturedcollection, relative /v2/images versionroot normalization only, URLauthority/dot/fragment/pathalias/queryinvalid rejected; oneUTF8marker andURL/markercycles inclinitialmarker guarded.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_core_test.go:TestImageCorePagingGuardsAndLocalConsumption`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Advertisednonmarkerkeyset exactinitial andvalues exactorderedsequence OR knownsingletonmembercollapse only; rejectextras/drops/changed/arbitrarysubsets thenrestorewholeownedoriginalquery+newmarker. ExplicitGo repair for Glanceserializer dict(params) collapse, rawresponse unchanged; no Link/first/schema/lastID/hint fallback.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Followingpages refresh latestordinarysourceheaders underfixedoriginalclient/provider/Type/Endpoint/base/Microversion, fixedoptionheadersoverride/liveauth, checksaroundacceptedbody/consumedrows/callbacks; no source/providerreplacement or responseURL routing.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`, `image/images_contracts_test.go:TestImageContractsResponseOwnershipAndNativePolicy`, `connectiontest/connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Strictactual200 only, acceptedread/Close/context/customcause/source/envelope/model/continuation/cycle failures yieldnilrow andwholeactualResponseError/no replay/closeonce; native retry/reauth/advancedheaders/same-scope redirect andbody/status ownership remain. Native/Get/List/generator inventory preserve unchanged endpoints.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_contracts_test.go:TestImageContractsResponseOwnershipAndNativePolicy`, `internal/cmd/sdkgen/glance_image_read_test.go:TestGlanceImageReadFacadePreservesNativeAndRegistry`.
- Every consumed list row uses the same 64 Body mappings plus computed location, aliases, Source defaults/descriptor conversion and unknown-property packing; each record owns raw row Wire, whole-page Envelope/Header/status and declared Resource independently. The exact constructor body key OpenStack-image-import-methods is consumed and split, while actual list HTTP import headers are ignored.
  Tests: `image/image_record_lists_test.go:TestImageRecordListProjectionPackingAndImportMethods`, `image/image_record_lists_test.go:TestImageRecordListEnvelopeConstructorAndStatusEvidence`.
- The pinned16 remote query names use JSON-domain Python Requests expansion/repetition/null omission; is_hidden maps to os_hidden with canonical precedence. Other canonical Body conditions are evaluated locally after projection; ordinary unknown names are discarded before marshaling. Dedicated controls reject their raw Filter spellings, and explicit invalid selected values fail before HTTP.
  Tests: `image/image_record_lists_test.go:TestImageRecordListSourceQueryAndLocalFilterClassification`, `image/image_record_lists_test.go:TestImageRecordListCompletePreflightAndLegacyBoundary`.
- ListImageRecords is lazy and repeatable, captures the option slice at construction and evaluates copied concrete options/current location once per iteration. Nil limit is omitted, explicit0 is retained, positive max-items hints absent/zero server limit and caps consumed raw rows before local filtering; Paginated false reads one page. Header/default Accept snapshots and source guards govern all pages.
  Tests: `image/image_record_lists_test.go:TestImageRecordListLazyRepeatedOptionSnapshots`, `image/image_record_lists_test.go:TestImageRecordListPagingRawCapsAndPartials`.
- The shared generic pager handles explicit body continuation forms and HTTP Link, dictionary links, falsey next, versioned exact collection aliases and the first server-limit reduction. Positive caller limit permits raw-row marker fallback even on a short page; empty pages stop, raw marker remains independent of caller record mutations, and cycles/scope/query changes retain page evidence.
  Tests: `image/image_record_lists_test.go:TestImageRecordListPagingRawCapsAndPartials`, `internal/rest/list_short_page_test.go:TestListShortPageMarkerPolicyUsesWireRowsAndStopsOnEmptyPage`.
- Actual200..399 pages require a valid object envelope and decoded consumed rows; accepted204 with an empty body still fails JSON decoding. Singleton plural objects are accepted by the shared Resource envelope policy. Late decode/IO/Close/cancellation/source failures preserve earlier All results and actual whole-page evidence, while early break/cap leaves unconsumed rows and continuations untouched.
  Tests: `image/image_record_lists_test.go:TestImageRecordListEnvelopeConstructorAndStatusEvidence`, `image/image_record_lists_test.go:TestImageRecordListPhysicalIOAndPostYieldGuards`, `image/image_record_lists_test.go:TestImageRecordListPagingRawCapsAndPartials`.
- Root Connection list pagination shares captured current location and the native provider/auth source with direct owned Get and Find. Existing typed ListImages/AllImages and native/generated finite APIs retain strict typed extraction and their prior paging/filter policies. Existing public Gophercloud helpers and shared fixtures/pager/filter engines are reused.
  Tests: `connectiontest/connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`, `image/image_record_lists_test.go:TestImageRecordListCompletePreflightAndLegacyBoundary`.

Python/Go differences:

- Pinned Python mutableResource reuse/Adapter configuredcache/session/microversion, typeddescriptor coercion/commonimageproperties/owner aliases/self stripping/unknownproperties packing/headerimport parsing are broader anddifferent from owned canonical ImageInfo/rawProperties/header evidence.
- Canonical nullable Go types/exactint64/tags/locations are deliberateGo shapevalidation, not complete Python descriptor coercion or output JSONSchema/range/enum/date/UUID/name-length parity; URLs and aliases remainpassive.
- Go fixedsource/body/status/preflight/context ownership andacceptedResponseError are newcontracts; native originalprebody/advancedMoreHeaders/reauthwrapperfield behavior remains and no livecloud/sourcepermission guarantee follows.
- Go restores originalrepeatquery after boundedvalidadvertisedcollapse; pinned Python/native follow collapsed serverquery, so filter/sort preservation is an explicitGo repair. Go has body-next-only continuation andlocalcap/no serverhint; Python Resource pager has broader Link/fallback/clientBody/JMESPath/filter behavior.
- Typed knownwire filters plus unknown Filters extension are not Pythonaliases/arbitraryResource attributes/clientfilter parity. Server stillowns filtering/normalization/configurabledefaultlimit/policyvisibility/order; no capability/discovery/schema gate or snapshot-consistency claim.
- Earlier contracts and differences above describe the preserved native/typed API profiles. The additional owned ImageRecord default profile supplies the missing declared Image projector, constructor packing, descriptor conversion, aliases, generic paging and lookup behavior described by the new contracts below; existing strict typed/native behavior is unchanged.
- The added ImageRecord APIs are fresh immutable Go snapshots for the named default profile. They do not mutate a supplied Python-like Resource or implement its dirty state, same-instance fetch, configured Adapter cache/conditional GET, dynamic session/microversion/base-path replacement, or arbitrary Python descriptor objects. These full SDK contracts remain explicitly tracked as SDK-R1, SDK-C1, SDK-J1 and SDK-S1 in docs/implementation-plan.md; API mapping here does not mean complete Python runtime equivalence or live cloud validation.
- The pinned Proxy._list JMESPath branch is reachable from images(**query); ImageRecord Filter rejects jmespath_filters and per-call base_path/session/microversion controls explicitly. Arbitrary JMESPath result shapes and configured session/cache are pending common SDK-J1/SDK-S1/SDK-C1 contracts, not supported by this mapping. Local Go comparisons distinguish bool/number and use exact decimal rather than Python True==1/binary-float behavior. Server query validation, visibility and sorting remain Glance responsibilities.

### `gophercloud:image/v2/images.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithUpdateOptions`

- Generated API accepts concrete UpdateOpts and the SDK owns the upstream builder and Extract invocation. All nine native concrete Patch types retain order and raw zero/false/empty values. Property add/replace includes string value; remove omits it, and property Names retain native unescaped JSON Pointer paths.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`.
- Nil/allocated-empty UpdateOpts both issue PATCH JSON[], nil tags is null and allocated-empty tags is[]. WithUpdateOptions replaces the whole base set, including nil replacement. Custom inherited Patch map extension remains available; the SDK does not claim it removed the element interface.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateEmptyReplacementAndInheritedPatchInterface`.
- Nil/error option and nonempty generic body/query/header/argument extensions fail before HTTP with OperationError context and original cause. Custom inherited Patch JSON marshaling error also fails before transport with its original cause; concrete UpdateOpts does not expose arbitrary external UpdateOptsBuilder.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateRejectsRequestExtensionsAndMarshalFailuresBeforeHTTP`.
- Native Update uses service ResourceBase, PATCH and v2.1 JSON-patch content type with default OkCodes200 only. Native microversion/source headers/live auth/context, rejected-response RetryFunc and accepted JSON-decoder retry remain upstream policies. Default rejected201/203/204/404/503 preserve actual status/expected/method/URL/body/header in native error.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateStrictHTTPContextAndRetryPolicy`.
- Extract preserves native Image field/timestamp/unknown Properties handling, float64-based size and unknown number decoding, last import-method/store-ID header values and original response identity. It does not inject declared raw view, Envelope/status or an owned acknowledgement into the native result.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractsRichImageUnknownPropertiesAndLastHeaders`.
- Valid empty object returns native Image, valid JSONnull may return nil,nil. Body decode failures and invalid known types preserve upstream error plus possible partial Image; OperationError keeps causes inspectable without fabricating owned ResponseError or replacing partial fields.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractorErrorsKeepNativePartialResults`.

Python/Go differences:

- This mapping closes the pinned Gophercloud Update declaration with library-owned outer builder/options/result handling. It does not close Python update_image: native explicit patches always send HTTP, preserve caller order and default strict200 instead of mutable Resource dirty/no-op/sorted-diff/descriptor behavior.
- The SDK accepts concrete UpdateOpts instead of arbitrary external UpdateOptsBuilder. Its inherited Patch element interface remains optional for arbitrary maps; concrete nine Patch types cover the pinned built-in requests. Nil Patch elements retain native unsafe behavior; input slice/map/custom object snapshots and arbitrary callback safety are not added by this facade.
- Native IDs go through ServiceURL, property Names are interpolated without JSON Pointer escaping, schema/enum/range/readonly/policy/permission checks remain server-owned, and configured native client/hooks can change retry/header/response policies. Owned literal/route/body/source guards and ACK contracts are separate APIs.
- Native Image extraction retains its typed/null/date/float64/unknown-key behavior and possible partial pointer plus error or nil,nil. Accepted body Close/decoder/retry policy and response evidence are not promoted to the owned raw ImageRecord guarantees.
- The current/original raw body snapshot foundation preserves sparse presence before descriptor projection for future owned commit. It is not a mutable Resource dirty/header/cache/session state machine, and Python update_image remains unresolved with SDK-R1/C1/S1 work outstanding.
- Ordinary project-owned modify_image is a core-user candidate under the pinned server default; public visibility, owner and location operations can require additional admin policy. Local HTTP tests/builds establish client contracts, not deployed permissions/storage or actual cloud outcomes.

### `python:cloud/_image/ImageCloudMixin/list_images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllCloudImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilterDeleted`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryShowAll`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- AllCloudImageRecords eagerly consumes the complete default owned image list once, keeps every consumed row in Inventory and removes rows whose Python-lowered string status equals deleted unless FilterDeleted is false. Value is the completed JSON array of kept declared views, [] for an empty result, and Images aligns with it.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListDeletedFilterShowAllAndLocation`.
- ShowAll true overrides filter_deleted before the v2 query choice and adds only member_status=all; explicit false keeps the query-free default. The shared Connection location is projected into every Resource while actual Wire location is retained.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListDeletedFilterShowAllAndLocation`, `connectiontest/connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- With the status filter enabled a null, numeric or array status fails with the actual list-page ResponseError and retains Inventory through the failing row without completed Value/Images; disabled filtering never inspects status. A late native page failure retains consumed Inventory.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordListStatusAndPageFailuresKeepPartialInventory`.
- Present filters, including explicit null or malformed JSON, invalid headers and nil options fail before HTTP; nil clears an earlier filter. A cancelling option stops later callbacks and HTTP while retaining the caller cause.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`, `image/image_record_cloud_test.go:TestCloudImageRecordOptionsCancelBeforeLaterCallbacks`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- The Go image Service is Glance v2 only, so supports_version(image, '2') is always the v2 branch. This owned profile requires paired Unicode JSON status strings; Python str.lower() also accepts lone surrogates.

### `python:cloud/_image/ImageCloudMixin/search_images`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.SearchImageRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- SearchImageRecords completes the default list_images inventory, including the deleted-status filter and no member query, before the shared Cloud identifier exact-or-fnmatch phase. No name or filter is forwarded as a server query or Proxy Body filter; deleted rows are never selectable.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`, `connectiontest/connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Ordered dictionary filters inspect the 65 declared Image fields plus location, including nested location dictionaries; declared null attributes match while an unknown top-level key is a local error after inventory with retained Inventory. Falsey filters keep the identifier selection.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`.
- A JSON string filter runs the shared JMESPath engine on the selected views and returns arbitrary JSON in Value without inventing Images. Show_all/filter_deleted arguments, malformed filters and nil options fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordSearchSelectionAndExpressions`, `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- Python logs and ignores bad fnmatch patterns and raises AttributeError for unknown filter keys; Go uses the shared cloudfilter identifier and mapping contracts and returns an invalid-option error carrying the local cause.

### `python:cloud/_image/ImageCloudMixin/get_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- Omitted or JSON-null filters delegate to the owned Proxy find_image engine with ignore_missing true: literal GET, clean 400/403/404 fallback to the visible and hidden name lists, no Cloud deleted-status filter, nil Value/Image when missing and terminal native errors. Value is the found declared view.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetFindPathAndMissing`, `connectiontest/connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Every other filter value, including falsey JSON, uses search_images and then Python truthiness, len and index-0 selection. More than one element, including string code-point length, is ImageRecordSelectionError wrapping ErrAmbiguous; objects have no index 0; a selected JSON null or empty result is missing; Image is set only for one ordinary selected row while expression values stay raw.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetFilteredSelection`, `connectiontest/connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Filter_deleted/show_all arguments and an invalid find identity fail before HTTP.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- The find path validates nonblank control-free identities like FindImageRecord. Unlike Cloud get_flavor, falsey non-null filters select the search path because the pinned source tests filters is not None.

### `python:cloud/_image/ImageCloudMixin/get_image_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetImageRecordByID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordQueryHeaders`

- GetImageRecordByID delegates to one strict literal GetImageRecord without query, Find fallback, missing-as-nil or deleted filtering; native 404 remains an error named GetImageRecordByID without nesting the delegated operation, and the Connection location is projected.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordGetByIDIsStrictLiteralGet`, `connectiontest/connection_image_cloud_records_test.go:TestConnectionCloudImageRecordsShareSourceLocationAndLiveToken`.
- Only headers are accepted. Filters, show_all, blank IDs and nil options fail before HTTP, and each option runs as one checked Get option so cancellation stops later callbacks.
  Tests: `image/image_record_cloud_test.go:TestCloudImageRecordArgumentsFailBeforeHTTP`, `image/image_record_cloud_test.go:TestCloudImageRecordOptionsCancelBeforeLaterCallbacks`.

Python/Go differences:

- Python returns connected mutable v1/v2 Image Resource objects and emits deprecation warnings; Go returns independently owned ImageRecord views with actual Wire/Envelope/header/status evidence, concrete per-helper options and no warnings. No Resource identity, adapter cache, session or arbitrary transport parity is claimed.
- Headers are a Go extension applied to every request of the logical call. Arguments that the pinned Cloud declaration does not have are rejected before HTTP instead of becoming Python TypeError, and option callbacks run once with context/source checks between them.
- Go accepts a literal string ID only and rejects blank, dot-segment and control identities before HTTP; it does not accept a Resource seed or Attributes like the Proxy GetImageRecord request.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
image = conn.image.get_image("image-id")
print(image.to_dict(), image.image_import_methods)

for image in conn.image.images(status="active", limit=10, max_items=20):
    print(image.to_dict())

image = conn.image.find_image("ubuntu", ignore_missing=False)
print(image.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required image ID for literal GET")
    find := flag.String("find", "", "optional exact image name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(value *image.ImageRecord) error {
    body, err := json.MarshalIndent(map[string]any{
        "resource": value.Resource, "wire": value.Wire,
        "status_code": value.StatusCode, "import_methods": value.ImportMethods,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, nameOrID string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    value, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID},
        image.WithImageRecordHeader("X-Request-Source", "image-records-example"))
    if err != nil { return err }
    if err := printRecord(value); err != nil { return err }

    for value, readErr := range service.ListImageRecords(ctx,
        image.WithImageRecordListFilter("status", "active"),
        image.WithImageRecordListLimit(10), image.WithImageRecordListMaxItems(20)) {
        if readErr != nil { return readErr }
        if err := printRecord(value); err != nil { return err }
    }

    if nameOrID != "" {
        found, err := service.FindImageRecord(ctx, nameOrID,
            image.WithFindImageRecordIgnoreMissing(false))
        if err != nil { return err }
        if err := printRecord(found); err != nil { return err }
    }
    return nil
}
```

## References

- [Download API](download.md)
- [Images](images.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1013-L1054)
- [Install](../docs/install.md)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L45-L63)
- [QueryParameters validation·transpose](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L301-L382)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L79-L260)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L128)
- [properties packing](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1221-L1230)
- [fetch translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1394)
- [constructor collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L494-L531)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Image Record Get Test](image_record_get_test.go)
- [Image Record Lists Test](image_record_lists_test.go)
- [Image Record Find Test](image_record_find_test.go)
- [Connection Image Records Test](../connectiontest/connection_image_records_test.go)
- [Python String Test](../internal/cloudfilter/python_string_test.go)
