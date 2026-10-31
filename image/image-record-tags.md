# Image Record Tags

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/add_tag`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.AddTag`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AddImageRecordTag`

- AddTag defaults to bodyless/query-free PUT on the dedicated fixed-ID endpoint; only actual204 acknowledges the request. Direct ID causes no metadata GET/list/PATCH/discovery/wait. Missing image or tag is a strict error; there is no IgnoreMissing option or nested404 suppression.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Literal tag accepts nonempty validUTF8 up to255 Unicode code points, preserves case/whitespace/Unicode/percent/query/fragment data and PathEscapes exactly once (%2F becomes%252F). Empty/dot/dotdot/slash/backslash/control/invalidUTF8/256runes reject before Name lookup/callback/HTTP. No trim/dedupe/URL decode or tags-set replacement.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- ID is validated and routed directly; explicit Name resolves exact literal matches across existing native pages once, including UUID-shaped names, before a fixed chosen-ID mutation. Absent/ambiguous/late surfaced native lookup failures or malformed resolved IDs send no mutation. Native Name LIST model/body ownership/continuation behavior is retained.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- Context/source/provider/type/origin/UTF8 base/source headers/version/Ref and literal tag preflight runs before caller callbacks or HTTP. Nil callbacks and callback/source/context failures preserve original causes. Ordinary option headers are the only extension; protected auth/framing/version/representation fields and conflicting supplied-map aliases fail locally.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- WithImageMutationOpts replaces the full concrete header policy. Header helpers snapshot input maps at creation; option slice and each callback config/map are copied before and after one invocation. Retained handles, later callbacks and parallel helper reuse cannot change prepared request policy.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Captured original client/provider/prefix and ordinary headers are used for Name LIST and mutation; the original provider latest auth token stays live. Safe source base/header changes retain captured requests, while provider replacement or foreign source retarget rejects before mutation.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`, `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Only actual original204 creates an acknowledgement with fixed request ImageID and literal Tag or compiled Action, owned opaque Body bytes, cloned Header and actual StatusCode. Empty/nonJSON/invalidUTF8 response bytes are allowed and never decode metadata, retarget from Location/body IDs, or establish current tags/status.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Accepted204 read/Close/context failures return retained partial/full acknowledgement plus ResponseError. Result/error Body/Header copies are independent; original read/Close/ctx.Err/custom cause survives. Accepted body closes exactly once without accepted replay. Native rejected/transport/prebody failures and unexpected200/201/202/206/404 keep nil result and their native evidence.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Configured native prebody retry/reauth/backoff preserves fixed method/origin/path/query and live auth, and may attempt the mutation again before acceptance. Body ownership tampering, including nil-to-JSONnull, RawBody, KeepResponseBody or JSONResponse, fails before the next attempt with original/hook/encoding causes; expanded OkCodes cannot bypass original204. Native ErrUnableToReauthenticate ErrOriginal/ErrReauth remain inspectable through fields. Changed-target/method redirects are guarded; identical-target configured redirect policy remains. Prior native APIs are preserved.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`.
- AddImageRecordTag takes exactly one literal ID or supplied ImageRecord and performs PUT /images/{once-escaped-id}/tags/{once-escaped-tag} without GET/List/body/query. ID constructs 65 declared fields with tags=[], properties=null and captured current location; supplied record preserves its own location and fetch receipt.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagLiteralConstructorBoundsAndCurrentLocation`, `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- After accepted HTTP, Add appends the literal tag including duplicates. Missing tags uses an independent empty list; nonnull scalar/object becomes a singleton; existing untyped raw members and caller record remain unchanged.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `image/image_record_tag_test.go:TestImageRecordTagLiteralConstructorBoundsAndCurrentLocation`.
- The named owned tag profile accepts actual200..399 and returns independent opaque mutation bytes/header/status in Acknowledgement. Response body is never decoded as Image or tags; supplied Record Wire/Envelope/Header/StatusCode remain the original fetch evidence, while a literal-ID constructor has no fetch evidence.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagAcceptsOpaque200Through399`, `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- Explicit-null tags and broken selected RawResource tags fail only after accepted HTTP. Acknowledgement remains inspectable with nil Record plus error; malformed JSON/nonUTF8 raw input is a Go-only representation boundary. Unrelated raw fields are passive, not reprojected.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagNullLocalListFailsAfterAcknowledgement`.
- Input records/raw bytes, option slice and each callback config are captured independently. Existing concrete ImageMutationOpts/Header/Headers helpers run once per operation, captured route/location/source and ordinary headers are guarded, auth stays live, and ID/tag/source/context/header validation precedes callbacks or HTTP.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagSourceHeaderSeedAndOptionSnapshots`, `image/image_record_tag_test.go:TestImageRecordTagPreflightAndGuardedCallbacks`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- Accepted Read/Close/context/source/hook failures retain possible opaque Acknowledgement and nil Record plus original causes. Rejected actual400..599 keeps nil result and native HTTP identity with physical Read/Close/cancel/binding causes. Clean native retry input stays unmodified; strict204 legacy mutation and native full-set PATCH behavior remain.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagAcceptedReadCloseAndStickyGuardAcknowledgements`, `image/image_record_tag_test.go:TestImageRecordTagNativeRejectionsRetryAndLegacy204`, `image/image_record_tag_test.go:TestImageRecordTagRejectedPhysicalFaults`.

Python/Go differences:

- The original nine contracts and old AddTag/RemoveTag APIs describe the retained strict204 Ref/ACK profile. New AddImageRecordTag/RemoveImageRecordTag provides the named immutable ImageRecord profile and actual200..399 local-list behavior; legacy profile differences are not limitations of the new profile.
- Pinned public Proxy returns None while TagMixin mutates and returns the same Resource. Go returns an independent Record and separate Acknowledgement; it does not return or mutate the caller object. Full same-instance/dirty/component/default alias lifecycle remains SDK-R1.
- Pinned Proxy._get_resource(existing Image) invokes Resource._update(**{}), which resets microversion, recollects components, updates connection-derived location and to_dict. Go deliberately preserves supplied location and unrelated RawResource fields; this incidental preparation lifecycle is not reproduced by the named immutable profile (SDK-R1/S1).
- Python Body tags uses a shared mutable default list and descriptor list coercion. Go defaults are private for each call and nonnull raw scalar/object wraps in one list. Explicit null errors after HTTP as Source; malformed JSON/nonUTF8 selected raw bytes are Go-only input errors. Complete Python Munch/subclass/arbitrary descriptor/runtime behavior remains SDK-R1.
- Pinned URL join strips outer slashes and leaves inner slashes; Go validates literal identity/tag and escapes each URI segment once. Empty/invalidUTF8/control/exact dot segments reject before HTTP, while whitespace-only/nonempty slash/backslash and tags longer than255 characters remain server-owned. No schema/permission/storage conclusion is inferred.
- Go owns actual response bytes/header/status and guards captured source/route/body/options with live native auth. Full configured Python Adapter/cache/conditional GET/dynamic session/base_path/microversion behavior remains SDK-C1/S1; no accepted response replay, metadata gate, automatic wait/rollback or current full-tag fetch is added.
- The new local tags list reflects seed plus this operation, not a server-confirmed complete set. Backend persistence, deployed policy overrides/permissions/length limits and actual cloud outcomes are not proved by local fixtures or example builds.

### `python:image/v2/remove_tag`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.RemoveTag`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.RemoveImageRecordTag`

- RemoveTag defaults to bodyless/query-free DELETE on the dedicated fixed-ID endpoint; only actual204 acknowledges the request. Direct ID causes no metadata GET/list/PATCH/discovery/wait. Missing image or tag is a strict error; there is no IgnoreMissing option or nested404 suppression.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Literal tag accepts nonempty validUTF8 up to255 Unicode code points, preserves case/whitespace/Unicode/percent/query/fragment data and PathEscapes exactly once (%2F becomes%252F). Empty/dot/dotdot/slash/backslash/control/invalidUTF8/256runes reject before Name lookup/callback/HTTP. No trim/dedupe/URL decode or tags-set replacement.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- ID is validated and routed directly; explicit Name resolves exact literal matches across existing native pages once, including UUID-shaped names, before a fixed chosen-ID mutation. Absent/ambiguous/late surfaced native lookup failures or malformed resolved IDs send no mutation. Native Name LIST model/body ownership/continuation behavior is retained.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`.
- Context/source/provider/type/origin/UTF8 base/source headers/version/Ref and literal tag preflight runs before caller callbacks or HTTP. Nil callbacks and callback/source/context failures preserve original causes. Ordinary option headers are the only extension; protected auth/framing/version/representation fields and conflicting supplied-map aliases fail locally.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsLiteralTagIdentityAndCompletePreflight`, `image/mutation_core_test.go:TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- WithImageMutationOpts replaces the full concrete header policy. Header helpers snapshot input maps at creation; option slice and each callback config/map are copied before and after one invocation. Retained handles, later callbacks and parallel helper reuse cannot change prepared request policy.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`.
- Captured original client/provider/prefix and ordinary headers are used for Name LIST and mutation; the original provider latest auth token stays live. Safe source base/header changes retain captured requests, while provider replacement or foreign source retarget rejects before mutation.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsPreparedOptionsAndLiveSource`, `image/mutation_core_test.go:TestImageMutationCoreSourcePreflightAndCapturedLiveAuth`, `image/mutation_options_test.go:TestImageMutationOptionsSnapshotReplacementAndLastWins`, `image/mutation_options_test.go:TestImageMutationOptionsCallbacksOwnHandlesAndCauses`, `image/mutation_options_test.go:TestImageMutationOptionsRejectHeaderOwnershipAndAliases`, `image/mutation_options_test.go:TestImageMutationOptionsParallelReusableHelpers`, `image/mutation_contracts_test.go:TestImageMutationsExactNameResolutionAndFailures`, `image/mutation_core_test.go:TestImageMutationCoreExactNameAndStrictLookupFailures`, `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Only actual original204 creates an acknowledgement with fixed request ImageID and literal Tag or compiled Action, owned opaque Body bytes, cloned Header and actual StatusCode. Empty/nonJSON/invalidUTF8 response bytes are allowed and never decode metadata, retarget from Location/body IDs, or establish current tags/status.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsFixedRoutesAndAcknowledgements`, `image/mutation_core_test.go:TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements`, `connectiontest/connection_image_mutations_test.go:TestConnectionImageMutationsShareClientAndKeepAcknowledgements`, `internal/cmd/sdkgen/glance_mutations_test.go:TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Accepted204 read/Close/context failures return retained partial/full acknowledgement plus ResponseError. Result/error Body/Header copies are independent; original read/Close/ctx.Err/custom cause survives. Accepted body closes exactly once without accepted replay. Native rejected/transport/prebody failures and unexpected200/201/202/206/404 keep nil result and their native evidence.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`.
- Configured native prebody retry/reauth/backoff preserves fixed method/origin/path/query and live auth, and may attempt the mutation again before acceptance. Body ownership tampering, including nil-to-JSONnull, RawBody, KeepResponseBody or JSONResponse, fails before the next attempt with original/hook/encoding causes; expanded OkCodes cannot bypass original204. Native ErrUnableToReauthenticate ErrOriginal/ErrReauth remain inspectable through fields. Changed-target/method redirects are guarded; identical-target configured redirect policy remains. Prior native APIs are preserved.
  Tests: `image/mutation_contracts_test.go:TestImageMutationsProviderHooksAndNativeCompatibility`, `image/mutation_core_test.go:TestImageMutationCoreNativeStatusesAndPrebodyOwnership`, `image/mutation_contracts_test.go:TestImageMutationsAcceptedFailuresAndStrictStatuses`, `image/mutation_core_test.go:TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence`.
- RemoveImageRecordTag takes exactly one literal ID or supplied ImageRecord and performs DELETE /images/{once-escaped-id}/tags/{once-escaped-tag} without GET/List/body/query. ID uses independent 65-field constructor defaults/current location and no fetch receipt; supplied location/other fields/fetch evidence are preserved.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagLiteralConstructorBoundsAndCurrentLocation`, `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- After accepted HTTP, Remove deletes the first exactly matching string only. Missing tags or absent local member is a local no-op, while native HTTP404 is an error. Scalars become singleton lists and nonstring/raw members remain passive; lone surrogate members do not match U+FFFD, valid surrogate pairs and literal escaped backslash do match.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `image/image_record_tag_test.go:TestImageRecordTagNativeRejectionsRetryAndLegacy204`.
- The named owned tag profile accepts actual200..399 and returns independent opaque mutation bytes/header/status in Acknowledgement. Response body is never decoded as Image or tags; supplied Record Wire/Envelope/Header/StatusCode remain the original fetch evidence, while a literal-ID constructor has no fetch evidence.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagAcceptsOpaque200Through399`, `image/image_record_tag_test.go:TestImageRecordTagOwnedSeedListSemanticsAndReceipts`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- Explicit-null tags and broken selected RawResource tags fail only after accepted HTTP. Acknowledgement remains inspectable with nil Record plus error; malformed JSON/nonUTF8 raw input is a Go-only representation boundary. Unrelated raw fields are passive, not reprojected.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagNullLocalListFailsAfterAcknowledgement`.
- Input records/raw bytes, option slice and each callback config are captured independently. Existing concrete ImageMutationOpts/Header/Headers helpers run once per operation, captured route/location/source and ordinary headers are guarded, auth stays live, and ID/tag/source/context/header validation precedes callbacks or HTTP.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagSourceHeaderSeedAndOptionSnapshots`, `image/image_record_tag_test.go:TestImageRecordTagPreflightAndGuardedCallbacks`, `connectiontest/connection_image_record_tags_test.go:TestConnectionImageRecordTagsShareOwnedSourceAndMutationReceipts`.
- Accepted Read/Close/context/source/hook failures retain possible opaque Acknowledgement and nil Record plus original causes. Rejected actual400..599 keeps nil result and native HTTP identity with physical Read/Close/cancel/binding causes. Clean native retry input stays unmodified; strict204 legacy mutation and native full-set PATCH behavior remain.
  Tests: `image/image_record_tag_test.go:TestImageRecordTagAcceptedReadCloseAndStickyGuardAcknowledgements`, `image/image_record_tag_test.go:TestImageRecordTagNativeRejectionsRetryAndLegacy204`, `image/image_record_tag_test.go:TestImageRecordTagRejectedPhysicalFaults`.

Python/Go differences:

- The original nine contracts and old AddTag/RemoveTag APIs describe the retained strict204 Ref/ACK profile. New AddImageRecordTag/RemoveImageRecordTag provides the named immutable ImageRecord profile and actual200..399 local-list behavior; legacy profile differences are not limitations of the new profile.
- Pinned public Proxy returns None while TagMixin mutates and returns the same Resource. Go returns an independent Record and separate Acknowledgement; it does not return or mutate the caller object. Full same-instance/dirty/component/default alias lifecycle remains SDK-R1.
- Pinned Proxy._get_resource(existing Image) invokes Resource._update(**{}), which resets microversion, recollects components, updates connection-derived location and to_dict. Go deliberately preserves supplied location and unrelated RawResource fields; this incidental preparation lifecycle is not reproduced by the named immutable profile (SDK-R1/S1).
- Python Body tags uses a shared mutable default list and descriptor list coercion. Go defaults are private for each call and nonnull raw scalar/object wraps in one list. Explicit null errors after HTTP as Source; malformed JSON/nonUTF8 selected raw bytes are Go-only input errors. Complete Python Munch/subclass/arbitrary descriptor/runtime behavior remains SDK-R1.
- Pinned URL join strips outer slashes and leaves inner slashes; Go validates literal identity/tag and escapes each URI segment once. Empty/invalidUTF8/control/exact dot segments reject before HTTP, while whitespace-only/nonempty slash/backslash and tags longer than255 characters remain server-owned. No schema/permission/storage conclusion is inferred.
- Go owns actual response bytes/header/status and guards captured source/route/body/options with live native auth. Full configured Python Adapter/cache/conditional GET/dynamic session/base_path/microversion behavior remains SDK-C1/S1; no accepted response replay, metadata gate, automatic wait/rollback or current full-tag fetch is added.
- The new local tags list reflects seed plus this operation, not a server-confirmed complete set. Backend persistence, deployed policy overrides/permissions/length limits and actual cloud outcomes are not proved by local fixtures or example builds.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")

result = conn.image.add_tag("image-id", "reviewed")
assert result is None

image = conn.image.get_image("image-id")
conn.image.remove_tag(image, "reviewed")
print(image.tags)
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
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    tag := flag.String("tag", "reviewed", "literal tag to add and then remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *tag); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printResult(value *image.ImageRecordTagResult, operationErr error) error {
    if value != nil {
        output := make(map[string]any)
        if ack := value.Acknowledgement; ack != nil {
            output["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "tag": ack.Tag,
                "status_code": ack.StatusCode, "body_bytes": len(ack.Body),
            }
        }
        if value.Record != nil {
            output["resource"] = value.Record.Resource
            output["wire"] = value.Record.Wire
            output["record_status_code"] = value.Record.StatusCode
        }
        body, err := json.MarshalIndent(output, "", "  ")
        if err != nil { return errors.Join(operationErr, err) }
        fmt.Println(string(body))
    }
    return operationErr
}

func run(ctx context.Context, cloud, imageID, tag string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    added, err := service.AddImageRecordTag(ctx, image.ImageRecordTagRequest{ID: imageID}, tag,
        image.WithImageMutationHeader("X-Request-Source", "image-record-tags-example"))
    if err := printResult(added, err); err != nil { return err }

    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    removed, err := service.RemoveImageRecordTag(ctx, image.ImageRecordTagRequest{Record: seed}, tag,
        image.WithImageMutationHeaders(map[string]string{"X-Request-Source": "image-record-tags-example"}))
    return printResult(removed, err)
}
```

## References

- [ImageRecord](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1196-L1220)
- [TagMixin add/remove](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L105-L145)
- [Install](../docs/install.md)
- [urljoin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L43-L50)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L839)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Image Record Tag Test](image_record_tag_test.go)
- [Connection Image Record Tags Test](../connectiontest/connection_image_record_tags_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
