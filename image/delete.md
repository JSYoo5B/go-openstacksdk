# Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/delete_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.DeleteImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDeleteImageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDeleteImageStore`
- `github.com/JSYoo5B/go-openstacksdk/image.WithDeleteImageIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.DeleteImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteStoreID`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteStoreRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordDeleteIgnoreMissing`

- Legacy typed DeleteImage profile only: DeleteImage defaults to whole-image DELETE and ignores clean missing resources. ID routes directly without metadata GET; nonnil store selects only /stores/{storeID}/{imageID}. Only actual204 constructs a result with fixed selected ImageID/StoreID, owned raw Body, cloned Header and actual StatusCode; no decoded model or fabricated response.
  Tests: `image/delete_contracts_test.go:TestDeleteImageWholeAndStoreRoutesAndEvidence`, `image/delete_core_test.go:TestDeleteCoreFixedScopeAndOwnedAcknowledgement`.
- Legacy typed DeleteImage profile only: Context/service/provider/image base/headers/ref and all options are validated before HTTP. Nil policy defaults StoreID=nil and IgnoreMissing=true; explicit false remains distinct. Explicit empty/unsafe/invalid UTF8 store fails locally instead of broadening to whole delete. Nil callbacks and caller callback failures preserve the invalid-option or original cause.
  Tests: `image/delete_contracts_test.go:TestDeleteImageCompletePreflight`, `image/delete_options_test.go:TestDeleteOptionsDefaultsAndExplicitStorePreflight`.
- Legacy typed DeleteImage profile only: Whole-policy helper snapshots both pointers at construction and fully replaces prior settings; leaf helpers replace one setting in order. Option slice and each callback configuration are owned. Custom callbacks run once before requests, and retained configuration cannot mutate later callbacks or the final policy. Reused snapshot helpers remain independent under concurrent application.
  Tests: `image/delete_contracts_test.go:TestDeleteImagePreparedSnapshotsAndSource`, `image/delete_options_test.go:TestDeleteOptionsFullReplacementAndConcurrentReuse`, `image/delete_options_test.go:TestDeleteOptionsCallbacksOnceAndIndependentRetainedHandles`.
- Legacy typed DeleteImage profile only: Explicit Name uses existing exact-name full-page Images resolver and duplicate detection, then a fixed safe DELETE ID. Only direct logical image NotFound from completed successful lookup, with no cause/context/source failure, is ignored by default. HTTP404, surfaced native body/decode/transport failures, ambiguity and nested404 causes remain errors. Native LIST body ownership and continuation semantics are retained.
  Tests: `image/delete_contracts_test.go:TestDeleteImageExactNameResolutionAndFailures`, `image/delete_core_test.go:TestDeleteCoreNameAbsenceAndSourceFailureBoundaries`.
- Legacy typed DeleteImage profile only: Privately owned DELETE404 is read and closed once before ignore decisions. Clean default404 returns nil,nil; clean strict404 returns nil plus typed logical NotFound preserving a value native ErrUnexpectedResponseCode with DELETE/fixedURL/expected204/actual404/rawBody/Header. Body read/Close/context failure preserves native404 and every cause and is never suppressed or converted into NotFound-only success.
  Tests: `image/delete_contracts_test.go:TestDeleteImageNotFoundPolicyAndClassification`, `image/delete_core_test.go:TestDeleteCoreOwnedMissingPreservesBodyAndContextFailures`.
- Legacy typed DeleteImage profile only: Accepted DELETE204 result survives partial read, Close and context failures together with accepted ResponseError. Result and error independently own bytes and headers. Original read/Close/native status and ctx.Err/context.Cause survive request operation wrapping. Nested404 during accepted204 does not erase acknowledgement or classify the operation as missing. Owned204/404 bodies close once without replay after body failure.
  Tests: `image/delete_contracts_test.go:TestDeleteImageAcceptedProofAndCancellation`, `image/delete_core_test.go:TestDeleteCoreAcknowledgementAndErrorEvidenceAreIndependent`, `image/delete_core_test.go:TestDeleteCoreOwnedMissingPreservesBodyAndContextFailures`.
- Legacy typed DeleteImage profile only: Captured prefix/headers and policy use the original provider live token; original source/provider and target guards run after callbacks and Name lookup. Legitimate captured source/header selection remains fixed while malformed or replaced sources fail before DELETE. Cancellation, parent deadline, HTTPClient timeout and original transport causes remain errors; all HTTP/test handles are closed.
  Tests: `image/delete_contracts_test.go:TestDeleteImagePreparedSnapshotsAndSource`, `image/delete_contracts_test.go:TestDeleteImageAcceptedProofAndCancellation`, `image/delete_core_test.go:TestDeleteCoreNameAbsenceAndSourceFailureBoundaries`.
- Legacy typed DeleteImage profile only: Configured pre-body reauthentication/retry/backoff remains available for native rejected/transport failures and may resend DELETE before acceptance. DELETE redirect following is disabled. Privately handled404 intentionally bypasses native RetryFunc; neither handled404 nor accepted204 body read/Close failures are retried. Native ErrUnableToReauthenticate retains independent ErrOriginal/ErrReauth fields; its causes are inspected through these fields after errors.As because that native wrapper does not Unwrap. Configured RetryFunc changes to KeepResponseBody/JSONResponse/JSONBody/RawBody are rejected before another DELETE with ErrInvalidOption, original pre-body failure and any callback cause retained, preserving the empty request and owned-response contract. The contract does not promise a single wire packet under configured retry policy.
  Tests: `image/delete_contracts_test.go:TestDeleteImageProviderHooksAndRedirects`, `image/delete_core_test.go:TestDeleteCoreActualProtocolAndNativeRetryBoundaries`.
- Legacy typed DeleteImage profile only: The upper workflow accepts actual204 only;200/202/206/redirect/403/409 or other unexpected status normally uses native errors and native rejected-body ownership without an acknowledgement result. If configured RetryFunc changes OkCodes so native Request accepts another status, the workflow owns its read/Close then rejects the actual status with raw body/header and causes rather than creating acknowledgement evidence. Store deletion never falls back to whole deletion. Server-owned last-location409/protected/status/policy decisions do not trigger metadata/store discovery, wait or cache/Swift/task cleanup.
  Tests: `image/delete_contracts_test.go:TestDeleteImageWholeAndStoreRoutesAndEvidence`, `image/delete_contracts_test.go:TestDeleteImageNativeDeleteCompatibility`, `image/delete_contracts_test.go:TestDeleteImageProviderHooksAndRedirects`, `image/delete_core_test.go:TestDeleteCoreActualProtocolAndNativeRetryBoundaries`.
- Legacy typed DeleteImage profile only: Existing native Images.Delete keeps its error-only generated callable and native202/204 semantics, while Collection.Delete/Remove and other native bindings remain unchanged. Generator checks native Delete context/client/id/result signature, ErrResult and Image canonical ID and rejects signature/result/identity drift. Upper whole/store guide adds no native operation, model, inventory row or store collection.
  Tests: `image/delete_contracts_test.go:TestDeleteImageNativeDeleteCompatibility`, `internal/cmd/sdkgen/glance_delete_test.go:TestGlanceDeletePreservesNativeAPIAndDocumentsStoreWorkflow`, `internal/cmd/sdkgen/glance_delete_test.go:TestGlanceDeleteRejectsNativeSignatureResultAndIdentityDrift`.
- Owned DeleteImageRecord selects one literal image ID or an SDK-produced ImageRecord's private raw current canonical id. Public projected id, Wire, Envelope, Header, passive URLs and property id do not retarget deletion. Explicit StoreID or StoreRecord.Resource.Body canonical id selects only the store route, without name fallback. Both image/store forms, missing/null/non-string identities, handcrafted ImageRecord without private state, nonblank/UTF8/control/dot validation failures are rejected before HTTP. Raw JSON identity decoding rejects unpaired escaped surrogates without treating literal U+FFFD, valid pairs or a literal backslash-u sequence as malformed. Valid identities are escaped once as a single path segment; identity and all caller channels are copied before callbacks.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordLiteralWholeAndStoreRoutesOwnOpaqueAcknowledgements`, `image/image_record_delete_test.go:TestDeleteImageRecordCompletePreflightRejectsAmbiguousOrUnsafeSelectors`, `image/record_identity_delete_test.go:TestDeleteImageRecordIdentityRejectsUnpairedPrivateCurrentID`, `image/record_identity_delete_test.go:TestDeleteImageRecordIdentityRejectsUnpairedStoreRecordID`, `image/record_identity_delete_test.go:TestDeleteImageRecordIdentityPreservesValidUnicodeTargets`.
- Owned whole deletion prepares the private current raw image body into the full65-field descriptor view with this call's current location before HTTP. It resets the plain ImportMethods attribute during preparation and consumes only the accepted DELETE response ImportMethods header afterward. A successful result returns an independent Record preserving current/original/dirty Body state and its prior fetched Wire/Envelope/Header/StatusCode plus passive metadata. Opaque DELETE bytes do not overlay attributes or clean pending Body. Literal-ID construction has no fabricated earlier fetch receipt; the actual DELETE receipt belongs only to Acknowledgement.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordProjectsPrivateBodyAndPreservesFetchedReceipt`, `image/image_record_delete_test.go:TestDeleteImageRecordKeepsPendingChangesForExplicitSubsequentCommit`, `image/image_record_delete_test.go:TestDeleteImageRecordLiteralWholeAndStoreRoutesOwnOpaqueAcknowledgements`.
- Owned store deletion extracts image/store canonical identities only and sends a fixed /stores/{storeID}/{imageID} DELETE. It performs no whole Image descriptor projection, response Image translation or ImportMethods update, and returns Record=nil even on success. Unrelated invalid raw Image descriptors or Store properties/description do not block that route; the same invalid Image projection still blocks whole deletion before HTTP. No discovery, metadata GET, alternate name lookup or whole-delete fallback is added.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordStoreOnlyExtractsIdentitiesBeforeProjection`.
- Owned concrete deletion options snapshot maps/pointers/StoreRecords and the option slice. FullOpts replaces all configuration; StoreID and StoreRecord helpers clear the other selector, and nil StoreRecord or empty full configuration explicitly restores whole mode. Explicit empty StoreID remains invalid. Ordinary headers merge, IgnoreMissing changes only its policy and callbacks apply once to independently copied configurations. Captured source/location/options and child identity stay fixed with the original provider's live token. Context/source/binding/outer guard failure stops later restoring callbacks and retains the original cause.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordStoreSelectorsAndFactorySnapshots`, `image/image_record_delete_test.go:TestDeleteImageRecordSourceOptionsLocationAndChildSnapshots`, `image/image_record_delete_test.go:TestDeleteImageRecordStopsAfterGuardFailureInOptions`, `image/image_record_delete_test.go:TestDeleteImageRecordCompletePreflightRejectsAmbiguousOrUnsafeSelectors`.
- Owned whole/store DELETE accepts actual200..399 and preserves an opaque actual acknowledgement with fixed chosen ImageID/StoreID and independent Body/Header/StatusCode. JSON null, arrays, malformed JSON and invalidUTF8 bytes are never decoded into Image data. Accepted Read/Close/context/source/outer guard failures return only a partial acknowledgement and error, with Record=nil, physical Close once, original causes and actual ResponseError evidence. Accepted handling failures are not automatically replayed or reclassified as missing because a nested cause mentions404.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordAcceptedStatusesKeepOpaqueBodyWithoutTranslation`, `image/image_record_delete_test.go:TestDeleteImageRecordAcceptedPhysicalFailuresReturnOnlyAcknowledgement`, `image/image_record_delete_test.go:TestDeleteImageRecordLiteralWholeAndStoreRoutesOwnOpaqueAcknowledgements`.
- Owned nil/defaulttrue IgnoreMissing suppresses only a clean final direct native DELETE404 after owned read/Close/context/source handling succeeds. False preserves native404 proof. Configured native retry hooks remain active, including a clean final404 after prebody retry; read/Close/cancellation/source/hook failures and nested transport404 are never hidden. Ignored missing returns nil result,nil error and creates no accepted acknowledgement or proof of backend absence.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordMissingPolicySuppressesOnlyCleanFinalNative404`.
- Owned rejected statuses and transport failures without an accepted response keep nil result and native status/body/header or original cause. Native prebody retry may resend the fixed empty-body DELETE and preserves live authentication/ordinary retry headers, while changed body ownership is rejected before another request and hook-expanded OkCodes cannot turn an actual out-of-policy status into a successful acknowledgement. Native hooks and the lower generated Delete binding are not replaced or mutated by this fixed owned policy.
  Tests: `image/image_record_delete_test.go:TestDeleteImageRecordNativeRejectionsNeverCreateAcknowledgement`, `image/image_record_delete_test.go:TestDeleteImageRecordNativeRetryKeepsFixedScopeAndActualStatusPolicy`.
- Connection and Service owned deletion share the existing Image/ImageV2 source binding, current call location, source headers and live authentication. Whole deletion separates the independent local descriptor Record and earlier fetch receipt from the actual deletion acknowledgement; store deletion contributes acknowledgement only. Caller mutations to a result, source receipt or later result do not alias the earlier input or another acknowledgement, and literal construction does not invent a fetched model.
  Tests: `image_record_delete_integration_test.go:TestConnectionOwnedImageDeletionSeparatesWholeRecordFromStoreAcknowledgement`, `image/image_record_delete_test.go:TestDeleteImageRecordProjectsPrivateBodyAndPreservesFetchedReceipt`.

Python/Go differences:

- Legacy typed DeleteImage profile only: Go uses explicit ID or exact Name Ref and concrete owned options. Pinned Python accepts ID or mutable Image/Store Resource objects and chooses store only when truthy; its empty/falsy store broadens to whole image, whereas explicit empty Go StoreID is invalid before HTTP.
- Legacy typed DeleteImage profile only: Pinned Python Proxy defaults ignore_missing=True, forwards it to Store leaf whose direct default isFalse, and discards both leaf return values. Go exposes actual204 raw Body/Header/StatusCode with fixed scope IDs; ignored clean404 or logical absence returns nil,nil and carries no success acknowledgement.
- Legacy typed DeleteImage profile only: Pinned Python exceptions.raise_from_response accepts status<400. Existing native images.Delete accepts202/204; upper DeleteImage deliberately accepts204 only. Unexpected statuses retain native error ownership rather than full Python response translation/session parity.
- Legacy typed DeleteImage profile only: Go privately handles404 to read/Close before suppression, retaining read/Close/context/native404 causes and preventing nested404 failure suppression. The handled404 bypasses native RetryFunc, an explicit difference from the rejected native404 hook; other configured pre-body reauth/retry/backoff remains.
- Legacy typed DeleteImage profile only: Go captured route/header/options and live original provider token, source guards, redirect rejection, independently owned204 result/error proof and errors.Is/As causes are concrete workflow policies. Python Resource/adapter/session and mutable request preparation are broader; Name LIST ownership and continuation retain existing native pager behavior.
- Legacy typed DeleteImage profile only: Go bounds native RetryFunc request-option mutation: changing KeepResponseBody, JSONResponse, JSONBody or RawBody fails before another DELETE while preserving the original rejected-request error and callback cause. This deliberate body-ownership guard is not complete arbitrary Python adapter/session hook parity.
- Owned DeleteImageRecord maps the finite immutable JSON-domain ImageRecord/StoreRecord convenience profile and returns independent local Record/actual acknowledgement instead of the pinned public proxy's None. Only SDK-produced private Image raw state is accepted; arbitrary mutable Resource/Munch/subclass, same-instance changes, dirty Header components and dynamic base_path/requires_id descriptor lifecycle remain SDK-R1. Adapter cache invalidation remains SDK-C1 and full dynamic session/microversion/exception/class/transport policy remains SDK-S1 in docs/implementation-plan.md; these global tasks are not closed by this mapping.
- Python branches on store truthiness: an empty string can select whole deletion while an ordinary Store Resource can stay truthy without a usable id. Its utils.urljoin performs falsey replacement, stringification and edge-slash trimming without identity escaping. Go requires nonblank literal canonical strings, rejects explicit empty/missing/null stores and ambiguous selectors, decodes raw JSON identities without silent lone-surrogate replacement, and escapes accepted UTF8 segments once. Arbitrary Python falsy/object/coercion/JSON-parser behavior is not runtime-equivalent.
- Pinned whole deletion reuses/prepares/translates the mutable Image with has_body=False and operation Resource headers/microversion kwargs; Store.delete_image consumes only IDs and invokes session.delete(url) without those per-call kwargs or Image projection. Go preserves that state distinction but applies captured source and concrete ordinary-header/microversion transport policy to its fixed routes, an explicit extension rather than full Adapter negotiation or dirty Header parity.
- Go whole success preserves old fetched Wire/Envelope/Header/StatusCode while only ImportMethods consumes the new response header; its acknowledgement separately owns the actual200..399 DELETE bytes/status/header. Accepted read/Close/context/source failures return partial acknowledgement and no Record. These cause-safe immutable receipt policies, sticky source guards, direct clean-final-native404 classification, accepted status bounding and no automatic post-accept replay are deliberate Go choices rather than Python response/exception object equivalence or rollback guarantees.
- Cloud ImageCloudMixin.delete_image delete_objects/Swift object cleanup, cache_delete_image, task cleanup, waits, generic Collection deletion and other Source declarations are separate work. This profile introduces no name lookup, extra metadata GET, store discovery, server role check, protected/status client gate or whole fallback. Default pinned whole policy is project ADMIN_OR_PROJECT_MEMBER; optional store deletion uses project ADMIN delete_image_location, but both branches form this one declared operation and do not add a second API count. Deployment overrides/backends/locations/protection and actual deletion completion remain server-owned.
- The contract audit is pinned-source static evidence plus named local Go tests and build gates. It executes no Python SDK runtime, authenticated cloud mutation or backend authorization outcome; native provider hooks may emit multiple pre-accept DELETEs and custom transport policies remain outside the owned receipt guarantee.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func deleteWholeImage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*image.DeleteImageResult, error) {
    return image.New(client).DeleteImage(ctx, resource.ID(id))
}
```

```python
conn.image.delete_image(image_id)
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func deleteStoreCopy(ctx context.Context, client *gophercloud.ServiceClient, id, storeID string) (*image.DeleteImageResult, error) {
    ignoreMissing := false
    return image.New(client).DeleteImage(ctx, resource.ID(id),
        image.WithDeleteImageOpts(image.DeleteImageOpts{
            StoreID: &storeID,
            IgnoreMissing: &ignoreMissing,
        }),
    )
}
```

```python
conn.image.delete_image(image_id, store=store_id, ignore_missing=False)
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image-from-store)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image)
- [Proxy.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)
- [Store.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75)
- [Delete Contracts Test](delete_contracts_test.go)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [Glance Delete Test](../internal/cmd/sdkgen/glance_delete_test.go)
