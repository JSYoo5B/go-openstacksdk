# Flavors

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/fetch_flavor_extra_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FetchExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsHeader`

- FetchExtraSpecs always performs one logical bodyless GET at the captured prefixed flavors/{id}/os-extra_specs route, with no flavor lookup or inline extra_specs shortcut. ID-only input seeds the requested ID; existing native or raw flavor fields are copied before options, keep their original fields, and only the Resource extra_specs value is replaced. Native Swap survives its json-excluded native field and native is_public/ephemeral wire aliases become source view names. An explicit route ID does not rewrite a supplied passive Resource ID; absent route ID uses the existing truthy string id/name/original_name identity.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- A missing response extra_specs becomes an empty object. Record.ExtraSpecs and independently owned actual whole-root Wire preserve present object, null, array, scalar, string, false and large precise JSON values. Resource.extra_specs reproduces Flavor's dict descriptor: objects and null remain, while nonnull nonobjects project to an empty object. Other actual response fields do not replace the existing flavor. Resource retains owned input metadata while record/Wire headers and status plus Envelope preserve the fresh physical response; input, Resource, Wire and raw result bytes do not alias.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- A selected microversion is retained without an inline-specs or 2.61 gate. With no selection, finite shared Nova discovery selects the advertised maximum capped at 2.61 or a versionless request for unavailable bounds; an explicit empty override bypasses discovery and issues a versionless private GET. Source token and headers remain live/owned, options apply once and mutable typed pointers/header inputs are snapshotted before later option mutations, while the original service client's version and route remain unchanged.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- Owned reads admit final200..399 but require a nonnull UTF-8 JSON object root; malformed/empty accepted JSON and parsed nonobject roots preserve the accepted receipt rather than being tolerated. HTTP403 retains original native status, expected codes, method, fixed URL and response body/header. Accepted read/Close nested404 and post-fetch source failures remain terminal ResponseErrors with the actual receipt and original cause, never a missing result, follow-up request or accepted-response replay. Invalid concrete input, nil options, callback cancellation, source drift, outer guard and spoofed auth header stop HTTP.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.

Python/Go differences:

- Python accepts a string or connected Flavor and mutates/returns the same resource. Go accepts a concrete ID/native Flavor/raw Resource carrier, returns an independently owned updated input view plus a raw specs value and physical receipt, and never mutates the supplied input. Raw existing fields and metadata are preserved rather than recreating every Flavor descriptor default or injecting computed Connection location.
- Explicit ID with an existing input is an additional route-only Go selection. With no explicit ID, only a validated safe UTF-8 string id/name/original_name can become the executable path; Python's arbitrary dynamic truthy identity and custom base_path/session controls are not exposed. Passive original IDs remain in the input view, even when a different validated string selects the request route.
- The Resource extra_specs object/null/empty projection matches the pinned type=dict descriptor; the additive ExtraSpecs field deliberately exposes the original nonobject value that the Python property converts to {}. Actual Wire and Envelope never contain invented defaults or input fields. ID-only Resource is a finite id plus extra_specs view, not a fully connected default-hydrated Python Flavor.
- Nil operation microversion uses a captured selected version or the existing finite guarded Nova discovery capped at Flavor2.61. Explicit empty is an additional versionless Go override. Full Keystoneauth discovery ordering/cache/session facilities retain independent SDK scope; discovery physical/source/JSON failures are explicit Go errors rather than blanket source DiscoveryFailure swallowing.
- Final200..399 and strict UTF-8 object parsing are the mapped Go transport boundary. Python accepts status<400 through its session and response.json text decoder; informational/session redirect/charset combinations are not asserted as identical. Physical read/Close/source/context faults retain typed receipt evidence. Native ListExtraSpecs remains a separate strict200 string-map API.
- This review maps the directly named unconditional fetch operation only. find_flavor/get_flavor/flavors optional inline enrichment, create/update/delete individual extra-specs, mutable Resource lifecycle, computed location and other service declarations keep independent tracking; none is incidentally promoted by this read.

### `gophercloud:compute/v2/flavors.ListExtraSpecs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListExtraSpecs`

- The existing generated native binding performs the bodyless selected-client GET at flavors/{id}/os-extra_specs with no query, lookup, discovery or new owned projection. Live token, source headers and both selected Compute version headers remain present. Successful200 returns the original map[string]string extraction ABI: missing/null specs are nil, an empty object is an empty map, a null member becomes an empty string and a numeric member preserves the native partial map with its json.UnmarshalTypeError.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorListExtraSpecsNativeBindingKeepsStrictStringMapABI`.
- Native201/204/403 remain HTTP failures with original expected200, method, exact member URL, body and response headers. Malformed JSON, truncated JSON and nonobject array root preserve the native syntax, io.ErrUnexpectedEOF and type causes respectively. Adding FetchExtraSpecs does not change this native API's result, selected microversion, route or extraction policy.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorListExtraSpecsNativeBindingKeepsStrictStringMapABI`.

Python/Go differences:

- Go retains the pinned native string-map and strict200 API and adds request.Wrap operation context around original errors. The raw source-shaped FetchExtraSpecs is a separate additive API and does not redefine native ListExtraSpecs or count other extra-spec declarations as completed.

### `python:compute/v2/get_flavor_extra_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpecsProperty`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsHeader`

- GetExtraSpecsProperty reads only the selected flavor/property member route with an empty body and no query, flavor GET, list or whole-specs GET. The concrete ID/native Flavor/RawResource input and selected source are captured before option callbacks; inline specs never skip this read, explicit ID determines the route, and the supplied existing extra_specs remain unchanged.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- The selected Value remains exact owned JSON rather than a string/dict conversion: string, precise large number, array, object and explicit null are retained. Missing properties return Value null and Present false, while explicit null is Present true. Wire contains only the actual flat response, and Value/Envelope/Header do not alias Wire. Actual final200/201/202/203/300 payloads retain their body/header/status receipt.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`.
- The property uses the same concrete read-options/source/version engine as FetchExtraSpecs: selected2.100 is retained, an unselected client performs one finite discovery and chooses2.61, and explicit empty is versionless without changing the caller client. Options execute once before discovery/member GET; input identity is already snapshotted when a callback changes caller models. Empty/dot/control property selections fail before callbacks and HTTP.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.
- Actual HTTP404 stays a strict native failure with original method/URL/body/header and no missing normalization. Accepted read/Close causes, source drift and nonobject response roots retain the actual partial receipt with nil Value/Wire and a ResponseError; nested404 cannot become ignored missing. Accepted processing errors make no SDK resend or native RetryFunc callback, while native provider handling of an actual HTTP failure is preserved.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.

Python/Go differences:

- Python accepts an ID or connected mutable Flavor; Go uses the concrete FlavorExtraSpecsRequest input and independent record, snapshots existing values, and exposes Present to distinguish missing from explicit null. The scalar selection does not update extra_specs or synthesize a Flavor Resource view.
- Python's cast(str | None, response.json().get(prop)) performs no runtime conversion. Go likewise retains arbitrary actual member JSON in Value, including exact large integers, rather than using native GetExtraSpec's string map.
- Go validates a nonempty valid property without controls/dot segments and escapes one literal path component, including slash/percent/space/reserved characters. Python utils.urljoin strips slash components; Go also requires a safe string route identity instead of allowing Python's broader dynamic identity values.
- Owned member reads accept final200..399 and require a valid UTF-8 nonnull JSON object; accepted failures include physical receipt and cause. WithFlavorExtraSpecsOptions/Microversion/Header are Go extensions; source's public named getter has no query/filter kwargs. Selected or finite2.61/versionless policy is shared with the separately reviewed FetchExtraSpecs.
- Full generic mutable Resource/session/cache and other flavor list/find declarations remain separate SDK goal work rather than additional required branches of this fixed single-property getter.

### `gophercloud:compute/v2/flavors.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Get`

- Get binds one bodyless queryless native GET to the selected ResourceBase flavors/{id} member using the live token/source headers and both selected Nova version headers. Native Flavor fields, unknown-field omission, numeric/string/blank swap conversion, missing public/specs zero defaults and nullable fields are preserved. Missing/null flavor wrappers return nil,nil; known-field and swap decode failures retain the native allocated/partially assigned Flavor together with their typed/numeric cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.
- Only native200 succeeds. Actual203/204/403 remain native failures with nil result, expected200, original method/URL/body/header and selected client unchanged; no list or specs lookup is introduced.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.

Python/Go differences:

- Go owns the exported concrete facade and wraps upstream errors while preserving native Get/Extract behavior and partial values. This native method does not add Cloud GetFlavorByID safe-result validation, ignored missing, Find fallback or optional specs enrichment.
- The returned Flavor is the native typed DTO; it does not synthesize Python descriptor defaults, unknown/location fields or connected mutable Resource state.

### `gophercloud:compute/v2/flavors.GetExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpec`

- GetExtraSpec makes one bodyless queryless native GET to flavors/{id}/os-extra_specs/{key}, retaining the selected token and both Nova version headers. Its result is the complete flat map[string]string, including unrelated string keys and a null member as an empty string; a missing requested property does not reduce the returned map. Null root retains nil map and empty object retains nonnil empty map.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.
- Numeric members retain the native partial map plus UnmarshalTypeError and an array root returns the native type failure. Only200 succeeds; actual203/204/404 preserve nil result and the original expected-code/method/URL/body/header proof without flavor lookup or ignored missing.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.

Python/Go differences:

- The library-owned facade preserves native route/result/status ABI and wraps errors; this whole string map is deliberately distinct from the owned raw scalar GetExtraSpecsProperty API.
- Native null string values become empty strings and nonstring JSON values produce native partial decode errors. The caller must check error even when a map is returned.

### `gophercloud:compute/v2/flavors.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailQuery`

- ListDetail owns a concrete request.Options builder while preserving the fixed flavors/detail native route. Default typed options omit is_public and all other query values. All eight native ListOpts fields reach their original wire spelling, raw limit/is_public extensions override typed values, and vendor query is retained. Options execute once at sequence construction while HTTP remains lazy; bodyless requests use the selected token and both version headers.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.
- Native Flavor/Swap/inline-specs DTOs and flavors_links continuation are preserved across two pages. Native200 empty and204 return no rows,300 page is accepted, and consumer break never fetches the next page. An empty native page stops even when it advertises a next link; no Python summary route, is_public default or item enrichment is invented.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies`.
- Late403 retains already yielded rows and yields one terminal error with original method/URL/body/header and expected200/204/300. First203, malformed JSON and typed malformed Flavor rows yield one terminal native error. A failing option produces no HTTP and retains the original callback cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.

Python/Go differences:

- Go exposes iter.Seq2[*Flavor,error] over the upstream native pager rather than returning a Pager and requiring caller extraction. Native FlavorPage/status/query behavior is preserved.
- Native fixed detailed-list paging does not add Python flavors(details=False), per-item get_extra_specs, semantic alias/Body filters, HTTP Link or marker fallback. Those separate public Python list/find declarations retain their own completion work.
- PrivateAccess/AllAccess behavior remains server-policy dependent; this mapping does not imply those requests are authorized for every core-user deployment.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavorByID`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDExtraSpecs`

- GetFlavorByID owns Cloud's default-false get_extra flag and strict GET-only behavior: one native flavor GET, no query/body/list/automatic discovery, and400/403/404 stay original HTTP errors rather than name fallback or nil success. It returns canonical native flavor fields and keeps the selected client2.55/live token.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`.
- Omitted/false enrichment makes no specs request. True fetches canonical returned ID once for missing/null/empty inline specs and keeps a nonempty inline map without another GET. Unrelated typed flavor fields are retained, caller query never leaks to specs, and actual specs404 is terminal. The existing native enrichment engine also preserves its missing/null-map and typed/transport/read/cancel error policy.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`.
- Typed FlavorByID options are reusable captured values and execute once per call. Nil service/context/option, unsafe ID and unsupported header/query/field/argument configurations fail before HTTP; source drift or option/cancellation error stops later callbacks and retains the original custom option/context causes.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`.
- The Cloud entry delegates to the already reviewed strict FindFallbackNever/IgnoreMissing(false) proxy-get mapping and checks the captured service/client/component binding around options and completion. The shared GET/enrichment implementation preserves live provider/version/context and rejects unsafe or null successful identity models before returning a Cloud result.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.

Python/Go differences:

- The pinned Cloud declaration delegates get_extra=False to compute.get_flavor and introduces no independent query/filter/list behavior. Go's concrete bool options own the strict Never/IgnoreMissing(false) combination instead of requiring callers to assemble it.
- Go returns the native Flavor DTO and requires a safe explicit input/returned ID. Native missing/null ExtraSpecs stays nil and missing IsPublic stays false, with no seeded mutable Python Resource identity/location/default reconstruction.
- The existing native GET/specs calls retain the configured microversion and native strict200 policy without automatic source Resource version discovery. Cloud binding checks do not replace the native provider's physical retry/reauth machine with owned MemberGet; the single-property API has its separate owned transport.
- Other Cloud/proxy flavor list/find/search and generic Resource/session/cache APIs remain in the original SDK goal independently of this ID-only alias.

### `python:compute/v2/flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListDetails`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilters`

- The owned flavor iterator defaults to /flavors/detail with Details=true and GetExtraSpecs=false; Details(false) selects /flavors. Semantic classification supports seven server-query attributes, min_disk/min_ram aliases, twelve local Body attributes and ignored unknown list attributes. Absent or semantic-null is_public sends literal None, while empty string and false remain distinct; canonical query spelling wins aliases in one bulk map, and conflicting semantic versus typed/raw query channels fail before HTTP.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `resource/filters_test.go:TestSemanticFilterSelectionKeepsMemberSpellingSeedAndDeferredUnknownErrors`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- Flat list rows project the thirteen declared Body values plus location into an owned view: raw id/name/original_name truthy fallback, omission-only name alias, absent defaults, explicit null, integer/boolean/float/dictionary response conversions and parsed response alias order. Nested flavor and vendor values remain actual Wire fields. Wire, row envelope, page headers and status stay independent; Service.ListFlavors captures current location once before options and adds it only to view copies.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordResponseAliasOrderAndIndependentReceipts`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONSourceConversionsAndOwnership`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONRejectsInvalidStringsAndNonfiniteWithCauses`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.
- Each iterator consumption prepares options lazily once and follows rel/href links, flavors_links, top-level next and HTTP Link. Initial effective positive limits allow last-consumed-raw-row logical markers even for filtered rows and caller-mutated views; the first advertised server limit is fixed without creating an initial-limit fallback. Raw row caps precede filters and avoid decoding unconsumed bad rows; first-page, empty-page and break stop later reads, while consumed malformed rows and late HTTP errors remain observable with actual receipts.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `internal/rest/list_test.go:TestListHTTPLinksAreOptInAndConflictingNextLinksFail`, `internal/rest/list_test.go:TestListContinuationCannotWidenCollectionFiltersOrMarkers`.
- GetExtraSpecs(true) enriches only yielded, locally matched rows whose projected dictionary is falsey. Nonempty inline dictionaries skip GET; missing, null, empty or normalized nonobject dictionaries fetch through the existing prepared extra-specs reader. Filtering, cap and break precede enrichment. The child has independent actual response evidence and Resource copies, uses the logical string identity without list query, and actual HTTP404 or accepted Close404 remains terminal with the original row retained.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Owned list/version policy preserves a selected client version, discovers at most 2.61 only when unselected, and supports explicit empty versionless override. One prepared policy and option application is shared with per-row enrichment; the original client is unchanged and live provider tokens remain in use. Model conversion, accepted-body failures, source/context failures and invalid controls retain their causes rather than fabricating successful rows or SDK resends.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- Service.ListFlavors supplies Connection location; direct leaf API.ListRecords has location=null. FlavorRecord separates the computed owned response view from actual Wire, row envelope, page metadata and optional child receipt rather than exposing a mutable Python Resource/session.
- Go query booleans use lowercase true/false and the shared query encoder supports scalar/null/scalar-array values. The common local matcher preserves JSON type distinctions and scalar-versus-object subset policy instead of Python bool-number equivalence, empty-dict truthiness and nested scalar attribute failures.
- Typed Limit/MaxItems zero means omission/unlimited and negatives are rejected; an effective wire limit is one positive integer and a marker is one nonblank string. Semantic and typed/raw wire query collisions are errors even for equal values; raw wire query can replace the matching typed limit or marker independently. Arbitrary session/path/resource_type, cache-bypass and deprecated Resource/JMESPath controls remain separate SDK goal scope.
- Owned list reads accept 200..399 but require complete nonnull UTF-8 JSON objects and flavors arrays or singleton objects. Accepted empty204 is an error rather than native empty-page compatibility. Invalid representations and descriptors retain actual response receipts; nonfinite float/overflow fails the finite JSON view, with audited IEEE-754/Unicode 16 conversions and independent actual raw values.
- Dictionary next links are an explicit Go compatibility extension; safe continuation checks prevent origin/route/filter widening and cycles while preserving reverse proxy prefixes. Logical markers and executable specs targets require string identities; numeric/container identities remain raw descriptor values and are never stringified into request paths.
- The named flavor list/conditional-enrichment workflow is mapped; global mutable Resource, session/cache/discovery lifecycle and unrelated native/Cloud flavor operations are not claimed complete by this row.

## References

- [User Read APIs](../../user-read-apis.md)
- [Flavor Extra Specs](../../flavor-extra-specs.md)
- [Flavor Property And Native](../../flavor-property-and-native.md)
- [Flavor Records](../../flavor-records.md)
