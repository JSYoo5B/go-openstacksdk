# Image Record Stage

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
record = conn.image.stage_image(record, filename="image.qcow2")

# with open("image.qcow2", "rb") as data:
#     record = conn.image.stage_image(record, data=data, size=0)
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
    filename := flag.String("filename", "", "required binary image filename")
    borrow := flag.Bool("borrow", false, "open a caller-owned file and lend its reader")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    infer := flag.Bool("infer-size", true, "infer total seekable length and restore cursor")
    size := flag.Int64("size", 0, "explicit signed X-OpenStack-Image-Size")
    flag.Parse()
    explicitSize := false
    flag.Visit(func(value *flag.Flag) { if value.Name == "size" { explicitSize = true } })
    if *imageID == "" || *filename == "" { log.Fatal("-image-id and -filename are required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *filename, *borrow, *viaService, *infer, *size, explicitSize); err != nil {
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

func run(ctx context.Context, cloud, imageID, filename string, borrow, viaService, infer bool, size int64, explicitSize bool) (err error) {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordStageRequest{Record: record, Filename: filename}
    mode := "filename"
    if borrow {
        data, openErr := os.Open(filename)
        if openErr != nil { return openErr }
        defer func() { err = errors.Join(err, data.Close()) }()
        input = image.ImageRecordStageRequest{Record: record, Data: data}
        mode = "borrowed-reader"
    }
    options := []image.ImageRecordStageOption{
        image.WithImageRecordStageOpts(image.ImageRecordStageOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-stage-example"},
        }),
        image.WithImageRecordStageHeader("X-SDK-Example-Mode", mode),
        image.WithImageRecordStageHeaders(map[string]string{"X-SDK-Example-Operation": "stage"}),
        image.WithImageRecordStageSizeInference(infer),
    }
    if explicitSize {
        options = append(options, image.WithImageRecordStageSize(size))
    } else {
        options = append(options, image.WithoutImageRecordStageSize())
    }
    var result *image.ImageRecordStageResult
    if viaService {
        result, err = service.StageImageRecord(ctx, input, options...)
    } else {
        result, err = conn.StageImageRecord(ctx, input, options...)
    }
    return printStage(result, err)
}

func printStage(result *image.ImageRecordStageResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["staged"] = result.Staged
        value["metadata"] = result.Metadata
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
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

- [`API.ImageData.StageImage/StageKnownImage`](v2/imagedata/README.md)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1403)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [Image Data](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L330-L360)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
