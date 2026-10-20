# Images

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCompletePreflight`, `image/images_contracts_test.go:TestImageContractsGetRouteAndReference`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Name is an explicit Go extension: captured existing exact native Images.ResolveID with duplicate/missing/model/pager boundaries, source and resolved safeID recheck, followed by fresh raw GET; Python get_image usesliteralID or Resource.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_contracts_test.go:TestImageContractsGetRouteAndReference`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- Concrete GetImageOpts Headers/full replacement/lastwins; immediate mapsnapshots and copiedbeforeafter initializedcallbacks; nil/custom error preflight; no builder or borrowed source/header mutation.
  Tests: `image/images_core_test.go:TestImageCoreCompletePreflight`, `image/images_options_test.go:TestImageOptionsImmediateSnapshotsAndPresence`, `image/images_options_test.go:TestImageOptionsReplacementExtensionsAndLastWins`, `image/images_options_test.go:TestImageOptionsLazyCallbacksAndRetainedOwnership`, `image/images_options_test.go:TestImageOptionsValidationAndParallelReusablePolicies`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`.
- ImageInfo explicitly decodes exact25 nullable canonicalfields atomically; exactint64/bool/stringpresence/tags and existing ImageLocation decoder; rawProperties projects noncanonical root values independently, no forced wirepropertiesobject/alias/ID/headerpseudoJSON seeding/outputschema gate.
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCanonicalPresenceAndAtomicDecoder`, `image/images_contracts_test.go:TestImageContractsCanonicalMetadataAndRawOwnership`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Source/provider/Type/Endpoint/base/Microversion and ordinaryheaders capturedbeforecallbacks throughName+firstGET, fixed optionheaders override/liveoriginalproviderauth; source/context checks beforework/afterName/accepted/afterdecode prevent retargeting.
  Tests: `image/images_core_test.go:TestImageCoreExactNameSourceAndOwnedFailures`, `image/images_options_test.go:TestImageOptionsLazyCallbacksAndRetainedOwnership`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
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
  Tests: `image/image_record_get_test.go:TestImageRecordGetReadCloseAndStickyGuardEvidence`, `image/image_record_get_test.go:TestImageRecordGetCompletePreflightAndLegacyBoundary`, `connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`.

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
  Tests: `image/images_core_test.go:TestImageCoreFixedGetAndCanonicalRawOwnership`, `image/images_core_test.go:TestImageCoreCanonicalPresenceAndAtomicDecoder`, `image/images_contracts_test.go:TestImageContractsCanonicalMetadataAndRawOwnership`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Onlycanonicalbody next, including short/emptyadvertisedpages; missing/null/emptyends, consumednextmuststring. Fixedsameorigin/exactescapedcapturedcollection, relative /v2/images versionroot normalization only, URLauthority/dot/fragment/pathalias/queryinvalid rejected; oneUTF8marker andURL/markercycles inclinitialmarker guarded.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_core_test.go:TestImageCorePagingGuardsAndLocalConsumption`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Advertisednonmarkerkeyset exactinitial andvalues exactorderedsequence OR knownsingletonmembercollapse only; rejectextras/drops/changed/arbitrarysubsets thenrestorewholeownedoriginalquery+newmarker. ExplicitGo repair for Glanceserializer dict(params) collapse, rawresponse unchanged; no Link/first/schema/lastID/hint fallback.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_contracts_test.go:TestImageContractsPagingAndConsumption`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
- Followingpages refresh latestordinarysourceheaders underfixedoriginalclient/provider/Type/Endpoint/base/Microversion, fixedoptionheadersoverride/liveauth, checksaroundacceptedbody/consumedrows/callbacks; no source/providerreplacement or responseURL routing.
  Tests: `image/images_core_test.go:TestImageCoreAdvertisedPagingAndQueryRestoration`, `image/images_contracts_test.go:TestImageContractsConcreteQueriesAndSource`, `image/images_contracts_test.go:TestImageContractsResponseOwnershipAndNativePolicy`, `connection_image_read_test.go:TestConnectionImageReadFacadeOwnsResponseAndPreservesFilters`.
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
  Tests: `connection_image_records_test.go:TestConnectionImageRecordsShareSourceLocationAndHiddenDiscovery`, `image/image_record_lists_test.go:TestImageRecordListCompletePreflightAndLegacyBoundary`.

Python/Go differences:

- Pinned Python mutableResource reuse/Adapter configuredcache/session/microversion, typeddescriptor coercion/commonimageproperties/owner aliases/self stripping/unknownproperties packing/headerimport parsing are broader anddifferent from owned canonical ImageInfo/rawProperties/header evidence.
- Canonical nullable Go types/exactint64/tags/locations are deliberateGo shapevalidation, not complete Python descriptor coercion or output JSONSchema/range/enum/date/UUID/name-length parity; URLs and aliases remainpassive.
- Go fixedsource/body/status/preflight/context ownership andacceptedResponseError are newcontracts; native originalprebody/advancedMoreHeaders/reauthwrapperfield behavior remains and no livecloud/sourcepermission guarantee follows.
- Go restores originalrepeatquery after boundedvalidadvertisedcollapse; pinned Python/native follow collapsed serverquery, so filter/sort preservation is an explicitGo repair. Go has body-next-only continuation andlocalcap/no serverhint; Python Resource pager has broader Link/fallback/clientBody/JMESPath/filter behavior.
- Typed knownwire filters plus unknown Filters extension are not Pythonaliases/arbitraryResource attributes/clientfilter parity. Server stillowns filtering/normalization/configurabledefaultlimit/policyvisibility/order; no capability/discovery/schema gate or snapshot-consistency claim.
- Earlier contracts and differences above describe the preserved native/typed API profiles. The additional owned ImageRecord default profile supplies the missing declared Image projector, constructor packing, descriptor conversion, aliases, generic paging and lookup behavior described by the new contracts below; existing strict typed/native behavior is unchanged.
- The added ImageRecord APIs are fresh immutable Go snapshots for the named default profile. They do not mutate a supplied Python-like Resource or implement its dirty state, same-instance fetch, configured Adapter cache/conditional GET, dynamic session/microversion/base-path replacement, or arbitrary Python descriptor objects. These full SDK contracts remain explicitly tracked as SDK-R1, SDK-C1, SDK-J1 and SDK-S1 in docs/implementation-plan.md; API mapping here does not mean complete Python runtime equivalence or live cloud validation.
- The pinned Proxy._list JMESPath branch is reachable from images(**query); ImageRecord Filter rejects jmespath_filters and per-call base_path/session/microversion controls explicitly. Arbitrary JMESPath result shapes and configured session/cache are pending common SDK-J1/SDK-S1/SDK-C1 contracts, not supported by this mapping. Local Go comparisons distinguish bool/number and use exact decimal rather than Python True==1/binary-float behavior. Server query validation, visibility and sorting remain Glance responsibilities.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "net/url"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func getImageExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.GetImage(ctx, resource.ID(imageID),
        image.WithGetImageOpts(image.GetImageOpts{}),
        image.WithGetImageHeaders(map[string]string{"X-Example": "image-get"}),
        image.WithGetImageHeader("X-Request-Id", "image-get-example"),
    )
}

func classicImagesExample(ctx context.Context, svc *image.Service, imageName, ownerID string) ([]*image.ImageInfo, error) {
    options := []image.ListImagesOption{
        image.WithListImagesOpts(image.ListImagesOpts{MaxItems: 0}),
        image.WithListImagesHeaders(map[string]string{"X-Example": "image-list"}),
        image.WithListImagesHeader("X-Request-Id", "image-list-example"),
        image.WithListImagesLimit(2),
        image.WithListImagesMarker(""),
        image.WithListImagesName(imageName),
        image.WithListImagesVisibility("private"),
        image.WithListImagesMemberStatus("all"),
        image.WithListImagesOwner(ownerID),
        image.WithListImagesStatus("active"),
        image.WithListImagesSizeMin(0),
        image.WithListImagesSizeMax(1 << 40),
        image.WithListImagesProtected(false),
        image.WithListImagesHidden(false),
        image.WithListImagesSortKeys("name", "created_at"),
        image.WithListImagesSortDirs("asc", "desc"),
        image.WithListImagesTags("ready", "approved"),
        image.WithListImagesCreatedAt("gte:2020-01-01T00:00:00Z"),
        image.WithListImagesUpdatedAt("lte:2030-01-01T00:00:00Z"),
        image.WithListImagesContainerFormat("bare"),
        image.WithListImagesDiskFormat("qcow2"),
        image.WithListImagesFilters(url.Values{
            "os_distro": {"ubuntu"}, "unused": {"remove-me"},
        }),
        image.WithListImagesFilter("architecture", "x86_64"),
        image.WithListImagesFilter("unused"),
        image.WithListImagesMaxItems(20),
        image.WithListImagesSinglePage(false),
    }
    rows := make([]*image.ImageInfo, 0)
    for value, err := range svc.ListImages(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, value)
    }
    return rows, nil
}

func modernImagesExample(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageInfo, error) {
    return svc.AllImages(ctx,
        image.WithListImagesID(imageID),
        image.WithListImagesSort("name:asc,created_at:desc"),
        image.WithListImagesMaxItems(20),
    )
}
```

```python
def images_example(conn, image_id, image_name, owner_id):
    image = conn.image.get_image(image_id)
    rows = list(conn.image.images(
        name=image_name, owner=owner_id, tag=["ready", "approved"],
        sort_key=["name", "created_at"], sort_dir=["asc", "desc"], limit=2,
    ))
    return image, rows
```

## References

- [generated/native Image API](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L243-L308)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L353-L550)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L517-L585)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L282-L329)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1034-L1055)
- [Locations](locations.md)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1756-L1827)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1855-L1871)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L564-L574)
- [Images Core Test](images_core_test.go)
- [Images Options Test](images_options_test.go)
- [Images Contracts Test](images_contracts_test.go)
- [Connection Image Read Test](../connection_image_read_test.go)
- [Glance Image Read Test](../internal/cmd/sdkgen/glance_image_read_test.go)
