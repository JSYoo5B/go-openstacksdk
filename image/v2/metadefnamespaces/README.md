# Metadefnamespaces

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/create_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateDisplayName`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateOwner`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateProtected`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateProperties`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateObjects`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateTags`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithCreateResourceTypeAssociations`

- Literal root namespace retains safe nonempty UTF8<=80rune identity and fixed source capture before callbacks; no Ref/Find/Name resolver or input/response-driven retargeting. One existing collection Create request uses original provider and reverse prefix with live original auth and captures ordinary headers/microversion, then rechecks source/context after preparation and accepted body.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreCompletePreflight`, `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreSourceAndUpdatePreservation`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedOptionsAndCapturedSource`, `connection_image_metadef_namespace_create_test.go:TestConnectionImageMetadefNamespaceNestedCreateSharesClientAndOwnsInput`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreAllDefinitionsOnePOST`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedAcceptedResponseEvidence`.
- Concrete FullOpts replaces complete scalar and nested configuration; last helper wins. Recursive maps/slices/raw attributes/optional pointers and callback option/config snapshots are independently owned. Callbacks execute once; helper reuse and concurrent Create prepare independent bodies. Original raw keys are checked before JSON marshal, callback causes retained.
  Tests: `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsDeepFactorySnapshots`, `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsReplacementAndPresence`, `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsCallbackAndRawValidation`, `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedOptionsAndCapturedSource`, `connection_image_metadef_namespace_create_test.go:TestConnectionImageMetadefNamespaceNestedCreateSharesClientAndOwnsInput`.
- Original Create201 is the only accepted decode status. Accepted read/Close/UTF8/JSON/model/context/source failure returns no typed Namespace and preserves actual whole bytes/header/status and original read/Close/ctx.Err/customCause in ResponseError, close once, without accepted-body replay or input-seeded nested models. Native rejected response ownership remains unchanged.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreAcceptedResponseOwnership`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedAcceptedResponseEvidence`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreNativeErrorsAndRetryBody`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedNativeRetriesAndServerFailures`.
- One flat collection POST201 combines required literal namespace, existing optional scalars and each supplied nested container, with no hidden GET/child writes/wait/cleanup. Nil properties/objects/tags/associations and nested object properties/Required omit; explicit nonnil empty encodes object/array. Property Type and Title value strings always encode, including empty Title; optional nil strings omit and explicit empty strings remain. Server defaults for omitted scalar settings remain unchanged.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreAllDefinitionsOnePOST`, `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreNilEmptyAndDefaults`, `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsReplacementAndPresence`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedCreateWireAndPresence`, `internal/cmd/sdkgen/glance_metadef_namespace_create_test.go:TestGlanceMetadefNamespaceNestedCreatePreservesRegistry`.
- Nested dictionary keys, Name, object Required entries, description, prefix and target accept validUTF8 only, including empty/control/route characters/>80; root route80 validation is separate, no trim/dedupe/CSV/key-name normalization. Property Type is nonemptyUTF8 without enum gate, Title validUTF8 empty allowed. Flat Attributes checks original keyUTF8 before marshal, requires full validUTF8 JSON values incl null/exact number tokens, and rejects exact nine owned keys. Local schema/WSME/DB/policy gates are not added; failures remain preHTTP with original causes.
  Tests: `image/v2/metadefnamespaces/nested_core_test.go:TestMetadefNamespaceCreateNestedCoreCompletePreflight`, `image/v2/metadefnamespaces/nested_options_test.go:TestMetadefNamespaceCreateNestedOptionsCallbackAndRawValidation`, `image/v2/metadefnamespaces/nested_contracts_test.go:TestMetadefNamespaceNestedDefinitionsAndRawSchema`.
- Create with no options posts metadefs/namespaces with only the namespace key, matching Python _create(MetadefNamespace, namespace=...).
  Tests: `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`.

Python/Go differences:

- Pinned Python MetadefNamespace recognizes resource_type_associations and inherited tags, but has no properties/objects descriptors; ordinary create proxy kwargs for those two are discarded under False unknown-body flags. Go covers all four actual server containers; this server augmentation is not invented Python operation parity.
- Python mutable Resource/dirty cache/same-object returns, arbitrary attrs/session/adapter/microversion and coercion framework remain broader than concrete snapshot Create. Go always sends an explicit requested Create without cached-state mutation.
- Go validates root safe route80 separately from UTF8-only nested JSON names/keys/Required; it does not locally reproduce pinned schema type enums, WSME mandatory/coercion/lost-key behavior, DB length/Unicode/CSV storage or server policy. Raw input forwarding and response echo do not prove keyword precision/storage round-trip.
- Pinned server creates parent then associations/objects/tags/properties sequentially; Invalid400 skips cleanup and other named failures attempt best-effort cleanup that may fail protected. Auto-created global resource types can survive. SDK adds no rollback/cleanup/child requests and fixtures do not establish deployed effects.
- Create201 nested content is submitted WSME echo rather than fresh persisted child fetch. Canonical Namespace response stays raw for nested definitions; generated child metadata and normalized schema/Required values are not inferred.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Python drops properties and objects kwargs because MetadefNamespace has no such Body attributes; Go can also send them with WithCreateProperties and WithCreateObjects.

### `python:image/v2/delete_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithDeleteIgnoreMissing`

- Literal namespace identity, fixed source capture and strict preflight: name is nonempty UTF8<=80runes; colon/Unicode/spaces remain, route-breaking/control/dot syntax rejected before callbacks/HTTP. No Find, Ref/name resolver, generic Resources or response-driven routing. Fixed original provider/prefix/header/microversion with live original auth; source checks after callbacks/body and before consumed rows/pages.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`.
- Concrete FullOpts replaces all configuration; helper values, maps, optional pointers and option slice are copied, per-callback config independently copied before/after and callback once per request/iteration. Nil/error callbacks and protected headers fail before HTTP; helper reuse/parallel iterations retain snapshots.
  Tests: `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Strict original Create201/Get-List-Update200/Delete204 accepted statuses. Accepted read/Close/UTF8/JSON/model/context/source failure preserves original bytes/header/status and read/Close/ctx.Err/customCause in owned ResponseError, without accepted-body replay. Delete actual204 preserves nonnil opaque acknowledgement plus error; close once. Native rejected response ownership is retained.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- DELETE204 has no body/query and returns selected requested Namespace with opaque actual acknowledgement evidence. IgnoreMissing nil=>true, explicitfalse native404 policy. True owns only clean physical404 and returns nil,nil after successful read/Close/context/source checks; only that404 bypasses RetryFunc. Missing/forbidden may be indistinguishable; defaultquiet404 is not absence proof; accepted204 handling error retains acknowledgement.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`, `internal/cmd/sdkgen/glance_metadef_namespaces_test.go:TestGlanceMetadefNamespacesRegistryAndConcretePolicy`.
- Delete sends DELETE metadefs/namespaces/{namespace} and by default treats 404 as success like Python ignore_missing=True.
  Tests: `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`.

Python/Go differences:

- Python mutable Resource/dirty cache/same-object returns, arbitrary attrs/session/adapter/microversion and coercion framework are broader than this concrete scalar/raw leaf.
- Go explicitly rejects route-breaking/control input, wrong canonical types and statuses outside201/200/204; Python generic Resource response/coercion differs.
- Go actual204 acknowledgement and cause-safe private clean404 ownership differ from Python None return and generic missing/native retry policy.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Go returns the actual 204 acknowledgement where Python returns None.

### `python:image/v2/get_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithGetHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithGetResourceType`

- Literal namespace identity, fixed source capture and strict preflight: name is nonempty UTF8<=80runes; colon/Unicode/spaces remain, route-breaking/control/dot syntax rejected before callbacks/HTTP. No Find, Ref/name resolver, generic Resources or response-driven routing. Fixed original provider/prefix/header/microversion with live original auth; source checks after callbacks/body and before consumed rows/pages.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`.
- Concrete FullOpts replaces all configuration; helper values, maps, optional pointers and option slice are copied, per-callback config independently copied before/after and callback once per request/iteration. Nil/error callbacks and protected headers fail before HTTP; helper reuse/parallel iterations retain snapshots.
  Tests: `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Strict original Create201/Get-List-Update200/Delete204 accepted statuses. Accepted read/Close/UTF8/JSON/model/context/source failure preserves original bytes/header/status and read/Close/ctx.Err/customCause in owned ResponseError, without accepted-body replay. Delete actual204 preserves nonnil opaque acknowledgement plus error; close once. Native rejected response ownership is retained.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- One fixed GET200 at requested namespace; ResourceType nil omits and explicitempty encodes resource_type=. Server alone applies property-name prefix; returned namespace/self/schema are passive. No local prefix rewrite, find, fallback or response-driven routing.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`, `internal/cmd/sdkgen/glance_metadef_namespaces_test.go:TestGlanceMetadefNamespacesRegistryAndConcretePolicy`.

Python/Go differences:

- Python mutable Resource/dirty cache/same-object returns, arbitrary attrs/session/adapter/microversion and coercion framework are broader than this concrete scalar/raw leaf.
- Go explicitly rejects route-breaking/control input, wrong canonical types and statuses outside201/200/204; Python generic Resource response/coercion differs.
- No authenticated cloud/server policy execution or full SDK parity is claimed.
- Typed optional ResourceType GET projection is an explicit Go wire convenience; pinned Python proxy does not expose a named resource_type argument.
- The pinned get_metadef_namespace accepts a literal namespace or mutable MetadefNamespace and fetches it directly. Go uses a required literal namespace string, makes a default body/query-free fixed GET, returns a fresh passive Namespace DTO and does not merge the request namespace or cached Resource attributes into omitted response fields. A response id remains raw rather than overriding the Namespace pointer or selecting another route.
- Go IsProtected reads the canonical protected JSON Boolean as a nullable pointer; Python exposes is_protected through the protected descriptor and applies Python bool coercion. Resource type associations, tags, properties, objects and other nested response values are available through independently owned Metadata.Body RawMessages instead of Python list/dict descriptors, tag default[] or mutable tag methods. Missing/null fields remain missing/null without synthetic defaults; Self is retained even though Python _translate_response removes self.
- The optional ResourceType selector is an additive typed Go wire projection; the named pinned getter exposes no such argument. Context and ordinary-header options replace adapter/session injection, and the selected client microversion remains unchanged. No additional version discovery is required by these pinned classes, whose inherited maximum microversion is None.
- The Go read accepts actual HTTP200 and a valid UTF-8 nonnull object, rejects wrong canonical types and unsafe/empty/over-80-rune literal identities before HTTP, and preserves accepted failure evidence. Python Resource accepts any status below400 and may retain request/cached attributes when JSON decoding raises ValueError. The raw DTO is the documented Go alternative; actual deployment and other named child/tag/mutation APIs are separate scopes.

### `python:image/v2/metadef_namespaces`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSortDir`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListResourceTypes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithListSinglePage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.All`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.NewWithDependencies`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListFilters`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListResourceTypes`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithRecordListSortDir`

- Literal namespace identity, fixed source capture and strict preflight: name is nonempty UTF8<=80runes; colon/Unicode/spaces remain, route-breaking/control/dot syntax rejected before callbacks/HTTP. No Find, Ref/name resolver, generic Resources or response-driven routing. Fixed original provider/prefix/header/microversion with live original auth; source checks after callbacks/body and before consumed rows/pages.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`.
- Concrete FullOpts replaces all configuration; helper values, maps, optional pointers and option slice are copied, per-callback config independently copied before/after and callback once per request/iteration. Nil/error callbacks and protected headers fail before HTTP; helper reuse/parallel iterations retain snapshots.
  Tests: `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Strict original Create201/Get-List-Update200/Delete204 accepted statuses. Accepted read/Close/UTF8/JSON/model/context/source failure preserves original bytes/header/status and read/Close/ctx.Err/customCause in owned ResponseError, without accepted-body replay. Delete actual204 preserves nonnil opaque acknowledgement plus error; close once. Native rejected response ownership is retained.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Reusable lazy List/All prepares per iteration, only six typed query keys. Limit nilomit/0wire/negativeinvalid; emptyMarker omitted, literalCSV resource_types unchanged, enums validated, server defaults/clamp delegated. MaxItems is local, SinglePage skipsnext only; no limit hint, arbitrary query, Python local Body filter or invented marker.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAdvertisedPaginationAndCaps`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesAdvertisedPagingAndControls`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Required nonnull namespaces array; rows decoded/check source only when consumed. Cap/break skip unused malformed rows and next; empty success stops; All nonnil empty and discards partials on errors. Only canonical advertised body next is followed after consumption. Exact /v2/metadefs/namespaces rootrelative maps to captured reverse-prefix collection while original Body stays unchanged. Sameorigin/path and initial key/value query retained, singleton bounded marker advances; changed target/query/escaped alias/dot/authority or repeated URL/marker rejected before HTTP. No first/schema/self/links/header/fallback.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAdvertisedPaginationAndCaps`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesAdvertisedPagingAndControls`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Owned namespace projection includes12 Body descriptors and location (13 fields): inherited id/name; canonical is_protected aliases protected in original member order with JSON truthiness/null; tags missing[]/null/scalar list; associations list-of-dict converts nonobject elements to{}. Present id includingnull authoritative else namespace. Declared Body recomputes current location; no declared Body retains supplied raw location.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsDeclaredProjectionAndOwnedChannels`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsDescriptorNullDefaultAndAliasOrder`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCurrentLocationAndStickyGuards`.
- Public namespace **query maps six server keys plus canonical declared Body local filters; visibility query takes precedence. Remote protected, unknown/tag query attrs are ignored; reserved heterogeneous/target/session/version controls rejected explicitly. Options own snapshots, full replacement/last wins, nil/errors/header/query conflicts fail before HTTP.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsSourceQueriesAndNormalizedLocalFilters`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCompletePreflightAndFaultReceipts`.
- Namespace Limit pointer distinguishes nil omission and explicit numericzero wire0. Zero disables marker fallback but follows explicit continuations; positive MaxItems hint replaces falseyzero or absentlimit. Positive requested limits persist; consumed raw-row cap precedes filters. Shared paging zero permission is opt-in and other services retain positive-only validation.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `internal/rest/list_zero_limit_test.go:TestListZeroLimitRequiresServiceOptIn`, `internal/rest/list_zero_limit_test.go:TestListZeroLimitRetainsExplicitLinksAndFalseyCapHint`.
- Actual200..399 requires whole UTF8 JSON, nonnull namespaces object envelope, array or single object; consumed rows nonnull objects. Constructor connection/microversion/_synchronized presence fails with original receipt/partial rows; self/other unknown fields remain passive actual Wire.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsActualStatusesAndMandatoryRepresentation`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`.
- Generic Body/HTTP Link continuation, short-page fallback using actual consumed id/namespace, empty stop, filters and cursor inheritance, cycle/drift/fixed path safety and exact relative /v2 collection aliases preserve pagination while AllRecords keeps partial rows.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsGenericPaginationAndRawMarkers`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions`, `internal/rest/list_versioned_path_test.go:TestListVersionedPathAliasRequiresExactRelativeCollection`.
- Accepted read/Close/UTF8/JSON/context/source errors preserve actual response and causes; source/outer invariants are sticky through preparation and body handling. Shared read boundary check precedes Close restoration, body closes once and accepted handling errors are never replayed.
  Tests: `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCurrentLocationAndStickyGuards`, `image/v2/metadefnamespaces/list_records_test.go:TestMetadefNamespaceRecordsCompletePreflightAndFaultReceipts`, `internal/rest/guarded_response_test.go:TestDoJSONGuardedObservesReadFailureBeforeCloseRestoresSource`.
- Connection Image/ImageV2 share the original provider/client and current location; actual non200 pages, descriptor projection, remote Wire values, snapshot headers and live token survive two pages. Older typed strict200 API remains separate and unchanged.
  Tests: `connection_image_metadef_record_lists_test.go:TestConnectionImageMetadefRecordListsShareLocationAndPaging`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.

Python/Go differences:

- This named mapping uses owned ListRecords/AllRecords; legacy typed CRUD/List/All keeps its original strict statuses and bounded projection. It does not assert all inherited mutable Resource/dirty/cache/session surfaces are implemented.
- Go fixes session/target/version/auth ownership, validates UTF8 and safe literal routes, retains independent receipts and raw Wire, and guards origin/path/query/cycles/conflicting links. Exact relative versioned collection links are allowed; arbitrary Python target or query replacements are excluded.
- Selected semantic query values use Go scalar/scalar-array encoding; object/nested-array query forms are rejected. Local comparison keeps bool and number distinct with exact decimal equality, unlike some Python equality/float/exception behavior.
- Dictionary links:{next:URL} is an explicit Go extension; Source dictionary iteration expects rel/href member objects. Namespace integer limit model does not preserve the dynamic Source difference between numeric0 and quoted0. Deprecated JMESPath applies to the Source generator before collection and may return heterogeneous non-list values; typed Go listing rejects that branch.
- Default server policy places these reader operations in core user; deployment visibility/ownership/policy and real cloud state are handled by the server and not proven by local fixtures.

### `python:image/v2/update_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateDisplayName`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateDescription`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateOwner`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces.WithUpdateProtected`

- Literal namespace identity, fixed source capture and strict preflight: name is nonempty UTF8<=80runes; colon/Unicode/spaces remain, route-breaking/control/dot syntax rejected before callbacks/HTTP. No Find, Ref/name resolver, generic Resources or response-driven routing. Fixed original provider/prefix/header/microversion with live original auth; source checks after callbacks/body and before consumed rows/pages.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Canonical Namespace is a validUTF8 nonnull object with exact-case optional string fields namespace/display_name/description/visibility/owner/self/schema and protected bool. Missing/null nil, empty/false present; literal nullable dates are unparsed. Nested definitions/links/unknown shapes stay independently owned raw JSON with exact large-number precision; no ID/Name/Links synthesis. Wrong types are atomic model failures with actual whole response proof.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`.
- Concrete FullOpts replaces all configuration; helper values, maps, optional pointers and option slice are copied, per-callback config independently copied before/after and callback once per request/iteration. Nil/error callbacks and protected headers fail before HTTP; helper reuse/parallel iterations retain snapshots.
  Tests: `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`.
- Strict original Create201/Get-List-Update200/Delete204 accepted statuses. Accepted read/Close/UTF8/JSON/model/context/source failure preserves original bytes/header/status and read/Close/ctx.Err/customCause in owned ResponseError, without accepted-body replay. Delete actual204 preserves nonnil opaque acknowledgement plus error; close once. Native rejected response ownership is retained.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- Shared unchanged REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect policy, blocks changed method/origin/path/query and owned KeepResponseBody/JSONResponse/RawBody/serialized JSON. Original accepted status enforced even if native hook expands OkCodes; unchanged serialized replacement may remain, nilbody differsJSONnull. Native reauth ErrOriginal/ErrReauth fields remain native.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesCanonicalModelsAndRawOwnership`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesResponseOwnershipAndNativeHooks`.
- One replacement PUT200 to fixed current namespace; required namespace body defaults current or optional supplied rename. Only supplied scalars encoded; omitted display_name/description cleared and visibility/protected/owner reset by server. No GET/RMW/PATCH, nested child update, automatic unprotect or rollback. Go always sends replacement, unlike Python dirty Resource cache no-op.
  Tests: `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence`, `image/v2/metadefnamespaces/core_test.go:TestMetadefNamespacesCoreCompletePreflight`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsValidationAndDefaults`, `image/v2/metadefnamespaces/options_test.go:TestMetadefNamespacesOptionsParallelReusableHelpers`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`, `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesOwnedPreparationAndSource`, `connection_image_metadef_namespaces_test.go:TestConnectionImageMetadefNamespacesShareClientAndPagingDefaults`, `internal/cmd/sdkgen/glance_metadef_namespaces_test.go:TestGlanceMetadefNamespacesRegistryAndConcretePolicy`.
- Update with no options sends PUT metadefs/namespaces/{namespace} with body {"namespace": current}, the body Python builds when it adds the namespace key before commit.
  Tests: `image/v2/metadefnamespaces/contracts_test.go:TestMetadefNamespacesFixedRoutesAndPayloads`.

Python/Go differences:

- Python mutable Resource/dirty cache/same-object returns, arbitrary attrs/session/adapter/microversion and coercion framework are broader than this concrete scalar/raw leaf.
- Go explicitly rejects route-breaking/control input, wrong canonical types and statuses outside201/200/204; Python generic Resource response/coercion differs.
- Nested definition POST attrs and Python cached Resource mutation/dirty update semantics remain outside this scalar-only create/replacement update.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Go can send only the scalar namespace fields; Python could also put resource_type_associations or tags in the PUT body, which Glance's namespace update ignores.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
)

type NamespaceEvidence struct {
    Created *metadefnamespaces.Namespace
    Read *metadefnamespaces.Namespace
    Rows []*metadefnamespaces.Namespace
    All []*metadefnamespaces.Namespace
    Updated *metadefnamespaces.Namespace
    Deleted *metadefnamespaces.Acknowledgement
}

func manageNamespace(ctx context.Context, service *image.Service, namespace, replacementNamespace, owner, resourceType string, pageLimit int) (out NamespaceEvidence, err error) {
    api := service.API.MetadefNamespaces
    description := "First line\nSecond line"
    protected := false
    out.Created, err = api.Create(ctx, namespace,
        metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{
            Description: &description, Protected: &protected,
        }),
        metadefnamespaces.WithCreateHeader("X-Request-Source", "example"),
        metadefnamespaces.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefnamespaces.WithCreateDisplayName("Example"),
        metadefnamespaces.WithCreateDescription(description),
        metadefnamespaces.WithCreateVisibility("private"),
        metadefnamespaces.WithCreateOwner(owner),
        metadefnamespaces.WithCreateProtected(false))
    if err != nil { return out, err }

    out.Read, err = api.Get(ctx, namespace,
        metadefnamespaces.WithGetOpts(metadefnamespaces.GetOpts{}),
        metadefnamespaces.WithGetHeader("X-Request-Source", "example"),
        metadefnamespaces.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        metadefnamespaces.WithGetResourceType(resourceType))
    if err != nil { return out, err }

    listOptions := []metadefnamespaces.ListOption{
        metadefnamespaces.WithListOpts(metadefnamespaces.ListOpts{}),
        metadefnamespaces.WithListHeader("X-Request-Source", "example"),
        metadefnamespaces.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefnamespaces.WithListLimit(pageLimit),
        metadefnamespaces.WithListMarker(""),
        metadefnamespaces.WithListVisibility("private"),
        metadefnamespaces.WithListSortKey("created_at"),
        metadefnamespaces.WithListSortDir("desc"),
        metadefnamespaces.WithListResourceTypes("OS::Glance::Image,OS::Nova::Server"),
        metadefnamespaces.WithListMaxItems(20),
        metadefnamespaces.WithListSinglePage(false),
    }
    for row, listErr := range api.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = api.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = api.Update(ctx, namespace,
        metadefnamespaces.WithUpdateOpts(metadefnamespaces.UpdateOpts{}),
        metadefnamespaces.WithUpdateHeader("X-Request-Source", "example"),
        metadefnamespaces.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefnamespaces.WithUpdateNamespace(replacementNamespace),
        metadefnamespaces.WithUpdateDisplayName("Example"),
        metadefnamespaces.WithUpdateDescription(description),
        metadefnamespaces.WithUpdateVisibility("private"),
        metadefnamespaces.WithUpdateOwner(owner),
        metadefnamespaces.WithUpdateProtected(false))
    if err != nil { return out, err }

    out.Deleted, err = api.Delete(ctx, replacementNamespace,
        metadefnamespaces.WithDeleteOpts(metadefnamespaces.DeleteOpts{}),
        metadefnamespaces.WithDeleteHeader("X-Request-Source", "example"),
        metadefnamespaces.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefnamespaces.WithDeleteIgnoreMissing(false))
    return out, err
}
```

```python
def manage_namespace(conn, namespace, replacement_namespace, owner, page_limit):
    created = conn.image.create_metadef_namespace(
        namespace=namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    fetched = conn.image.get_metadef_namespace(namespace)
    rows = list(conn.image.metadef_namespaces(
        limit=page_limit, visibility="private", sort_key="created_at",
        sort_dir="desc", resource_types="OS::Glance::Image,OS::Nova::Server"))
    updated = conn.image.update_metadef_namespace(
        namespace, namespace=replacement_namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    conn.image.delete_metadef_namespace(replacement_namespace, ignore_missing=False)
    return created, fetched, rows, updated
```

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
)

func createNestedNamespace(ctx context.Context, service *image.Service, namespace string) (*metadefnamespaces.Namespace, error) {
    description := "A compute shape"
    prefix, target := "hw:", "metadata"
    properties := map[string]metadefnamespaces.PropertyDefinition{
        "mode": {
            Type: "string", Title: "",
            Attributes: map[string]json.RawMessage{
                "enum": json.RawMessage(`["small","large"]`),
                "default": json.RawMessage(`"small"`),
            },
        },
    }
    objects := []metadefnamespaces.ObjectDefinition{{
        Name: "shape", Description: &description,
        Properties: map[string]metadefnamespaces.PropertyDefinition{
            "vcpus": {
                Type: "integer", Title: "VCPUs",
                Attributes: map[string]json.RawMessage{"minimum": json.RawMessage(`0`)},
            },
        },
        Required: []string{"vcpus"},
    }}
    tags := []metadefnamespaces.TagDefinition{{Name: "example"}}
    associations := []metadefnamespaces.ResourceTypeAssociationDefinition{{
        Name: "OS::Nova::Server", Prefix: &prefix, PropertiesTarget: &target,
    }}
    return service.API.MetadefNamespaces.Create(ctx, namespace,
        metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{}),
        metadefnamespaces.WithCreateProperties(properties),
        metadefnamespaces.WithCreateObjects(objects),
        metadefnamespaces.WithCreateTags(tags),
        metadefnamespaces.WithCreateResourceTypeAssociations(associations))
}
```

```python
def create_namespace_known_containers(conn, namespace):
    return conn.image.create_metadef_namespace(
        namespace=namespace,
        resource_type_associations=[{"name": "OS::Nova::Server"}],
        tags=[{"name": "example"}])


def create_namespace_server_body(conn, namespace):
    # Adapter HTTP: properties/objects are not pinned MetadefNamespace descriptors.
    response = conn.image.post("/metadefs/namespaces", json={
        "namespace": namespace,
        "properties": {"mode": {"type": "string", "title": ""}},
        "objects": [{"name": "shape", "properties": {
            "vcpus": {"type": "integer", "title": "VCPUs"}},
            "required": ["vcpus"]}],
        "tags": [{"name": "example"}],
        "resource_type_associations": [{"name": "OS::Nova::Server"}]})
    response.raise_for_status()
    return response
```

## References

- [Metadef Object Namespace Record Lists](../../metadef-object-namespace-record-lists.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1390-L1485)
- [Metadef Namespace](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#update-namespace)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L342-L387)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L261-L340)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L552-L603)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L647-L661)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Metadef Namespaces Test](../../../connection_image_metadef_namespaces_test.go)
- [Glance Metadef Namespaces Test](../../../internal/cmd/sdkgen/glance_metadef_namespaces_test.go)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L705-L925)
- [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L25-L57)
- [Metadef Namespace](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py#L23-L54)
- [Tag](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L38)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L133-L261)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#create-namespace)
- [Nested Core Test](nested_core_test.go)
- [Nested Options Test](nested_options_test.go)
- [Nested Contracts Test](nested_contracts_test.go)
- [Connection Image Metadef Namespace Create Test](../../../connection_image_metadef_namespace_create_test.go)
- [Glance Metadef Namespace Create Test](../../../internal/cmd/sdkgen/glance_metadef_namespace_create_test.go)
