# Metadef Property Record Write

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

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
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One flat collection POST201 sends required literal name plus nonnil Type/Title and only supplied description/raw attributes. Type is nonempty validUTF8 without a local enum; empty Title is valid, Description permits multiline/empty without an added length bound. Attributes keys are validUTF8, values any validUTF8JSON including null, with exact name/type/title/description/self/schema/created_at/updated_at/namespace_name keys reserved; the readonly keyword is forwarded. Server Schema additionalProperties=false and WSME field/type narrowing remain server behavior; no discovery/GET/schema gate/rollback or round-trip promise.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`, `internal/cmd/sdkgen/glance_metadef_properties_test.go:TestGlanceMetadefPropertiesRegistryAndConcreteScope`.
- Owned CreateRecord always sends one flat collection POST with actual200..399 acceptance, including POST {} when no recognized Body attributes remain. Only supplied raw values among 19 declared Body fields are transposed to their wire names; name/type/title are not SDK-required and missing descriptor defaults are not sent. Supplied id/name, including null, arrays/objects, empty/slash/long names, are passive Body data and cannot append a child or retarget the fixed parent. No discovery/GET/schema gate/rollback is added.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`.
- Source construction eagerly reads descriptor values before HTTP without replacing stored raw inputs. Owned create validates the same shared property projection before sending: nullable untyped/int/bool/list/dict fields, explicit null retention, missing min_length/min_items0 and require_unique_itemsfalse, inherited id presence or name fallback, plus fixed namespace URI and captured location form a 21-field Resource. Raw request values such as numeric strings or boolean strings remain unchanged on the wire while the returned view is converted.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`, `image/v2/metadefproperties/record_get_test.go:TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull`.
- Concrete create opts/header(s)/attribute(s) helpers use full-config replacement, later-option selection and independent map/slice/raw JSON factory snapshots with copy-before/copy-after callback ownership. Plural attributes replace the selection and singular attributes replace a key. Unordered-map canonical names win over wire aliases including null, so superseded alias encoding failures are hidden; ordinary unknown fields including value/location are ignored before encoding. Reserved bindings/route/session/version controls, nil/nonsemantic options, known marshal errors and owned-header conflicts fail before HTTP.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Present recognized response fields overlay the private request seed; omitted values retain that seed, ordered parsed dictionary aliases use their last effective value, and unknown response fields remain Wire-only. Resource, actual Wire, physical Envelope and each Header own independent bytes with actual status/header metadata; fixed Namespace and location stay request provenance and Key is nil. Returned id/name/links do not select a follow-up route.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesActualStatusesOverlayAndOwnedChannels`, `connectiontest/connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Owned create requires whole-response validUTF8 but accepts empty/opaque/syntactically invalid JSON after actual200..399 as seeded/default success with Wire nil and an actual receipt. Valid parsed nonobjects, descriptor/read/Close/context/source failures retain actual Body/Header/StatusCode and all causes in ResponseError without replay. Native rejected-status/body/retry/reauth ownership stays native, response-less failures invent no receipt, and native OkCodes/request JSON mutations cannot widen the owned boundary.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`.
- Source/context/outer lifetime checks precede location and options; Connection location is snapshotted before callbacks, and sticky source/namespace/context guards surround each callback and physical/read/Close boundary. Both Connection Image and ImageV2 entry points supply the same owned current location and live original authentication, while direct construction defaults location to null. Existing required Type/Title raw Create and its strict201 policy remain separate unchanged contracts.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`, `connectiontest/connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.

Python/Go differences:

- Owned CreateRecord maps the public fresh Resource.new/create operation: supplied declared raw attributes only, no SDK-required name/type/title, passive supplied id, eager descriptor view and empty POST {}. The original raw Create keeps its required literal name/Type/Title and strict201 policy; server schema/WSME/DB acceptance remains server behavior.
- Go attributes are encoding/json snapshots; canonical map keys take precedence over wire aliases including null because Go maps lack Python insertion order. Python ordered kwargs use the later recognized alias. Arbitrary Python runtime objects, mutable attribute references and cache/dirty instance identity are not reproduced; unknown actual response fields are retained separately in Wire.
- Go rejects explicit bound/generic route/session/version controls under its fixed namespace/client policy. Some names collide in Python bindings, while base_path is a legitimate Source dynamic override and __conflicting_attrs can inject constructor attributes; these dynamic paths are deliberately unavailable in this bounded mapping.
- Go adds independent Resource/Wire/Envelope/Header/status evidence, whole UTF8/JSON shape validation, explicit header ownership, context/native errors and sticky target/source guards. Source default to_dict has20 Body/computed fields while Go additionally exposes fixed namespace_name and typed Namespace; captured explicit Go zone follows the existing documented location policy.
- Local reused transport fixtures and compile-only examples establish payload/projection/ownership contracts. Authenticated cloud creation, actual Python execution, server schema/protection/policy/DB conversions or full Resource/session parity are not proven.

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
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Canonical Property requires a whole validUTF8 nonnullJSONobject. Exact-case optional nullable name/type/title/description/self/schema/literal dates remain pointers; no ID/namespace/name/links seed or time parsing. Unknown keywords including numbers/default remain independent RawMessages in Body; canonical type failures are atomic and retain actual response evidence. Key is separate list-only wire-map-key provenance, nil on CRUD, and never rewrites passive Name or raw Body.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesCanonicalModelsAndRawOwnership`.
- Six concrete option families use FullOpts replacement and last-option-wins with deep pointer/header/RawMessage snapshots. Custom callbacks run once with fresh ownership before and after; nil/error callbacks and invalid/conflicting owned headers fail before HTTP. Plural any Attributes helpers JSON-marshal an owned factory snapshot, validate original keys before encoding, wholly replace attributes and retain marshal causes; singular Attribute adds/replaces one key. Raw FullOpts and encoding/json arbitrary any semantics are explicit boundaries. Lazy option slices and concurrent helper reuse are owned.
  Tests: `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsParallelReusableHelpers`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFiniteDictionaryAndLocalControls`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Strict original statuses are Create201, Get/List/Update200 and Delete/DeleteAll204. Accepted read/Close/wholeUTF8/JSON/canonical/context/source failures preserve actual Body/Header/status and original causes in ResponseError, close once and never replay. Actual204 opaque acknowledgement retains requested Namespace/childName, with nil Name for DeleteAll, even on handling failures. Native rejected-body ownership stays native.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest retains configured prebody retry/reauth/backoff and identical-target redirect policy. Method/origin/path/query/serializedbody and KeepResponseBody/JSONResponse/RawBody/JSONBody ownership are guarded; expanded native OkCodes cannot change original accepted statuses. Read/Close/ctx.Err/customCause and native reauth ErrOriginal/ErrReauth evidence remain available. No accepted-body replay.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesResponseOwnershipAndNativeHooks`.
- One full replacement PUT200 uses fixed current-child route; body name defaults current unless explicit rename. Type/Title are required again and only supplied Description/raw attributes are sent, so omitted optional keywords reset according to server behavior. Names, typed values, raw/any snapshots and exact reserved attributes are checked before HTTP. No GET/read-modify-write/dirty cache/PATCH/automatic retention/rollback; Schema/WSME/DB conversions are not a round-trip promise.
  Tests: `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreFixedScopeAndCRUD`, `image/v2/metadefproperties/core_test.go:TestMetadefPropertiesCoreCompletePreflight`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsValidationAndDefaults`, `image/v2/metadefproperties/options_test.go:TestMetadefPropertiesOptionsCallbackAndFactoryOwnership`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesFixedScopedRoutesAndPayloads`, `image/v2/metadefproperties/contracts_test.go:TestMetadefPropertiesOwnedPreparationAndSource`, `connectiontest/connection_image_metadef_properties_test.go:TestConnectionImageMetadefPropertiesShareClientAndFixedScope`.
- Owned UpdateRecord snapshots exactly one concrete ID or RawResource identity before callbacks. Explicitly present id including null takes precedence over name; only missing id permits name fallback. The selected child must be nonempty validUTF8 safe literal<=80runes even on no-op, and missing/conflicting/unsafe identities fail before callbacks. Other input Resource fields, invalid nonidentity descriptors and old receipt metadata are discarded, matching the public Source Resource._get_id reduction rather than reusing a mutable existing definition; attrs id/value cannot retarget either input form.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Every supplied recognized non-id Body attribute is fresh dirty input, including null, false, zero, empty values, descriptor-default values and name equal to selected id. One fixed-child query-free PUT transposes only these18 supplied fields to wire names and preserves their raw JSON values; seeded id, missing defaults, old Resource fields and synthesized name are not sent. Name is a passive rename body value while request id stays fixed. There is no read-modify-write, PATCH, implicit Type/Title requirement or promise that server PUT retains omitted fields.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesKeepAllRawDirtyFieldsSeparateFromView`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`.
- After full source/location/options/header/descriptor validation, absence of recognized dirty Body fields returns an independent 21-field local Resource without HTTP. Header-only, ordinary-unknown-only or attribute-selection-cleared updates do not force PUT. Local result contains only selected id, descriptor defaults/nulls, fixed namespace and captured current location; Wire/Envelope/Header are nil, StatusCode0 and Resource receipt metadata is absent, so it does not claim a fetched or changed server definition.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteEmptyConstructionAndIdentityOnlyUpdate`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `connectiontest/connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Concrete update opts/header(s)/attribute(s) preserve full-replacement/later-option selection and deep map/slice/raw JSON callback/factory ownership. Canonical attribute presence including null wins over a wire alias and hides superseded alias marshal failures; unknown fields are ignored before encoding. All input forms reserve id/value plus fixed route/session/version/binding controls; nil/nonsemantic options, known encoding/conversion errors and invalid/conflicting protected headers fail before HTTP, including no-op calls.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteFactoriesSelectionAndCapturedLocation`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteInputAttributeAndHeaderPreflight`.
- Actual200..399 response fields overlay only the private selected-id-plus-explicit-attributes seed, retaining omitted inputs and preserving parsed alias ordering. Empty/opaque/invalid JSON leaves the seed/default view with Wire nil but actual Envelope/Header/status; valid nonobject or descriptor/wholeUTF8 failure retains accepted evidence. Resource/Wire/page/headers are independently owned, Key stays nil, fixed Namespace/location are not overwritten and returned id/name are passive response data. Both Connection routes preserve the same current location and fresh identity-only/no-op state.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesActualStatusesOverlayAndOwnedChannels`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `connectiontest/connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.
- Source/context/outer validation occurs before location/options and selected Resource identity is immutable before callbacks. Sticky source/parent/context guards run around every callback and physical/read/Close boundary and also protect the local no-op result. Accepted read/Close/context/source errors retain all actual receipt/cancellation causes without replay; native retry/reauth, rejected-response evidence and request JSON/OkCodes ownership remain guarded. Old raw Update still requires Type/Title and strict200, independently of owned dirty/no-op behavior.
  Tests: `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWriteGuardsAlsoCoverLocalNoop`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesSuccessDecodeAndAcceptedFaultReceipts`, `image/v2/metadefproperties/record_write_test.go:TestMetadefPropertyRecordWritesNativeBodyOwnershipAndLegacyContracts`, `connectiontest/connection_image_metadef_property_write_records_test.go:TestConnectionImageMetadefPropertyWritesShareLocationAndFreshState`.

Python/Go differences:

- Pinned public update_metadef_property first reduces a Property Resource to Resource._get_id and constructs a fresh id-bound resource; its other Body and dirty state are not retained. Owned UpdateRecord likewise snapshots only identity and fresh explicit attrs. Direct mutable Resource.commit/cache reuse is a separate Source surface, while old raw Update retains its required Type/Title full-replacement strict200 behavior.
- Go validates a fixed safe literal identity before callbacks and no-op, and reserves attrs id for every concrete input. Normal Source string/Resource-reduced IDs collide with new(id=value,**attrs), but Source None/nonliteral/dynamic Resource paths can behave differently and its no-op precedes URL/required-ID preparation. Those paths are outside the fixed Go identity domain.
- Go map canonical names beat wire aliases including null, unlike Python ordered kwargs, and any values use encoding/json factory snapshots. Raw supplied values are transmitted without descriptor conversion; defaults affect the declared view and do not enter a request. Unknown inputs are ignored while actual unknown response fields remain Wire-only.
- No-op Go results add explicit no-receipt evidence: Wire/Envelope/Header nil and status0 with an independent21-field declared Resource. Actual HTTP with tolerated JSON failure instead keeps actual status/header/body. Source returns its new mutable Resource; Go neither fetches old fields nor infers existence or server update success from local no-op.
- Go adds fixed scope/header/context/native error and sticky target/source ownership, rejects dynamic route/session/version controls including legitimate Source base_path, and captures Connection location before options. Its extra namespace_name/Namespace and explicit zone policy remain documented differences from default Source to_dict.
- Local reused fixtures and compile-only consumers verify dirty payload/no-op/projection/receipt contracts. No authenticated cloud mutation, actual Python execution, server schema/protection/policy/DB conversion, omitted-field preservation or full Resource/session parity is proven.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
namespace = "OS::Compute::Libvirt"
id = "hw_cpu_policy"

created = conn.image.create_metadef_property(
    namespace, name=id, type="string", title="CPU Policy",
    description="CPU allocation policy", is_readonly=False, min_length=0,
)
updated = conn.image.update_metadef_property(
    id, namespace, name=id, type="string", title="CPU Policy",
    description="Updated CPU allocation policy", is_readonly=False, min_length=0,
)
local = conn.image.update_metadef_property(id, namespace)
print(created.to_dict(), updated.to_dict(), local.to_dict())
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
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    properties "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    namespace := flag.String("namespace", "OS::Compute::Libvirt", "literal namespace")
    id := flag.String("id", "hw_cpu_policy", "property name")
    action := flag.String("action", "snapshot", "snapshot, create, or update")
    title := flag.String("title", "CPU Policy", "property title")
    description := flag.String("description", "CPU allocation policy", "property description")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *namespace, *id, *action, *title, *description); err != nil {
        log.Fatal(err)
    }
}

func printFailure(err error) error {
    var accepted *resource.ResponseError
    var unexpected gophercloud.ErrUnexpectedResponseCode
    var receipt map[string]any
    switch {
    case errors.As(err, &accepted):
        receipt = map[string]any{
            "status_code": accepted.StatusCode, "header": accepted.Header,
            "body": string(accepted.Body),
        }
    case errors.As(err, &unexpected):
        receipt = map[string]any{
            "status_code": unexpected.Actual, "header": unexpected.ResponseHeader,
            "body": string(unexpected.Body),
        }
    default:
        return nil
    }
    data, printErr := json.MarshalIndent(receipt, "", "  ")
    if printErr != nil { return printErr }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, namespace, id, action, title, description string) error {
    if action != "snapshot" && action != "create" && action != "update" {
        return fmt.Errorf("unknown action %q", action)
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return err }
    attributes := map[string]any{
        "name": id, "type": "string", "title": title,
        "description": description, "is_readonly": false, "min_length": 0,
    }
    var record *properties.Record
    switch action {
    case "create":
        record, err = scope.CreateRecord(ctx,
            properties.WithRecordCreateAttributes(attributes),
            properties.WithRecordCreateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    case "update":
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            properties.WithRecordUpdateAttributes(attributes),
            properties.WithRecordUpdateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    default:
        record, err = scope.UpdateRecord(ctx, properties.RecordRequest{ID: id},
            properties.WithRecordUpdateHeader("X-Request-Source", "metadef-property-write-example"),
        )
    }
    if err != nil { return errors.Join(err, printFailure(err)) }
    data, err := json.MarshalIndent(map[string]any{
        "namespace": record.Namespace,
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope),
        "header": record.Header, "status_code": record.StatusCode,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Metadefproperties](v2/metadefproperties/README.md)
- [Metadef Property Records](metadef-property-records.md)
- [Metadef Property Record List](metadef-property-record-list.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1841)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L811)
- [constructor·attribute collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L538-L609)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231-L1336)
- [commit no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2017)
- [descriptor get/set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208-L291)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
