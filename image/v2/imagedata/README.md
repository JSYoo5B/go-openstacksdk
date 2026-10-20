# Imagedata

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/stage_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.StageImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.StageKnownImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.WithStageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.WithStageSize`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.WithStageHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.WithStageHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.StageImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.StageImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordStageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordStageHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordStageHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordStageSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithoutImageRecordStageSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordStageSizeInference`

- Legacy native/ref strict200/204 stage profile only: StageImage performs a fresh explicit-ID metadata GET before streaming, while StageKnownImage snapshots only supplied native ID/status before option callbacks and skips that initial GET. Both require exact queued status before PUT and leave the supplied model unchanged.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageFreshKnownAndBorrowedReader`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`, `image/v2/imagedata/stage_core_test.go:TestStageCoreFreshCanonicalQueuedAndWholeNativeDecode`.
- Legacy native/ref strict200/204 stage profile only: Explicit Name uses the existing Images exact-name native page resolver once, followed by a fresh metadata GET. Requested or resolved ImageID remains fixed even when accepted metadata or Location describes another ID.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageQueuedNativePreflightAndNameBinding`.
- Legacy native/ref strict200/204 stage profile only: Initial metadata is a flat UTF-8 object accepted as200, decoded as a complete native Image before exact lowercase canonical status is required. Missing/null/case-only/nonstring/nonqueued status or unrelated native decode failure prevents PUT and reader consumption.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageQueuedNativePreflightAndNameBinding`, `image/v2/imagedata/stage_core_test.go:TestStageCoreFreshCanonicalQueuedAndWholeNativeDecode`.
- Legacy native/ref strict200/204 stage profile only: The SDK borrows a nonnil reader through a Read-only view, streaming from its current position without caller Close, Seek, size inspection or buffering. Typed-nil input and canceled preflight fail before HTTP or reading.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageClosedOptionsAndPreflight`, `api/glance_stage_contracts_test.go:TestGlanceStageStreamFailuresAndCallerCancellation`, `api/glance_stage_contracts_test.go:TestGlanceStageFreshKnownAndBorrowedReader`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`, `image/v2/imagedata/stage_core_test.go:TestStageCorePreflightSourceAndCustomCancellationCauses`.
- Legacy native/ref strict200/204 stage profile only: Size is an optional nonnegative int64 pointer: nil omits and zero is preserved in X-OpenStack-Image-Size only. No Content-Length or whole-file length is inferred, and no caller data is truncated to the stated size.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageSizeAndOptionSnapshots`, `image/v2/imagedata/stage_options_test.go:TestStageOptionsSizeNilZeroAndInvalidConfiguration`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`.
- Legacy native/ref strict200/204 stage profile only: Binary PUT uses a private fixed-target Provider with reauthentication, retry, backoff and redirects disabled, preserving the original client settings. Partial consumption followed by native status, transport or reader failure is not semantically replayed or followed by metadata GET.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageStreamFailuresAndCallerCancellation`, `api/glance_stage_contracts_test.go:TestGlanceStagePutNeverReplaysOrInvokesProviderCallbacks`, `image/v2/imagedata/stage_core_test.go:TestStageCoreBinaryFailuresDoNotReplayOrRedirect`.
- Legacy native/ref strict200/204 stage profile only: StageImageResult retains independently copied actual PUT204 Acknowledgement evidence. Outer Image is assigned only after successful final native GetResult extraction, including import-method/store-ID header projection; outer Body/Header/StatusCode retain original accepted final GET200 evidence even on read or decode errors, unaffected by that model projection.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageLiveProviderFixedRoutesAndNativeABI`, `api/glance_stage_contracts_test.go:TestGlanceStageAcceptedAndFollowupEvidence`, `api/glance_stage_contracts_test.go:TestGlanceStageFreshKnownAndBorrowedReader`, `image/v2/imagedata/stage_core_test.go:TestStageCoreAcknowledgementAndFollowupFailureEvidence`, `image/v2/imagedata/stage_core_test.go:TestStageCoreGETHeaderProjectionMatchesNativeAndPreservesRawEvidence`.
- Legacy native/ref strict200/204 stage profile only: Binary octet-stream Content-Type and empty Accept are SDK-owned, distinct from captured ordinary metadata headers. Size/auth/framing/version overrides are guarded; ordinary header helpers canonicalize aliases with sequential last-wins and reject conflicting map aliases.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageLiveProviderFixedRoutesAndNativeABI`, `api/glance_stage_contracts_test.go:TestGlanceStageClosedOptionsAndPreflight`, `image/v2/imagedata/stage_options_test.go:TestStageOptionsHeaderAuthorityAndCanonicalLastWins`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`.
- Legacy native/ref strict200/204 stage profile only: WithStageOpts snapshots Size and header maps and replaces the complete config; individual helpers apply afterward. Parsed custom options and reusable concurrent options remain independent; each caller retains ownership of its reader.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageLiveProviderFixedRoutesAndNativeABI`, `api/glance_stage_contracts_test.go:TestGlanceStageSizeAndOptionSnapshots`, `image/v2/imagedata/stage_options_test.go:TestStageOptionsOwnedSnapshotsReplacementAndConcurrentReuse`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`.
- Legacy native/ref strict200/204 stage profile only: Before any accepted PUT204 the result is nil. Accepted PUT read or followup GET failures return the actual acknowledgement plus original errors without synthetic uploading/active models, extra recovery GET, import POST or cleanup DELETE. Context causes remain inspectable alongside original failures.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageClosedOptionsAndPreflight`, `api/glance_stage_contracts_test.go:TestGlanceStageStreamFailuresAndCallerCancellation`, `api/glance_stage_contracts_test.go:TestGlanceStageAcceptedAndFollowupEvidence`, `image/v2/imagedata/stage_core_test.go:TestStageCoreAcknowledgementAndFollowupFailureEvidence`, `image/v2/imagedata/stage_core_test.go:TestStageCorePreflightSourceAndCustomCancellationCauses`.
- Legacy native/ref strict200/204 stage profile only: Source client/provider/type/header/origin validity is checked before options and subsequent phases. Captured ServiceClient routes, Microversion and header maps remain fixed; the original Provider token is read at dispatch. Each metadata GET keeps the configured JSON retry/auth policy and selected HTTPClient transport/timeout, while binary PUT disables replay callbacks on its private provider.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageClosedOptionsAndPreflight`, `api/glance_stage_contracts_test.go:TestGlanceStageLiveProviderFixedRoutesAndNativeABI`, `image/v2/imagedata/stage_core_test.go:TestStageCorePreflightSourceAndCustomCancellationCauses`, `image/v2/imagedata/stage_core_test.go:TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases`.
- Legacy native/ref strict200/204 stage profile only: Existing native Stage/Upload/Download functions, StageResult alias and specialized inventory placeholder remain separate. Actual compiled signature and canonical Image ID/status tag guards reject drift without inventing a resource collection or changing generated native declarations.
  Tests: `api/glance_stage_contracts_test.go:TestGlanceStageLiveProviderFixedRoutesAndNativeABI`, `internal/cmd/sdkgen/glance_stage_test.go:TestGlanceStageKeepsActualNativeStreamingAndNoInventedCollection`, `internal/cmd/sdkgen/glance_stage_test.go:TestGlanceStageRejectsNativeSignatureIdentityAndStatusDrift`.
- One captured owned ID/Record seed uses private current identity and source snapshots before callbacks. There is no initial GET, name lookup, store/import discovery or developer builder. Ambiguous selectors, non-SDK Record, typed-nil reader, unsafe fixed identity, nil context/source/option and protected headers are local Go errors before data IO or HTTP.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageRejectsInvalidInputsAndOwnedHeadersBeforeIO`, `image/image_record_stage_test.go:TestImageRecordStageCapturesRequestOptionsLocationAndOrdinaryHeadersOnce`.
- The _get_resource existing-update branch is mapped through full current65 descriptor projection, captured current location and ImportMethods reset before queued status, filename opening, size inference or PUT. Private current/original/dirty body and actual prior receipts are owned; public Record/receipt edits are passive and cannot forge routing or the pending seed.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageProjectsPrivateDescriptorsBeforeQueuedOrHTTP`, `image/image_record_stage_test.go:TestImageRecordStageCapturesRequestOptionsLocationAndOrdinaryHeadersOnce`.
- The exact canonical cached status must be the string queued. Missing/null/nonstring/different case/nonqueued status and a literal-ID constructor seed fail before data IO or HTTP. A caller can explicitly GetImageRecord first; this public helper does not fetch status or synthesize uploading/active.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageRequiresExactQueuedBeforeAnyDataIO`, `image/image_record_stage_test.go:TestImageRecordStageProjectsPrivateDescriptorsBeforeQueuedOrHTTP`.
- Explicit borrowed Data replaces retained private data; nil Data falls back to private borrowed data or sends an empty PUT. Plain data is outside Resource/Wire/body fields. A borrowed reader keeps its current cursor, is never SDK-closed, and remains available through returned owned records and subsequent immutable operations.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageNilDataAndBorrowedFallbackPreserveCurrentCursor`, `image/image_record_stage_test.go:TestImageRecordStageFilenameAndBorrowedFileHaveSeparateLifetimes`.
- A nonempty Filename is mutually exclusive with explicit Data and opens only after prepared exact queued state. The helper owns and closes the opened file after the whole workflow, including final metadata GET and all failures, and never retains a closed owned file in a returned record. Open/close failures preserve native causes; caller files supplied as Data remain borrowed.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageFilenameAndBorrowedFileHaveSeparateLifetimes`, `image/image_record_stage_test.go:TestImageRecordStageRequiresExactQueuedBeforeAnyDataIO`.
- Default size inference probes io.Seeker current position, total end and restores the captured position. It uses total length rather than remaining bytes. ESPIPE means unavailable size; other seek errors and sticky guard failures stop PUT. Go attempts cursor cleanup after an end-seek failure, a documented improvement over Source get_file_size without finally; cleanup never clears a terminal guard/error.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageSizePolicyUsesTotalLengthWithoutChangingCursor`, `image/image_record_stage_test.go:TestImageRecordStageSizeInferenceKeepsSeekFailuresAndESPIPEUnknown`.
- Explicit signed int64 Size, including zero and negative values, emits exact decimal X-OpenStack-Image-Size only on PUT and skips seeking. It sets neither Content-Length nor a truncation limit. A nil Size infers by default; WithImageRecordStageSizeInference(false) never seeks. Python bool and unbounded integer sizes are explicit typed representation differences.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageSizePolicyUsesTotalLengthWithoutChangingCursor`.
- The six concrete factories own headers and size pointers. Whole opts replaces complete configuration; singular/plural headers merge with canonical sequential last-wins. WithoutImageRecordStageSize clears only explicit Size and preserves inference policy; WithImageRecordStageSizeInference(true) reenables default inference. Parsed custom callbacks run once with owned before/after snapshots and sticky context/source/outer guards.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageOptionsKeepBulkMergeFullReplacementAndGuardOwnership`, `image/image_record_stage_test.go:TestImageRecordStageCapturesRequestOptionsLocationAndOrdinaryHeadersOnce`.
- Exactly one SDK binary PUT uses captured fixed /images/{privateID}/stage, once-escaped identity, octet-stream Content-Type and empty Accept. Captured ordinary headers and original live authentication/transport/timeout remain distinct. The binary provider disables SDK retry, reauth, backoff and redirects; accepted-body or reader failure never replays consumed data.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageFixedRoutesAndOpaqueAcceptedReceipts`, `image/image_record_stage_test.go:TestImageRecordStageBinaryRejectionsNeverReplayReauthenticateOrBackoff`, `image/image_record_stage_test.go:TestImageRecordStageBorrowedReadErrorsAndTransportCausesStopFinalFetch`.
- Actual accepted200..399 PUT produces independent Staged Body/Header/Status evidence. The opaque acknowledgement may be empty, invalid UTF8, malformed JSON or any JSON shape and is not decoded as image metadata. Location/self/file and incidental acknowledgement body identity cannot select a new route or fabricate a body/status baseline.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageFixedRoutesAndOpaqueAcceptedReceipts`, `image/image_record_stage_test.go:TestImageRecordStageAcceptedPUTHandlingFailuresRetainOnlyActualStageReceipt`.
- Unlike silent action/import translation, Image.stage calls _translate_response(has_body=False), whose raise_from_response rejects actual400..599 before final fetch. Native rejection/reader/transport causes are retained without accepted-stage invention or automatic recovery GET. Physical read/Close/source/context handling failure after acceptance retains only real Staged acknowledgement and terminal causes.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageBinaryRejectionsNeverReplayReauthenticateOrBackoff`, `image/image_record_stage_test.go:TestImageRecordStageBorrowedReadErrorsAndTransportCausesStopFinalFetch`, `image/image_record_stage_test.go:TestImageRecordStageAcceptedPUTHandlingFailuresRetainOnlyActualStageReceipt`.
- Successful stage header-only translation consumes import-method header with literal comma splitting and updates owned Header/Status/ImportMethods while retaining private current/original/dirty body state. Record Envelope contains the actual opaque stage ACK and Wire is nil; it does not clean body state or decode that ACK as image metadata. Staged remains independent response evidence and the caller seed remains unchanged.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageInvalidFetchPreservesPendingBodyForLaterCommit`, `image/image_record_stage_test.go:TestImageRecordStageFinalFetchErrorsKeepBothPhaseEvidenceAndPendingSeed`.
- After successful PUT handling and header-only translation, one mandatory seeded metadata fetch targets the same captured private ID. No initial GET, name resolution, polling, import POST, wait, checksum or cleanup DELETE is added. Final GET retains the normal safe native JSON auth/retry policy, independently from the binary one-attempt policy.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageFixedRoutesAndOpaqueAcceptedReceipts`, `image/image_record_stage_test.go:TestImageRecordStageFinalFetchRetainsSafeNativeRetryFixedTargetAndLiveAuth`, `image/image_record_stage_test.go:TestImageRecordStageRejectedBodyCloseAndFinalFetchReauthKeepNativePolicy`.
- Actual accepted200..399 final GET Metadata is independent from Staged. Valid JSON object overlays the prepared current seed, applies descriptor/unknown-property/header projection and cleans the baseline. Invalid JSON syntax tolerates and preserves pending current/original/dirty state for later commit; invalid object shape/conversion errors preserve real response evidence and causes without fabricated completion.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageValidFetchOverlaysSeedAndCleansRawBody`, `image/image_record_stage_test.go:TestImageRecordStageInvalidFetchPreservesPendingBodyForLaterCommit`, `image/image_record_stage_test.go:TestImageRecordStageFinalFetchErrorsKeepBothPhaseEvidenceAndPendingSeed`.
- Final fetch rejection, accepted read/Close/parser, original-source/outer/context failures and owned filename cleanup preserve actual prior phase receipts and inspectable native causes, without replay or extra recovery requests. Before successful final translation, a prepared pending seed is retained when available; independently owned receipts cannot mutate Record state. Physical response bodies close once and native rejection owns rejected bodies.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageAcceptedPUTHandlingFailuresRetainOnlyActualStageReceipt`, `image/image_record_stage_test.go:TestImageRecordStageFinalFetchErrorsKeepBothPhaseEvidenceAndPendingSeed`, `image/image_record_stage_test.go:TestImageRecordStageFilenameAndBorrowedFileHaveSeparateLifetimes`.
- Service and Connection share captured Image/ImageV2 source, current location, fixed identity and cause wrapping. The executable public guide and declared tests cover owned callbacks/headers/receipts/data lifecycle, while existing native/ref StageImage, StageKnownImage and generated Stage signatures remain a separate legacy profile. This finite immutable JSON mapping leaves SDK-R1/C1/S1 and other image workflows explicit.
  Tests: `image/image_record_stage_test.go:TestImageRecordStageOptionsKeepBulkMergeFullReplacementAndGuardOwnership`, `image/image_record_stage_test.go:TestImageRecordStageCapturesRequestOptionsLocationAndOrdinaryHeadersOnce`, `image_record_stage_integration_test.go:TestConnectionOwnedImageStageSharesSourceAndRetainsBorrowedDataThroughProperties`, `image_record_stage_integration_test.go:TestConnectionOwnedImageStageRejectsInvalidInvocation`.

Python/Go differences:

- Principal public graph: Source checks exact cached queued status after _get_resource without initial GET, filename/data selection, automatic total-file size, one stage PUT/header-only translation, then mandatory seeded fetch. Owned StageImageRecord implements that complete finite graph on SDK-produced immutable private JSON state. Literal ID fails queued preflight; callers explicitly fetch first. Existing native/ref StageImage and StageKnownImage remain the strict200/204 legacy profile, with their previous API/test references retained and scoped.
- Legacy native/ref strict200/204 stage profile only: Pinned Python _get_resource does not GET a bare ID, and Proxy.stage_image checks the supplied mutable Image cached status. Go StageImage(Ref) freshly GETs metadata, with an explicit Name convenience resolver; StageKnownImage copies only native ID/status and skips the initial GET.
- Legacy native/ref strict200/204 stage profile only: Python filename and existing Resource.data inputs are separate from explicit borrowed io.Reader. Python infers seekable whole-file size, restoring its cursor, and accepts bool or negative int; Go omits automatic size and accepts only optional nonnegative int64, preserving zero and current-position streaming.
- Legacy native/ref strict200/204 stage profile only: The Go binary provider disables SDK reauthentication, retry, backoff and redirects to prevent reader replay. Caller RoundTripper internals and release of a blocking Read remain outside the one-SDK-PUT guarantee; the underlying reader is never closed or sought.
- Legacy native/ref strict200/204 stage profile only: Go keeps actual accepted PUT204 Acknowledgement evidence separate from the final GET model/response, and can return partial evidence with an error. Python mutates and fetches the same Resource and returns it, without this owned partial-result structure.
- Legacy native/ref strict200/204 stage profile only: Go fresh metadata uses strict accepted200, full native Image decoding, UTF-8 and canonical queued status; native extension map/number/timestamp projection remains native. Safe fixed routes and protected SDK-owned options/header namespaces are deliberate Go policies rather than arbitrary injected builders.
- Source returns and mutates the same arbitrary Resource/Munch/subclass, dynamically reads descriptor/header/URI/original/dirty state and current location, and retains opened filename data without closing in this helper. Go snapshots private current/original/dirty body and actual receipts, captures one current location and fixed source/identity, accepts explicit io.Reader/string Filename, closes owned files after final GET/all failures and clears closed-file data. Borrowed data is retained by interface and never SDK-closed. Arbitrary Python data truthiness and PathLike/runtime descriptors remain representation/lifecycle bounds.
- Source omitted size calls get_file_size only on truthy seekable()/tell()/seek() data, returns total file size, restores only the successful sequence, and treats OSError errno ESPIPE as None without finally. Go default inference uses io.Seeker current/end/start, includes empty/nontruthy Go reader objects, and attempts restore even after end error; ESPIPE means unknown while other errors/guard changes are terminal. Explicit signed int64 incl0/negative is sent decimal; Python int subclasses bool yield True/False header and bigint is unbounded. WithImageRecordStageSizeInference(false) opts out, WithoutSize clears explicit Size while preserving inference. The older four-flow audit proposal of opt-in compatibility inference is superseded only for this owned Stage unit.
- Stage calls _translate_response(has_body=False): actual400..599 raises and prevents final fetch. Accepted200..399 opaque bytes/headers/status remain independent Staged; body is not JSON-decoded and pending body stays dirty. The owned Record additionally stores actual stage ACK Envelope/Header/Status with Wire=nil rather than carrying a previous GET receipt forward. Final accepted200..399 metadata JSON object overlays/cleans while malformed syntax preserves pending body; shape/conversion and physical handling errors return owned partial receipts/seed with inspectable causes. Source mutates before raising and exposes no equivalent immutable partial-result structure; HTTP1xx/arbitrary dynamic adapter response domains are not this fixed protocol.
- Captured fixed private ID is path-escaped once and protected headers/constructor controls prevent arbitrary session/URL injection. Binary PUT disables SDK retry/reauth/backoff/redirect replay; final GET keeps ordinary safe native JSON policy and original provider live token. Source Adapter skip_cache=False/cache invalidation/metrics/global request/custom microversion sessions and arbitrary response exception behavior remain SDK-C1/S1, and full Resource class/constructor/backing lifecycle remains SDK-R1, as documented in docs/implementation-plan.md. A custom RoundTripper may replay internally and a blocked caller Read still needs caller cooperation.
- Default project-scoped stage policy is modify_image with ADMIN_OR_PROJECT_MEMBER plus repository access; stage deserializer requires deployed glance-direct and octet-stream. Server status transition, staging store/quota/capacity/size enforcement and deployment policy overrides are not emulated or validated by local SDK tests. This finite owned go_mapping does not promote public create_image, deprecated upload_image, import_image or download_image or claim authenticated cloud success.

### `gophercloud:image/v2/imagedata.Upload`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.Upload`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.RawClient`

- Upload PUTs the caller reader once to images/{raw imageID}/file with application/octet-stream and service MoreHeaders, ignores the response body and accepts only 204; other statuses keep native ErrUnexpectedResponseCode under Upload/imagedata context without an owned receipt.
  Tests: `image/v2/imagedata/native_contracts_test.go:TestNativeImageDataUploadAndStage`.

Python/Go differences:

- This declaration maps the Gophercloud native data call only; it is not Python upload/stage/download parity, which the owned image record data workflows cover.
- The SDK adds only request.Wrap operation context and, for Download, the request.Download wrapper whose Header field repeats the native body; native URLs, OkCodes, retries and stream ownership are unchanged.

### `gophercloud:image/v2/imagedata.Stage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.Stage`

- Stage PUTs the caller reader once to images/{raw imageID}/stage with application/octet-stream, ignores the response body and accepts only 204 under Stage/imagedata context.
  Tests: `image/v2/imagedata/native_contracts_test.go:TestNativeImageDataUploadAndStage`.

Python/Go differences:

- This declaration maps the Gophercloud native data call only; it is not Python upload/stage/download parity, which the owned image record data workflows cover.
- The SDK adds only request.Wrap operation context and, for Download, the request.Download wrapper whose Header field repeats the native body; native URLs, OkCodes, retries and stream ownership are unchanged.

### `gophercloud:image/v2/imagedata.Download`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.Download`

- Download GETs images/{imageID}/file, accepts only 200 and hands the open native body to the caller without reading or closing it; the returned Header field is the same ReadCloser from native Extract. Rejected statuses such as 204 and 206 return a Download/imagedata error and no stream.
  Tests: `image/v2/imagedata/native_contracts_test.go:TestNativeImageDataDownloadOwnershipAndStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native data call only; it is not Python upload/stage/download parity, which the owned image record data workflows cover.
- The SDK adds only request.Wrap operation context and, for Download, the request.Download wrapper whose Header field repeats the native body; native URLs, OkCodes, retries and stream ownership are unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func stageExistingImage(ctx context.Context, client *gophercloud.ServiceClient, id string, data io.Reader) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageImage(ctx, resource.ID(id), data)
}
```

```python
image = conn.image.get_image(image_id)
image = conn.image.stage_image(image, data=data)
```

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
)

func stageKnownImage(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, data io.Reader, remainingBytes int64) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageKnownImage(ctx, image, data,
        imagedata.WithStageSize(remainingBytes),
        imagedata.WithStageHeader("X-Trace", "stage-image"),
    )
}
```

```python
image = conn.image.stage_image(image, data=data, size=remaining_bytes)
```

## References

- [Proxy.stage_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image.stage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [get_file_size](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#stage-binary-image-data)
- [Imageimport](../imageimport/README.md)
- [Glance Stage Contracts Test](../../../api/glance_stage_contracts_test.go)
- [Stage Core Test](stage_core_test.go)
- [Stage Options Test](stage_options_test.go)
- [Glance Stage Test](../../../internal/cmd/sdkgen/glance_stage_test.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/imagedata/requests.go)
- [Image Record Download](../../image-record-download.md)
