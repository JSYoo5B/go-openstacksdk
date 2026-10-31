# Image Record Upload

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/upload_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.UploadImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithoutImageUploadSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadField`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadFields`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadOwner`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadProtected`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadHidden`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadMinDisk`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadMinRAM`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageUploadTags`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.UploadImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UploadImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithoutImageRecordUploadSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordUploadSizeInference`

- Legacy typed strict201/204 upload profile only: One fixed POST201 metadata creation followed by one PUT204 binary upload uses canonical response ID, defaults qcow2/bare/private, and performs no additional detail GET, queued status gate, checksum/import/Task/Swift/wait/cleanup.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreFixedRoutesAndIndependentEvidence`, `image/upload_image_contracts_test.go:TestImageUploadContractsWireAndOwnership`, `connectiontest/connection_image_upload_test.go:TestConnectionImageUploadFacadeOwnsPhases`, `internal/cmd/sdkgen/glance_image_upload_test.go:TestGlanceImageUploadPreservesNativeAndRegistry`.
- Legacy typed strict201/204 upload profile only: Concrete full opts replaces whole configuration; plural Fields replaces all raw overrides; singular field/typed helpers overwrite exact keys; Headers merges ordinary headers. Maps/raw bytes/slices/pointers and initialized callbacks are independently owned across reuse and callback retention.
  Tests: `image/upload_image_options_test.go:TestImageUploadOptionsFactoriesAndSnapshots`, `image/upload_image_options_test.go:TestImageUploadOptionsReplacementAndTypedPresence`, `image/upload_image_options_test.go:TestImageUploadOptionsRetainedCallbacksAndSource`, `image/upload_image_options_test.go:TestImageUploadOptionsParallelReuseAndHeaderPolicies`, `image/upload_image_contracts_test.go:TestImageUploadContractsPreflightAndSnapshots`.
- Legacy typed strict201/204 upload profile only: Flat precise JSON owns request name while forwarding other canonical/custom keys without descriptor aliases/coercion/schema/enum/readonly/range gates. Final formats must be nonempty UTF8 JSON strings; null/false/zero/negative/exact numbers remain server data.
  Tests: `image/upload_image_options_test.go:TestImageUploadOptionsReplacementAndTypedPresence`, `image/upload_image_options_test.go:TestImageUploadOptionsFactoriesAndSnapshots`, `image/upload_image_contracts_test.go:TestImageUploadContractsWireAndOwnership`.
- Legacy typed strict201/204 upload profile only: Required valid UTF8 nonblank Name and nonnil/not typed nil borrowed Data, nil callback/factory marshal cause/raw key/value/header/size validation stop before any HTTP/Reader Read/Seek/Close/length inference; literal key edges reject Python-strip-changing whitespace including U001C..001F.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreCompletePreflight`, `image/upload_image_contracts_test.go:TestImageUploadContractsPreflightAndSnapshots`, `image/upload_image_options_test.go:TestImageUploadOptionsFactoriesAndSnapshots`.
- Legacy typed strict201/204 upload profile only: Original service/provider/Type/Endpoint/base/reverse prefix/microversion and ordinary headers captured before callbacks remain fixed across phases; actual transport uses live original auth. Private JSON/binary media is installed without mutating source; explicit native metadata RetryFunc advanced header policy remains native.
  Tests: `image/upload_image_options_test.go:TestImageUploadOptionsRetainedCallbacksAndSource`, `image/upload_image_options_test.go:TestImageUploadOptionsParallelReuseAndHeaderPolicies`, `image/upload_image_contracts_test.go:TestImageUploadContractsSourceAndBinaryPolicy`, `image/upload_image_contracts_test.go:TestImageUploadContractsNativeMetadataPolicy`, `connectiontest/connection_image_upload_test.go:TestConnectionImageUploadFacadeOwnsPhases`.
- Legacy typed strict201/204 upload profile only: Actual accepted POST201 raw Body/Header/Status remains in Metadata on read/Close/context/source/decode error. Reused atomic nullable25 ImageInfo owns raw Body+Properties independently without header pseudoJSON, seeded IDs or output enum/date parsing. Missing/null/empty/unsafe canonical ID retains created evidence and prevents PUT.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreCanonicalMetadataAndChosenID`, `image/upload_image_core_test.go:TestImageUploadCoreAcceptedPhaseFailures`, `image/upload_image_contracts_test.go:TestImageUploadContractsPartialEvidence`.
- Legacy typed strict201/204 upload profile only: Canonical ID alone selects once-escaped fixed file PUT; Location/self/file/incidental aliases cannot redirect. Safe Unicode/space/colon/percent/question/hash literals are retained while empty/dot/dotdot/slash/backslash/control/invalidUTF8 fail before binary use.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreCanonicalMetadataAndChosenID`, `image/upload_image_contracts_test.go:TestImageUploadContractsWireAndOwnership`, `image/upload_image_contracts_test.go:TestImageUploadContractsPartialEvidence`.
- Legacy typed strict201/204 upload profile only: Borrowed current-position binary reader exposes only Read and receives no Close/Seek/GetBody/Len/buffering. Exactly one SDK attempt preserves configured timeout/transport/liveauth and disables native reauth/retry/backoff/redirect; failures retain completed metadata/image and native causes without replay.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreNativePolicyAndNoReplay`, `image/upload_image_core_test.go:TestImageUploadCoreCancellationAndBorrowedFailures`, `image/upload_image_contracts_test.go:TestImageUploadContractsSourceAndBinaryPolicy`, `image/upload_image_contracts_test.go:TestImageUploadContractsPartialEvidence`.
- Legacy typed strict201/204 upload profile only: Nil size omits owned size header, zero/nonnegative int64 emits exact X-OpenStack-Image-Size only on PUT; WithoutSize clears prior size. No automatic file length/seek/hash/type inference. Python Image.upload omitted-size total-file inference is explicitly different.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreFixedRoutesAndIndependentEvidence`, `image/upload_image_options_test.go:TestImageUploadOptionsReplacementAndTypedPresence`, `image/upload_image_contracts_test.go:TestImageUploadContractsWireAndOwnership`.
- Legacy typed strict201/204 upload profile only: Only actual accepted204 creates independent opaque acknowledgement, retained with result and whole ResponseError on read/Close/context/custom cause/source failures; closes response body once/no resend. Native/generated Create+Upload and legacy Upload/optional wait compatibility remain unchanged; deprecated Python same mutable created Resource/session/cache semantics remain partial.
  Tests: `image/upload_image_core_test.go:TestImageUploadCoreAcceptedPhaseFailures`, `image/upload_image_core_test.go:TestImageUploadCoreNativePolicyAndNoReplay`, `image/upload_image_contracts_test.go:TestImageUploadContractsPartialEvidence`, `image/upload_image_contracts_test.go:TestImageUploadContractsNativeMetadataPolicy`, `internal/cmd/sdkgen/glance_image_upload_test.go:TestGlanceImageUploadPreservesNativeAndRegistry`.
- Capture service/API/Images/raw Provider/Endpoint/base/type/microversion, ordinary headers and sticky ancestor guard before caller marshaling, options or cloud-location callbacks. Owned Attrs/maps/raw bytes/helpers must be independently reusable; Connection and Service share this prepared source rather than a user builder.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadCapturesAttributesOptionsLocationAndHeadersOnce`, `image/image_record_upload_test.go:TestImageRecordUploadOptionMergeReplacementAndStickyCallbackGuards`, `image/image_record_upload_test.go:TestImageRecordUploadFactoryOptionsRemainReusableAfterResultAndConfigEdits`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadRejectsInvalidInvocation`.
- Pinned Proxy.upload_image emits RemovedInSDK50Warning first, then rejects falsey explicit disk/container parameters before private create. No required name, format enum validation or qcow2/bare/private defaults exist in this Source method. A Go-owned default profile must be named and documented; raw falsy/nonstring metadata values must not be rejected merely by the legacy final-format validator.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadFormalFormatsRequireTruthinessWithoutNameOrDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder`.
- Private Proxy._create handles truthy __conflicting_attrs by overwriting constructor attrs and removing the hook; empty/falsey hooks remain unknown metadata properties. bound base_path controls create route while inner conflicting base_path is ordinary metadata. resource_type/self/connection/_synchronized collisions fail in Source; microversion is constructor state later replaced by the negotiated create value. Go fixed target/control rejection is an explicit finite-domain difference, not full dynamic builder parity.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder`, `image/image_record_upload_test.go:TestImageRecordUploadRejectsFixedControlsAndInvalidConstructorBeforeHTTP`.
- Normalize all 64 inherited/declared Body fields, owner/owner_id wire alias, canonical and wire names, unknown-property packing, constructor import-method plain attribute and current location. Materialize the 65-field descriptor view before HTTP without writing getter-converted/default/null/tag values back into current/original/dirty components or submitted metadata.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAllDeclaredFieldsSubmitRawWireWithoutProjectionDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadDescriptorViewAndMetadataReceiptsRemainIndependent`, `image/image_record_upload_test.go:TestImageRecordUploadRejectsFixedControlsAndInvalidConstructorBeforeHTTP`.
- Unsynchronized Source _ComponentManager marks every supplied raw Body key dirty including id; _original_body separately exempts a truthy id. POST uses all constructor current Body values, not the update helper pending dirty set that excludes id. Flatten properties over top-level keys: a dict overwrites canonical raw keys, a truthy string remains properties, other truthy scalar and falsey values are omitted. Container/disk truthy parameter precheck is not repeated after conflicting/property overrides.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAllDeclaredFieldsSubmitRawWireWithoutProjectionDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder`, `image/image_record_upload_test.go:TestImageRecordUploadInvalidMetadataSyntaxPreservesConstructorDirtyIDAndLaterCommit`.
- Image Resource.create defaults POST with requires_id=False, resource_key=None, has_body=True/create_returns_body=None. Submit one flat JSON metadata POST on the captured source through normal metadata retry/auth policy; there is no initial GET/list/name resolution, duplicate check or queued-status condition.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadFormalFormatsRequireTruthinessWithoutNameOrDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadMetadataRetryKeepsSafeJSONScopeHeadersAndLiveAuth`.
- Accepted create response read/Close/context/source failures retain independently copied actual Metadata body/header/status and appropriate submitted Record state. A rejected/native failed POST cannot start data assignment/size inference/PUT; preserve native error causes and rejection body policy without automatic recovery GET or DELETE.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAcceptedPhysicalFailuresKeepActualPhaseReceipts`, `image/image_record_upload_test.go:TestImageRecordUploadRejectedMetadataNeverAssignsDataOrStartsBinary`.
- Resource._translate_response raises for HTTP >=400. Valid JSON object body discards self, normalizes aliases/unknown properties, overlays sparse raw constructor state and cleans the whole baseline. Empty valid {} still creates empty properties; actual response import-method header replaces the constructor list. Body/header/status/current location remain separate from Wire and raw metadata bytes.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAllDeclaredFieldsSubmitRawWireWithoutProjectionDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadDescriptorViewAndMetadataReceiptsRemainIndependent`, `image/image_record_upload_test.go:TestImageRecordUploadCreationResponseMayReplaceArbitraryInitialIdentity`, `image/image_record_upload_test.go:TestImageRecordUploadAcceptedBoundariesKeepOpaquePUTAndNeverFetch`.
- Source catches ValueError for invalid JSON syntax and still translates headers/view without overlaying or cleaning pending constructor Body. Valid non-object JSON or invalid projected descriptors are errors. Go strict UTF-8/object/Unicode bounds and atomic partial receipts must be explicit and tested, including input id retained through a tolerated invalid create response.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadInvalidMetadataSyntaxPreservesConstructorDirtyIDAndLaterCommit`, `image/image_record_upload_test.go:TestImageRecordUploadMetadataTranslationFailuresKeepActualCreatedEvidence`.
- Use the private post-creation raw id, which may come from a response overlay or an unchanged constructor seed, to choose a once-escaped fixed /images/{id}/file route. Location/self/file/public projected view must not redirect. Source raw urljoin accepts missing/non-string/slashed ids differently; Go nonblank exact Unicode-string/control/dot validation is a documented bound. Identity failure retains created metadata before any binary IO.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadCreationResponseMayReplaceArbitraryInitialIdentity`, `image/image_record_upload_test.go:TestImageRecordUploadIdentityValidationOccursOnlyAfterMetadataAndBeforeBinaryIO`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.
- Only after successful metadata translation does Proxy overwrite img.data with the passed data unconditionally, including None/falsey. Plain data is outside Body, dirty state and HTTP receipts. nil Go Data performs an empty PUT; it is not a required Reader and cannot accidentally retain prior/fallback data. Bytes/strings are caller io.Reader adapters in Go; typed-nil is a local concrete-input bound.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadFormalFormatsRequireTruthinessWithoutNameOrDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadRejectsFixedControlsAndInvalidConstructorBeforeHTTP`, `image/image_record_upload_test.go:TestImageRecordUploadBorrowedReadAndTransportFailuresKeepCreatedRecord`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.
- Deprecated upload has no Filename parameter or SDK-owned file helper. Constructor Attributes filename remains ordinary metadata and never opens or redirects a file; a caller-open io.Reader file is borrowed at its current cursor and never closed by this helper. Open/Close responsibility stays with the caller.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder`, `image/image_record_upload_test.go:TestImageRecordUploadCallerOpenedFilenameRemainsBorrowedAndNeverReopened`.
- After metadata creation/data assignment, nil Size defaults to total seekable-file length, restores the original cursor and sends remaining bytes from the original position. Non-seekable data and ESPIPE leave size unknown; other seek errors stop before PUT and retain metadata. Reuse the guarded Stage inference engine; its extra restoration after failure and io.Seeker interface bound are explicit lifecycle differences.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadInfersTotalSizeAfterPOSTAndRestoresBorrowedCursor`, `image/image_record_upload_test.go:TestImageRecordUploadSeekFailuresRetainCreatedMetadataAndESPIPEUnknown`.
- Explicit signed int64 Size preserves negative/zero/positive exact X-OpenStack-Image-Size and bypasses inference. WithoutSize clears only explicit Size and preserves the independent DisableSizeInference flag; default inference resumes when enabled. The inference optout suppresses all Seek/Len inspection. Source isinstance(int) bool/subclasses/Python str(True/False) are outside typed Go int64; Source size validation occurs after metadata creation.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadInfersTotalSizeAfterPOSTAndRestoresBorrowedCursor`, `image/image_record_upload_test.go:TestImageRecordUploadSeekFailuresRetainCreatedMetadataAndESPIPEUnknown`.
- Reuse imageRecordDataOnce for octet-stream Content-Type, empty Accept, signed optional size and a Read-only borrowed view. Fix the route and source, retain live provider auth and original transport/timeout, disable binary reauth/retry/backoff/redirect/GetBody and perform one consumed-stream attempt. Native HTTP rejection, transport/reader errors and sticky guards cannot cause semantic replay.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAcceptedBoundariesKeepOpaquePUTAndNeverFetch`, `image/image_record_upload_test.go:TestImageRecordUploadCapturesAttributesOptionsLocationAndHeadersOnce`, `image/image_record_upload_test.go:TestImageRecordUploadMetadataRetryKeepsSafeJSONScopeHeadersAndLiveAuth`, `image/image_record_upload_test.go:TestImageRecordUploadBinaryRejectionsNeverReplayReauthenticateOrBackoff`, `image/image_record_upload_test.go:TestImageRecordUploadBorrowedReadAndTransportFailuresKeepCreatedRecord`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.
- Pinned Image.upload returns the session.put Response without _translate_response or exceptions.raise_from_response; Proxy discards it and returns the same created Image. Go returns explicit native rejection errors plus owned actual Upload receipt. Accepted PUT body including malformed/non-UTF8 bytes is opaque and cannot replace/clean Body or import methods/location/status/header/Envelope/Wire from metadata creation.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadAcceptedBoundariesKeepOpaquePUTAndNeverFetch`, `image/image_record_upload_test.go:TestImageRecordUploadBinaryRejectionsNeverReplayReauthenticateOrBackoff`.
- After accepted metadata, size/identity/file/read/transport/rejection/accepted-PUT handling errors retain the created Record and Metadata independently; retain Upload only for actual accepted response evidence. No uploading/active synthetic status, metadata rollback/delete, checksum verification, polling, import/task/Swift branch or compensating fetch is present.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadIdentityValidationOccursOnlyAfterMetadataAndBeforeBinaryIO`, `image/image_record_upload_test.go:TestImageRecordUploadSeekFailuresRetainCreatedMetadataAndESPIPEUnknown`, `image/image_record_upload_test.go:TestImageRecordUploadAcceptedPhysicalFailuresKeepActualPhaseReceipts`, `image/image_record_upload_test.go:TestImageRecordUploadBinaryRejectionsNeverReplayReauthenticateOrBackoff`, `image/image_record_upload_test.go:TestImageRecordUploadBorrowedReadAndTransportFailuresKeepCreatedRecord`.
- Neither Proxy.upload_image nor Image.upload fetches after PUT. Resource.create conditional fetch requires create_returns_body is False, but pinned Image inherits None, so the branch is unreachable for this selected graph. The created clean or pending state and import-method list must survive accepted or rejected PUT; explicit caller Get/Wait is a separate operation.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadInvalidMetadataSyntaxPreservesConstructorDirtyIDAndLaterCommit`, `image/image_record_upload_test.go:TestImageRecordUploadAcceptedBoundariesKeepOpaquePUTAndNeverFetch`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.
- Forward arbitrary owned metadata attrs including owner/public/community visibility and properties to the server; do not invent client RBAC/enum/range gates. Pinned add_image/upload_image are core user operations, public visibility/foreign owner admin branches are in the same metadata signature and community is member/admin. Retain borrowed data across later owned immutable operations; exact caller lifetime/cursor is caller-owned.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadCapturesAttributesOptionsLocationAndHeadersOnce`, `image/image_record_upload_test.go:TestImageRecordUploadBinaryRejectionsNeverReplayReauthenticateOrBackoff`, `image/image_record_upload_test.go:TestImageRecordUploadCallerOpenedFilenameRemainsBorrowedAndNeverReopened`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.
- Preserve the 10 legacy UploadImage contracts, 17 test references and public API profile; add complete owned UploadImageRecord/metadata foundation rather than changing generated/native strict201/204 behavior. Promote only selected python:image/v2/upload_image after actual focused/full45/generator/guide/install gates. Full modern public create_image remains unresolved 12-contract/20-file graph; mutable class extensions R1, cache C1 and dynamic Adapter/session S1 remain explicit shared work.
  Tests: `image/image_record_upload_test.go:TestImageRecordUploadFormalFormatsRequireTruthinessWithoutNameOrDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadFactoryOptionsRemainReusableAfterResultAndConfigEdits`, `connectiontest/image_record_upload_integration_test.go:TestConnectionOwnedImageUploadSharesSourceAndBorrowedDataThroughImportUpdateAndStage`.

Python/Go differences:

- Complete finite owned deprecated Proxy.upload_image graph: full64 raw Body/65 descriptor view, required truthy raw formal formats without defaults, private Proxy._create/Resource.create constructor and flat metadata POST, sparse translated or pending invalid-JSON response, unconditional borrowed data assignment and default total-length inference followed by one opaque PUT. This go_mapping is not full SDK supported parity.
- Legacy typed strict201/204 upload profile only: Pinned deprecated Proxy.upload_image requires explicit truthy disk/container formats and returns the same mutable creation Resource after Image.upload; Go defaults qcow2/bare/private, requires valid Name/Reader, and returns owned actual phase evidence with precise ImageInfo.
- Legacy typed strict201/204 upload profile only: Python Image.upload may seek a seekable source to infer total length and restore its cursor; Go performs no length inference/Seek/Close and uses explicit nil/zero/nonnegative size only on binary PUT.
- Legacy typed strict201/204 upload profile only: Python Resource descriptors, aliases/type conversions, Adapter session/header translation and cached mutable model semantics differ from exact raw JSON and independent snapshots. Full parity remains unresolved.
- Legacy typed strict201/204 upload profile only: Task/Swift/checksum/import/wait/failed-upload cleanup belong separate create_image private helpers, not this deprecated proxy declaration. No synthetic operation or native/resource inventory additions.
- Go uses immutable owned JSON-domain Records/options/phase receipts; Source returns the same mutable Image with dynamic class/descriptors/plain data. Shared dynamic Resource extension R1, optional cache C1 and complete Adapter/discovery/session S1 remain unresolved globally.
- Formal Go format options own same-named initial Attributes keys because a Go map cannot model Python duplicate argument binding. Source __conflicting_attrs and properties flattening still override the prepared raw metadata. Name/data/visibility/formats have no implicit Go defaults in this owned profile; legacy typed upload retains its separate defaults and required Name/Reader behavior.
- Source Resource.create translates and raises HTTP>=400, while Image.upload returns an untranslated session Response under Proxy.request raise_exc=False. Go accepts actual200..399 and returns explicit native400..599 errors with creation evidence; binary retry/reauth/backoff/redirect are disabled to avoid replaying borrowed streams. Metadata keeps safe native JSON retry/auth.
- Go fixes the captured service target and validates the translated private literal Unicode-string id after accepted metadata; Source urljoin supports additional dynamic/nonstring/slash behavior. Top base_path/resource_type/self/connection/_synchronized/microversion controls are bounded; inner conflicting base_path/resource_type remain ordinary metadata and constructor collisions remain local errors.
- Data is borrowed io.Reader or nil for an empty PUT, with caller-owned file lifetime; Attributes filename remains metadata. Go signed int64 size and io.Seeker inference do not represent Python bool/int-subclass/string/bytes/custom-file dynamic semantics. Cursor cleanup also restores after end-seek failures; explicit inference optout and independently clearing Size are Go options.
- Pinned deprecated upload and constructor _compute_attributes emit Python deprecation warnings. The new Go owned API does not reproduce Python warning/filter behavior. It performs no initial/final GET, PUT translation, status inference, checksum, task/Swift/import/wait or automatic rollback.
- The full20-file/12-contract modern public create_image graph and separate download_image graph remain unresolved and unchanged. Closing this private metadata/create plus deprecated upload mapping does not promote those operations, dynamic Source objects, authenticated backend policy or universal runtime parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
with open("ubuntu.qcow2", "rb") as source:
    record = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="ubuntu-example",
        visibility="private",
        filename="ubuntu.qcow2",
    )
print(record.id, record.status)
```

```sh
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -service -size 0
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -infer-size=false
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
    "path/filepath"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    filename := flag.String("file", "", "required caller-owned image file")
    name := flag.String("name", "ubuntu-example", "image metadata name")
    visibility := flag.String("visibility", "private", "server image visibility")
    container := flag.String("container-format", "bare", "explicit image container format")
    disk := flag.String("disk-format", "qcow2", "explicit image disk format")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    size := flag.Int64("size", 0, "explicit signed size hint; omission uses inference")
    inferSize := flag.Bool("infer-size", true, "infer total seekable length when size is omitted")
    flag.Parse()
    if *filename == "" { log.Fatal("-file is required by this example") }
    sizePresent := false
    flag.Visit(func(value *flag.Flag) {
        if value.Name == "size" { sizePresent = true }
    })
    options := []image.ImageRecordUploadOption{
        image.WithImageRecordUploadOpts(image.ImageRecordUploadOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-upload-example"},
        }),
        image.WithImageRecordUploadContainerFormat(*container),
        image.WithImageRecordUploadDiskFormat(*disk),
        image.WithImageRecordUploadSizeInference(*inferSize),
    }
    if sizePresent { options = append(options, image.WithImageRecordUploadSize(*size)) }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *filename, *name, *visibility, *viaService, options); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, filename, name, visibility string, viaService bool, options []image.ImageRecordUploadOption) (err error) {
    source, err := os.Open(filename)
    if err != nil { return err }
    defer func() { err = errors.Join(err, source.Close()) }()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordUploadRequest{
        Data: source,
        Attributes: map[string]any{
            "name": name, "visibility": visibility,
            // This is metadata; the SDK does not open a filename attribute.
            "filename": filepath.Base(filename),
        },
    }
    var result *image.ImageRecordUploadResult
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        result, err = service.UploadImageRecord(ctx, input, options...)
    } else {
        result, err = conn.UploadImageRecord(ctx, input, options...)
    }
    return printUpload(result, err)
}

func printUpload(result *image.ImageRecordUploadResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        for name, receipt := range map[string]*image.ImageUploadResponse{
            "metadata": result.Metadata, "uploaded": result.Uploaded,
        } {
            if receipt != nil {
                value[name] = map[string]any{
                    // Response bytes may be empty or non-JSON; []byte is base64.
                    "body": receipt.Body, "header": receipt.Header,
                    "status_code": receipt.StatusCode,
                }
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource, "wire": result.Record.Wire,
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

- [Image Record Waits](image-record-waits.md)
- [Install](../docs/install.md)
- [Upload Image](upload-image.md)
- [Create Import](create-import.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
