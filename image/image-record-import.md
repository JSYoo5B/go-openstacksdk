# Image Record Import

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/import_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.API.ImportImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.API.ImportKnownImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportMethod`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportURI`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportRemoteRegion`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportRemoteImageID`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportRemoteServiceInterface`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportStore`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportStores`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportAllStores`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportAllStoresMustSucceed`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportField`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.WithImportMethodField`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ImportImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ImportImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportMethod`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportURI`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportRemoteRegion`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportRemoteImageID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportRemoteServiceInterface`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportAllStores`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportAllStoresMustSucceed`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportStore`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportStores`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportField`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportFields`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportMethodField`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordImportMethodFields`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.StoreRecord.ImportIdentity`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo.StoreImportIdentity`

- Legacy native/ref strict202 import profile only: ImportImage starts with a fresh metadata GET of the explicit ID. ImportKnownImage copies only supplied native ID/container/disk formats before option callbacks and sends no GET. The default request is one glance-direct POST with no empty URI or seeded model fields; incidental response IDs never retarget the requested import ID.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportFreshAndKnownAcknowledgements`, `image/v2/imageimport/import_core_test.go:TestImportCoreKnownSeedAndHTTPBodySnapshots`.
- Legacy native/ref strict202 import profile only: An explicit Name uses the existing Images collection exact-name and advertised native paging resolver once, then freshly GETs the resolved ID rather than reusing listed formats. Case decoys, missing names, duplicate matches and errors on later pages terminate before metadata GET or import POST.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportFreshFormatsAndAllPageNameBinding`.
- Legacy native/ref strict202 import profile only: Fresh metadata must be a flat UTF-8 object accepted as200, decode the complete native Image shape, and contain canonical lowercase nonempty container_format/disk_format strings. Missing/null/case-only/typed-invalid formats or unrelated malformed native fields/time preserve actual accepted GET body/header/status with ResponseError and never POST.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportFreshFormatsAndAllPageNameBinding`, `image/v2/imageimport/import_core_test.go:TestImportCoreFreshCanonicalFormatsAndNativeDecodeEvidence`.
- Legacy native/ref strict202 import profile only: Owned method options default empty Method to glance-direct and preserve unknown nonempty server methods. Web URI is required and prohibited for other methods. Glance-download requires region+image ID and emits optional interface only when present; incomplete/orphan/wrong-method remote values reject before lookup.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportMethodsAndRootStoreControls`, `image/v2/imageimport/import_options_test.go:TestImportOptionsMethodBodyAndOwnedNamespaces`.
- Legacy native/ref strict202 import profile only: Singular Store selects only X-Image-Meta-Store while plural Stores selects only the root JSON array. Explicit false root booleans remain present; nil booleans and nil/empty Stores omit. Store plus nonempty Stores or true AllStores conflicts, and explicit empty Store/empty entries reject; an inherited legacy store header conflicts with plural/all-store selection before HTTP.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportMethodsAndRootStoreControls`, `image/v2/imageimport/import_options_test.go:TestImportOptionsMethodBodyAndOwnedNamespaces`, `image/v2/imageimport/import_core_test.go:TestImportCorePreflightAndSourceBeforeEveryPhase`, `api/glance_import_contracts_test.go:TestGlanceImportExtensionsAndPreflightAreSDKOwned`, `api/glance_import_contracts_test.go:TestGlanceImportLiveProviderFixedRoutesAndSourceRechecks`.
- Legacy native/ref strict202 import profile only: Root Fields and MethodFields are separate SDK-owned JSON extension namespaces. Snapshot encoding retains nested JSON, null and exact json.Number values; protected core keys and aliases cannot replace method/store/booleans or method name/URI/remote controls. Non-JSON values fail before HTTP.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportExtensionsAndPreflightAreSDKOwned`, `api/glance_import_contracts_test.go:TestGlanceImportOptionSnapshotsReplacementAndReuse`, `image/v2/imageimport/import_options_test.go:TestImportOptionsMethodBodyAndOwnedNamespaces`, `image/v2/imageimport/import_options_test.go:TestImportOptionsOwnedSnapshotsReplacementAndConcurrentReuse`.
- Legacy native/ref strict202 import profile only: Ordinary headers are canonicalized with sequential last-wins helpers, frozen and independently reused. Different values under case aliases and protected auth/transport/version/store headers reject. A configured explicit image version header must agree with the selected source Microversion.
  Tests: `image/v2/imageimport/import_options_test.go:TestImportOptionsHeaderAuthorityAndCanonicalLastWins`, `image/v2/imageimport/import_core_test.go:TestImportCorePreflightAndSourceBeforeEveryPhase`, `api/glance_import_contracts_test.go:TestGlanceImportExtensionsAndPreflightAreSDKOwned`, `api/glance_import_contracts_test.go:TestGlanceImportLiveProviderFixedRoutesAndSourceRechecks`.
- Legacy native/ref strict202 import profile only: WithImportOpts snapshots pointers, Stores slices, header maps and both nested JSON extension maps and replaces the whole options value; individual helpers apply afterward. Bulk empty restores the default method and clears prior controls. Parsed custom configs, supplied KnownImage mutation during options and concurrent reusable options do not alter frozen request data or another response.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportOptionSnapshotsReplacementAndReuse`, `image/v2/imageimport/import_options_test.go:TestImportOptionsOwnedSnapshotsReplacementAndConcurrentReuse`, `image/v2/imageimport/import_core_test.go:TestImportCoreKnownSeedAndHTTPBodySnapshots`.
- Legacy native/ref strict202 import profile only: ImportResult reports the fixed ImageID and only actual accepted202 Body/Header/StatusCode. Empty, arbitrary acknowledgement and large-number bytes are passive owned evidence, not an active Image/Task model. A foreign Location is retained without following it or deriving identity.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportHTTPFailuresAndAcceptedEvidence`, `api/glance_import_contracts_test.go:TestGlanceImportFreshAndKnownAcknowledgements`, `image/v2/imageimport/import_core_test.go:TestImportCoreKnownSeedAndHTTPBodySnapshots`.
- Legacy native/ref strict202 import profile only: Native HTTP/transport, context and accepted body-read failures stay terminal submission errors without fabricated acknowledgements or semantic resend/cleanup. Accepted202 read failure returns the actual partial result and ResponseError; a concurrently expired context is joined with the original native404 evidence instead of hiding either cause.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportHTTPFailuresAndAcceptedEvidence`, `api/glance_import_contracts_test.go:TestGlanceImportReadTransportAndCancellationAreTerminal`, `image/v2/imageimport/import_core_test.go:TestImportCoreAcceptedReadFailureAndNativeErrors`, `image/v2/imageimport/import_core_test.go:TestImportCoreCanceledNativeErrorsPreserveBothCauses`.
- Legacy native/ref strict202 import profile only: The workflow captures client settings/headers and fixes ResourceBase-aware metadata GET/import POST routes while sharing the original Provider. Live token rotation, request context, selected Microversion and configured middleware survive GET/POST; configured401 reauth and503 retry retain the same import body and target. Invalidating source changes after options or GET stop before the next phase, redirects cannot change the image, and explicit Store overlays the configured legacy value without modifying the original client.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportLiveProviderFixedRoutesAndSourceRechecks`, `api/glance_import_contracts_test.go:TestGlanceImportFreshAndKnownAcknowledgements`, `image/v2/imageimport/import_core_test.go:TestImportCorePreflightAndSourceBeforeEveryPhase`, `image/v2/imageimport/import_core_test.go:TestImportCoreAcceptedReadFailureAndNativeErrors`.
- Legacy native/ref strict202 import profile only: Existing native Create/Get and error-only Create result, Image aliases and the unsupported specialized inventory row remain separate. Generator gates verify actual native functions/method constants and canonical string Image identity/formats, reject drift or unrelated opt-in, preserve generated native declarations byte-for-byte and do not invent an import resource collection.
  Tests: `api/glance_import_contracts_test.go:TestGlanceImportNativeCreateGetAndImageModelsStayUnchanged`, `internal/cmd/sdkgen/glance_import_test.go:TestGlanceImportKeepsActualNativeCallsAndNoInventedCollection`, `internal/cmd/sdkgen/glance_import_test.go:TestGlanceImportRejectsNativeIdentityFormatsAndConstantsDrift`.
- Select literal ID or SDK-produced Record private current ID once; escape once and ignore public Resource/Wire/Envelope/Location for routing. Source _get_resource performs no GET/list/find. ID-only Image has no formats and fails locally before POST.
  Tests: `image/image_record_import_test.go:TestImageRecordImportPreflightRejectsUnsafeSelectorsDescriptorsJSONAndProtectedNamespaces`, `image/image_record_import_test.go:TestImageRecordImportOpaqueAcceptedResponsesKeepPreparedRecordAndIndependentReceipt`, `image_record_import_integration_test.go:TestConnectionOwnedImageImportRejectsInvalidInvocation`.
- Existing Image _get_resource._update({}) preserves full current/original/dirty/presence, refreshes current location, resets plain image_import_methods and materializes every declared descriptor before HTTP. Prior fetch receipts and private borrowed data stay unchanged; immutable returned record does not alias caller.
  Tests: `image/image_record_import_test.go:TestImageRecordImportOpaqueAcceptedResponsesKeepPreparedRecordAndIndependentReceipt`, `image/image_record_import_test.go:TestImageRecordImportAcknowledgementDoesNotCleanPendingBodyOrDiscardBorrowedData`, `image/image_record_import_test.go:TestImageRecordImportCapturesPrivateImageStoreOptionsAndLocationBeforeCallbacks`.
- Unrelated descriptor projection may fail before import HTTP. container_format and disk_format are untyped Body values, requiring Python truthiness of both private current values rather than canonical nonempty strings. Any truthy complete JSON is allowed; missing/null/false/0/empty-string/list/object fail. SDK Source adds no local method status or format allow-list check.
  Tests: `image/image_record_import_test.go:TestImageRecordImportFormatsUseRawPythonTruthinessWithoutStatusOrAllowList`, `image/image_record_import_test.go:TestImageRecordImportPreflightRejectsUnsafeSelectorsDescriptorsJSONAndProtectedNamespaces`.
- Method default glance-direct applies only when omitted. Explicit None/null, empty/unknown string, boolean, number, list or object pass through as method.name; Source does not runtime-enforce its str annotation. Truthy URI still requires equality to exact string web-download. Preserve nil-omitted versus explicit raw-null via owned JSON presence, never blanket empty-to-default or non-string validation.
  Tests: `image/image_record_import_test.go:TestImageRecordImportMethodPreservesOmittedNullUnknownAndAdminBranchValues`.
- Any Python-truthy JSON URI value requires method equal to exact string web-download and enters method.uri unchanged. None/null/false/0/empty-string/list/object URI is omitted under every method. web-download without URI is submitted. Source annotations do not enforce URI string at runtime; legacy required-URI or Go string-only forms are separate profiles.
  Tests: `image/image_record_import_test.go:TestImageRecordImportURIUsesTruthyDomainAndExactWebDownloadGuard`.
- Remote region+image ID+service interface enter all3 method.glance_* fields iff all3 values are Python-truthy, independent of method. Complete non-string JSON triples (bool/number/list/object) enter unchanged; incomplete/None/falsey tuples silently omit. SDK Source does not runtime-enforce annotations, require a tuple for glance-download, or reject complete remote values under another method.
  Tests: `image/image_record_import_test.go:TestImageRecordImportRemoteTupleRequiresAllThreeTruthyValuesUnderAnyMethod`.
- Singular store and each plural store convert locally without GET/list/discovery. None singular omits; each plural None constructs bare Store with ID None. String/numeric/bool/list literals become id values; JSON dict/Munch constructs Store from its attributes and consumes id, with missing id default None and no name fallback. Store Resource uses private canonical raw id. Owned StoreRecord selection retains raw JSON ID plus missing/null semantics before callbacks, never public Resource/Wire/Envelope identity. Source dynamic-class bounds remain explicit.
  Tests: `image/image_record_import_test.go:TestImageRecordImportCapturesPrivateImageStoreOptionsAndLocationBeforeCallbacks`, `image/image_record_import_test.go:TestImageRecordImportPluralStoreConstructorsPreserveRawIDsOrderDuplicatesAndMissingNull`, `image/v2/serviceinfo/store_import_identity_test.go:TestStoreRecordImportIdentityRetainsPrivateIDAndReturnsOwnedCopies`, `image/v2/serviceinfo/store_import_identity_test.go:TestStoreRecordImportIdentityMissingAndForgedInputs`, `image/v2/serviceinfo/store_import_identity_test.go:TestStoreImportIdentityConstructorValuesAndOwnership`.
- Truthy all_stores conflicts with singular or nonempty plural stores; explicit false does not. Singular non-None store conflicts with nonempty plural stores. Empty/nil plural is no stores. Source checks/converts in Proxy and repeats store conflicts in Image; Go preserves selection presence and snapshots it.
  Tests: `image/image_record_import_test.go:TestImageRecordImportFlagsPreserveFalseyPresenceAndRejectTruthyStoreConflicts`, `image/image_record_import_test.go:TestImageRecordImportStoreSelectorsRejectConflictsUnsafeHeadersAndConstructorControls`.
- Singular store emits BOTH root stores:[same canonical raw ID] and X-Image-Meta-Store; empty string is preserved in array and empty header. Plural raw JSON IDs preserve order/duplicates and null/bool/number/list/object values, body only; empty stores sequence omits key. Source bodies do not validate the raw singular header value. Go singular non-string/null rejection is an explicit header-representation boundary, without claiming an equivalent Source None-header/session error.
  Tests: `image/image_record_import_test.go:TestImageRecordImportSingularStoreEmitsBodyAndCompatibilityHeaderAndRemovesConfiguredDecoys`, `image/image_record_import_test.go:TestImageRecordImportPluralStoreConstructorsPreserveRawIDsOrderDuplicatesAndMissingNull`, `image/image_record_import_test.go:TestImageRecordImportStoreSelectorsRejectConflictsUnsafeHeadersAndConstructorControls`, `image/v2/serviceinfo/store_import_identity_test.go:TestStoreImportIdentityConstructorValuesAndOwnership`.
- Root all_stores and all_stores_must_succeed preserve every non-None JSON value, including false, zero, empty string/list/object or other nonbool values; Source SDK never runtime-enforces bool annotations. all_stores conflict uses Python truthiness. Omitted/None fields stay absent; server owns default false/true and rejects nonbool. Typed bool helpers can coexist with raw JSON opts; backend success/cleanup remain server-owned.
  Tests: `image/image_record_import_test.go:TestImageRecordImportFlagsPreserveFalseyPresenceAndRejectTruthyStoreConflicts`.
- Concrete With options own pointers/maps/slices/store selectors, full-config replacement and ordinary-header merge. Compiled method/root namespaces cannot be overwritten by optional extension fields. These concrete JSON/option policies are Go additions, without caller builders.
  Tests: `image/image_record_import_test.go:TestImageRecordImportExtensionAndFullOptionsFactoriesAreSnapshotsAndReusable`, `image/image_record_import_test.go:TestImageRecordImportCapturesPrivateImageStoreOptionsAndLocationBeforeCallbacks`, `image/image_record_import_test.go:TestImageRecordImportPreflightRejectsUnsafeSelectorsDescriptorsJSONAndProtectedNamespaces`.
- Capture original Service/API/Images/provider/source/type/origin/microversion, ordinary source headers, option callbacks and current location once with sticky ancestor/context guards. SDK owns auth/framing/version/representation/store namespaces. Configured X-Image-Meta-Store is removed from captured child only. Shared RequestHeaderPolicy validates selected store-header value or explicit absence at physical dispatch/retry so native callbacks cannot add, remove or change store selection; ordinary native retry headers retain their existing policy. Store conversion remains local JSON without provider binding.
  Tests: `image/image_record_import_test.go:TestImageRecordImportCapturesPrivateImageStoreOptionsAndLocationBeforeCallbacks`, `image/image_record_import_test.go:TestImageRecordImportOptionAndNestedMarshalerGuardsStopLaterWorkWithCauses`, `image/image_record_import_test.go:TestImageRecordImportSingularStoreEmitsBodyAndCompatibilityHeaderAndRemovesConfiguredDecoys`.
- Submit one logical POST /images/{private ID}/import with frozen JSON/route/store-header policy. Native clean retry/reauth may resend exactly that request with live original provider token, while sticky source/ancestor/context and selected-value/absent store-header guards stop changed scope before physical resend. No lookup/discovery/stage/task/fetch/wait/checksum/cleanup/follow-Location is inserted; custom RoundTripper internal replay remains an explicit transport bound.
  Tests: `image/image_record_import_test.go:TestImageRecordImportNativeRetryFreezesBodyTargetStoreHeadersAndKeepsLiveAuthentication`, `image/image_record_import_test.go:TestImageRecordImportSafeReauthenticationRetriesOnlyCapturedJSONSubmission`, `image/image_record_import_test.go:TestImageRecordImportOptionAndNestedMarshalerGuardsStopLaterWorkWithCauses`.
- Image.import_image returns raw response without translate: opaque ACK does not parse/overlay/clean Body, read import-method headers, change image status or previous fetch receipts. Prepared Record keeps pending state and empty ImportMethods. Independent Acknowledgement preserves actual Body/Header/Status even invalid JSON/UTF-8.
  Tests: `image/image_record_import_test.go:TestImageRecordImportOpaqueAcceptedResponsesKeepPreparedRecordAndIndependentReceipt`, `image/image_record_import_test.go:TestImageRecordImportAcknowledgementDoesNotCleanPendingBodyOrDiscardBorrowedData`.
- Source Proxy.request defaults raise_exc=False and Import never calls raise_from_response, allowing raw400..599. Owned Go broad200399 plus explicit native400599 errors are documented policy. Rejection has no accepted ACK; accepted Read/Close/source/context failures retain only actual partial ACK and inspectable causes, without synthetic completion/recovery.
  Tests: `image/image_record_import_test.go:TestImageRecordImportAcceptedPhysicalFailuresRetainOnlyActualPartialAcknowledgement`, `image/image_record_import_test.go:TestImageRecordImportNativeRejectionsPreserveErrorEvidenceAndNeverAcknowledge`.
- glance-direct/web-download/glance-download and exact copy-image are mandatory same-public-function branches plus shared Connection facade. Glance exact copy_image policy is project-admin; ordinary modify_image is admin-or-project-member/owner. Server policy/quota/status/store/task/backend and SDK-R1/C1/S1 remain explicit; legacy API/tests/native ABI/catalog stay unchanged.
  Tests: `image/image_record_import_test.go:TestImageRecordImportMethodPreservesOmittedNullUnknownAndAdminBranchValues`, `image_record_import_integration_test.go:TestConnectionOwnedImageImportSharesSourceAndPreservesBorrowedData`, `image_record_import_integration_test.go:TestConnectionOwnedImageImportRejectsInvalidInvocation`.

Python/Go differences:

- Principal public graph: Source _get_resource converts supplied Image locally and projects current descriptors/location/import-method reset, validates truthy container/disk formats, converts singular/plural Store inputs, builds one interoperable import JSON request and returns a raw asynchronous POST response. Owned ImportImageRecord implements that complete finite graph with immutable SDK-produced private JSON state and concrete SDK-owned selectors. ID-only seed has no formats and fails locally; callers explicitly fetch first. Native/ref ImportImage and ImportKnownImage remain separate legacy strict202 profiles with existing API/test references retained and scoped.
- Legacy native/ref strict202 import profile only: Python _get_resource constructs or accepts mutable Image/Store Resources without a GET, and the proxy requires supplied container/disk formats. Go ImportImage(Ref) freshly fetches metadata, resolving an explicit Name once through existing exact-name collection policy; ImportKnownImage copies only native ID/container/disk strings and does not fetch.
- Legacy native/ref strict202 import profile only: The Go known-image/native shape does not reproduce arbitrary supplied Resource subclasses, dict/Munch construction, dirty state or cached descriptor defaults. Fresh GET performs native whole-image decoding and requires canonical lowercase nonempty format strings with valid UTF-8; the requested ID stays fixed even if response ID differs.
- Legacy native/ref strict202 import profile only: Python singular Store sends both a stores array and compatibility header. Go Store is header-only and plural Stores is body-only, rejects configured store-header conflicts with plural/all-store requests, and permits an explicit typed Store to replace the configured legacy value.
- Legacy native/ref strict202 import profile only: Go requires web-download URI and glance-download remote region+image ID, permits optional omitted interface for the server default, and rejects orphan/wrong-method remote values. Python can send web without URI or silently omit an incomplete remote tuple. Unknown nonempty methods remain server-owned.
- Legacy native/ref strict202 import profile only: Concrete SDK options and JSON extensions snapshot owned request data, enforce namespace/header/source boundaries, and expose original Provider transport configuration rather than arbitrary caller native builders or injected Python sessions. Cloud custom extension support remains server-owned.
- Legacy native/ref strict202 import profile only: Go returns an actual202 ImportResult acknowledgement with partial accepted-read evidence, rather than the raw Python requests.Response. It does not imply completed storage, return a synthetic active Image/Task or automatically stage, discover, wait, clean up or follow Location.
- Source mutates the same arbitrary Image/Store Resource, dict/Munch or subclass and dynamically reads body/descriptor/current-location state. Go snapshots full private current/original/dirty/presence and borrowed data, projects a captured location, resets plain ImportMethods before dispatch and preserves earlier passive fetch receipts. Imported response is not translated, overlaid or used to clean Body; independent actual ACK and immutable Record expose different ownership/lifecycle. Source dynamic class/constructor/object domains remain SDK-R1.
- Owned JSON Method distinguishes omitted default glance-direct from explicit null/empty/non-string/unknown values. URI uses Python truthiness and exact web-download equality; complete truthy remote region/image/interface values emit together independent of method, while incomplete tuples omit. Non-None root all_stores/must_succeed raw values preserve presence, including false and nonbool JSON for server validation. Store selectors consume only canonical Attributes id, use null for missing identity, preserve all plural JSON IDs/order/duplicates, and singular empty string enters both stores array and compatibility header. Concrete typed With helpers coexist with owned raw JSON; legacy stricter URI/remote/empty-method/singular-header-only behavior is unchanged.
- Source Image.import_image never calls _translate_response or raise_from_response and Proxy.request defaults raise_exc=False, so arbitrary adapters can return raw400..599 without an exception. The owned Go profile deliberately uses native400..599 errors and accepts actual200..399 opaque acknowledgements; malformed JSON/UTF-8 bytes stay passive. Accepted physical/source/context failures return actual partial ACK with inspectable causes and no fabricated completed Record or recovery. Singular non-string/null store ID fails the Go HTTP-header representation check before POST; no equivalent Source None-header/session exception is claimed.
- The SDK owns concrete JSON/header snapshots, fixed private image identity, source/provider/origin/microversion and sticky outer/context guards before one logical POST. Shared RequestHeaderPolicy protects the selected store compatibility header value or its absence at physical dispatch and native retry; callbacks cannot change store selection while ordinary native headers retain configured policy. Clean native JSON auth/retry keeps captured body/route with live original provider token; guard drift prevents resend. No initial lookup, method discovery, stage, task, final fetch, wait, checksum, cleanup or follow-Location is inserted. Full adapter cache/invalidation/metrics/global request configuration and arbitrary session/response policy remain SDK-C1/S1; custom RoundTripper internal behavior is outside the logical-request guarantee. Raw JSON precision, strict UTF-8/paired-surrogate validation and constructor/header namespaces are explicit Go representation policies, without arbitrary Python object or byte-for-byte serializer equivalence.
- Pinned Glance selects exact copy-image through project-admin copy_image, and other methods through modify_image admin-or-project-member/owner. This mandatory admin method shares the same public Import signature; ordinary glance-direct/web-download/glance-download remain core user priority. Image state/format/UUID/URI, deployed import methods/stores, quota, lock/proxy-worker and asynchronous backend task outcome stay server-owned. This finite owned go_mapping does not promote public create_image, upload_image or download_image or establish authenticated cloud/storage completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
response = conn.image.import_image(
    record,
    method="web-download",
    uri="https://images.example.org/image.qcow2",
    stores=["fast", "archive"],
    all_stores_must_succeed=False,
)
print(response.status_code)
```

```sh
go run . -cloud dev -image-id ID
go run . -cloud dev -image-id ID -method web-download -uri https://images.example.org/image.qcow2 -stores fast,archive -all-stores-must-succeed=false
go run . -cloud dev -image-id ID -service -method glance-download -remote-region RegionTwo -remote-image-id REMOTE_ID -remote-interface public
go run . -cloud dev -image-id ID -method copy-image -store archive
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
    "strings"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    method := flag.String("method", "glance-direct", "server import method")
    uri := flag.String("uri", "", "web-download source URI")
    region := flag.String("remote-region", "", "remote Glance region")
    remoteID := flag.String("remote-image-id", "", "remote Glance image ID")
    remoteInterface := flag.String("remote-interface", "", "remote Glance service interface")
    store := flag.String("store", "", "singular store; explicit empty string is retained")
    stores := flag.String("stores", "", "comma-separated plural stores, preserving order and duplicates")
    allStores := flag.Bool("all-stores", false, "explicit all_stores value")
    allMust := flag.Bool("all-stores-must-succeed", false, "explicit all_stores_must_succeed value")
    flag.Parse()
    present := make(map[string]bool)
    flag.Visit(func(value *flag.Flag) { present[value.Name] = true })
    if *imageID == "" { log.Fatal("-image-id is required") }
    options := []image.ImageRecordImportOption{
        image.WithImageRecordImportOpts(image.ImageRecordImportOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-import-example"},
        }),
        image.WithImageRecordImportMethod(*method),
        image.WithImageRecordImportURI(*uri),
        image.WithImageRecordImportRemoteRegion(*region),
        image.WithImageRecordImportRemoteImageID(*remoteID),
        image.WithImageRecordImportRemoteServiceInterface(*remoteInterface),
    }
    if present["store"] {
        options = append(options, image.WithImageRecordImportStore(image.ImageRecordImportStore{ID: *store}))
    }
    if present["stores"] {
        selected := make([]image.ImageRecordImportStore, 0)
        for _, id := range strings.Split(*stores, ",") { selected = append(selected, image.ImageRecordImportStore{ID: id}) }
        options = append(options, image.WithImageRecordImportStores(selected...))
    }
    if present["all-stores"] { options = append(options, image.WithImageRecordImportAllStores(*allStores)) }
    if present["all-stores-must-succeed"] { options = append(options, image.WithImageRecordImportAllStoresMustSucceed(*allMust)) }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *viaService, options); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID string, viaService bool, options []image.ImageRecordImportOption) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordImportRequest{Record: record}
    var result *image.ImageRecordImportResult
    if viaService {
        result, err = service.ImportImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ImportImageRecord(ctx, input, options...)
    }
    return printImport(result, err)
}

func printImport(result *image.ImageRecordImportResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        if ack := result.Acknowledgement; ack != nil {
            // Opaque response bytes may not be JSON. []byte renders as base64.
            value["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "body": []byte(ack.Body),
                "header": ack.Header, "status_code": ack.StatusCode,
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "envelope_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if operationErr != nil { value["error"] = operationErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

## References

- [Image Record Stage](image-record-stage.md)
- [Install](../docs/install.md)
- [`API.ImageImport.ImportImage/ImportKnownImage`](v2/imageimport/README.md)
- [CreateAndImport](create-import.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L436)
- [Image Records](image-records.md)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L317-L390)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L330-L343)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
