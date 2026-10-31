# Image Record Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/deactivate_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DeactivateImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.DeactivateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeactivateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionHeaders`

- Legacy typed strict204 profile only: DeactivateImage defaults to bodyless/query-free POST on the dedicated fixed-ID endpoint; only actual204 acknowledges the request. Direct ID causes no metadata GET/list/PATCH/discovery/wait. Missing image or tag is a strict error; there is no IgnoreMissing option or nested404 suppression.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: ID is validated and routed directly; explicit Name resolves exact literal matches across existing native pages once, including UUID-shaped names, before a fixed chosen-ID mutation. Absent/ambiguous/late surfaced native lookup failures or malformed resolved IDs send no mutation. Native Name LIST model/body ownership/continuation behavior is retained.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- Legacy typed strict204 profile only: Context/source/provider/type/origin/UTF8 base/source headers/version/Ref and literal tag preflight runs before caller callbacks or HTTP. Nil callbacks and callback/source/context failures preserve original causes. Ordinary option headers are the only extension; protected auth/framing/version/representation fields and conflicting supplied-map aliases fail locally.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Legacy typed strict204 profile only: WithImageMutationOpts replaces the full concrete header policy. Header helpers snapshot input maps at creation; option slice and each callback config/map are copied before and after one invocation. Retained handles, later callbacks and parallel helper reuse cannot change prepared request policy.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Legacy typed strict204 profile only: Captured original client/provider/prefix and ordinary headers are used for Name LIST and mutation; the original provider latest auth token stays live. Safe source base/header changes retain captured requests, while provider replacement or foreign source retarget rejects before mutation.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`, `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Only actual original204 creates an acknowledgement with fixed request ImageID and literal Tag or compiled Action, owned opaque Body bytes, cloned Header and actual StatusCode. Empty/nonJSON/invalidUTF8 response bytes are allowed and never decode metadata, retarget from Location/body IDs, or establish current tags/status.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Accepted204 read/Close/context failures return retained partial/full acknowledgement plus ResponseError. Result/error Body/Header copies are independent; original read/Close/ctx.Err/custom cause survives. Accepted body closes exactly once without accepted replay. Native rejected/transport/prebody failures and unexpected200/201/202/206/404 keep nil result and their native evidence.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Configured native prebody retry/reauth/backoff preserves fixed method/origin/path/query and live auth, and may attempt the mutation again before acceptance. Body ownership tampering, including nil-to-JSONnull, RawBody, KeepResponseBody or JSONResponse, fails before the next attempt with original/hook/encoding causes; expanded OkCodes cannot bypass original204. Native ErrUnableToReauthenticate ErrOriginal/ErrReauth remain inspectable through fields. Changed-target/method redirects are guarded; identical-target configured redirect policy remains. Prior native APIs are preserved.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`.
- Owned action selects one literal image ID or an SDK-produced ImageRecord's private raw current canonical id and snapshots it before location/options callbacks. Public Resource id, Wire, Envelope, Header, passive URLs and properties cannot retarget POST. Ambiguous or missing selectors, handcrafted Record without private state, null/non-string/malformed raw identity, invalidUTF8/unpaired escaped surrogates, control/dot/blank identities and invalid context/source/ordinary headers fail before HTTP. Accepted UTF8 literal identity is escaped once as a single segment; no image Name lookup or automatic metadata GET is introduced.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`, `image/image_record_actions_test.go:TestImageRecordActionsStrictRawUnicodeIdentity`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`, `image/image_record_actions_test.go:TestImageRecordActionsSnapshotInputsOptionsHeadersAndLocation`, `image/image_record_delete_test.go:TestDeleteImageRecordCompletePreflightRejectsAmbiguousOrUnsafeSelectors`.
- Owned action follows the pinned _get_resource(existing)._update({}) local preparation in its finite immutable profile: full65-field descriptor projection runs before POST using private raw current and this call's current location, including descriptor conversion failures. Plain ImportMethods resets to a nonnil empty list. The independent prepared Record preserves private original/current/dirty Body state, prior fetched Wire/Envelope/Header/StatusCode and passive timestamps/Links. Literal construction has no fabricated fetch receipt, and projection does not clean pending Body.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsProjectPrivateCurrentBodyAndPreserveFetchReceipts`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`.
- Owned compiled deactivate/reactivate sends fixed bodyless/query-free POST /images/{id}/actions/{action}; it adds no arbitrary action, status gate, GET/list/discovery/wait or cleanup. Actual200..399 bytes are opaque even for empty/null/array/invalid JSON/invalidUTF8. Acknowledgement records fixed chosen ImageID/Action and independent actual Body/Header/StatusCode without synthesizing a server image status.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`.
- Owned successful action returns the prepared independent Record without translating action response Body or headers. Response id/status/location and import-method headers remain acknowledgement evidence only; local image status, prior fetch receipt, raw pending state and the preparation's empty ImportMethods are unchanged. A subsequent explicit owned Update can still commit pending Body, and caller/result channels do not alias their input or another acknowledgement.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsProjectPrivateCurrentBodyAndPreserveFetchReceipts`, `image/image_record_actions_test.go:TestImageRecordActionsKeepPendingRawChangesForExplicitCommit`, `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`.
- Owned concrete action FullOpts snapshots and replaces the full ordinary-header configuration; Header/Headers merge with their captured inputs. The option slice and each callback config/map are copied before and after one invocation, retaining custom option/context causes and preventing retained config or location callbacks from changing captured identity/options/source. Guard checks around callbacks stop later restoring callbacks after an observed context/source/binding/outer failure. Protected transport/auth/representation/version headers and nil callbacks fail before HTTP.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsSnapshotInputsOptionsHeadersAndLocation`, `image/image_record_actions_test.go:TestImageRecordActionsStopAfterGuardFailureInOptions`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`.
- Owned accepted Read/Close/context/source/outer-guard failures return only a partial actual acknowledgement plus cause-safe ResponseError and Record=nil. Acknowledgement/error bytes and headers are independent, physical body closes once and accepted processing failures do not automatically replay POST. Nested404 causes on accepted processing are not classified as missing and never suppress partial evidence.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsAcceptedPhysicalFailuresReturnOnlyPartialAcknowledgement`.
- Owned Go action deliberately rejects actual400..599 with nil result and inspectable native status/body/header proof, and never ignores404. Transport failures without an accepted response retain their original causes. This is explicit Go error policy: the pinned Python Image._action calls only session.post, the public proxy discards its Response, and Proxy.request defaults raise_exc=False, so an ordinary HTTP400..599 Response returned by the default successful-transport chain can yield None rather than a Source status exception.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsNativeRejectionsIncludeMissingAndNeverAcknowledge`, `image/image_record_actions_test.go:TestImageRecordActionsNativeRetryPreservesFixedScopeLiveAuthAndOwnership`.
- Owned native prebody retry preserves the compiled method/fixed scope/empty body and original provider live auth while retaining ordinary retry headers and hook causes. Changed owned body/response representation is rejected before another attempt and hook-expanded OkCodes cannot fabricate acceptance outside200..399. Connection/Service delegate through the existing shared Image/ImageV2 source, current location and operation labels, with independent Record/fetch/ack receipts and cause-safe invalid invocation. No arbitrary Python session/cache/hook runtime equivalence is inferred.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsNativeRetryPreservesFixedScopeLiveAuthAndOwnership`, `connectiontest/image_record_actions_integration_test.go:TestConnectionOwnedImageActionsPreserveRecordAndSeparateReceipts`, `connectiontest/image_record_actions_integration_test.go:TestConnectionOwnedImageActionsRejectInvalidInvocation`.

Python/Go differences:

- Principal action status difference: pinned Image._action is session.post(url) without raise_from_response or _translate_response, and both public wrappers discard the returned Response. The complete default Proxy.request chain forwards raise_exc=False; ordinary HTTP400..599 Responses can therefore produce None on successful transport, unlike Go's explicit native status errors and strict404 rejection. Owned Go accepts actual200..399 and returns an inspectable opaque acknowledgement. This is deliberately go_mapping, not supported or identical Source accepted/error-status behavior; configured session/transport exceptions remain separate.
- Legacy typed strict204 profile only: Pinned Proxy methods accept direct ID strings or mutable Image Resource objects and return None. Go uses explicit ID/Name Ref and concrete owned headers, returns actual acknowledgement evidence, and introduces exact Name lookup as a typed convenience.
- Legacy typed strict204 profile only: Go intentionally accepts actual204 only, owns accepted body bytes/headers and preserves result+ResponseError on read/Close/context failures. Complete Python response translation, adapter/session exceptions, request preparation and arbitrary hooks remain broader.
- Legacy typed strict204 profile only: Go snapshots captured route/ordinary headers/options with live original auth and enforces bounded route/source/request-body guards. Existing native Name continuation/body behavior and configured same-target redirect/prebody retry policies remain unchanged; no single wire-attempt promise under configured retries.
- Legacy typed strict204 profile only: Pinned Image._action calls session.post and neither decodes a response nor applies raise_from_response explicitly; adapter exceptions remain native to the Python session. Go deliberately requires actual204 and returns owned acknowledgement evidence without updating a local status cache.
- Owned immutable JSON-domain ImageRecord uses SDK private canonical current state or literal ID, materializes the complete65 view/current location and resets ImportMethods before the compiled action. Python may reuse and mutate the same Resource/Munch/subclass and dict backing state. Arbitrary constructor/descriptor/default/history/dirty Header/base_path/requires_id lifecycle remains SDK-R1, Adapter cache invalidation remains SDK-C1, and dynamic session/microversion/class/exception/metrics/transport policy remains SDK-S1 in docs/implementation-plan.md; these global requirements are not completed by this mapping.
- Python utils.urljoin applies falsy replacement, object stringification and edge-slash trimming; its public action accepts no per-call attrs/options and _action supplies no explicit body/headers/microversion kwargs or _prepare_request negotiation. Go bounds identities to strict validUTF8 JSON string/literal segments, rejects lone-surrogate retargeting and escapes once, while captured source and concrete ordinary-header options are explicit transport extensions. Arbitrary Python objects, mutable routing and parser/coercion behavior are not runtime-equivalent.
- Neither pinned action translates response data into the Resource. Go returns a prepared independent Record with raw pending/original state and old fetch receipts intact, ImportMethods still empty, plus a separate actual acknowledgement; response import-method/status/id/location data never refreshes that Record. Accepted Read/Close/context/source failures return partial acknowledgement with Record=nil, owned causes/bytes/headers and no automatic accepted replay. These ownership/source-guard policies differ from Python's discarded Response and are not server completion or rollback guarantees.
- Pinned deactivate/reactivate defaults are project-scoped ADMIN_OR_PROJECT_MEMBER and server-owned active/deactivated transitions, authorization, deployment policy and idempotence. Go adds no role/status preflight, IgnoreMissing, metadata GET, name lookup, wait, cache update or cleanup. No authenticated cloud, Python SDK runtime or backend mutation was executed by this audit. Public create_image, property helper, upload/stage/import/download and their separate mandatory Source graphs are not closed by these two action mappings.

### `python:image/v2/reactivate_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ReactivateImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ReactivateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ReactivateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordActionHeaders`

- Legacy typed strict204 profile only: ReactivateImage defaults to bodyless/query-free POST on the dedicated fixed-ID endpoint; only actual204 acknowledges the request. Direct ID causes no metadata GET/list/PATCH/discovery/wait. Missing image or tag is a strict error; there is no IgnoreMissing option or nested404 suppression.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: ID is validated and routed directly; explicit Name resolves exact literal matches across existing native pages once, including UUID-shaped names, before a fixed chosen-ID mutation. Absent/ambiguous/late surfaced native lookup failures or malformed resolved IDs send no mutation. Native Name LIST model/body ownership/continuation behavior is retained.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- Legacy typed strict204 profile only: Context/source/provider/type/origin/UTF8 base/source headers/version/Ref and literal tag preflight runs before caller callbacks or HTTP. Nil callbacks and callback/source/context failures preserve original causes. Ordinary option headers are the only extension; protected auth/framing/version/representation fields and conflicting supplied-map aliases fail locally.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Legacy typed strict204 profile only: WithImageMutationOpts replaces the full concrete header policy. Header helpers snapshot input maps at creation; option slice and each callback config/map are copied before and after one invocation. Retained handles, later callbacks and parallel helper reuse cannot change prepared request policy.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Legacy typed strict204 profile only: Captured original client/provider/prefix and ordinary headers are used for Name LIST and mutation; the original provider latest auth token stays live. Safe source base/header changes retain captured requests, while provider replacement or foreign source retarget rejects before mutation.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`, `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Only actual original204 creates an acknowledgement with fixed request ImageID and literal Tag or compiled Action, owned opaque Body bytes, cloned Header and actual StatusCode. Empty/nonJSON/invalidUTF8 response bytes are allowed and never decode metadata, retarget from Location/body IDs, or establish current tags/status.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Accepted204 read/Close/context failures return retained partial/full acknowledgement plus ResponseError. Result/error Body/Header copies are independent; original read/Close/ctx.Err/custom cause survives. Accepted body closes exactly once without accepted replay. Native rejected/transport/prebody failures and unexpected200/201/202/206/404 keep nil result and their native evidence.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Legacy typed strict204 profile only: Configured native prebody retry/reauth/backoff preserves fixed method/origin/path/query and live auth, and may attempt the mutation again before acceptance. Body ownership tampering, including nil-to-JSONnull, RawBody, KeepResponseBody or JSONResponse, fails before the next attempt with original/hook/encoding causes; expanded OkCodes cannot bypass original204. Native ErrUnableToReauthenticate ErrOriginal/ErrReauth remain inspectable through fields. Changed-target/method redirects are guarded; identical-target configured redirect policy remains. Prior native APIs are preserved.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`.
- Owned action selects one literal image ID or an SDK-produced ImageRecord's private raw current canonical id and snapshots it before location/options callbacks. Public Resource id, Wire, Envelope, Header, passive URLs and properties cannot retarget POST. Ambiguous or missing selectors, handcrafted Record without private state, null/non-string/malformed raw identity, invalidUTF8/unpaired escaped surrogates, control/dot/blank identities and invalid context/source/ordinary headers fail before HTTP. Accepted UTF8 literal identity is escaped once as a single segment; no image Name lookup or automatic metadata GET is introduced.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`, `image/image_record_actions_test.go:TestImageRecordActionsStrictRawUnicodeIdentity`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`, `image/image_record_actions_test.go:TestImageRecordActionsSnapshotInputsOptionsHeadersAndLocation`, `image/image_record_delete_test.go:TestDeleteImageRecordCompletePreflightRejectsAmbiguousOrUnsafeSelectors`.
- Owned action follows the pinned _get_resource(existing)._update({}) local preparation in its finite immutable profile: full65-field descriptor projection runs before POST using private raw current and this call's current location, including descriptor conversion failures. Plain ImportMethods resets to a nonnil empty list. The independent prepared Record preserves private original/current/dirty Body state, prior fetched Wire/Envelope/Header/StatusCode and passive timestamps/Links. Literal construction has no fabricated fetch receipt, and projection does not clean pending Body.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsProjectPrivateCurrentBodyAndPreserveFetchReceipts`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`.
- Owned compiled deactivate/reactivate sends fixed bodyless/query-free POST /images/{id}/actions/{action}; it adds no arbitrary action, status gate, GET/list/discovery/wait or cleanup. Actual200..399 bytes are opaque even for empty/null/array/invalid JSON/invalidUTF8. Acknowledgement records fixed chosen ImageID/Action and independent actual Body/Header/StatusCode without synthesizing a server image status.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`.
- Owned successful action returns the prepared independent Record without translating action response Body or headers. Response id/status/location and import-method headers remain acknowledgement evidence only; local image status, prior fetch receipt, raw pending state and the preparation's empty ImportMethods are unchanged. A subsequent explicit owned Update can still commit pending Body, and caller/result channels do not alias their input or another acknowledgement.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsProjectPrivateCurrentBodyAndPreserveFetchReceipts`, `image/image_record_actions_test.go:TestImageRecordActionsKeepPendingRawChangesForExplicitCommit`, `image/image_record_actions_test.go:TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses`.
- Owned concrete action FullOpts snapshots and replaces the full ordinary-header configuration; Header/Headers merge with their captured inputs. The option slice and each callback config/map are copied before and after one invocation, retaining custom option/context causes and preventing retained config or location callbacks from changing captured identity/options/source. Guard checks around callbacks stop later restoring callbacks after an observed context/source/binding/outer failure. Protected transport/auth/representation/version headers and nil callbacks fail before HTTP.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsSnapshotInputsOptionsHeadersAndLocation`, `image/image_record_actions_test.go:TestImageRecordActionsStopAfterGuardFailureInOptions`, `image/image_record_actions_test.go:TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP`.
- Owned accepted Read/Close/context/source/outer-guard failures return only a partial actual acknowledgement plus cause-safe ResponseError and Record=nil. Acknowledgement/error bytes and headers are independent, physical body closes once and accepted processing failures do not automatically replay POST. Nested404 causes on accepted processing are not classified as missing and never suppress partial evidence.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsAcceptedPhysicalFailuresReturnOnlyPartialAcknowledgement`.
- Owned Go action deliberately rejects actual400..599 with nil result and inspectable native status/body/header proof, and never ignores404. Transport failures without an accepted response retain their original causes. This is explicit Go error policy: the pinned Python Image._action calls only session.post, the public proxy discards its Response, and Proxy.request defaults raise_exc=False, so an ordinary HTTP400..599 Response returned by the default successful-transport chain can yield None rather than a Source status exception.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsNativeRejectionsIncludeMissingAndNeverAcknowledge`, `image/image_record_actions_test.go:TestImageRecordActionsNativeRetryPreservesFixedScopeLiveAuthAndOwnership`.
- Owned native prebody retry preserves the compiled method/fixed scope/empty body and original provider live auth while retaining ordinary retry headers and hook causes. Changed owned body/response representation is rejected before another attempt and hook-expanded OkCodes cannot fabricate acceptance outside200..399. Connection/Service delegate through the existing shared Image/ImageV2 source, current location and operation labels, with independent Record/fetch/ack receipts and cause-safe invalid invocation. No arbitrary Python session/cache/hook runtime equivalence is inferred.
  Tests: `image/image_record_actions_test.go:TestImageRecordActionsNativeRetryPreservesFixedScopeLiveAuthAndOwnership`, `connectiontest/image_record_actions_integration_test.go:TestConnectionOwnedImageActionsPreserveRecordAndSeparateReceipts`, `connectiontest/image_record_actions_integration_test.go:TestConnectionOwnedImageActionsRejectInvalidInvocation`.

Python/Go differences:

- Principal action status difference: pinned Image._action is session.post(url) without raise_from_response or _translate_response, and both public wrappers discard the returned Response. The complete default Proxy.request chain forwards raise_exc=False; ordinary HTTP400..599 Responses can therefore produce None on successful transport, unlike Go's explicit native status errors and strict404 rejection. Owned Go accepts actual200..399 and returns an inspectable opaque acknowledgement. This is deliberately go_mapping, not supported or identical Source accepted/error-status behavior; configured session/transport exceptions remain separate.
- Legacy typed strict204 profile only: Pinned Proxy methods accept direct ID strings or mutable Image Resource objects and return None. Go uses explicit ID/Name Ref and concrete owned headers, returns actual acknowledgement evidence, and introduces exact Name lookup as a typed convenience.
- Legacy typed strict204 profile only: Go intentionally accepts actual204 only, owns accepted body bytes/headers and preserves result+ResponseError on read/Close/context failures. Complete Python response translation, adapter/session exceptions, request preparation and arbitrary hooks remain broader.
- Legacy typed strict204 profile only: Go snapshots captured route/ordinary headers/options with live original auth and enforces bounded route/source/request-body guards. Existing native Name continuation/body behavior and configured same-target redirect/prebody retry policies remain unchanged; no single wire-attempt promise under configured retries.
- Legacy typed strict204 profile only: Pinned Image._action calls session.post and neither decodes a response nor applies raise_from_response explicitly; adapter exceptions remain native to the Python session. Go deliberately requires actual204 and returns owned acknowledgement evidence without updating a local status cache.
- Owned immutable JSON-domain ImageRecord uses SDK private canonical current state or literal ID, materializes the complete65 view/current location and resets ImportMethods before the compiled action. Python may reuse and mutate the same Resource/Munch/subclass and dict backing state. Arbitrary constructor/descriptor/default/history/dirty Header/base_path/requires_id lifecycle remains SDK-R1, Adapter cache invalidation remains SDK-C1, and dynamic session/microversion/class/exception/metrics/transport policy remains SDK-S1 in docs/implementation-plan.md; these global requirements are not completed by this mapping.
- Python utils.urljoin applies falsy replacement, object stringification and edge-slash trimming; its public action accepts no per-call attrs/options and _action supplies no explicit body/headers/microversion kwargs or _prepare_request negotiation. Go bounds identities to strict validUTF8 JSON string/literal segments, rejects lone-surrogate retargeting and escapes once, while captured source and concrete ordinary-header options are explicit transport extensions. Arbitrary Python objects, mutable routing and parser/coercion behavior are not runtime-equivalent.
- Neither pinned action translates response data into the Resource. Go returns a prepared independent Record with raw pending/original state and old fetch receipts intact, ImportMethods still empty, plus a separate actual acknowledgement; response import-method/status/id/location data never refreshes that Record. Accepted Read/Close/context/source failures return partial acknowledgement with Record=nil, owned causes/bytes/headers and no automatic accepted replay. These ownership/source-guard policies differ from Python's discarded Response and are not server completion or rollback guarantees.
- Pinned deactivate/reactivate defaults are project-scoped ADMIN_OR_PROJECT_MEMBER and server-owned active/deactivated transitions, authorization, deployment policy and idempotence. Go adds no role/status preflight, IgnoreMissing, metadata GET, name lookup, wait, cache update or cleanup. No authenticated cloud, Python SDK runtime or backend mutation was executed by this audit. Public create_image, property helper, upload/stage/import/download and their separate mandatory Source graphs are not closed by these two action mappings.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
conn.image.deactivate_image("image-id")

# conn.image.reactivate_image("image-id")
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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    action := flag.String("action", "deactivate", "deactivate or reactivate")
    fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record first")
    viaService := flag.Bool("service", false, "use image.Service for the selected POST")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    if *action != "deactivate" && *action != "reactivate" {
        log.Fatal("-action must be deactivate or reactivate")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *action, *fromRecord, *viaService); err != nil {
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

func run(ctx context.Context, cloud, imageID, action string, fromRecord, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordActionRequest{ID: imageID}
    var service *image.Service
    if fromRecord || viaService {
        service, err = conn.Image(ctx)
        if err != nil { return err }
    }
    if fromRecord {
        record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
        if err != nil { return err }
        input = image.ImageRecordActionRequest{Record: record}
    }
    options := []image.ImageRecordActionOption{
        image.WithImageRecordActionOpts(image.ImageRecordActionOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-actions-example"},
        }),
        image.WithImageRecordActionHeader("X-SDK-Example-Action", action),
        image.WithImageRecordActionHeaders(map[string]string{"X-SDK-Example-Mode": "owned"}),
    }
    var result *image.ImageRecordActionResult
    if viaService {
        if action == "deactivate" {
            result, err = service.DeactivateImageRecord(ctx, input, options...)
        } else {
            result, err = service.ReactivateImageRecord(ctx, input, options...)
        }
    } else if action == "deactivate" {
        result, err = conn.DeactivateImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ReactivateImageRecord(ctx, input, options...)
    }
    return printAction(result, err)
}

func printAction(result *image.ImageRecordActionResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["acknowledgement"] = result.Acknowledgement
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "earlier_receipt_status_code": result.Record.StatusCode,
                "earlier_receipt_body_bytes": len(result.Record.Envelope),
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

- [DeactivateImage/ReactivateImage](mutations.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L301-L328)
- [Image Actions](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L102)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1117-L1137)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L262-L282)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L806)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1112-L1189)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
