# Objects

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/get_object_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithGetMetadataNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutGetMetadataNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.HeadMetadata`

- GetMetadata sends one bodyless query-free normal HEAD with strict actual200 at the captured effective ResourceBase/Endpoint and once-escaped literal container/object. No parent HEAD, lookup, contents GET or metadata cache; normal server symlink semantics remain.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Newest nil omits X-Newest, explicit false/true emits its literal value on Getter HEAD, and Without clears. Concrete whole opts replaces configuration; singular/plural headers merge with original ASCII validation, alias rejection and last singular overwrite.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsReplacementAndRetainedHandles`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Reusable helper factories own header maps/Newest pointers; initialized callback storage is copied after every callback. Nil callbacks and retained handle mutation cannot alter another invocation; option and source/custom context causes are joined.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsReplacementAndRetainedHandles`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Actual HEAD atomically projects initialized lowercased suffix Values including presentempty, nullable exact signed64 ContentLength and nullable literal ETag/content/timestamp/expiry/manifest/cache fields. Original nonASCII keys, case aliases, duplicate/multivalue known/custom headers and malformed counters fail with nil Metadata and retained raw response.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreAtomicProjection`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsAtomicHeaders`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Required literal names permit once-escaped object hierarchy and percent/space/Unicode/question/hash/colon, while empty/invalidUTF8/backslash/control/DEL and dot path components fail before HTTP with ErrInvalidOption and native name cause. Nil API/context/source/provider, unsafe Endpoint/effectivebase/type/header and canceled contexts fail before callbacks/HTTP.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsCompletePreflightAndCauses`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Only actualHEAD200 creates independent Body/Header/Status. Accepted read/Close/context/custom cause/source/projection failures preserve raw result plus whole independent ResponseError; one Close, no accepted replay and original strconv.NumError are retained. Unknown raw headers may repeat and remain passive.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCorePhaseFailuresAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsAtomicHeaders`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Captured source/provider/raw bases/Type/Microversion/ordinary headers remain stable, original live auth and native prebody retry/backoff/reauth/timeout/transport/fixedtarget redirects remain. SDK method/body/status ownership holds while advanced RetryFunc.MoreHeaders stays providerpolicy; native Get200/204 and Update201/202 ABI remain intact.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSourceAndLiveAuth`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsSourceAndNativePolicy`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`, `internal/cmd/sdkgen/swift_object_metadata_test.go:TestSwiftObjectMetadataPreservesNativeAndRegistry`.
- One query-free HEAD for explicit literal container/object names; final200..399 accepted; actual headers/body/status and atomic existing MetadataInfo projection retain Source/native/context errors.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- This callable implements the Proxy string-name branch. Pinned get_object_metadata always performs HEAD for names and existing Object resources; mutable Resource input/return semantics are tracked separately.
- Observed custom wire-alias collisions remain valid HEAD observations. Go preserves both suffixes; the POST-only collision boundary applies to setters/deleters, not this getter.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/set_object_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.SetMetadataFromFresh`

- SetMetadata always performs a fresh owned HEAD200 at symlink=get then one query-free bodyless full-custom POST202. Nil/empty maps and unchanged values still request both phases; no final HEAD, payload transfer, parent creation, synthetic updated state, whole RMW restart or cache.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreEmptyWritesAndNineHeaders`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Caller metadata is snapshotted before callbacks. Original ASCII token suffix-only keys reject full X-Object-Meta prefix and case aliases; literal legalUTF8 values preserve spaces/percent/HTAB and presentempty without Python str coercion, falsey omission, trim or URL encoding.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsCompletePreflightAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`.
- Set merges supplied custom suffixes into the fresh complete custom snapshot and resends all resulting custom metadata. Only the nine observed mutable system wire headers are carried; observed→sourceordinary→POSToption precedence retains explicit empty overrides. Auth/readonly/unknown/sysmeta/transient/SLO response fields are never blindly copied.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreEmptyWritesAndNineHeaders`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Mutation owns HEAD symlink=get, validates observed known symlink headers and rejects X-Symlink-Target presence beforePOST with ErrUnsupported and raw Before. This bounds current versioned aliases; later link changes may apply POST then307 and retain native error evidence rather than prove unchanged state.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSymlinkAndStrictStatus`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsMutationEvidence`.
- Mutation opts are concrete initialized header maps with whole replacement/canonical merge, original key validation and independent factory/callback snapshots. MetadataOpts headers apply only POST; sourceordinary headers apply fresh read and captured POST. Request/auth/body/Newest/custom/symlink/sysmeta prefixes are reserved; exact generic system wire headers are allowed.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsReplacementAndRetainedHandles`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsCompletePreflightAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Accepted HEAD raw Body/Header/Status are retained as Before on read/Close/context/source/projection/symlink and prePOST failure, with nil Acknowledgement. Completed Before remains immutable through native POST errors; failedaccepted HEAD Metadata remains nil and actual proof/cause is preserved.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreAtomicProjection`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCorePhaseFailuresAndCauses`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSourceAndLiveAuth`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsAtomicHeaders`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsMutationEvidence`.
- Only actualPOST202 creates opaque Acknowledgement, preserved independently even on accepted read/Close/context/source failure plus whole ResponseError. Before is observed priorstate and POST headers are not projected as refreshedmetadata; accepted responses are closed once and never replayed.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCorePhaseFailuresAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Missing404/unexpected/native/transport errors are never ignored, preserve original typed status/body/header/custom/context/reauthfield causes, and retain completed Before on second-phase failure. Captured source/liveauth and fixed method/target/serializednilbody/strictstatus survive native perphase prebody policies without an automatic whole RMW retry or CAS promise.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSymlinkAndStrictStatus`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSourceAndLiveAuth`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsSourceAndNativePolicy`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `internal/cmd/sdkgen/swift_object_metadata_test.go:TestSwiftObjectMetadataPreservesNativeAndRegistry`.
- Name-created Source Object has no last_modified_at: query-free HEAD then POST, including empty setter; no final HEAD because Object override does not use refresh parameter.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterFreshMergeFalseyAliasesAndNoFinalHEAD`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterEverySystemAliasAndEmptyInput`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- Falsey JSON values omitted, all existing custom metadata preserved including empty strings, exact full prefix and all nine system canonical/wire aliases; Source incoming/system descriptor precedence preserved within the explicit observed wire-alias collision boundary; unordered observed collisions cannot claim full dictionary-order parity.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterFreshMergeFalseyAliasesAndNoFinalHEAD`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterEverySystemAliasAndEmptyInput`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- Source Header delete_after conversion reuses DescriptorIntegerJSON (unsigned digit strings/Unicode/arbitrary integers, signed/space/nonnumeric→zero, nondecimal isdigit→error); bool descriptor preserves bool(nonempty string).
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterFreshMergeFalseyAliasesAndNoFinalHEAD`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataSetterEverySystemAliasAndEmptyInput`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- POST-only Go boundary for surviving ambiguous observed custom aliases; successful HEAD proof retained with ErrInvalidOption, without POST or manufactured acknowledgement. Read/no-op deletion remains allowed; deletion that removes an alias can proceed after resolving the collision.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataObservedAliasesRejectOnlyBeforePOST`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- This callable implements the Proxy string-name branch. Existing cached Object inputs can skip fresh HEAD in Python; mutable Resource input/cache is tracked separately rather than silently treated as complete.
- Incoming Go-map alias collisions are rejected before HTTP. Separately, when POST is actually needed, surviving observed custom suffixes that map to the same final wire header are rejected with ErrInvalidOption and successful HEAD response proof: Go http.Header loses original response order, while Python resolves these aliases by ordered last assignment. Getter and nil/empty/absent-key deletion no-ops are unchanged; deleting one conflicting suffix can remove the ambiguity.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/delete_object_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DeleteMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DeleteMetadataAndRefresh`

- DeleteMetadata always performs fresh own HEAD200 then one full-custom bodyless POST202, including nil/empty key slices and absent suffixes. It is custom metadata subtraction, never HTTPDELETE or X-Remove headers; no final refresh, inferred no-op or suppression of missing errors.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreEmptyWritesAndNineHeaders`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Delete snapshots the caller key slice before callbacks and rejects full-prefix, empty, original nonASCII/malformed suffix tokens, duplicates and casealiases before HTTP. Names/source/header/context validation preserves native typed/option/custom causes with no lookup or coercion.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsCompletePreflightAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Delete removes only specified suffixes from fresh complete custom Values, preserves all other custom entries including empty and carries exactly nine bounded observed mutable systemheaders with source/option override precedence. Exact system wire overrides are separate from custom keys; Python alias deletions are not synthesized.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreRoutesAndMerge`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreEmptyWritesAndNineHeaders`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Delete shares mutation whole-opts replacement, canonical singular/plural merge, original token validation, retained callback/factory ownership and parallel reusable options. All preparation/input capture completes before HTTP; POST option headers do not alter the HEAD read phase.
  Tests: `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsFactorySnapshots`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsReplacementAndRetainedHandles`, `objectstorage/v1/objects/metadata_options_test.go:TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsPreflightAndOptions`.
- Delete reads own symlink=get metadata, validates known symlink response headers and rejects link presence beforePOST while retaining actual Before. Current versioned aliases are deliberately bounded; native307 after application remains an error with priorphase proof and no unchanged-state or rollback claim.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSymlinkAndStrictStatus`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsMutationEvidence`.
- Delete retains acceptedHEAD raw Before on firstphase handling or prePOST error, completed Before without Acknowledgement on nativePOSTfailure, and actualPOST202 Acknowledgement on accepted handling failure. Raw fields, response errors and typed/custom causes are independent with one Close/no accepted replay; no response state is synthesized.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCorePhaseFailuresAndCauses`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsAtomicHeaders`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_object_metadata_test.go:TestConnectionObjectMetadataSharesClientAndOwnsPhases`.
- Strict HEAD200/POST202 and missing404 preservation remain separate from native Get200/204 and Update201/202. Source/provider/raw bases/liveauth and method/URL/body/status guards retain perphase prebody native policy, advanced MoreHeaders and reauthfield boundaries without automatically restarting the read/merge/write flow.
  Tests: `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSourceAndLiveAuth`, `objectstorage/v1/objects/metadata_core_test.go:TestObjectMetadataCoreSymlinkAndStrictStatus`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsSourceAndNativePolicy`, `objectstorage/v1/objects/metadata_contracts_test.go:TestObjectMetadataContractsWireAndIdentity`, `internal/cmd/sdkgen/swift_object_metadata_test.go:TestSwiftObjectMetadataPreservesNativeAndRegistry`.
- Nil/empty keys are noHTTP; absent/case-mismatched/full-prefixed keys are HEAD-only no-op; actual deletion is query-free HEAD/POST/final HEAD with actual refresh data.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataDeleteNoOpsAndObservedFinalHEAD`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataDeleteAfterPreservesSourceBranch`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- Exact lower-case observed custom suffixes and system canonical keys; ordinary duplicates collapse during removal; delete_after deletes delete_at without raising deleted flag and exposes missing key as MetadataKeyError.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataDeleteNoOpsAndObservedFinalHEAD`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataDeleteAfterPreservesSourceBranch`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- POST-only Go boundary for surviving ambiguous observed custom aliases; successful HEAD proof retained with ErrInvalidOption, without POST or manufactured acknowledgement. Read/no-op deletion remains allowed; deletion that removes an alias can proceed after resolving the collision.
  Tests: `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataObservedAliasesRejectOnlyBeforePOST`, `objectstorage/v1/objects/metadata_python_test.go:TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- This callable implements the Proxy string-name branch. Existing cached Object inputs can skip fresh HEAD in Python; mutable Resource input/cache is tracked separately rather than silently treated as complete.
- Incoming Go-map alias collisions are rejected before HTTP. Separately, when POST is actually needed, surviving observed custom suffixes that map to the same final wire header are rejected with ErrInvalidOption and successful HEAD response proof: Go http.Header loses original response order, while Python resolves these aliases by ordered last assignment. Getter and nil/empty/absent-key deletion no-ops are unchanged; deleting one conflicting suffix can remove the ambiguity.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/copy_object`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Copy`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.ObjectScope.Copy`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCopyQuery`

- The existing ObjectScope.Copy resolves its literal resource reference under the bound container and delegates native COPY. Its fixture asserts one source COPY, Destination /target/copy, version-id=version,201 and returned ETag=checksum; this verifies Go capability only. This one fixture is separate native-capability evidence, not a Python parity or full COPY policy matrix.
  Tests: `api/swift_resources_contracts_test.go:TestSwiftScopeBindsUploadMetadataUpdateAndDownload`.

Python/Go differences:

- Pinned Proxy.copy_object(self)->None is only a three-line stub that raises NotImplementedError immediately; the pinned Python unit test asserts that exception. It implements no HTTP copy workflow, source/destination arguments, options or returned object.
- Existing Go API.Copy and ObjectScope.Copy remain native capability: source COPY accepts201, CopyOpts.Destination is /container/object, ObjectVersionID extends the query and CopyHeader extracts response headers. This available Go operation is not an implementation of the pinned Python stub.
- The pinned upload_object name is a class assignment to create_object, not a separate declaration/workflow/catalog operation. Its canonical review remains create_object; no upload_object alias review or additional parity count is introduced.

Unresolved boundaries:

- Pinned source has no executable copy workflow to port. Unsupported applies only to this pinned Python direct stub; Go native API.Copy/ObjectScope.Copy remain separately available, and reproducing NotImplementedError is not needed.
- Reviewing all29 direct Swift Proxy declarations does not complete the22 separately catalogued ObjectStoreCloudMixin workflows or inherited Proxy/Resource/session/cache/descriptor behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Objects.InContainer(ctx, resource.ID("assets"))
if err != nil { return err }
object, err := scope.Get(ctx, "images/logo ?.png")
if err != nil { return err }
fmt.Println(object.Name, object.Bytes, object.ContentType, object.Metadata)

for object, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(object.Name, object.Hash)
}

header, err := scope.Create(ctx, "docs/readme.txt", objects.CreateOpts{
    Content: strings.NewReader("hello"),
    Metadata: map[string]string{"Owner": "sdk"},
})
if err != nil { return err }
_ = header

download, err := scope.Download(ctx, resource.ID("docs/readme.txt"))
if err != nil { return err }
defer download.Close()
_, err = io.Copy(destination, download)
if err != nil { return err }
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListObjects(ctx context.Context, conn *sdk.Connection, containerName string) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Objects.InContainer(ctx, resource.ID(containerName))
	if err != nil {
		return err
	}
	for object, err := range scope.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(object.Container, object.Name, object.Subdir, object.Bytes)
	}
	return nil
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func ChangeObjectMetadata(ctx context.Context, conn *sdk.Connection, newest bool) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	container, key := "assets", "docs/readme ?.txt"
	getOptions := []objects.GetMetadataOption{
		objects.WithGetMetadataOpts(objects.GetMetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-read"},
		}),
		objects.WithGetMetadataHeader("X-Example", "read"),
		objects.WithGetMetadataHeaders(map[string]string{"X-Example-Trace": "metadata"}),
	}
	if newest {
		getOptions = append(getOptions, objects.WithGetMetadataNewest(true))
	} else {
		getOptions = append(getOptions, objects.WithoutGetMetadataNewest())
	}
	before, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		if before != nil {
			fmt.Println("HEAD", before.StatusCode)
		}
		return err
	}
	if before.Metadata.ContentLength != nil {
		fmt.Println("object bytes", *before.Metadata.ContentLength)
	}
	mutationOptions := []objects.MetadataOption{
		objects.WithMetadataOpts(objects.MetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-write"},
		}),
		objects.WithMetadataHeader("Content-Disposition", "attachment"),
		objects.WithMetadataHeaders(map[string]string{"Cache-Control": "private"}),
	}
	change, err := service.Objects.SetMetadata(ctx, container, key,
		map[string]string{"owner": "sdk", "empty": ""}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	change, err = service.Objects.DeleteMetadata(ctx, container, key,
		[]string{"owner"}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	refreshed, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		return err
	}
	fmt.Println("stored empty value", refreshed.Metadata.Values["empty"])
	return nil
}

func printObjectMetadataPhases(change *objects.MetadataChangeResult) {
	if change == nil {
		return
	}
	if change.Before != nil {
		fmt.Println("before HEAD", change.Before.StatusCode)
	}
	if change.Acknowledgement != nil {
		fmt.Println("POST", change.Acknowledgement.StatusCode)
	}
}
```

```python
obj = conn.object_store.get_object_metadata(
    "docs/readme ?.txt", container="assets"
)
same = conn.object_store.set_object_metadata(obj, owner="sdk", empty="")
assert same is obj
conn.object_store.delete_object_metadata(obj, keys=["owner"])
refreshed = conn.object_store.get_object_metadata(obj)
print(refreshed.metadata)
```

## Limitations

- Reads, uploads and waits report what one request observed. Swift replicas, eventual consistency, proxy caching and middleware decide when other clients see a change; a `HEAD` 404 or a matching metadata value is not a lock or a cluster-wide guarantee.
- Segmented uploads cannot be published atomically or rolled back. Unique segment prefixes and the returned cleanup evidence show what was written.
- `StreamObject` returns an `io.ReadCloser`: close it, and read it from one goroutine at a time.
- Text reads decode only UTF-8, a byte-order mark or a supported declared charset. Undeclared non-UTF-8 text returns `ErrUnsupported` with the original bytes; see [no charset guessing](../../../docs/sdk-support-ledger.md#deliberate-runtime-differences).

## References

- [`copy_object`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L531-L533)
- [Test Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/tests/unit/object_store/v1/test_proxy.py#L251-L252)
- [Listing](../../../docs/listing.md)
- [Containers](../containers/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Obj](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222)
- [Storage Object Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L631)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/obj/server.py#L665)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [atomic header projection](metadata_contracts_test.go)
