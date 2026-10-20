# Metadefproperties

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/create_metadef_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateType`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateTitle`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordCreateAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NewWithDependencies`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One flat collection POST201 sends required literal name plus nonnil Type/Title and only supplied description/raw attributes. Type is nonempty validUTF8 without a local enum; empty Title is valid, Description permits multiline/empty without an added length bound. Attributes keys are validUTF8, values any validUTF8JSON including null, with exact name/type/title/description/self/schema/created_at/updated_at/namespace_name keys reserved; the readonly keyword is forwarded. Server Schema additionalProperties=false and WSME field/type narrowing remain server behavior; no discovery/GET/schema gate/rollback or round-trip promise.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_properties_test.go:TestGlanceMetadefPropertiesRegistryAndConcreteScope`.
- Owned CreateRecord always sends one flat collection POST with actual200..399 acceptance, including POST {} when no recognized Body attributes remain. Only supplied raw values among 19 declared Body fields are transposed to their wire names; name/type/title are not SDK-required and missing descriptor defaults are not sent. Supplied id/name, including null, arrays/objects, empty/slash/long names, are passive Body data and cannot append a child or retarget the fixed parent. No discovery/GET/schema gate/rollback is added.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`.
- Source construction eagerly reads descriptor values before HTTP without replacing stored raw inputs. Owned create validates the same shared property projection before sending: nullable untyped/int/bool/list/dict fields, explicit null retention, missing min_length/min_items0 and require_unique_itemsfalse, inherited id presence or name fallback, plus fixed namespace URI and captured location form a 21-field Resource. Raw request values such as numeric strings or boolean strings remain unchanged on the wire while the returned view is converted.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- Concrete create opts/header(s)/attribute(s) helpers use full-config replacement, later-option selection and independent map/slice/raw JSON factory snapshots with copy-before/copy-after callback ownership. Plural attributes replace the selection and singular attributes replace a key. Unordered-map canonical names win over wire aliases including null, so superseded alias encoding failures are hidden; ordinary unknown fields including value/location are ignored before encoding. Reserved bindings/route/session/version controls, nil/nonsemantic options, known marshal errors and owned-header conflicts fail before HTTP.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Present recognized response fields overlay the private request seed; omitted values retain that seed, ordered parsed dictionary aliases use their last effective value, and unknown response fields remain Wire-only. Resource, actual Wire, physical Envelope and each Header own independent bytes with actual status/header metadata; fixed Namespace and location stay request provenance and Key is nil. Returned id/name/links do not select a follow-up route.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesActualStatusesOverlayAndOwnedChannels`, `connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Owned create requires whole-response validUTF8 but accepts empty/opaque/syntactically invalid JSON after actual200..399 as seeded/default success with Wire nil and an actual receipt. Valid parsed nonobjects, descriptor/read/Close/context/source failures retain actual Body/Header/StatusCode and all causes in ResponseError without replay. Native rejected-status/body/retry/reauth ownership stays native, response-less failures invent no receipt, and native OkCodes/request JSON mutations cannot widen the owned boundary.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`.
- Source/context/outer lifetime checks precede location and options; Connection location is snapshotted before callbacks, and sticky source/namespace/context guards surround each callback and physical/read/Close boundary. Both Connection Image and ImageV2 entry points supply the same owned current location and live original authentication, while direct construction defaults location to null. Existing required Type/Title raw Create and its strict201 policy remain separate unchanged contracts.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`, `connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.

Python/Go differences:

- Owned CreateRecord maps the public fresh Resource.new/create operation: supplied declared raw attributes only, no SDK-required name/type/title, passive supplied id, eager descriptor view and empty POST {}. The original raw Create keeps its required literal name/Type/Title and strict201 policy; server schema/WSME/DB acceptance remains server behavior.
- Go attributes are encoding/json snapshots; canonical map keys take precedence over wire aliases including null because Go maps lack Python insertion order. Python ordered kwargs use the later recognized alias. Arbitrary Python runtime objects, mutable attribute references and cache/dirty instance identity are not reproduced; unknown actual response fields are retained separately in Wire.
- Go rejects explicit bound/generic route/session/version controls under its fixed namespace/client policy. Some names collide in Python bindings, while base_path is a legitimate Source dynamic override and __conflicting_attrs can inject constructor attributes; these dynamic paths are deliberately unavailable in this bounded mapping.
- Go adds independent Resource/Wire/Envelope/Header/status evidence, whole UTF8/JSON shape validation, explicit header ownership, context/native errors and sticky target/source guards. Source default to_dict has20 Body/computed fields while Go additionally exposes fixed namespace_name and typed Namespace; captured explicit Go zone follows the existing documented location policy.
- Local reused transport fixtures and compile-only examples establish payload/projection/ownership contracts. Authenticated cloud creation, actual Python execution, server schema/protection/policy/DB conversions or full Resource/session parity are not proven.

### `python:image/v2/get_metadef_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithGetResourceType`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordGetAttributes`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One fixed child GET200; ResourceType nil omits query, explicit empty sends resource_type= and nonempty is a literal validUTF8 control-free query. Actual server association prefix stripping is not reproduced locally. Passive returned Name/Key/self/schema cannot retarget scope; no cache reuse, selector, fallback, discovery or follow-up request.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`.
- Named strict getter boundary audit distinguishes Python **query as _get controls or Resource seed attributes from an HTTP query. Existing strict200 fresh DTO retains actual descriptor keywords/default omission/aliases raw, rejects304 without cache reuse and independently decodes later success. WithGetResourceType remains a deliberate Go server-query extension; this raw getter itself does not implement Resource seeding/default/coercion/fetch translation. The additive owned GetRecord contracts below close that formerly unresolved named getter gap.
  Tests: `image/v2/metadefproperties/get_mapping_test.go:TestMetadefPropertyGetMappingDescriptorKeywordsRemainRaw`, `image/v2/metadefproperties/get_mapping_test.go:TestMetadefPropertyGetMappingDoesNotReuseCachedOrSeededResults`.
- GetRecord snapshots concrete ID-or-Resource input before dependency/options callbacks, retains recognized initial attributes, applies option attrs before selecting final ID or alternate name and never mutates the input. String+id attribute is a preflight collision; Resource id attr may retarget before HTTP. Required fixed literal parent/child and zero query/body GET reject unsafe or null/nonstring identity; response fields cannot retarget. Known seed descriptor errors and reserved route/cache/session/class/version controls fail preflight, while unknown ordinary attrs and superseded alias encoding failures are ignored.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSeedAttributesAndResponseOverlay`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordInputAndAttributesPreflight`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSupersededAliasDoesNotExposeEncodingFailure`.
- Owned Resource projects eighteen declared Body descriptors plus inherited id, current Connection location and fixed Go namespace_name provenance. Missing min_length/min_items default0 and require_unique_items false; explicit null is retained. Raw untyped JSON, list singleton wrapping and dict object-or-empty fallback, shared integer descriptor and Python truthiness are applied after seed/response overlay. Omitted response fields preserve seeds. Maps prefer canonical presence including null; parsed response aliases follow dictionary member order with duplicate first slot/last value.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordFixedRouteDefaultsAndOwnedReceipts`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSeedAttributesAndResponseOverlay`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- Actual200..399 successful fetch always reads/closes once and verifies whole UTF8. Parsed nonnull object overlays declared Body only and retains complete independent Wire; empty/opaque/syntax-invalid JSON returns seeded/default Resource with nil Wire and actual Envelope/Header/status. Valid parsed null/array/scalar and descriptor/read/Close/context/source failures are atomic accepted ResponseErrors preserving receipt and causes. Native400+ errors retain native ownership; no accepted-body replay.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordStatusParsingAndMissingPolicy`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordAcceptedBodyErrorsPreserveAllCauses`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- RecordGet FullOpts replace configuration, headers and recognized semantic attribute capture with factory/callback ownership before and after. Bulk attrs replace and singular attrs overlay; callbacks run once, latest source ordinary headers are captured and live Provider auth is preserved. Connection location is captured before options and fixed namespace URI means seed/response computed location never replaces it. Resource/Wire/Envelope/headers own independent bytes and only actual response metadata is attached.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordOptionFactoriesAndRetainedStorage`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordPreflightOrderAndCallbackCauses`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordFixedRouteDefaultsAndOwnedReceipts`.
- Getter reuses REST/fixedrequest native prebody retry/reauth/backoff policies with fixed method/target/body/framing/representation/status ownership. Source/namespace/outer/context guards cover callbacks, physical attempts, read/Close and final projection; observed failure stays sticky after target restoration. Retry mutation cannot expand actual accepted range or replay an accepted-body failure.
  Tests: `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordNativeRetryPoliciesAndOwnership`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordSourceAndOuterGuardAtPhysicalBoundaries`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordPreflightOrderAndCallbackCauses`.
- Both high-level Connection.Image API and versioned Connection.ImageV2 inject CurrentLocation and share the original service client/provider. Owned configured facts survive caller mutation and option-updated authentication is live; view preserves seeds/default conversion with actual203 receipt and passive response fields. Original typed Get retains strict200 raw Body and additive resource_type query independently.
  Tests: `connection_image_metadef_property_record_test.go:TestConnectionImageMetadefPropertyRecordUsesOwnedLocationAndLiveAuth`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordLegacyGetRemainsRawAndStrict`.

Python/Go differences:

- Go mapping uses an immutable owned concrete RecordRequest and fresh GET; Python mutates existing Resource/cache/dirty/session state and accepts broader dict/Munch/None parent inputs. Go requires fixed literal namespace and selected string child, explicitly rejects route/cache/class/session/version overrides, and does not add finder/discovery or conditional cached returns.
- Resource view has21 fields: pinned Python default to_dict20 plus fixed namespace_name provenance. Unknown/self/schema/date fields remain only in actual Wire/Envelope; Source Body descriptor names, null/defaults, list/dict/bool/int projection and response overlay are owned. Response identity is passive, and canonical map aliases have deterministic precedence while Python kwargs use dictionary order.
- Shared JSON integer projection preserves arbitrary integer JSON and bool, truncates IEEE754 float values, uses Unicode16 digit properties and omits CPython configurable digit-string limits. Bool numeric truthiness compares exact decimal zero, so underflow/JSON backend and nonfinite runtime behavior may differ. Arbitrary Go attributes use encoding/json snapshots, with deferred failures only for finally selected declared attrs.
- Always-on fixed query-free owned fetch accepts actual200..399 and tolerates successful syntax-invalid JSON, but requires wholeUTF8 and parsed nonnull objects. Original strict200 raw typed Get remains a separate API, with WithGetResourceType as a deliberate Go server-query extension. Python public resource_type/value/namespace_name kwargs cause duplicate-binding TypeError beforeHTTP.
- Connection current location including explicitly configured Zone is preserved; pinned Python current_location normally has zone=None. Direct New has null location, while NewWithDependencies supplies a concrete callback. Fixed namespace URI causes seed/response location to remain passive in Wire.
- Local transport fixtures and compile/install builds establish the named getter Go contract; server schema/policy/DB defaults, authenticated side effects and live Python execution are not proven. This mapping does not claim whole Python Resource/session lifecycle parity or change the remaining metadata CRUD/list inventory.

### `python:image/v2/metadef_properties`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NewWithDependencies`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- List is lazy and finite: one query/body-free collection GET200 with required nonnull properties dictionary; All collects a fresh request and returns nil on failure. Whole syntax and dictionary keys are checked before consumption, wire encounter order preserved, duplicate keys keep first slot and last value as Python JSONdict. Only consumed nonnull object values are decoded; MaxItems0 unlimited, negative invalid and cap/break skip unused malformed rows. Each row Key preserves wire identity separately from passive Name; no sorting, paging/query/hint/next/fallback, key autorouting or Python client Body filtering.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreOrderedDictionaryConsumption`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Owned ListRecords is lazy and each iteration captures a fresh source/location/options snapshot for one query/body-free fixed-parent GET accepting actual200..399. Whole-page validUTF8 and a nonnull root JSON object with a mandatory nonnull properties dictionary are required; empty dictionary succeeds. Dictionary encounter order is retained, duplicate keys keep their first position and last value, and continuation body fields or HTTP Link never create another request.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsOrderedDictionaryProjectionAndReceipts`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsStatusAndMandatoryWholePageValidation`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsLazySnapshotsAndReiteration`.
- Dictionary key seeds declared name before present response attributes overlay it; inherited id preserves explicit presence including null and otherwise uses the projected name. The shared property projector evaluates all declared descriptors before filtering with aliases, nullable defaults and Source conversions; its 21-field Resource includes fixed namespace URI and captured current location, while Wire contains only the actual row and Envelope is the whole physical page. Key is independent provenance; returned names/ids and empty/slash/long keys remain passive. Both Connection image entry points supply the same owned location; direct New defaults it to null and response location remains in Wire.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsOrderedDictionaryProjectionAndReceipts`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsNamesAndKeysRemainPassive`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`, `connection_image_metadef_property_list_records_test.go:TestConnectionImageMetadefPropertyListsShareOwnedLocationAndFiniteRoutes`.
- Exactly the 19 canonical Body descriptor names are local filters evaluated after Source-style descriptor projection; caller filter values remain owned JSON snapshots without descriptor coercion. Wire aliases, unknown attributes and limit/marker are ignored before encoding, including their otherwise invalid values. Reserved route/session/version/namespace/control attributes are rejected before HTTP; singular filters replace a key and plural filters replace the prior selection. Nested dictionary subset filtering uses shared JSON semantics with bool/number distinction and exact decimal comparison; max_items selects the property descriptor and never becomes a paging cap.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsCanonicalFiltersUseProjectedValues`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsCompletePreflightAndSourceLifetime`.
- Typed MaxItems is a separate Go raw-entry consumption cap before later row decoding, projection and filtering: zero is unlimited and negative is preflight-invalid. Whole-page UTF8/JSON syntax is validated first, but early break or cap skips unused row semantics. AllRecords returns a nonnil empty successful slice and retains earlier matching records alongside a later accepted-response/source/descriptor error; projection of a consumed malformed descriptor fails even when another filter would exclude it.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsCapsBreakAndPartialResults`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsStatusAndMandatoryWholePageValidation`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsGuardBeforeNextDecodeAndEmptyCompletion`.
- Owned list options execute once per iteration with immutable option-slice/filter/header factory snapshots and copy-before/copy-after callback ownership; full opts replace and later options win. Source/context/location validation precedes callbacks, reserved/nil/error/header conflicts fail before HTTP, and retained callback mutations do not change an in-flight request. Sticky parent/source guards run around callbacks and physical/read/Close/next-row/empty-completion boundaries; JSON Accept is explicitly owned across caller source headers and native retry callbacks.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsLazySnapshotsAndReiteration`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsCompletePreflightAndSourceLifetime`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsGuardBeforeNextDecodeAndEmptyCompletion`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsOwnExplicitJSONAcceptAcrossSourceAndRetry`.
- Resource, Wire, row Header, Key and physical-page Envelope have independent storage and actual status/header metadata. Accepted read/Close/UTF8/JSON/descriptor/context/source failures retain the actual response and all causes without replay; rejected statuses and response-less failures retain native error/retry/reauth ownership. Fixed request/response guards prevent native OkCodes or callback mutation from relaxing the owned boundary. Existing strict200 raw List/All keep their separate passive DTO/no-filter/no-name-seed and nil-on-error behavior.
  Tests: `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsOrderedDictionaryProjectionAndReceipts`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsAcceptedFaultsAndNativeOwnership`, `image/v2/metadefproperties/list_records_test.go:TestMetadefPropertyListRecordsLegacyFiniteDTOIsSeparate`.

Python/Go differences:

- Pinned Python specialized Property.list seeds name from each dictionary key, overlays value/URI fields, constructs eager descriptor-backed resources and applies canonical Body filters without pagination. Owned ListRecords maps this finite operation; original strict200 raw List/All remain separate and do not acquire that projection or filtering.
- Go adds Key provenance, the fixed namespace_name Resource field, an explicit raw-entry MaxItems cap and partial AllRecords results. Source max_items is a property descriptor filter, not a generic consumption cap. Wire aliases/unknown/limit/marker remain ignored; Go rejects reserved generic list/session/version/route controls as a fixed-parent/client policy rather than exposing Python dynamic Resource adapters.
- Shared Go JSON filtering distinguishes booleans from numbers, compares decimal numbers exactly and treats dictionary filters on nonobject actual values as nonmatches instead of Python object-method failures. Caller filter values are JSON snapshots and are not descriptor-coerced. Unknown response fields remain available in Wire rather than joining the declared Resource view.
- Source default to_dict has 20 Body/computed fields; Go additionally exposes fixed namespace_name and typed Namespace. Go captures Connection current location before options once per iteration; the fixed-URI Source collector rule keeps response location wire-only. Mutable Resource dirty/cache state, arbitrary Python objects and adapter/session controls remain outside this bounded mapping.
- Local reused transport fixtures and compile-only consumers verify wire/projection/ownership contracts. Authenticated cloud behavior, server schema/policy/protection/DB defaults, actual Python execution and full Resource/session parity are not proven.

### `python:image/v2/update_metadef_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateName`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateType`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateTitle`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordUpdateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithRecordUpdateAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NewWithDependencies`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One full replacement PUT200 uses fixed current-child route; body name defaults current unless explicit rename. Type/Title are required again and only supplied Description/raw attributes are sent, so omitted optional keywords reset according to server behavior. Names, typed values, raw/any snapshots and exact reserved attributes are checked before HTTP. No GET/read-modify-write/dirty cache/PATCH/automatic retention/rollback; Schema/WSME/DB conversions are not a round-trip promise.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Owned UpdateRecord snapshots exactly one concrete ID or RawResource identity before callbacks. Explicitly present id including null takes precedence over name; only missing id permits name fallback. The selected child must be nonempty validUTF8 safe literal<=80runes even on no-op, and missing/conflicting/unsafe identities fail before callbacks. Other input Resource fields, invalid nonidentity descriptors and old receipt metadata are discarded, matching the public Source Resource._get_id reduction rather than reusing a mutable existing definition; attrs id/value cannot retarget either input form.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Every supplied recognized non-id Body attribute is fresh dirty input, including null, false, zero, empty values, descriptor-default values and name equal to selected id. One fixed-child query-free PUT transposes only these18 supplied fields to wire names and preserves their raw JSON values; seeded id, missing defaults, old Resource fields and synthesized name are not sent. Name is a passive rename body value while request id stays fixed. There is no read-modify-write, PATCH, implicit Type/Title requirement or promise that server PUT retains omitted fields.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`.
- After full source/location/options/header/descriptor validation, absence of recognized dirty Body fields returns an independent 21-field local Resource without HTTP. Header-only, ordinary-unknown-only or attribute-selection-cleared updates do not force PUT. Local result contains only selected id, descriptor defaults/nulls, fixed namespace and captured current location; Wire/Envelope/Header are nil, StatusCode0 and Resource receipt metadata is absent, so it does not claim a fetched or changed server definition.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Concrete update opts/header(s)/attribute(s) preserve full-replacement/later-option selection and deep map/slice/raw JSON callback/factory ownership. Canonical attribute presence including null wins over a wire alias and hides superseded alias marshal failures; unknown fields are ignored before encoding. All input forms reserve id/value plus fixed route/session/version/binding controls; nil/nonsemantic options, known encoding/conversion errors and invalid/conflicting protected headers fail before HTTP, including no-op calls.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Actual200..399 response fields overlay only the private selected-id-plus-explicit-attributes seed, retaining omitted inputs and preserving parsed alias ordering. Empty/opaque/invalid JSON leaves the seed/default view with Wire nil but actual Envelope/Header/status; valid nonobject or descriptor/wholeUTF8 failure retains accepted evidence. Resource/Wire/page/headers are independently owned, Key stays nil, fixed Namespace/location are not overwritten and returned id/name are passive response data. Both Connection routes preserve the same current location and fresh identity-only/no-op state.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesActualStatusesOverlayAndOwnedChannels`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Source/context/outer validation occurs before location/options and selected Resource identity is immutable before callbacks. Sticky source/parent/context guards run around every callback and physical/read/Close boundary and also protect the local no-op result. Accepted read/Close/context/source errors retain all actual receipt/cancellation causes without replay; native retry/reauth, rejected-response evidence and request JSON/OkCodes ownership remain guarded. Old raw Update still requires Type/Title and strict200, independently of owned dirty/no-op behavior.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`, `connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.

Python/Go differences:

- Pinned public update_metadef_property first reduces a Property Resource to Resource._get_id and constructs a fresh id-bound resource; its other Body and dirty state are not retained. Owned UpdateRecord likewise snapshots only identity and fresh explicit attrs. Direct mutable Resource.commit/cache reuse is a separate Source surface, while old raw Update retains its required Type/Title full-replacement strict200 behavior.
- Go validates a fixed safe literal identity before callbacks and no-op, and reserves attrs id for every concrete input. Normal Source string/Resource-reduced IDs collide with new(id=value,**attrs), but Source None/nonliteral/dynamic Resource paths can behave differently and its no-op precedes URL/required-ID preparation. Those paths are outside the fixed Go identity domain.
- Go map canonical names beat wire aliases including null, unlike Python ordered kwargs, and any values use encoding/json factory snapshots. Raw supplied values are transmitted without descriptor conversion; defaults affect the declared view and do not enter a request. Unknown inputs are ignored while actual unknown response fields remain Wire-only.
- No-op Go results add explicit no-receipt evidence: Wire/Envelope/Header nil and status0 with an independent21-field declared Resource. Actual HTTP with tolerated JSON failure instead keeps actual status/header/body. Source returns its new mutable Resource; Go neither fetches old fields nor infers existence or server update success from local no-op.
- Go adds fixed scope/header/context/native error and sticky target/source ownership, rejects dynamic route/session/version controls including legitimate Source base_path, and captures Connection location before options. Its extra namespace_name/Namespace and explicit zone policy remain documented differences from default Source to_dict.
- Local reused fixtures and compile-only consumers verify dirty payload/no-op/projection/receipt contracts. No authenticated cloud mutation, actual Python execution, server schema/protection/policy/DB conversion, omitted-field preservation or full Resource/session parity is proven.

### `python:image/v2/delete_metadef_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.DeleteRecord`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Individual DELETE204 defaults IgnoreMissing nil=>true, including zero/full options; explicit false retains native rejected404 policy/retry. Only the physically owned clean404 with successful read/Close/context/source handling becomes nil,nil and bypasses native404 RetryFunc. Forbidden resources can be reported404; quiet missing is not absence proof. Nested/native/Close/context/source/name-preflight failures are never ignored; no parentGET/find/unprotect/wait/cleanup.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`.
- Owned DeleteRecord selects exactly one immutable concrete ID or RawResource identity before any option callback. Resource is cloned; explicitly present id including null takes priority over name, while missing id falls back to name. Missing/conflicting/null/nonstring/unsafe/overlong identity fails before callbacks or HTTP; unrelated definition fields and malformed unused raw values are ignored. Selected child and fixed namespace are escaped once and cannot be retargeted by retained input mutations or response content.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordResourceIdentityIsSnapshotAndOnlySelection`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordCompleteInputPreflight`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordOptionsOwnPointersHeadersAndIdentity`.
- Owned individual deletion sends a body/query-free fixed-child DELETE and accepts actual200..399 as opaque bytes without UTF8/JSON/descriptor decoding. Acknowledgement preserves selected Namespace/Name, actual Body/Header/StatusCode and independent storage; accepted read/Close/context/source handling failures return the captured acknowledgement with ResponseError/all causes and do not replay the accepted response. Rejected or response-less failures have no invented acknowledgement.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsFixedRoutesAndOpaqueStatuses`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`.
- IgnoreMissing nil or true privately accepts only an actually received physical404 and bypasses native404 RetryFunc, returning the actual404 acknowledgement even when handling fails. This is evidence of a handled missing response, not proof of successful deletion or resource absence. Explicit false preserves native rejected404 status/body/header/retry policy; a transport error claiming404 is never converted to handled physical missing. No parent fetch/find/unprotect/wait/cleanup is added.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsMissingReceiptsAndNativeErrors`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`.
- Existing DeleteOption helpers retain full-replacement/last-wins and deep header/pointer snapshots; callbacks run once with sticky source/parent/context checks before and after each callback and at physical/read/Close boundaries. Native prebody retry/reauth remains owned by Gophercloud while fixed target/response ownership and actual acceptance cannot be widened by native callbacks. Old raw Delete stays strict204 and keeps its clean handled404 nil,nil result.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsSourceFirstAndPerCallbackGuard`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordOptionsOwnPointersHeadersAndIdentity`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsNativeStatusExpansionAndLegacySeparation`.

Python/Go differences:

- Pinned public delete_metadef_property selects generic Resource/string IDs and returns None. Go uses a fixed namespace scope plus immutable concrete ID/RawResource identity and returns actual opaque acknowledgement evidence. Unrelated property descriptors do not participate in deletion; dirty/cache/session/version adapter state and dynamic parent lookup are not exposed.
- Owned DeleteRecord accepts actual200..399 and defaults to handled physical404, matching the bounded body-free deletion/missing policy. Its handled404 includes an actual404 receipt and can return acknowledgement plus handling error; this is evidence rather than successful deletion or absence proof. Existing raw Delete remains strict204 with clean handled404 nil,nil.
- Go imposes literal route validation, explicit header ownership, context/native error typing and sticky source/parent/request/response guards. Native rejected-response retry and reauth policies remain available, and accepted body handling is never replayed.
- Local transport fixtures verify fixed routes, opaque receipts, missing/native faults and immutable ownership. No authenticated cloud deletion, actual Python execution, server protection/policy/schema validation, wait/cleanup side effects or full Resource/session parity is proven.

### `python:image/v2/delete_all_metadef_properties`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.DeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.WithDeleteAllHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties.NamespaceScope.DeleteAllRecords`

- Zero-HTTP InNamespace validates required literal namespace and captures API/client/original provider/type/endpoint/base/microversion for scope lifetime. Parent and child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected and colon/Unicode/space escaped once. No Ref/find/parentGET or response-driven routing. Each call copies latest ordinary source headers, uses live original authentication and rechecks scope/source/context before callbacks, after accepted body and consumed rows.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Collection DELETE204 is strict for all404 and has no IgnoreMissing option. Opaque acknowledgement fixes Namespace and nil Name; no child enumeration, count, unprotect, transaction, wait, cleanup, automatic retry after accepted body or swallowed server policy failure.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`.
- Owned DeleteAllRecords sends one query/body-free DELETE to the fixed namespace property collection, accepts actual200..399 and treats every body as opaque bytes without UTF8/JSON/descriptor decoding. Acknowledgement owns actual Body/Header/StatusCode, fixed Namespace and nil Name; no child enumeration, deletion count, parent lookup, transaction, unprotect, wait or cleanup is inferred.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsFixedRoutesAndOpaqueStatuses`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`.
- Collection deletion has no IgnoreMissing policy: every physical404 or response-less transport404 stays a native failure with status/body/header/retry causes and no invented acknowledgement. Native callbacks cannot expand the actual owned acceptance set. Accepted read/Close/context/source handling failures return the actual opaque acknowledgement and response evidence with all causes, close once and are never replayed.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsMissingReceiptsAndNativeErrors`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsNativeStatusExpansionAndLegacySeparation`.
- Existing DeleteAllOption helpers preserve full-replacement/last-wins and deep header snapshots, callback-once copy-before/copy-after ownership and source-first preflight. Sticky fixed namespace/source/context checks surround every callback and physical/read/Close boundary; original native prebody retry/reauth and request/response ownership guards remain active. Existing raw DeleteAll remains a separate strict204 operation.
  Tests: `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsSourceFirstAndPerCallbackGuard`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordOptionsOwnPointersHeadersAndIdentity`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets`, `image/v2/metadefproperties/delete_records_test.go:TestMetadefPropertyDeleteRecordsNativeStatusExpansionAndLegacySeparation`.

Python/Go differences:

- Pinned delete_all_metadef_properties obtains a generic namespace Resource and delegates to its one body-free collection deletion, while the public proxy returns None. Go keeps the existing fixed scope/client and returns actual opaque acknowledgement evidence with nil child Name; dynamic Resource/session/cache/version/parent lookup state is not exposed.
- Owned DeleteAllRecords accepts actual200..399 and preserves native404 failure without an IgnoreMissing option. Existing raw DeleteAll remains strict204. Neither bounded API enumerates children or establishes a count, transaction, protected-resource cleanup or asynchronous completion.
- Go retains explicit header snapshots, context/native errors and sticky source/parent/request/response guards. Captured acknowledgements accompany accepted handling errors; native rejected-response policy remains native and accepted bytes are never replayed.
- Local transport fixtures establish finite fixed-route/receipt/ownership behavior. Authenticated cloud side effects, actual Python execution, server protection/policy/DB behavior and full Resource/session parity are not proven.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
)

type PropertyEvidence struct {
    Created *metadefproperties.Property
    Read *metadefproperties.Property
    Rows []*metadefproperties.Property
    All []*metadefproperties.Property
    Updated *metadefproperties.Property
    Deleted *metadefproperties.Acknowledgement
    Cleared *metadefproperties.Acknowledgement
}

func manageProperties(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out PropertyEvidence, err error) {
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    propertyType, title := "string", "Hypervisor Type"
    description := "First line\nSecond line"
    attributes := map[string]any{"enum": []string{"kvm", "qemu"}, "default": "kvm"}
    out.Created, err = scope.Create(ctx, name,
        metadefproperties.WithCreateOpts(metadefproperties.CreateOpts{
            Type: &propertyType, Title: &title,
            Attributes: map[string]json.RawMessage{"enum": json.RawMessage(`["kvm","qemu"]`)},
        }),
        metadefproperties.WithCreateHeader("X-Request-Source", "example"),
        metadefproperties.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefproperties.WithCreateType(propertyType),
        metadefproperties.WithCreateTitle(title),
        metadefproperties.WithCreateDescription(description),
        metadefproperties.WithCreateAttributes(attributes),
        metadefproperties.WithCreateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefproperties.WithGetOpts(metadefproperties.GetOpts{}),
        metadefproperties.WithGetHeader("X-Request-Source", "example"),
        metadefproperties.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        metadefproperties.WithGetResourceType(""))
    if err != nil { return out, err }

    listOptions := []metadefproperties.ListOption{
        metadefproperties.WithListOpts(metadefproperties.ListOpts{}),
        metadefproperties.WithListHeader("X-Request-Source", "example"),
        metadefproperties.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefproperties.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefproperties.WithUpdateOpts(metadefproperties.UpdateOpts{}),
        metadefproperties.WithUpdateHeader("X-Request-Source", "example"),
        metadefproperties.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefproperties.WithUpdateName(replacementName),
        metadefproperties.WithUpdateType(propertyType),
        metadefproperties.WithUpdateTitle(title),
        metadefproperties.WithUpdateDescription(description),
        metadefproperties.WithUpdateAttributes(attributes),
        metadefproperties.WithUpdateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefproperties.WithDeleteOpts(metadefproperties.DeleteOpts{}),
        metadefproperties.WithDeleteHeader("X-Request-Source", "example"),
        metadefproperties.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefproperties.WithDeleteIgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefproperties.WithDeleteAllOpts(metadefproperties.DeleteAllOpts{}),
        metadefproperties.WithDeleteAllHeader("X-Request-Source", "example"),
        metadefproperties.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

```python
def manage_properties(conn, namespace, name, replacement_name, clear_namespace=False):
    created = conn.image.create_metadef_property(
        namespace, name=name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    fetched = conn.image.get_metadef_property(name, namespace)
    rows = list(conn.image.metadef_properties(namespace))
    updated = conn.image.update_metadef_property(
        name, namespace, name=replacement_name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    conn.image.delete_metadef_property(replacement_name, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
    return created, fetched, rows, updated
```

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
)

func deleteOwnedProperties(ctx context.Context, scope *metadefproperties.NamespaceScope, id string, clearNamespace bool) (member, collection *metadefproperties.Acknowledgement, err error) {
    member, err = scope.DeleteRecord(ctx, metadefproperties.RecordRequest{ID: id},
        metadefproperties.WithDeleteIgnoreMissing(false),
        metadefproperties.WithDeleteHeader("X-Request-Source", "owned-delete-example"),
    )
    if err != nil || !clearNamespace { return member, nil, err }
    collection, err = scope.DeleteAllRecords(ctx,
        metadefproperties.WithDeleteAllHeader("X-Request-Source", "owned-delete-example"),
    )
    return member, collection, err
}
```

```python
def delete_owned_properties(conn, namespace, id, clear_namespace=False):
    conn.image.delete_metadef_property(id, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
```

## References

- [Metadef Property Records](../../metadef-property-records.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1947)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L254-L389)
- [Schema.raw](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L86-L103)
- [property schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L692-L806)
- [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L23-L62)
- [controller.update](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L185-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [server.show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L98-L146)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#show-property-definition)
- [MetadefProperty.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L64-L96)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-properties)
- [core tests](core_test.go)
- [option tests](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection contract](../../../connection_image_metadef_properties_test.go)
- [generator contract](../../../internal/cmd/sdkgen/glance_metadef_properties_test.go)
- [Metadef Property Record List](../../metadef-property-record-list.md)
- [Metadef Property Record Write](../../metadef-property-record-write.md)
