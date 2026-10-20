# Create

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/create_object`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.CreateObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectMetadataValue`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectMD5`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectSHA256`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectSegmentSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCreateObjectSegmentSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectUseSLO`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCreateObjectUseSLO`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateObjectGenerateChecksums`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCreateObjectGenerateChecksums`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.ObjectCreateUnconfirmedSegmentError.Error`

- Explicit Data selects one ordinary binary PUT, preserves nonnil empty input and native201/202 raw acknowledgement, and bypasses capabilities, HEAD, hashing and segmentation.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsBytesWireAndSnapshots`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreChecksumDecisions`.
- File and borrowed Reader use one immutable bounded-memory spool; Reader starts at its current cursor and is never closed or sought. Size, checksums and every replay use those owned bytes.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsFileReaderAndHashes`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreOwnedSources`.
- File-style checksum defaults, supplied-one recomputation, false-generation comparison-only hashes and matching stale skip preserve source behavior; supplied/generated metadata is distinct from request ETag.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsFileReaderAndHashes`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreChecksumDecisions`.
- Fresh capabilities share strict Info decoding and maximum-first/minimum-second int64 selection, default1GiB, explicit0 and clean physical404/412 fallback2684354561. Empty files can upload at size0; invalid bounds or nonempty effective0 retain proof before writes.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsFileReaderAndHashes`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreCapabilityProjection`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreEmptyAndZeroBounds`.
- Literal container/object and ResourceBase route once-escaped URLs; ordered SLO raw JSON and DLO empty body with once-percent-quoted slash-terminated generation preserve logical Content-Type.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsSLOAndDLOWire`.
- Unique cryptographic segment prefixes and fixed numeric width preserve stable manifest order. Observed segment requests force If-None-Match:*, omit ETag, and SLO requests force Accept after redirect aliases while retaining logical Content-Type.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsSLOAndDLOWire`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadNativeOwnershipGuards`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`.
- Clean segment201 establishes observed ownership and preserves an optional literal ETag with one surrounding quote pair removed. Segment202 retains actual acknowledgement but returns ObjectCreateUnconfirmedSegmentError, never establishes ownership and blocks manifest publication.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsSLOAndDLOWire`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsAcceptedFaultsAndUnconfirmedSegments`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`.
- Eligible segment failures receive at most one second logical round and manifest failures at most three logical attempts. Recovered errors stay in attempt history while eventual success returns nil.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsNativeReplayAndPhysicalEvidence`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsCleanupAndStickyAmbiguity`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadManifestAttemptsAndAmbiguity`.
- Native physical retry/backoff/reauth and permitted same-target redirects receive independent frozen body views with exact framing; previous Close cannot consume a later view. Callbacks cannot replace body, factory or success-code ownership.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsNativeReplayAndPhysicalEvidence`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadNativePhysicalReplay`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadNativeOwnershipGuards`.
- Actually started physical uploads retain independent raw responses/status/headers and native causes, including rejected body Read/Close errors and response-plus-transport faults. Acknowledgement and error proof do not alias attempts; accepted processing faults never resend.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsBytesWireAndSnapshots`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsNativeReplayAndPhysicalEvidence`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsAcceptedFaultsAndUnconfirmedSegments`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadPhysicalFaultEvidence`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadNativePhysicalReplay`.
- Segment and manifest uncertainty is cumulative across physical and logical attempts. Earlier transport or server uncertainty survives later clean rejection, and accepted processing/context faults keep ambiguity; manifest201/202 is acknowledgement without publication completion.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsSLOAndDLOWire`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsCleanupAndStickyAmbiguity`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadManifestAttemptsAndAmbiguity`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadPhysicalFaultEvidence`.
- Terminal segment/manifest failure cleans only exact observed Created && !Ambiguous names before possible publication under live context/source. Unconfirmed names and ambiguous or canceled manifest outcomes preserve the generation for recovery.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsCleanupAndStickyAmbiguity`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadManifestAttemptsAndAmbiguity`.
- Exact cleanup retains separate DELETE202/204/clean404 acknowledgements and joins cleanup failures after the original upload error. Cleanup uses no listing, HEAD or multipart controls, and acknowledgements do not establish cluster completion.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsCleanupAndStickyAmbiguity`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadManifestAttemptsAndAmbiguity`.
- Full options replace configuration; map/pointer/Data snapshots, canonical last-wins overlays and Without helpers preserve defaults and reuse. Callbacks run once and retained option state cannot alter later phases.
  Tests: `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsBytesWireAndSnapshots`, `objectstorage/v1/objects/create_options_test.go:TestObjectCreateOptionsFactorySnapshots`, `objectstorage/v1/objects/create_options_test.go:TestObjectCreateOptionsDefaultsAndReplacement`, `objectstorage/v1/objects/create_options_test.go:TestObjectCreateOptionsParallelReuse`.
- Missing/multiple sources, typed-nil Reader, invalid filename/digest/segment size, nil options/context and reserved/alias headers reject before Reader consumption or HTTP. Callback cancellation/source/body-ownership causes remain joined and terminal.
  Tests: `objectstorage/v1/objects/create_options_test.go:TestObjectCreateOptionsPreflightAndCallbackCauses`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadNativeOwnershipGuards`.
- Connection exposes CreateObject and IsObjectStale through the shared service client, with fresh capture and independent phase/result proof.
  Tests: `connection_objectstorage_create_object_test.go:TestConnectionObjectCreateSharesClientAndOwnsSources`.
- Swift generated native/service files remain byte-exact, accounts/containers/objects gain no generated Collections, and the service guide references CreateObject/IsObjectStale plus SegmentPrefix/SLO-DLO policy.
  Tests: `internal/cmd/sdkgen/swift_object_create_test.go:TestSwiftObjectCreatePreservesNativeAndRegistry`.

Python/Go differences:

- Go requires exactly one explicit Data/Filename/Reader source; nil and present empty bytes differ, no filename=name fallback, and borrowed Reader is an owned-spool file-workflow extension. Python data returns mutable Object while filename returns None.
- Go preserves separate capabilities/discovery/ordinary/segment/manifest/cleanup responses and all actual upload attempts, immutable data/spool replay, controlled closure and sticky source/context errors. This corrects FileSegment pos/seek/Close and future.result retry defects rather than reproducing them.
- Go unique slash-terminated generation and physical If-None-Match:* establish clean201 observed segment ownership; segment202 becomes an actual202 semantic error and blocks publication. Exact conservative cleanup replaces deterministic neighbors and image-task-only cleanup, preserving cumulative ambiguous manifest outcomes.
- Go strict typed digests/options/metadata and explicit0 size differ from Python coercion/falsyzero. It preserves data checksum defaults, file supplied-one recomputation and false-generation comparison fallback, while correcting legacy suffix lookup. SDK hash metadata is not request ETag.
- Native201/202 ordinary/manifest acknowledgement compatibility remains; no default/manifest request ETag is computed from JSON, and SLO raw JSON preserves logical Content-Type. Native generated Objects.Create remains separate for lower-level extensions.

Unresolved boundaries:

- Python mutable Object/container precedence, dirty header/metadata state, descriptor str(any) coercion, adapter/session/microversion/cache and broad exception-class behavior are not reproduced wholesale.
- Unique prefixes, conditional segment ownership and cleanup evidence are observed client safeguards; server races, eventual consistency, backend quota/permissions and partial writes cannot establish atomic publication, readable DLO, rollback or cluster completion.
- Python cloud image-task/autocreated-object integration, broad async/query/native extensions, heartbeat upload reports and completion waits remain separate workflows.

### `python:object_store/v1/is_object_stale`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.IsObjectStale`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithIsObjectStaleOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithIsObjectStaleHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithIsObjectStaleHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithIsObjectStaleMD5`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithIsObjectStaleSHA256`

- Fresh bodyless HEAD uses literal names and effective ResourceBase with native200/204 and clean physical404. No capability GET, upload, implicit filename or cleanup occurs.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsLazyFilesAndLiteralHEAD`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreLazyFilesAndComparison`.
- Explicit filename remains required, but HEAD404 and sufficient supplied digests do not open it. Existing object with neither local digest lazily reads/hashes once; later IO/hash failures retain Discovery and nil Stale.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsLazyFilesAndLiteralHEAD`, `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsHashProjectionAndAtomicErrors`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreLazyFilesAndComparison`.
- Only current and legacy SDK digest headers project with one UTF8/control-safe value and no alias duplicates. Present empty current values shadow legacy, literal case is preserved, and unrelated ETag/size/date do not substitute.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsHashProjectionAndAtomicErrors`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreChecksumProjection`.
- Stale=false requires at least one local digest and a matching remote literal for every available local digest; a missing remote digest is stale. Nullable decision and remote hashes commit only after clean complete evidence; dirty404, accepted read/Close/context/source and projection failures keep raw HEAD proof without a decision.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsHashProjectionAndAtomicErrors`, `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsDirtyMissingAndLiveGuards`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreLazyFilesAndComparison`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreAcceptedFailuresAndStickyGuards`.
- Full/header/digest option factories own snapshots with canonical overlays and reusable callback state; malformed digests or filename, nil options/context and reserved headers fail before HTTP.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsDirtyMissingAndLiveGuards`, `objectstorage/v1/objects/stale_options_test.go:TestObjectStaleOptionsSnapshotsAndReplacement`, `objectstorage/v1/objects/stale_options_test.go:TestObjectStaleOptionsPreflightAndCauses`.
- Native HEAD retry preserves captured option/source headers until permitted native overrides, uses live auth, does not rerun callbacks, and snapshots rejected error proof before callback mutation. Accepted source/context faults stay sticky and do not commit a decision.
  Tests: `objectstorage/v1/objects/stale_contracts_test.go:TestIsObjectStaleContractsDirtyMissingAndLiveGuards`, `objectstorage/v1/objects/stale_options_test.go:TestObjectStaleOptionsNativeSnapshotAndProof`, `objectstorage/v1/objects/stale_core_test.go:TestObjectStaleCoreAcceptedFailuresAndStickyGuards`.
- Connection exposes CreateObject and IsObjectStale through the shared service client, with fresh capture and independent phase/result proof.
  Tests: `connection_objectstorage_create_object_test.go:TestConnectionObjectCreateSharesClientAndOwnsSources`.
- Swift generated native/service files remain byte-exact, accounts/containers/objects gain no generated Collections, and the service guide references CreateObject/IsObjectStale plus SegmentPrefix/SLO-DLO policy.
  Tests: `internal/cmd/sdkgen/swift_object_create_test.go:TestSwiftObjectCreatePreservesNativeAndRegistry`.
- IsObjectStale HEADs the object, compares x-sdk-md5 and x-sdk-sha256 metadata with the local file hashes and reports a missing object as stale, as Python does.
  Tests: `objectstorage/v1/objects/python_rejudge_test.go:TestPythonRejudgeObjectStaleRequests`.

Python/Go differences:

- Go requires explicit nonempty local filename and literal container/object, returns nullable Stale plus owned raw Discovery/local/remote hashes, while Python returns bool through mutable Resource metadata.
- Go nativeHEAD200/204 and clean physical404 are explicit; dirty/nested404, read/Close/context/source/projection and later file errors retain real HEAD proof with no committed decision. Filename is not read for clean404 or sufficient supplied digests.
- Go projects only relevant current/legacy hash headers with strict single-value/case-alias/UTF8 control validation, preserves absent versus empty and literal digest case, and corrects pinned full-header legacy fallback after BaseResource prefix stripping. Neither implementation substitutes ETag/size/date for SDK hashes.
- Concrete digest/header options and callback snapshots reject malformed digests and reserved headers before HTTP and preserve fixed original source plus live auth rather than Python descriptor/adapter coercion.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "bytes"
    "context"
    "errors"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showPhase(name string, phase *objects.ObjectCreatePhaseResult) {
    if phase == nil {
        return
    }
    for _, attempt := range phase.Attempts {
        if attempt.Response != nil {
            fmt.Printf("%s logical=%d physical=%d status=%d response-bytes=%d error=%v\n",
                name, attempt.LogicalAttempt, attempt.PhysicalAttempt,
                attempt.Response.StatusCode, len(attempt.Response.Body), attempt.Error)
        } else {
            fmt.Printf("%s logical=%d physical=%d transport-error=%v\n",
                name, attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
        }
    }
}

func showCreate(result *objects.CreateObjectResult) {
    if result == nil {
        return
    }
    fmt.Printf("source=%s mode=%s size=%d skipped=%t prefix=%q manifest-ambiguous=%t\n",
        result.Source, result.Mode, result.Size, result.Skipped,
        result.SegmentPrefix, result.ManifestAmbiguous)
    if result.Capabilities != nil {
        fmt.Printf("segment-size=%d fallback=%t\n",
            result.Capabilities.Size, result.Capabilities.UsedFallback)
    }
    if result.Discovery != nil {
        fmt.Println("stale HEAD", result.Discovery.StatusCode)
    }
    showPhase("ordinary PUT", result.Ordinary)
    for _, segment := range result.Segments {
        fmt.Printf("segment=%q offset=%d size=%d created=%t ambiguous=%t\n",
            segment.Name, segment.Offset, segment.Size, segment.Created, segment.Ambiguous)
        showPhase("segment PUT", segment.Upload)
    }
    showPhase("manifest PUT", result.Manifest)
    for _, cleanup := range result.Cleanup {
        fmt.Println("cleanup target", cleanup.Name)
        showPhase("cleanup DELETE", cleanup.Deletion)
    }
}

func showError(err error) {
    var proof *resource.ResponseError
    if errors.As(err, &proof) {
        fmt.Println("response error", proof.StatusCode, "bytes", len(proof.Body))
    }
    var unconfirmed *objects.ObjectCreateUnconfirmedSegmentError
    if errors.As(err, &unconfirmed) {
        fmt.Println("unconfirmed segment", unconfirmed.Name)
    }
}

func uploadExamples(ctx context.Context, conn *sdk.Connection,
    filename, knownMD5, knownSHA256 string) error {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        return err
    }
    options := []objects.CreateObjectOption{
        objects.WithCreateObjectOpts(objects.CreateObjectOpts{
            Headers: map[string]string{"Content-Type": "application/octet-stream"},
        }),
        objects.WithCreateObjectHeader("X-Trans-Id-Extra", "sdk-upload"),
        objects.WithCreateObjectHeaders(map[string]string{"Cache-Control": "private"}),
        objects.WithCreateObjectMetadata(map[string]string{"owner": "sdk"}),
        objects.WithCreateObjectMetadataValue("empty", ""),
        objects.WithCreateObjectMD5(knownMD5),
        objects.WithCreateObjectSHA256(knownSHA256),
        objects.WithoutCreateObjectSegmentSize(),
        objects.WithoutCreateObjectUseSLO(),
        objects.WithoutCreateObjectGenerateChecksums(),
    }
    result, err := service.Objects.CreateObject(ctx, "backups", "empty ?.bin",
        objects.CreateObjectInput{Data: []byte{}}, options...)
    showCreate(result)
    if err != nil {
        showError(err)
        return err
    }
    stale, err := service.Objects.IsObjectStale(ctx, "backups", "archive ?.tar", filename,
        objects.WithIsObjectStaleOpts(objects.IsObjectStaleOpts{}),
        objects.WithIsObjectStaleHeader("X-Trace", "stale"),
        objects.WithIsObjectStaleHeaders(map[string]string{"X-Trans-Id-Extra": "sdk-stale"}),
        objects.WithIsObjectStaleMD5(knownMD5),
        objects.WithIsObjectStaleSHA256(knownSHA256))
    if stale != nil {
        if stale.Discovery != nil {
            fmt.Println("standalone stale HEAD", stale.Discovery.StatusCode)
        }
        if stale.Stale != nil {
            fmt.Println("observed stale", *stale.Stale)
        }
    }
    if err != nil {
        showError(err)
        return err
    }
    result, err = service.Objects.CreateObject(ctx, "backups", "archive ?.tar",
        objects.CreateObjectInput{Filename: filename}, options...)
    showCreate(result)
    if err != nil {
        showError(err)
        return err
    }
    readerOptions := append([]objects.CreateObjectOption{}, options...)
    readerOptions = append(readerOptions,
        objects.WithCreateObjectMD5(""),
        objects.WithCreateObjectSHA256(""),
        objects.WithCreateObjectSegmentSize(1<<30),
        objects.WithCreateObjectUseSLO(false),
        objects.WithCreateObjectGenerateChecksums(false))
    result, err = service.Objects.CreateObject(ctx, "backups", "stream ?.bin",
        objects.CreateObjectInput{Reader: bytes.NewReader([]byte("reader contents"))},
        readerOptions...)
    showCreate(result)
    if err != nil {
        showError(err)
    }
    return err
}

func main() {}
```

```python
conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "archive.tar", filename=local_filename)
conn.object_store.create_object(
    "backups", "dynamic.tar", filename=local_filename, use_slo=False
)
stale = conn.object_store.is_object_stale("backups", "archive.tar", local_filename)
```

## References

- [Info](../info.md)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L529)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L691-L909)
- [FileSegment.seek/reset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L448-L482)
- [Swift PUT API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L153-L299)
- [Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L173-L281)
- [SLO validation/finalization](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1341-L1612)
- [DLO validation](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/dlo.py#L422-L461)
- [Create core](create_core_test.go)
- [Create options](create_options_test.go)
- [Create Upload Test](create_upload_test.go)
- [Create Contracts Test](create_contracts_test.go)
- [stale core](stale_core_test.go)
- [stale options](stale_options_test.go)
- [Stale Contracts Test](stale_contracts_test.go)
- [Connection](../../../connection_objectstorage_create_object_test.go)
- [generator](../../../internal/cmd/sdkgen/swift_object_create_test.go)
