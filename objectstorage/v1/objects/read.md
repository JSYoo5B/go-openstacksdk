# Read

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/get_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfNoneMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadRange`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadFilename`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadMultipartManifest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadSymlink`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadVersionID`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadBufferSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DownloadObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetCloudObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetWriter`

- One fixed literal GET returns raw binary bytes; no HEAD/list/cache before content
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreWireAndStatus`, `connection_objectstorage_object_read_test.go:TestConnectionObjectReadSharesClientAndOwnsTransfer`.
- Literal Range/conditional headers, UTC dates and four actual query names; all option maps/pointers owned
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsOptionsAndPreflight`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsSnapshots`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsHeaderAndDateInputs`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsCallbacksAndDefaults`.
- Ordinary/source reserved headers and complete preflight fail before HTTP
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsOptionsAndPreflight`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsPreflight`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsCallbacksAndDefaults`.
- 200/206 opaque body and 304 NotModified/no read; no multipart/checksum/contentlength inference
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreWireAndStatus`.
- Nullable exact-int64 and literal metadata projection, raw headers and buffers independently owned
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsMetadataAndEvidence`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreResponseAndSource`, `connection_objectstorage_object_read_test.go:TestConnectionObjectReadSharesClientAndOwnsTransfer`.
- Read/Close/custom context faults preserve accepted status/header and actual partial bytes; Complete tracks observed EOF
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsMetadataAndEvidence`, `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreResponseAndSource`.
- Opening source drift retains accepted proof; native rejected payload/error cleanup ABI preserved
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsMetadataAndEvidence`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreResponseAndSource`.
- Fixed target/live auth/normal versus advanced native retry policy, ownership guards/actual status gate and preserved generated Download
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsNativePolicy`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamBoundaries`, `internal/cmd/sdkgen/swift_object_read_test.go:TestSwiftObjectReadPreservesNativeAndRegistry`.
- The outfile path opens after the accepted GET, receives the decoded bytes and is closed by the SDK; a clean 404 never opens or truncates it. A borrowed writer is flushed when it supports Flush and never closed.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetPathOutputBinaryOwnedCloseAndMissingNoTruncation`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetBorrowedWriterFlushOnlyAndNoStatisticalTextRead`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetFileWriterAndFalseyOutputOptions`.
- The remember_content text uses the cloud text decoding: Requests header charset rules, a byte-order mark and supported declared charsets, with undeclared non-UTF-8 bytes returned as an explicit error with the actual response.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetRequestsHeaderCharsetDefaultsReplacementAndAliases`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetUnmarkedBOMAndExplicitUTF16UTF32`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetStatisticalCharsetGapIsExplicitWithOwnedActualEvidence`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDetectorGapIncludesUTF8NULMalformedBOMAndKnownPythonCodecs`.

Python/Go differences:

- Go returns owned raw byte/evidence result; Python returns a mutable Object and only remembers decoded response.text when requested.
- Go requires explicit literal parent/object and has no outfile path/open/flush or Resource data mutation option.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Python's outfile and remember_content arguments map to separate calls: GetObject returns bytes, DownloadObject writes to a caller-supplied writer, and GetCloudObjectText returns the decoded text or, with WithCloudGetOutfile or WithCloudGetWriter, writes to a path the SDK opens and closes or to a borrowed writer it only flushes.
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.

### `python:object_store/v1/download_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DownloadObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfNoneMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadRange`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadFilename`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadMultipartManifest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadSymlink`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadVersionID`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadBufferSize`

- Borrowed writer receives binary content through one GET, no HEAD/no lifecycle ownership
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreBorrowedWriter`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsSnapshots`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsPreflight`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsCallbacksAndDefaults`.
- 304 writes nothing; accepted initial metadata/header/status and projection errors remain observable
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsMetadataAndEvidence`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreWireAndStatus`, `connection_objectstorage_object_read_test.go:TestConnectionObjectReadSharesClientAndOwnsTransfer`.
- Bounded copy uses configured buffers and excludes ReaderFrom/WriterTo shortcuts
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreBorrowedWriter`.
- Successful Write n defines BytesWritten; short/invalid counts and simultaneous Read+Write errors retained
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreBorrowedWriter`.
- Complete independently records observed EOF plus full delivery, even full-count Write error/Close fault; skipped canceled writes stay incomplete
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreBorrowedWriter`.
- Context/source checked between Read and Write; response Close once, underlying causes and metadata proof retained without whole-body buffering
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_core_test.go:TestObjectReadCoreBorrowedWriter`.
- DownloadObject sends one GET for the object and writes the body bytes Python download_object returns.
  Tests: `objectstorage/v1/objects/python_rejudge_test.go:TestPythonRejudgeObjectDownloadRequest`.

Python/Go differences:

- Go DownloadObject copies into a borrowed writer and returns a byte count plus metadata/evidence; Python download_object returns response.content bytes.
- The concrete Go ownership/typed options replace mutable Object/attrs request preparation, while existing native Download remains available.
- Go percent-encodes a slash inside the object name as %2F; Python sends it literally and Swift resolves both to the same object.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:object_store/v1/stream_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.StreamObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfNoneMatch`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfModifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectReadIfUnmodifiedSince`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadRange`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadFilename`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadMultipartManifest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadSymlink`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadVersionID`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectReadBufferSize`

- GET eager and payload read deferred; caller buffer controls Read size; zero-length Read does not consume underlying body
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamOwnershipAndCleanup`.
- 304 yields nonnil closed-empty body/NotModified, no read, and observable Close failure
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWireAndBytes`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamOwnershipAndCleanup`.
- Projection/source opening failure returns accepted proof with nil metadata/body and closes without draining
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsMetadataAndEvidence`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsPreflight`.
- Valid returned n contributes BytesRead before joined errors; exact EOF evidence independently sets Complete
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamOwnershipAndCleanup`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamGuards`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamBoundaries`.
- Early/idempotent Close exposes stable cleanup cause once; later Read/Close never reopens body
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamOwnershipAndCleanup`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamGuards`.
- Initial raw header/status error proof remains immutable despite exported result mutation; later underlying/context/source causes and counts preserved
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsWriterAndStreamOwnership`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamOwnershipAndCleanup`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamGuards`, `connection_objectstorage_object_read_test.go:TestConnectionObjectReadSharesClientAndOwnsTransfer`.
- Prebody retry/actual status guard cannot retarget or resend exposed body; existing native Download status/header/query ABI remains exact
  Tests: `objectstorage/v1/objects/read_contracts_test.go:TestObjectReadContractsNativePolicy`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsSnapshots`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsHeaderAndDateInputs`, `objectstorage/v1/objects/read_options_test.go:TestObjectReadOptionsCallbacksAndDefaults`, `objectstorage/v1/objects/read_stream_test.go:TestObjectReadStreamBoundaries`, `internal/cmd/sdkgen/swift_object_read_test.go:TestSwiftObjectReadPreservesNativeAndRegistry`.

Python/Go differences:

- Go exposes an eager, explicitly closeable guarded stream with delivered-byte counters and owned evidence; Python exposes requests.iter_content byte chunks.
- Go terminal auto-close/stable Close and strict projected metadata are deliberate contracts, rather than Requests iterator/session behavior.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- StreamObject returns an io.ReadCloser with delivered-byte counters instead of a Requests iter_content iterator of chunk_size bytes; WithObjectReadBufferSize sets the read buffer, the caller must close the stream and read it from one goroutine at a time, and the read does not check payload integrity; see Server guarantees outside the SDK in docs/sdk-support-ledger.md.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func readObjects(ctx context.Context, api *objects.API, etag string, asOf time.Time, version string) error {
	common := []objects.ObjectReadOption{
		objects.WithObjectReadOpts(objects.ObjectReadOpts{
			Headers:           map[string]string{"X-Audit": "initial"},
			IfModifiedSince:   &asOf,
			IfUnmodifiedSince: &asOf,
		}),
		objects.WithObjectReadHeaders(map[string]string{"X-Audit": "read"}),
		objects.WithObjectReadHeader("X-Audit", "read-final"),
		objects.WithObjectReadNewest(false),
		objects.WithObjectReadBufferSize(32 * 1024),
	}
	getOpts := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfUnmodifiedSince(),
		objects.WithObjectReadIfModifiedSince(asOf),
		objects.WithObjectReadIfNoneMatch(etag),
	)
	buffered, err := api.GetObject(ctx, "backups", "report.txt", getOpts...)
	if buffered != nil {
		fmt.Printf("GET status=%d bytes=%d not-modified=%t complete=%t\n",
			buffered.StatusCode, len(buffered.Body), buffered.NotModified, buffered.Complete)
	}
	if err != nil {
		return err
	}

	copyOpts := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfModifiedSince(),
		objects.WithObjectReadIfUnmodifiedSince(asOf),
		objects.WithObjectReadIfMatch(etag),
		objects.WithObjectReadRange("bytes=0-1023"),
		objects.WithObjectReadFilename("report.part"),
		objects.WithObjectReadVersionID(version),
	)
	var output bytes.Buffer
	copied, err := api.DownloadObject(ctx, "backups", "report.txt", &output, copyOpts...)
	if copied != nil {
		fmt.Printf("copied=%d complete=%t\n", copied.BytesWritten, copied.Complete)
	}
	if err != nil {
		return err
	}

	plain := append(append([]objects.ObjectReadOption(nil), common...),
		objects.WithoutObjectReadIfModifiedSince(),
		objects.WithoutObjectReadIfUnmodifiedSince(),
		objects.WithoutObjectReadNewest(),
	)
	manifestOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		objects.WithObjectReadMultipartManifest("get"),
	)
	streamed, err := api.StreamObject(ctx, "backups", "manifest", manifestOpts...)
	if err != nil {
		return err
	}
	_, readErr := io.Copy(io.Discard, streamed.Body)
	closeErr := streamed.Body.Close()
	fmt.Printf("stream bytes=%d complete=%t\n", streamed.BytesRead, streamed.Complete)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}

	linkOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		objects.WithObjectReadSymlink("get"),
	)
	_, err = api.GetObject(ctx, "backups", "report-link", linkOpts...)
	return err
}

func main() {}
```

```python
import openstack

conn = openstack.connect()

remembered = conn.object_store.get_object(
    "report.txt", container="backups", remember_content=True
)
print("remembered text characters:", len(remembered.data))

data = conn.object_store.download_object("report.txt", container="backups")
print("downloaded bytes:", len(data))

chunks = conn.object_store.stream_object(
    "report.txt", container="backups", chunk_size=32 * 1024
)
for chunk in chunks:
    print("chunk bytes:", len(chunk))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288-L396)
- [Swift GET API reference](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L16-L151)
- [Swob](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/swob.py#L1350-L1478)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L103-L171)
- [Read Core Test](read_core_test.go)
- [Read Stream Test](read_stream_test.go)
- [concrete options](read_options_test.go)
- [Read Contracts Test](read_contracts_test.go)
- [Connection Objectstorage Object Read Test](../../../connection_objectstorage_object_read_test.go)
- [Swift Object Read Test](../../../internal/cmd/sdkgen/swift_object_read_test.go)
