# Python Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/update_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordBasePath`

- Scalar ID/no recognized attrs performs no PUT/GET and keeps seed without HTTP receipt; id itself is not dirty.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordNoopSeedHasNoReceiptOrPUT`.
- Container dirty attributes are flat unwrapped JSON PUT without id; nullable/raw values and Source list/HREF projection remain distinct from received fields and owned request/seed.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordFlatDirtyBodyNullableAndRawValues`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordAliasesOrderResponseMergeAndReferenceProjection`.
- Malformed JSON/empty body keeps seed; valid nonobject and malformed HREF errors preserve accepted physical status/header/body.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordEmptyInvalidJSONAndShapeErrorsKeepActualProof`.
- Concrete options snapshot raw attrs, headers, version/basepath and literal escaped IDs; unsafe constructor/source/context/header inputs fail before HTTP.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordOptionsSnapshotBasePathAndSelectedVersion`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordPreflightAndConstructorCollisions`.
- Synthetic final100..399 accepts,400..999 preserves native proof; physical read/Close/cancel/source faults suppress callbacks; clean retry preserves body while tampering blocks replay and failedreauth retains original401.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordAcceptedAndRejectedStatusPolicy`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordPhysicalFailuresNeverReplayAndRetainNativeProof`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordCleanRetryAndFailedReauthPreserveBody`.
- Shared charset subset decodes Latin1, BOM UTF16/32 or explicitLE/BE projection while actual envelope/headers/error proof remain physical; unsupported codecs remain explicit.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordRequestsCharsetGapIsExplicit`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordExplicitCharsetsDecodeViewsKeepPhysicalEnvelope`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordDecodedShapeFailureKeepsOriginalEncodedProof`.

Python/Go differences:

- Pinned Source commit discards dirty id before checking requires_commit, correcting the older review that said id is in the body. New API maps only scalar-ID newly seeded closed JSON commits; no-attrs is no-op.
- Source unknown ordinary attrs are ignored; Go raw attributes must be valid UTF8 JSON before selection. Go map bulk alias conflicts use sorted keys, while sequential factories preserve order.
- Accepted HTTP evidence is separated from projected Resource/received Wire; Committed false never proves server execution did not occur. Live Barbican support is not established by Source emulation.
- Existing Container/Resource or dictionary seed synchronized/dirty/equal noPUT state, pending dirty/recommit and same-object behavior are not exposed by this scalar-ID API. Full Python kwargs duplicate binding/constructor and printf URI templates are not reproduced; empty/control/dot-segment ID commits are restricted.
- Requests response charset sniffing/full codecs, BOM-less bare utf16/utf32 native-endian choice and permissive nonfinite JSON remain outside the shared normalized subset; cp1006/unsupported charset retain ErrUnsupported with actual response proof.
- Independent Source audit executed 11 AST-extracted assertions for component dirty tracking, attribute consumption, commit/body preparation and list conversion; this was not Proxy/decoder/full-runtime or live-cloud execution. Full-runtime import was blocked by unavailable cached keystoneauth1.adapter, without dependency installation.

### `python:key_manager/v1/update_order`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordBasePath`

- String-ID Order commit removes id from dirty state; unknown-only/no attributes makes no HTTP or receipt, while known null/false attributes send one unwrapped PUT. Alias option and response member order use the last canonical value; meta request JSON stays raw and descriptor view is projected separately.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordNoop`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordDirtyAliasesAndOwnedReceipt`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordAliasPriority`.
- Final 100..399 statuses commit the actual response; invalid/empty JSON retains seed, valid nonobject or invalid HREF reports a response error with physical body/status. Owned request/seed/resource/wire/header state is independent; configured literal route/header/microversion remain scoped to the selected client.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordResponseDomain`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordAcceptedStatusAndFault`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordOptionsOwnInputsAndLiteralPath`.
- Preflight rejects unsafe dirty IDs, scalar-constructor collisions and selected-route mutation before HTTP. Bounded shared charset normalization decodes Latin1 into Resource/Wire while preserving original receipt/error bytes.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordPreflightAndClientGuard`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordCharsetKeepsPhysicalProof`.

Python/Go differences:

- The Go mapping deliberately accepts a newly seeded string ID plus closed finite JSON attributes, rather than mutable Python Resource/dictionary seeds, Python keyword binding or existing dirty/equality/recommit history. ID is not included in the PUT body; the previous remaining text incorrectly claimed it was.
- Dirty PUT IDs must be nonblank UTF-8 literal segments without controls/dot segments; custom BasePath is a literal service-relative path rather than a Python URI printf template. Source dynamic constructor/binding collisions are represented by concrete Go options and preflight errors.
- The explicit Go JSON profile preserves finite raw JSON number literals and supports UTF-8/Latin1 plus UTF-16/32 BOM or explicit endianness. Requests implicit charset guessing, other codec namespaces, BOM-less bare UTF-16/32 native-endian and permissive NaN/Infinity are outside this profile. Unknown attributes remain discarded; no caller builder is required.
- Go returns independent input/resource/wire/raw HTTP observations rather than mutating and returning the same Python object. This operation mapping does not claim live Barbican PUT support or full SDK-R1/C1/S1 parity.

### `python:key_manager/v1/update_secret`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordBasePath`

- secrets.API.Update sends PUT secrets/{id} with the raw payload as body and UpdateOpts.ContentType as the Content-Type header, accepts only 204 and turns 200 into an error.
  Tests: `keymanager/v1/secrets/python_parity_test.go:TestPythonSecretUpdateSendsRawPayload`.
- No-op ID-only and unknown/location/microversion/if_match-only inputs make no PUT or GET and retain only a seed Resource, no status/headers/Wire; ID itself is not dirty.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordNoopSeedHasNoReceiptOrPUT`.
- Known attrs are flat unwrapped JSON PUT, nullable values remain distinct, payload is JSON rather than raw upload, raw content_types maps to Source descriptor view, and seed/request/response do not alias.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordFlatDirtyBodyNullableAndRawValues`.
- Request aliases obey option order; response aliases obey JSON insertion order, null/id presence and passive secret_ref/secret_id projection.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordAliasesOrderResponseMergeAndReferenceProjection`.
- Empty or valid-UTF8 malformed JSON responses keep request seed like Source ValueError; valid nonobject or malformed HREF projection errors retain actual accepted receipt.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordEmptyInvalidJSONAndShapeErrorsKeepActualProof`.
- Options snapshot attrs/headers/basepath/version; literal URL escaping and private selectedversion retain sharedclient. Nil/context/source/constructor/unsafe controls fail before commit.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordOptionsSnapshotBasePathAndSelectedVersion`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordPreflightAndConstructorCollisions`.
- Original100..399 status policy, native400..999 proof, read/Close/customcancel/restoredsource terminal errors block more PUT/retry/auth/backoff. Clean native retry keeps bytes, alteredbody terminates before replay and failedreauth retains401original.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordAcceptedAndRejectedStatusPolicy`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordPhysicalFailuresNeverReplayAndRetainNativeProof`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordCleanRetryAndFailedReauthPreserveBody`.
- UTF16/UTF32 with a BOM or explicit LE/BE and ISO8859-1/Latin1 decode through shared responsejson.Normalize only for JSON Resource/Wire projection. Actual Envelope/Header/Status and ResponseError.Body retain original physical bytes, including normalized-shape errors. Unsupported cp1006/remaining namespace keeps ErrUnsupported+actual proof; no-op/dirty PUT semantics remain unchanged.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordExplicitCharsetsDecodeViewsKeepPhysicalEnvelope`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordDecodedShapeFailureKeepsOriginalEncodedProof`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordRequestsCharsetGapIsExplicit`.

Python/Go differences:

- secrets.API.Update is Barbican's payload upload; it returns only an error rather than a Secret.
- Pinned openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/Gophercloudv2.15.0/Go1.25.0. New UpdateRecord maps a string-ID seeded JSON commit. Existing API.Update remains rawpayload upload; no generated/native transport alone establishes named parity.
- Actual Source commit discards dirty id before requires_commit: scalar ID does not appear in the PUT body, correcting the earlier review. No dirtyattrs means no PUT/GET and no HTTP receipt.
- Concrete option factories own attrs/rawbytes/header maps/basepath/version pointers. Sequential attribute factories and response JSON preserve alias order; unordered Go input maps use sorted key order. Explicit Headers/Microversion are Go transport controls; Python attrs microversion does not override commit sessionversion.
- SDK-owned Seed/Request/Resource/Wire/Envelope/Header/StatusCode distinguish input from actual response. Committed means accepted body/Close observed; false never proves no remote execution, and a nonnil Resource on error can be a request seed.
- Literal service-relative BasePath and bounded nonblank escaped commit IDs replace arbitrary URI templates/falsey Source ID routes. Captured source/headers/body stay guarded through native hooks; returned HREFs never trigger metadata/payload reads.
- SDK-R1/C1/S1 Resource/class/session/cache foundations remain separate gaps. Actual Source61 observations use Proxy/Secret/Resource/Requests2.34.2/urllib32.8.0/CPython3.14.8 with adapterput returning memoryHTTPResponse. No liveBarbican/auth/retry/ACL proof. Worker no Go execution; root owns final checks.
- Cycle11 closes the declared BOM-or-explicit-LE/BE UTF16/UTF32 plus Latin1 response charset subset through the shared responsejson normalizer; actual wire byte receipts remain separate from normalized views. This does not complete the two named remaining areas or the full Requests loader/codec namespace. Existing standalone mains and Source61 oracle are unchanged; root owns Go validation.
- Python mutable Secret/Resource same-instance state, pending changes, dirty-state equality, MRO/constructor/kwargs binding and URI templates are replaced by the closed string-ID UpdateRecord API (deliberate design choice).
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Only standard finite JSON numbers are accepted: NaN, Infinity and overflowing literals such as 1e400 return ErrUnsupported with the original response, where Python's json loads them as floats, and a latest microversion must be written as "latest"; see Deliberate runtime differences in docs/sdk-support-ledger.md.

### `python:key_manager/v1/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Container`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Order`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Secret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForContainer`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForOrder`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithFailures`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Quota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ProjectQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForProjectQuota`

- Order WaitFor with WithUnlimitedWait and WithFailureStates("ERROR") repeats GET orders/{id}, returns on a case-insensitive target match, fails with resource.FailedStateError only for ERROR, keeps polling through other error-like states and reports progress 0 after each nonterminal observation.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForStatus`.
- WithStatusAttribute selects another string field and WithTimeout ends the wait with an error wrapping context.DeadlineExceeded.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForStatus`.
- Owned concrete Container/Order/Secret constructors snapshot synchronized Resource and physical evidence; cached lower-equivalent nullable/empty/string status returns without HTTP and inputs/results do not alias.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCachedStatesAndIndependentTargetSnapshot`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`.
- Shared Collection.Wait/WaitDeleted and prepared WaitPolicy supply native context/timer/progress mechanics. Source defaults are status-unbounded/delete120/interval2, explicit timeout zero/negative exhausts before polling, zero interval shortens the pause. Compound callback/cancel/source errors retain the last accepted receipt.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDefaultTimeoutsAndZeroIntervalPolicy`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCallbackFailureCancellationAndSourceAreTerminal`.
- Guarded common REST preserves rejected and accepted Read/Close, sticky source change and custom cancellation causes with actual status/body/header; native RetryFunc nil does not replay faulty physical responses. Clean503 retries retain owned selected version/headers and live token.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCleanNativeRetryPreservesSelectedPolicyAndLiveToken`.
- State normalization preserves Python Unicode lower, null and empty values. Source failure ERROR defaults and explicit empty failures disable detection; callback0 occurs only on nonterminal observations. Declared custom nullable string attributes are available.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNullableAndInvalidSelectedAttributesKeepProof`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitFailureDefaultsExplicitEmptyAndCustomAttribute`.
- Synchronized descriptor values persist on omission, incoming null/false/zero replaces prior values, malformed generic metadata leaves existing view and records actual response, and fixed RequestID does not follow returned id/HREF.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitSynchronizedSeedMergeAndGenericMalformedBody`.
- Secret polling follows prior Resource payload_content_type before current response content_types.default. Current payload_content_type alone does not choose the first payload. Complete metadata+payload success precedes status completion and binary/text proof is owned.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitPayloadSelectionUsesPreviousSynchronizedResource`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitMetadataDefaultPayloadAndStrictFailures`.
- Option/state preflight, clone isolation and captured source/header/version ownership prevent HTTP when the owned contract is invalid.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitOptionOwnershipAndPreflight`.
- Partial finite SecretStore GET polling added to existing Container/Order/Secret targets; actual raw/default/nullable projection, immutable literal identity, synchronized presence merge and Source bool/ref descriptors reuse existing guarded reader and prepared wait policy. Other known fetchable targets remain missing; no full named mapping claimed.
  Tests: `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitLiteralRouteRawProjectionAndDefaults`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitNativeCachedSnapshotKeepsExplicitIdentityAndNoFakeEnvelope`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitOrderedAliasesAndSynchronizedMerge`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitBooleanDescriptorUsesSourceFloatTruthiness`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDescriptorAndShapeErrorsKeepActualPage`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDeleteCleanAbsenceAndDisabledBackendLimitation`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitPreflightFailureAndCallbackPolicies`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCompoundPhysicalFaultsNeverRetryOrReauthenticate`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCleanRetryReauthOwnHeadersAndLiveToken`.
- Partial concrete SecretACL and effective Quota target constructors own stock parent/singleton routes and do not append response IDs/HREFs or the internal current-project selector. Literal defaults have no fabricated HTTP status, byte Envelope or Wire; nullable/untyped declared values and actual envelope/object/header/status are independently owned.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitOwnsParentRouteAndSourceDictDefaults`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitOwnsNoIDRouteAndEnvelopeRawDescriptors`.
- ForSecretACL snapshots actual Metadata.Body or native Read/ACLRef when Body is absent, retains immutable native SecretID, and does not fabricate Wire/Envelope. ForQuota requires original Body. Cached normalized string/null matches return before timers/HTTP and constructor/result snapshots do not alias. SecretACL URI secret_id is available for declared getattr selection without inserting it into to_dict Body.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitSnapshotsCachedNativeAndURIWithoutFakeEnvelope`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitCachedSnapshotAndNullableDefaults`.
- SecretACL read descriptor preserves objects/None and converts other JSON-domain values to empty dict exactly as pinned fields._convert_type(dict); literal Wire remains raw. Incoming declared null/false/zero/object/large-integer presence overlays synchronized values; omission and generic malformed/empty JSON retain previous descriptors. Quota unwraps quotas when present and uses root-object fallback otherwise.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitRawPresenceMergeAndReadDictCoercion`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitMergesFlatFallbackAndNullPresence`.
- SecretACL and ordinary Quota do not declare status; default status wait is unsupported before HTTP rather than invented null/active. Declared selected attributes must be string or null: cached non-string rejects before HTTP, fetched non-string/nonobject/envelope-invalid/UTF8-invalid retains actual proof. Numeric quota values are not converted to status strings.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitMissingStatusAndTypedAttributePreflight`, `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaWaitNonobjectAndSelectedValueErrorsKeepActualProof`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitEnvelopeShapeAndNumericAttributesAreExplicitFailures`.
- Status-free deletion fetches GET, ignores undeclared wire status:deleted, calls progress0 only after nonterminal accepted observations, and completes only on clean GET404 while returning the original Resource snapshot plus actual absence receipt. Status GET404 remains an error. No DELETE or configuration-deletion acknowledgement is synthesized.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaDeleteSkipsUndeclaredStatusAndRetainsOriginalAt404`.
- Both finite targets reuse reader captured source/header/version and common prepared WaitPolicy. Twenty combined physical fault lanes retain actual method/URL/status/body/header and Read/Close/sticky-source/custom-cancel/context causes without retry/reauth. Native 401 pointer ErrUnableToReauthenticate exposes actual response via ErrOriginal and physical causes via ErrReauth; 404 physical faults do not prove deletion.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaPhysicalFailuresNeverBecomeAbsenceOrRetry`.
- Clean 401 renewal and 503 retry remain native for both finite targets, preserve fixed stock routes and selected version/header policy, observe refreshed token and leave original client policy unmodified.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaCleanRetryReauthPreservesStockRouteAndOwnedHeaders`.
- Concrete admin ProjectQuota target fetches only the original fixed escaped project route with project_quotas envelope/root fallback and seven nullable untyped Body descriptors. Raw extensions and actual receipt remain separate; no status is invented and cached/default matching nullable strings return without polling.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitFixedRouteRawDescriptorsAndDefaults`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitCachedRecordOwnsSnapshotsAndNoStatus`.
- ProjectQuota metadata merges prior fields through generic malformed JSON and explicit-null observations, while invalid shapes/numeric selected attributes retain response errors. Deletion polls GET and completes only on clean404, ignoring undeclared status:deleted; accepted/rejected Close faults cannot become absence or replay.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitMergeIgnoresMalformedJSONAndPreservesNull`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitDeleteOnlyCleanGET404`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitShapeNumericAndPhysicalFailures`.

Python/Go differences:

- Legacy generated collection waits retain their historical five-minute policy. The concrete waits.API lane defaults status to unlimited/ERROR and deletion120s with interval2s, matching the pinned helper defaults.
- Legacy resource.WithPollInterval requires positive durations; concrete waits.WithInterval(0) uses Source100ms/min-positive-timeout coercion.
- Timeouts and failures are Python ResourceTimeout and ResourceFailure; Go returns context deadline errors and resource.FailedStateError.
- Finite SDK-owned concrete Target constructors with immutable routes and independent record snapshots replace Python arbitrary Resource subclasses, same-object mutation and dynamic getattr. All known pinned fetchable concrete targets now include admin ProjectQuota; arbitrary class/subtyping remains SDK-R1 as a deliberate finite Go API boundary.
- Source normalized nullable/empty/string states use PythonLower and private exact comparison tokens, not broad Unicode EqualFold. Go failures accept strings only, callbacks are typed func(int)error, declared attributes must be nullable strings.
- The SDK validates source/headers and applied option callbacks before cached-state success; Source returns its cached object before consulting session or many invalid timer/failure settings. Negative interval is a pre-HTTP Go guard rather than Python sleep-time ValueError after an observation.
- Status timeout defaults to unlimited, deletion timeout120s, interval2s. Nil Timeout is omission; explicit Unlimited removes the deletion SDK limit. Explicit zero/negative timeout fails before first GET unless a cached status already matches. Zero interval uses Source 100ms/min-positive-timeout pause.
- Fixed original route, selected microversion or absence and reserved headers survive native attempts. Python mutable response id can change subsequent fetch routing; Go preserves RequestID and leaves response id/HREF passive. Live auth token and ordinary native retry headers remain native.
- Source custom Secret.fetch does not raise HTTP errors itself; Go admits100..399 and rejects400..599 strictly with full physical causes, so payload404 cannot establish metadata absence. Accepted UTF-8 text is decoded strictly, opaque bytes remain Payload plus JSON base64 Resource values.
- Container/Order generic malformed/empty JSON preserves synchronized attributes as Source ValueError handling does; valid JSON null/nonobject fails. Secret malformed JSON fails. Go requires valid UTF-8 metadata and eagerly projects finite declared casts/aliases using the existing shared projector.
- Source projectors provide nullable descriptor defaults and locationnull; canonical wire aliases win deterministically. Unknown metadata is retained only in Wire. No Connection location/session HTTP cache, dirty-resource commit or arbitrary Python runtime value parity is claimed.
- Clean metadata404 deletion returns an owned original Resource view plus the new404 proof channels rather than Source object identity. Fetched deleted returns the updated view. Errors retain last actual evidence but return Resource=nil.
- Fixed literal/request selector routing never follows response HREF/native convenience ID; Source mutable id routing is not reproduced.
- Cached native Metadata.Body has no original object key order or byte envelope: sorted-key constructor aliases and no fabricated Envelope.
- Computed location ownednull and strict UTF8 finite JSON/declared attributes remain existing SDK-R1/C1 policies.
- Pinned generic Resource.fetch requires_id=True overrides SecretACL.requires_id=False and does not carry the public getter requires_id=False option. Generic waiting on a getter ACL/Quota with id=None fails before polling, or appends an artificial ID if set. Concrete Go targets deliberately correct this pinned request preparation defect by owning stock /secrets/{secretID}/acl and /quotas routes.
- SecretACL URI secret_id is selected from immutable request scope but is not added to Resource Body/to_dict; current-project is an internal wait selector and is never a wire project/member ID or synthesized Resource.id.
- No status descriptor is invented for ACL/Quota: default status waits fail preflight; deletion skips absent status and waits for clean GET404. Stock ACL deletion restores implicit default ACL GET200 and project override deletion leaves effective quota GET200; GET absence observation does not prove settings deletion.
- ACL wait read projection follows Source getter dict coercion while existing native secretacls API uses its independent stock object validation. Strict UTF8, declared string/null wait attributes and owned-null computed location remain bounded SDK policies.
- Admin ProjectQuota is a member route unlike effective Quota; its original project ID is seeded and immutable while response IDs remain passive. Like Quota/ACL it has no status descriptor and stock override reset can leave GET200, so absence waiting is not proof of settings deletion.
- This mapping covers finite SDK-owned concrete target constructors rather than arbitrary Python Resource/runtime subclass/cache/location behavior. SDK-R1/C1/S1 and actual deployment/permission/release validation remain explicit whole-SDK gaps.

### `python:key_manager/v1/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Container`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Order`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Secret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForContainer`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForOrder`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecret`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithInterval`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretStore`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.SecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForSecretACL`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.Quota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ProjectQuota`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits.ForProjectQuota`

- Order WaitForDeletion with WithTimeout(120s) repeats GET orders/{id} until 404 and then succeeds.
  Tests: `keymanager/v1/orders/python_parity_test.go:TestPythonOrderWaitForDelete`.
- Owned concrete Container/Order/Secret constructors snapshot synchronized Resource and physical evidence; cached lower-equivalent nullable/empty/string status returns without HTTP and inputs/results do not alias.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCachedStatesAndIndependentTargetSnapshot`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold`.
- Shared Collection.Wait/WaitDeleted and prepared WaitPolicy supply native context/timer/progress mechanics. Source defaults are status-unbounded/delete120/interval2, explicit timeout zero/negative exhausts before polling, zero interval shortens the pause. Compound callback/cancel/source errors retain the last accepted receipt.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDefaultTimeoutsAndZeroIntervalPolicy`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCallbackFailureCancellationAndSourceAreTerminal`.
- Guarded common REST preserves rejected and accepted Read/Close, sticky source change and custom cancellation causes with actual status/body/header; native RetryFunc nil does not replay faulty physical responses. Clean503 retries retain owned selected version/headers and live token.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCleanNativeRetryPreservesSelectedPolicyAndLiveToken`.
- Deletion always performs the first metadata GET even when cached deleted. A clean metadata404 returns original Resource with actual404 evidence, fetched deleted returns fetched Resource; null/nonstring/forbidden/server errors are terminal.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitDeletionOriginalAndFetchedReturnContracts`.
- Secret payload404 is a terminal marked payload failure rather than absence of the Secret. Metadata and payload native proof stay separate, and ACTIVE metadata cannot complete status wait when payload failed.
  Tests: `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerSecretWaitMetadataDefaultPayloadAndStrictFailures`, `keymanager/v1/waits/python_wait_contracts_test.go:TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay`.
- Partial finite SecretStore GET polling added to existing Container/Order/Secret targets; actual raw/default/nullable projection, immutable literal identity, synchronized presence merge and Source bool/ref descriptors reuse existing guarded reader and prepared wait policy. Other known fetchable targets remain missing; no full named mapping claimed.
  Tests: `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitLiteralRouteRawProjectionAndDefaults`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitNativeCachedSnapshotKeepsExplicitIdentityAndNoFakeEnvelope`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitOrderedAliasesAndSynchronizedMerge`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitBooleanDescriptorUsesSourceFloatTruthiness`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDescriptorAndShapeErrorsKeepActualPage`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitDeleteCleanAbsenceAndDisabledBackendLimitation`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitPreflightFailureAndCallbackPolicies`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCompoundPhysicalFaultsNeverRetryOrReauthenticate`, `keymanager/v1/waits/python_secret_store_contracts_test.go:TestPythonKeymanagerSecretStoreWaitCleanRetryReauthOwnHeadersAndLiveToken`.
- Partial concrete SecretACL and effective Quota target constructors own stock parent/singleton routes and do not append response IDs/HREFs or the internal current-project selector. Literal defaults have no fabricated HTTP status, byte Envelope or Wire; nullable/untyped declared values and actual envelope/object/header/status are independently owned.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitOwnsParentRouteAndSourceDictDefaults`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitOwnsNoIDRouteAndEnvelopeRawDescriptors`.
- ForSecretACL snapshots actual Metadata.Body or native Read/ACLRef when Body is absent, retains immutable native SecretID, and does not fabricate Wire/Envelope. ForQuota requires original Body. Cached normalized string/null matches return before timers/HTTP and constructor/result snapshots do not alias. SecretACL URI secret_id is available for declared getattr selection without inserting it into to_dict Body.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitSnapshotsCachedNativeAndURIWithoutFakeEnvelope`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitCachedSnapshotAndNullableDefaults`.
- SecretACL read descriptor preserves objects/None and converts other JSON-domain values to empty dict exactly as pinned fields._convert_type(dict); literal Wire remains raw. Incoming declared null/false/zero/object/large-integer presence overlays synchronized values; omission and generic malformed/empty JSON retain previous descriptors. Quota unwraps quotas when present and uses root-object fallback otherwise.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitRawPresenceMergeAndReadDictCoercion`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitMergesFlatFallbackAndNullPresence`.
- SecretACL and ordinary Quota do not declare status; default status wait is unsupported before HTTP rather than invented null/active. Declared selected attributes must be string or null: cached non-string rejects before HTTP, fetched non-string/nonobject/envelope-invalid/UTF8-invalid retains actual proof. Numeric quota values are not converted to status strings.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLWaitMissingStatusAndTypedAttributePreflight`, `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaWaitNonobjectAndSelectedValueErrorsKeepActualProof`, `keymanager/v1/waits/python_quota_contracts_test.go:TestPythonKeymanagerQuotaWaitEnvelopeShapeAndNumericAttributesAreExplicitFailures`.
- Status-free deletion fetches GET, ignores undeclared wire status:deleted, calls progress0 only after nonterminal accepted observations, and completes only on clean GET404 while returning the original Resource snapshot plus actual absence receipt. Status GET404 remains an error. No DELETE or configuration-deletion acknowledgement is synthesized.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaDeleteSkipsUndeclaredStatusAndRetainsOriginalAt404`.
- Both finite targets reuse reader captured source/header/version and common prepared WaitPolicy. Twenty combined physical fault lanes retain actual method/URL/status/body/header and Read/Close/sticky-source/custom-cancel/context causes without retry/reauth. Native 401 pointer ErrUnableToReauthenticate exposes actual response via ErrOriginal and physical causes via ErrReauth; 404 physical faults do not prove deletion.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaPhysicalFailuresNeverBecomeAbsenceOrRetry`.
- Clean 401 renewal and 503 retry remain native for both finite targets, preserve fixed stock routes and selected version/header policy, observe refreshed token and leave original client policy unmodified.
  Tests: `keymanager/v1/waits/python_acl_contracts_test.go:TestPythonKeymanagerACLQuotaCleanRetryReauthPreservesStockRouteAndOwnedHeaders`.
- Concrete admin ProjectQuota target fetches only the original fixed escaped project route with project_quotas envelope/root fallback and seven nullable untyped Body descriptors. Raw extensions and actual receipt remain separate; no status is invented and cached/default matching nullable strings return without polling.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitFixedRouteRawDescriptorsAndDefaults`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitCachedRecordOwnsSnapshotsAndNoStatus`.
- ProjectQuota metadata merges prior fields through generic malformed JSON and explicit-null observations, while invalid shapes/numeric selected attributes retain response errors. Deletion polls GET and completes only on clean404, ignoring undeclared status:deleted; accepted/rejected Close faults cannot become absence or replay.
  Tests: `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitMergeIgnoresMalformedJSONAndPreservesNull`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitDeleteOnlyCleanGET404`, `keymanager/v1/waits/python_project_quota_contracts_test.go:TestPythonProjectQuotaWaitShapeNumericAndPhysicalFailures`.

Python/Go differences:

- Legacy generated collection waits retain their historical five-minute policy. The concrete waits.API lane defaults status to unlimited/ERROR and deletion120s with interval2s, matching the pinned helper defaults.
- Finite SDK-owned concrete Target constructors with immutable routes and independent record snapshots replace Python arbitrary Resource subclasses, same-object mutation and dynamic getattr. All known pinned fetchable concrete targets now include admin ProjectQuota; arbitrary class/subtyping remains SDK-R1 as a deliberate finite Go API boundary.
- Source normalized nullable/empty/string states use PythonLower and private exact comparison tokens, not broad Unicode EqualFold. Go failures accept strings only, callbacks are typed func(int)error, declared attributes must be nullable strings.
- The SDK validates source/headers and applied option callbacks before cached-state success; Source returns its cached object before consulting session or many invalid timer/failure settings. Negative interval is a pre-HTTP Go guard rather than Python sleep-time ValueError after an observation.
- Status timeout defaults to unlimited, deletion timeout120s, interval2s. Nil Timeout is omission; explicit Unlimited removes the deletion SDK limit. Explicit zero/negative timeout fails before first GET unless a cached status already matches. Zero interval uses Source 100ms/min-positive-timeout pause.
- Fixed original route, selected microversion or absence and reserved headers survive native attempts. Python mutable response id can change subsequent fetch routing; Go preserves RequestID and leaves response id/HREF passive. Live auth token and ordinary native retry headers remain native.
- Source custom Secret.fetch does not raise HTTP errors itself; Go admits100..399 and rejects400..599 strictly with full physical causes, so payload404 cannot establish metadata absence. Accepted UTF-8 text is decoded strictly, opaque bytes remain Payload plus JSON base64 Resource values.
- Container/Order generic malformed/empty JSON preserves synchronized attributes as Source ValueError handling does; valid JSON null/nonobject fails. Secret malformed JSON fails. Go requires valid UTF-8 metadata and eagerly projects finite declared casts/aliases using the existing shared projector.
- Source projectors provide nullable descriptor defaults and locationnull; canonical wire aliases win deterministically. Unknown metadata is retained only in Wire. No Connection location/session HTTP cache, dirty-resource commit or arbitrary Python runtime value parity is claimed.
- Clean metadata404 deletion returns an owned original Resource view plus the new404 proof channels rather than Source object identity. Fetched deleted returns the updated view. Errors retain last actual evidence but return Resource=nil.
- Fixed literal/request selector routing never follows response HREF/native convenience ID; Source mutable id routing is not reproduced.
- Cached native Metadata.Body has no original object key order or byte envelope: sorted-key constructor aliases and no fabricated Envelope.
- Computed location ownednull and strict UTF8 finite JSON/declared attributes remain existing SDK-R1/C1 policies.
- Pinned generic Resource.fetch requires_id=True overrides SecretACL.requires_id=False and does not carry the public getter requires_id=False option. Generic waiting on a getter ACL/Quota with id=None fails before polling, or appends an artificial ID if set. Concrete Go targets deliberately correct this pinned request preparation defect by owning stock /secrets/{secretID}/acl and /quotas routes.
- SecretACL URI secret_id is selected from immutable request scope but is not added to Resource Body/to_dict; current-project is an internal wait selector and is never a wire project/member ID or synthesized Resource.id.
- No status descriptor is invented for ACL/Quota: default status waits fail preflight; deletion skips absent status and waits for clean GET404. Stock ACL deletion restores implicit default ACL GET200 and project override deletion leaves effective quota GET200; GET absence observation does not prove settings deletion.
- ACL wait read projection follows Source getter dict coercion while existing native secretacls API uses its independent stock object validation. Strict UTF8, declared string/null wait attributes and owned-null computed location remain bounded SDK policies.
- Admin ProjectQuota is a member route unlike effective Quota; its original project ID is seeded and immutable while response IDs remain passive. Like Quota/ACL it has no status descriptor and stock override reset can leave GET200, so absence waiting is not proof of settings deletion.
- This mapping covers finite SDK-owned concrete target constructors rather than arbitrary Python Resource/runtime subclass/cache/location behavior. SDK-R1/C1/S1 and actual deployment/permission/release validation remain explicit whole-SDK gaps.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
err := secrets.New(client).Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"})
```

```go
order, err := orders.New(client).WaitFor(ctx, resource.ID("o1"), "ACTIVE",
	resource.WithUnlimitedWait(),          // Python wait=None
	resource.WithFailureStates("ERROR"),  // Python failures=None
)
err = orders.New(client).WaitForDeletion(ctx, resource.ID("o1"),
	resource.WithTimeout(120*time.Second),
)
```

## References

- [key manager v1 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py)
- [Metadata Fetch](metadata-fetch.md)
- [Metadata Create](metadata-create.md)
- [Metadata Delete](metadata-delete.md)
- [`update_container`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L142)
- [Python Update Record](containers/python-update-record.md)
- [`update_order`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L243)
- [Python Update Record](orders/python-update-record.md)
- [`update_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L345)
- [Python Update Record](secrets/python-update-record.md)
- [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615)
- [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L654)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [resource.wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [resource.wait_for_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2670)
- [Container](containers/python-update-record.md)
- [Order](orders/python-update-record.md)
