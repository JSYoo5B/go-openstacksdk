# Keypairs

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithGetOptions`

- Keypair Get directly uses the requested name at the selected reverse-prefix os-keypairs/{name} route and makes one bodyless GET with no LIST/fallback. Omitted or explicit empty owner sends no query; nonempty owner is the escaped user_id query, independent of response numeric id/name.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Native Name/Fingerprint/PublicKey/PrivateKey/UserID/Type fields are extracted. Missing/null Type stays empty without an ssh default; missing Name is not synthesized from the request. Missing/null keypair envelope returns nil,nil. Malformed JSON remains an error, while a known-field type error preserves the native partial model and original UnmarshalTypeError.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- The selected Compute2.55 headers, configured source header and live token reach every keypair GET without changing the client/provider/resource-base. Only native200 succeeds;403/404/unexpected203 retain actual/expected status, method, exact owner-qualified URL, body and response header instead of a missing success or unrelated request.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.

Python/Go differences:

- Python accepts a name/Keypair Resource, uses name as Resource ID and can retain a seeded name; Go requires the explicit name string and returns actual native Name/Fingerprint/PublicKey/PrivateKey/UserID/Type fields. Numeric wire id, source ssh type default, created/deleted/location/unknown descriptors and mutable Resource methods are not synthesized. Native missing/null wrapper may be nil,nil and known-field decode errors may retain a partial model.
- Optional owner is concrete GetOpts.UserID; empty string omits it like the source truthy check. Go preserves the selected client version rather than Keypair Resource discovery/max2.10 selection or an automatic owner version change. A foreign-owner403 remains an HTTP error; local fixtures do not grant or verify real-cloud admin permissions.
- The documented native strict200 response and typed projection are Go alternatives to Python adapter-wide response/seed/session policy. No list fallback, keypair creation/deletion or whole Resource lifecycle support is promoted through this named getter.

### `python:compute/v2/delete_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.DeleteKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteHeader`

- DeleteKeypair treats the supplied name as the logical keypair ID and makes one bodyless DELETE at the selected prefixed member route without GET, LIST or wait. Empty/default owner is omitted, an explicit Unicode/special-character owner and vendor query are URL-escaped, and selected live token/configured headers/both Compute2.1 version headers are retained without adding an owner-version gate or changing the client.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- IgnoreMissing defaults to true; explicit true ignores ordinary404 and explicit false preserves the native404 cause plus ErrNotFound. Owned final200..399 is success with passive non-JSON response bytes and no extraction or follow-up request; HTTP403 is preserved with actual/expected status, exact DELETE URL, body/header and operation context. Native Delete202/204 and strict404, and Remove's default/strict missing policy, remain separate unchanged APIs.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- Concrete typed options are snapshotted; last option wins, bulk resets typed owner/missing selections while retaining extensions, and caller query/header aliases cannot alter the prepared request. Nil API/client/context/options, unsafe name, null missing policy, protected owner/ignore_missing queries, unsupported body fields and spoofed authentication headers fail without HTTP. Pre-canceled or option-canceled contexts preserve context.Canceled, custom cause and the original option error; captured-source failures stop later callbacks.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedPreflightAndOptionOwnership`.
- Accepted read/Close, nested404, cancellation and source failures remain terminal ResponseErrors with physical202 body/header and original cause, do not become ErrNotFound or ignored-missing success, and do not invoke native retry after acceptance. A native503 retry callback that changes the captured source is stopped before a second DELETE while retaining original HTTP evidence; options are not replayed.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteAcceptedFailuresAndNativeRetryGuards`.

Python/Go differences:

- Python accepts a keypair Resource or its logical name ID and returns None. Go accepts a validated single-segment/text name and returns error only; nil represents accepted deletion or deliberately ignored ordinary missing, with no completion wait. UserID is a concrete string so empty is omitted, and nullable IgnoreMissing distinguishes unset/default true from false while rejecting JSON null rather than applying Python dynamic truthiness.
- Go retains the caller's selected microversion and sends explicit owner query literally; it does not perform Python session discovery, change the version or add a new local2.10 owner gate. Nova policy determines foreign-owner permissions, with403 preserved. Full connected Resource/session lifecycle remains separate SDK scope.
- Owned DeleteKeypair accepts final200..399 and reads passive body bytes while retaining physical read/Close/source/context evidence. Native Delete remains strict202/204 and404; native Remove retains its existing lookup/missing policy. The separate Cloud ComputeCloudMixin/delete_keypair operation and arbitrary Resource name resolution are not promoted by this direct Proxy mapping.

### `gophercloud:compute/v2/keypairs.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithGetOptions`

- The direct keypair-name GET uses the selected ResourceBase and makes no LIST or unrelated request. Default and explicit-empty UserID omit the owner query; a supplied owner is escaped as user_id. The library-owned concrete GetOpts builder retains live provider authentication, configured headers and both selected Compute2.55 version headers without changing the selected client.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Native Extract preserves the six string fields Name/Fingerprint/PublicKey/PrivateKey/UserID/Type and ignores unknown wire fields. Missing/null type remains empty, missing name is not seeded from the requested name, and missing/null keypair wrappers return nil,nil. Malformed JSON remains an error; a nonstring known field preserves the native partial KeyPair and json.UnmarshalTypeError.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Only native GET200 is accepted. Actual403/404/unexpected203 preserve ErrUnexpectedResponseCode with actual/expected status, method, exact owner-bearing URL, response body and header, and do not trigger a fallback request or convert missing to success.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.

Python/Go differences:

- The upstream optional GetOptsBuilder interface is replaced by concrete GetOption/WithGetOptions and a library-owned serializer; API.Get returns the extracted native KeyPair and wraps failures with operation context. Native aliases, owner omission, strict200 status and partial-result extraction are preserved. The documented Python get_keypair mapping is a separate catalog operation.

### `gophercloud:compute/v2/keypairs.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithDeleteQuery`

- Native Delete issues one bodyless DELETE directly at the selected prefixed os-keypairs/{name} route. Omitted options send no query; the library-owned concrete DeleteOpts builder escapes an explicit owner as user_id and binds an escaped vendor query extension alongside it. Live token, configured source headers and both selected Compute2.55 version headers reach the actual request.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- Native202/204 succeeds without JSON extraction while200/404 remains strict ErrUnexpectedResponseCode. Failure preserves expected202/204, actual status, DELETE method, exact URL and original passive response body/header. Default/strict Remove404 compatibility remains unchanged and does not alter direct native Delete's missing policy.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.

Python/Go differences:

- The upstream DeleteOptsBuilder interface is replaced by concrete DeleteOption/WithDeleteOptions and a library-owned query serializer with WithDeleteQuery extensions. API.Delete returns ExtractErr with operation context; native direct-name route, owner omission, strict202/204 and strict404 behavior are preserved. New DeleteKeypair's default-ignore/broad-status policy is an independent Python operation mapping and is not applied to this native function.

### `python:compute/v2/create_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.CreateKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreatePublicKey`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateType`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateHeader`

- CreateKeypair performs one fixed selected-ResourceBase collection POST at os-keypairs with keypair envelope and no query, preliminary GET/LIST, completion wait or cleanup. No attributes produce an empty keypair object. All eight declared body fields plus id/is_deleted aliases accept explicit null, empty, scalar, array or object JSON values; name is body data without member-path ID validation. Unknown semantic attributes are discarded before serialization. Selected live token, source headers and both Compute2.1 version headers are retained without a new local type or owner gate.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Original dirty attributes form the Resource seed; no type=ssh field is synthesized into the POST. Omitted final type projects as ssh only in the view, while null and empty remain literal. Input and response numeric id become name only when canonical name is absent; present name including null wins. Resource id independently mirrors name. Raw deleted values are transmitted unchanged and shared BodyFieldBoolean projects is_deleted only in the view; response null stays null, nonempty false-string is true and empty array/zero is false. Bulk and response deleted win over is_deleted; individual semantic options use the last canonical value.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Actual wrapped keypair or flat object fields overlay recognized seed fields, with response omission preserving input and present null replacing it. The independently owned nine-field Resource excludes unknown attributes; independently owned Wire preserves only actual response fields, literal id/deleted aliases, unknown/vendor fields and large JSON numbers. View defaults and input are not inserted into Wire. Envelope, record/resource/wire headers and status preserve the physical response, and mutations do not cross these records.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Accepted final200..399 includes actual200/201/202/203, bodyless204 and flat300. Empty, opaque and malformed JSON leave the seeded Resource and nil Wire; parseable nonobject roots or null/array keypair envelopes fail with accepted ResponseError and body/header/status receipt. HTTP403/404 retain the native actual/expected status, POST method, exact collection URL and original body/header; create404 is not ignored. Physical read/Close nested404 and post-response source failure retain accepted201 evidence and original cause rather than tolerated JSON failure, with zero SDK resend.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Bulk/individual semantic inputs, vendor values and callback-added attrs/raw fields are snapshotted before later mutations. Bulk replaces semantic choices while retaining extensions; callbacks execute once per call. Core name/id/type/user_id/deleted/is_deleted extensions and unsupported query fail before HTTP. Captured API/source changes and callback cancellation stop later callbacks and HTTP, preserving the original callback error, context.Canceled and custom cancellation cause. A native503 retry hook that changes the source cannot issue a second allocation and retains the original HTTP evidence; allowed token refresh reaches the fixed request.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.

Python/Go differences:

- Python returns a connected mutable Keypair and accepts dynamic attrs. Go returns a finite stateless source-shaped Resource view plus independently owned actual Wire and response receipt; all nine declared view attributes and eight body fields are represented by raw JSON. String convenience options coexist with concrete map/bulk attributes for nullable or untyped values. Unknown semantic attrs are discarded and explicit vendor fields use protected library-owned extensions. Full Resource/session/location/discovery remains separate shared SDK scope, without blocking this finite named create contract.
- Canonical name over id, including present null, matches Keypair._consume_attrs for input and response. For deleted/is_deleted collisions Python generic mapping follows input or response insertion order; Go deterministically favors canonical deleted in bulk and response maps. Individual Go options apply sequentially with the last canonical choice winning. These documented map/option choices are not insertion-order emulation.
- Go uses the caller-selected client microversion unchanged rather than performing Python discovery and selecting the discovered maximum capped at Keypair2.10 when no session default exists. Name/type/owner semantics and permissions are left to Nova; an owner fixture is not a real cloud authorization result. The separate Cloud ComputeCloudMixin/create_keypair wrapper is not promoted by this direct Proxy operation.
- Go tolerates invalid-UTF8 accepted bodies as unparseable with seeded Resource and nil Wire, preserving the envelope; Python response charset detection and replacement-character decoding are not reproduced. Valid nonobject JSON and physical read/Close/source/context failures remain explicit receipt errors. Native Create retains its required string Name, typed model and strict200/201 policy as a separately reviewed source declaration.

### `gophercloud:compute/v2/keypairs.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithCreateField`

- Native Create preserves required Name preflight and sends one authenticated POST to the selected prefixed os-keypairs collection without query or lookup. Concrete CreateOpts binds Name, UserID, Type and PublicKey together with a vendor object extension in the actual keypair body. A protected user_id extension and omitted required Name fail before HTTP; the application need not implement CreateOptsBuilder.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Native Extract preserves all six typed string fields Name/Fingerprint/PublicKey/PrivateKey/UserID/Type and ignores unknown vendor fields. A nonstring name preserves the decoded fingerprint in a partial KeyPair and json.UnmarshalTypeError; a null keypair envelope returns nil,nil. SDK-owned seed/default/alias/RawResource handling is not substituted for native extraction.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Only native POST200/201 is accepted, with actual fixtures for both. Actual202 remains ErrUnexpectedResponseCode with expected200/201, original POST method, exact collection URL, original response body and header. The additive CreateKeypair200..399 policy does not widen native Create.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.

Python/Go differences:

- The upstream CreateOptsBuilder interface is replaced by concrete CreateOpts/CreateOption and a library-owned serializer with WithCreateOptions/WithCreateField. API.Create returns the extracted native KeyPair and wraps failures with operation context. Native required Name, optional string fields, strict200/201 and partial/null extraction remain intact; source-shaped nullable attributes, view-only ssh and Resource/Wire belong to the independent Python CreateKeypair mapping.

### `python:compute/v2/keypairs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListFilters`

- Wrapped and flat rows produce a nine-attribute source-shaped Resource: nested keypair fields override outer fields, canonical name overrides literal id including null/empty, logical id follows name, omitted type defaults to ssh and deleted uses response Boolean truthiness. Resource fields and headers are independent of the complete actual outer-row Wire, raw row Envelope and record headers, including precise extension numbers.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`.
- Semantic user_id/limit/marker go to the server while id/created_at/is_deleted/fingerprint/name/private_key/public_key/type filter the normalized Resource locally. Unknown attributes and wire deleted as a semantic key are ignored. Option maps/pointers/carriers are owned; list construction is lazy and each repeated iteration applies callbacks once to fresh state.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`.
- Advertised body links, keypairs_links, top-level next and HTTP Link continue the fixed collection. A filtered-out last row still supplies its normalized name as fallback marker, caller mutations do not change that marker, raw max_items precedes local filtering, single-page and break stop requests, and an empty page stops even with next. A first server-introduced positive limit does not activate marker fallback. Late403 retains already yielded results and actual HTTP evidence.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsContinuationLogicalMarkersCapsAndLateErrors`, `internal/rest/list_test.go:TestListDictionaryLinksAreExplicitAndGuarded`.
- A selected Compute version is retained; otherwise finite Nova discovery selects at most2.10. Explicit empty overrides selection for versionless reads on a private client. Invalid controls/collisions/options and changed source fail before subsequent work; accepted page read/source faults remain ResponseError with actual body/header/status and cannot be replayed or mistaken for missing.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding`.
- The plural keypairs envelope accepts an array or one object. Owned accepted204 still requires a list representation rather than silently becoming an empty native page; parsed or physical failures retain accepted response evidence.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding`, `internal/rest/list_test.go:TestListOptionalReaderAndRepresentationsKeepDefaultPolicy`.

Python/Go differences:

- Concrete With-functions and iter.Seq2 replace Python kwargs and its generator. The owned Resource exposes the nine declared keypair Body attributes; computed Connection location and mutable Python Resource methods are not synthesized by this leaf API. Wire and raw row Envelope retain the actual response without normalized aliases/defaults.
- Typed Limit/MaxItems are nonnegative with zero meaning omitted/unbounded; a present pagination limit must be one positive integer and marker one nonempty string. Raw queries override same typed queries; semantic query collisions with typed/raw inputs are rejected rather than silently overwriting them. Unknown semantic fields are discarded; explicit raw query extensions remain separate.
- Operation controls use dedicated options. Arbitrary base_path, deprecated jmespath_filters and allow_unknown_params are explicitly unsupported instead of becoming wire filters. Continuations are guarded to the captured collection route and query; inconsistent/cyclic links fail. Dictionary links.next is an installed Go compatibility extension; the pinned source ordinary-next dictionary does not follow that form.
- Nil operation microversion uses the captured selected version or the shared finite Nova discovery capped at2.10. Explicit empty deliberately performs versionless reads; no mutation of the original client or automatic upgrade to the newer keypair pagination version occurs. This does not reproduce KSA's entire discovery/cache/session system.
- Source JSON fields remain passive raw values, while malformed object shapes, physical accepted faults and Go context/source errors are explicit. The native List/All results and single-page policy remain separate and unchanged.
- Canonical deleted wins a conflicting is_deleted response field deterministically; the pinned Python body component may depend on insertion order when both wire and alias names are present.

### `python:compute/v2/find_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.FindKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindIgnoreMissing`

- Find always attempts the escaped fixed member GET for accepted literal UTF-8 input and optional truthy owner. A successful member response returns its input-name/owner-seeded Resource with response fields overriding seed including null, while Wire and Envelope contain only actual response data. Opaque/malformed/empty accepted bodies retain the seed with nil Wire; valid nonobject/envelope failures retain a receipt beside the error.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`, `resource/find_identity_response_id_test.go:TestCollectionFindIdentityPassiveIDsAreOnlyCompared`.
- Only clean actual400/403/404 permits a full owned list fallback with the same owner, version, headers and live provider token; no name query hint is added. Passive numeric names do not equal a string identity. Unique matches consume the complete list, duplicate matches raise ErrAmbiguous, late list errors remain errors, default missing returns nil,nil and strict missing returns ErrNotFound.409/500 do not fall back.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`.
- Automatic ceiling2.10 selection occurs once across direct GET and fallback, and concrete options apply once for the whole find. Explicit empty preserves the original selected client while issuing both phases versionless with owned option pointers/maps/headers and final missing-policy precedence.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`, `api/compute_keypair_find_test.go:TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors`.
- Nil/invalid inputs, options, canceled context, outer guard and changed source fail before subsequent callbacks/HTTP. Accepted read/Close/source failures with nested404 are terminal and preserve the actual member receipt, or discovery evidence without fabricating a member result. Native retry source mutation cannot issue another member/fallback; live reauthentication preserves the fixed member and owner.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors`.

Python/Go differences:

- FindKeypair is an additive concrete-option API returning the owned KeypairRecord. Existing Find(resource.Ref), Get and their native KeyPair ABI retain their explicit lookup and strict native decode policies. Default missing is true without Python's deprecation warning.
- Literal identity validation rejects empty/whitespace-only, invalid UTF-8, controls and exact '.' or '..' identities; reserved characters and internal spaces use escaped always-member GET through the audited direct-route hook. Passive response name/id values never choose another target.
- The declared nine Body fields are projected without Python mutable Resource or computed Connection location. Accepted malformed/nonJSON body tolerance preserves a seeded Resource with nil Wire; physical failures and parsed invalid objects return an error with actual accepted receipt. Missing/duplicate failures use Go errors.Is sentinels. Invalid UTF-8 response bytes also use the Go tolerant opaque-body path; Python HTTP text decoding is not reproduced.
- Microversion selection uses the shared finite Nova reader once. Explicit empty is a Go versionless override; source/session discovery caches and warning facilities are not replicated. Foreign-owner access still depends on deployment policy and its HTTP failures remain visible.

### `gophercloud:compute/v2/keypairs.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithListQuery`

- The unchanged native List binding serializes typed UserID then applies raw wire overrides/extensions using its SDK-owned concrete builder. GET retains selected ResourceBase, live token, source header and both2.55 version headers, with no body. Raw limit is a request extension rather than a native ListOpts field.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.
- Native nested keypair extraction preserves six string fields and ignores vendor fields; omitted/null type stays empty and null nested keypair yields its zero native model. SinglePageBase does not follow body or HTTP next links. The tested empty204 without JSON Content-Type is an empty native page;200/300 retain normal extraction and201 remains outside native pager codes.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.
- Unexpected201 and403 preserve native actual/expected200,204,300 statuses, GET method, exact owner-bearing URL, response body/header. Invalid known string types preserve json.UnmarshalTypeError and stop before yielding partially decoded rows.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.

Python/Go differences:

- The optional upstream ListOptsBuilder is replaced by concrete ListOption/WithListOptions plus a library-owned serializer, and the native pager/extractor are exposed as iter.Seq2. Native ListOpts, six-string KeyPair alias,200/204/300 pager acceptance and SinglePageBase behavior remain intact. Additive ListRecords implements the separately reviewed Python-style list instead of changing this native operation.

## References

- [User Read APIs](../../user-read-apis.md)
- [User Actions](../../user-actions.md)
- [Keypairs Console](../../keypairs-console.md)
- [Keypairs List Find](../../keypairs-list-find.md)
- [Keypairs Cloud](../../keypairs-cloud.md)
