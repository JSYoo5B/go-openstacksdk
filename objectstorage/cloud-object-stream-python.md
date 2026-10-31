# Cloud Object Stream Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/stream_object`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.StreamObject`
- `github.com/JSYoo5B/go-openstacksdk.Connection.StreamObjectChunks`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.StreamObjectChunks`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectStreamQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.StreamCloudObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudStreamQueryString`

- Library-owned factory returns a lazy, single-use closeable binary chunk iterator; default1024 and query_string ignored even when malformed. No discovery/options/GET before iteration; connection supplies the cached Swift Cloud facade. Receipt is actual status/header and each returned chunk is independently owned.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamLazyDefaultBinaryChunksAndSingleUse`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamLazyAliasIgnoredQueryAndCleanMissing`, `connectiontest/connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamDiscoversLazilyAndSharesCachedFacade`.
- Typed concrete ChunkSize/QueryString options and pointer snapshots require no application interface builder. Short physical reads are assembled into bounded owned chunks, nil default and explicit sizes are copied before mutation; nil/cancelled context/client/options and unsafe literal names fail without HTTP.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamChunkSizeSnapshotShortReadsAndValidation`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNilCancelledAndOptionSourceMutationBeforeHTTP`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamNilCancelledAndClosedHandleStayLazy`, `connectiontest/connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamPreflightAndClosedHandleNeverDiscover`.
- Source-valid physical100..399 status range includes199/203/204/206 and provided304 body; native400+ errors retain actual status/header/body. Only a clean404 after successful physicalRead/Close and encoded-body validation becomes empty success, retaining receipt404; mixed404 physical/decode failures remain errors.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamSourceSuccessStatusAnd304Body`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamClean404OnlyEmptyAndOtherNativeStatuses`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamMissingReadCloseFaultsAreNotSuppressed`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCompressedRejectionsValidateBeforeIgnoring404`, `connectiontest/connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamMissingReturnsActualReceipt`.
- EOF/error/break/consumerClose cleans physical body once; callbackClose prevents GET; Close during GET adopts and closes late unread body plus actual receipt. Check Err/Close for early-break cleanup failure; consumerClose prevents next Read; private receipt and unary/native error snapshots resist caller header/body mutation.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamBreakClosesOnceAndRetainsCleanupFailure`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCloseFromOptionsAndConsumerStopsBeforeAnotherRead`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamConcurrentCloseDuringGETOwnsLateBody`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamPartialReadCloseAndImmutableCurrentChunkProof`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNativePolicyErrorCopiesUnaryActualReceipt`, `connectiontest/connection_swift_object_stream_test.go:TestConnectionSwiftChunkStreamCleanupErrorsKeepPrivateProof`.
- Bodyless GET pins selected source/provider/account route/version/header facts through options, read, Close, auth/retry. Restored source plus custom cancellation/read/Close retain actual physical partial proof; native401 physical fault keeps ErrOriginal+ErrReauth and suppresses all retry/reauth/backoff hooks. Clean401 renewal and429/498 backoff reuse fixed request.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamStickyRestoredSourceAndCancellationAfterActualRead`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamNative401PhysicalFaultDoesNotInvokeHooks`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamCleanNativeRenewalAndBackoffReuseFixedRoute`.
- Requests baseline Accept:*/* and explicit Accept-Encoding:gzip, deflate, zstd fix request defaults without X-Newest. Owned caller override is snapshotted and native retry cannot change it. Real default Go transport preserves actual compressed Content-Encoding/Content-Length while bytes stream decoded, with no accidental auto-gzip double decoding.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamEncodingOverrideIsOwnedThroughCallbacksAndNativeRetry`, `objectstorage/cloud_object_stream_test.go:TestSwiftCloudObjectStreamDefaultTransportGzipAndZstandard`.
- Incremental gzip/x-gzip, zlib/rawdeflate including raw zlib-prefix fallback, zstd and reverse ordered chains up to5 match installed Requests decoder selection; unrecognized single/br remains opaque. Empty bodies yield nothing. Clean gzip truncatedtrailer/partialheader and post-complete tail normalize after actualEOF; badmagic/CRC and physical faults never become successfulEOF, including encoded404.
  Tests: `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamRequestsCodecsAndReverseEncodingChain`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamEmptyBodiesAndMalformedCodecKeepReceipt`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamDeflateFallsBackAfterValidZlibPrefix`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamGzipCleanEOFTrailerTailHeaderAndCompressed404`, `objectstorage/v1/objects/python_cloud_stream_test.go:TestPythonCloudObjectStreamGzipEOFNormalizationNeverHidesPhysicalFaults`.

Python/Go differences:

- This closes the named cloud fixed string container/object branch only. Python dynamically accepted mutable Object/Container, dirtyattrs, inherited metadata and arbitrary sessions are excluded typed Go semantics; Proxy.stream_object and native Objects.StreamObject review judgments are unchanged. SDK-R1/C1/S1 constructor/resource/session/cache foundations remain unsupported and supported=0.
- Go ChunkSize is typed positive int1..16777216; nil selects SDK1024. Python None/negative/bool/zero/float branches and HTTP-versus-chunk-validation precedence are deliberate Go defaults/safety differences; invalidtyped options are rejected beforeGET.
- Go assembles short Reads/framing pieces into owned bounded chunks, whereas Requests physicalchunked response can yield shorter wire-boundary segments; binary content and maximum requested chunk bound are preserved, not identical transportsegmentation.
- CloudObjectStream/Chunks/Err/Close/Receipt are SDK-owned implementations, single-use like Source generator. SDK automatic EOF/error/breakcleanup and partialbytes+error improve Go ownership. Close may stop pendingGET and capture latebody; serialiteration/Close only, blockingRead interruption uses context rather than concurrentClose as with shared guarded reader.
- Go validates safe UTF8 literalSwift names and PathEscapes complete object segment; Source utils.urljoin strips outer slashes and preserves inner object slash as path. Name/route/framing/auth/version guards and malformedtyped preflight are deliberate Go protocol policy.
- Receipt exposes actual status/header snapshot plus errorchain proof without buffering successful object payload. physical Read errors retain only that current compressed/raw Read partialbytes; decoded partial bytes are yielded separately. Rejected physical bytes are already owned by nativeGophercloud and codec validation never rereads/recloses that response.
- Decoder comparison uses CPython3.14.8, Requests2.34.2 andurllib3.2.8.0 with gzip/xgzip/deflate/zstd installed; zstd implementation is pinned klauspost/compressv1.18.0. Single optionalbr/unknown stays opaque as in this runtime, and optionalPython decoder installation/session customization is not genericResource/transport parity.
- NativeGophercloud clean401/429/498 callbacks remain provider-owned with SDK guard; ErrUnableToReauthenticate does notUnwrap its original/reauth branches. Accepted status policy is Source100..399 and does not expand existing nativeStreamObject200/206/304 semantics. Source exception rendering/KSA authentication/cache behavior remain foundationdifferences.
- Actual pinned Source execution has41 observations using real cloud/Proxy/Object/Requests/urllib3 and only memoryHTTP adapter/Connectionstub. Finalroot focused3packages passed28newgroups(leaf21/cloud3/Connection4); PythonmainAST and byteexactPython/Go guide fences passed. These are not livecloud authentication, SwiftACL or releaseacceptance. Review file is a candidate pending canonicalmerge; fullsupported count remains0.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--chunk-size", type=int, default=1024)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    stream = conn.stream_object(
        args.container, args.object,
        query_string=args.query, resp_chunk_size=args.chunk_size,
    )
    try:
        for chunk in stream:
            sys.stdout.buffer.write(chunk)
    finally:
        stream.close()
```

```go
package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func run(ctx context.Context, cloud, container, object, query string, chunkSize int) (err error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    stream := conn.StreamObjectChunks(ctx, container, object,
        objectstorage.WithObjectStreamChunkSize(chunkSize),
        objectstorage.WithObjectStreamQueryString(query),
    )
    defer func() {
        err = errors.Join(err, stream.Close())
        if receipt := stream.Receipt(); receipt != nil {
            fmt.Fprintf(os.Stderr, "status=%d encoding=%q\n",
                receipt.StatusCode, receipt.Header.Get("Content-Encoding"))
        }
    }()
    for chunk, readErr := range stream.Chunks() {
        var writeErr error
        if len(chunk) != 0 {
            var written int
            written, writeErr = os.Stdout.Write(chunk)
            if writeErr == nil && written != len(chunk) { writeErr = io.ErrShortWrite }
        }
        if readErr != nil || writeErr != nil {
            return errors.Join(readErr, writeErr)
        }
    }
    return stream.Err()
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name")
    query := flag.String("query", "", "ignored by the pinned cloud stream_object")
    chunkSize := flag.Int("chunk-size", 1024, "decoded bytes per chunk; 1..16777216")
    timeout := flag.Duration("timeout", 5*time.Minute, "whole download deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [cloud stream_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L484)
- [Object.stream/_download](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L319)
- [Install](../docs/install.md)
- [Python Cloud Stream Test](v1/objects/python_cloud_stream_test.go)
- [Cloud Object Stream Test](cloud_object_stream_test.go)
