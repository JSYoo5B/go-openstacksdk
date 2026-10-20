# Cloud Object Get Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.DownloadObject`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.Cloud.GetObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetCloudObjectText`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetStream`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithObjectGetWriter`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithoutObjectGetContent`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage.WithoutObjectGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetChunkSize`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetStream`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetOutfile`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCloudGetWriter`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCloudGetContent`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutCloudGetOutfile`

- Eager closed string-name cloud get_object ignores declaredquery_string/stream, default response.text returns lowercase tupleheaders plus optionaltext/actualreceipt. Memory ignores resp_chunk_size, byte/text empty and unknown/repeated headers retain presence; clean404 Missing retains actualreceipt404 without header/body tuple.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDefaultTextLowerHeadersEagerAndIgnoredControls`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetEagerLowerHeadersTextAndMissingReceipt`.
- Concrete owned options snapshot pointerChunkSize/Outfile and distinguish defaultNone(memory) from falseynonNone(emptypath/OmitContent headers-only): falsebranch does no acceptedbodyread/text/write/decoder/chunkvalidation and closesbody. Output reset returns memory; nil/client/context/cancel/options/unsafe names/conflictingborrowedtargets fail withoutHTTP.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetOptionsSnapshotsPreflightAndDestinationReset`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetFalseyNonNoneOutfileSkipsReadWriteTextAndChunkValidation`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetNilCancelledAndStatisticalGapStayExplicit`.
- Nonempty path opens afteracceptedGET/status/headercopy, writes opaque decodedbinary and closes itsownedhandle; clean404 never opens/truncates path. Borrowedwriter success flushes onlyifFlush supported, nevercloses; noFlush io.Writer works withoutdeveloperbuilder. Destination bytes count actualwrites and binarypath bypasses textdetector.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetPathOutputBinaryOwnedCloseAndMissingNoTruncation`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetBorrowedWriterFlushOnlyAndNoStatisticalTextRead`, `objectstorage/cloud_object_get_test.go:TestSwiftCloudObjectGetFileWriterAndFalseyOutputOptions`.
- Accepted decoderconstruction follows output/pathopen; missingdirectory wins overbadgzip or6encodingchain ctorerror, existingpath truncatesbeforectorfailure, falseypath neverdecoderinit. Source raw integerheader bad is preserved withoutnew eagervalidation; actualreadfault keeps itsphysicalproof.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetPathOpenFailurePrecedesBodyCodecAndAlwaysClosesResponse`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDecoderConstructorsFollowPathOpenAndSkipFalseyOutfile`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetTranslatedIntegerHeaderKeepsRawValueAndReadFailuresRemainFirst`.
- Write/shortwrite/flush failures retain actualHTTP receipt, currentdecodedchunk/acceptedBytesWritten and independentcleanup causes; failedRead nevercallsdestination, writefailure skipsFlush, borrowedClose neverruns. Postwrite sourcechange/customcancel remainssticky whenphysicalClose restoresendpoint and errorchain retains read/Close/actual401original+reauth withouthooks/replay.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetWriterFailureShortWriteFlushFailurePreserveStageAndCleanup`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetAcceptedReadFaultNeverCallsDestination`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetWriterSourceCancellationIsStickyAcrossRestoringClose`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetBufferedPhysicalFaultsAnd401OriginalRemainActual`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetOptionMutationGuardsSourceBeforeDestinationOrHTTP`.
- Subset textsemantics matchRequests headerpermissivesplit/case-sensitive defaulttextLatin1/defaultJSONUTF8/lastcharsetwins, explicitASCII/Latin1/cp1252/UTF8SIG andUTF16/32 includingshortreplacement/highsurrogate+oddbyte, rawIANA ShiftJIS alias andnormalBOM. Defaultstatistical unmarkednonUTF8/NUL/malformedBOM andknownunsupportedPythoncodecs return explicitErrUnsupported with actualfullreceived-body evidence ratherthansilentdeclaredcompletion.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetRequestsHeaderCharsetDefaultsReplacementAndAliases`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetUnmarkedBOMAndExplicitUTF16UTF32`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetStatisticalCharsetGapIsExplicitWithOwnedActualEvidence`, `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetDetectorGapIncludesUTF8NULMalformedBOMAndKnownPythonCodecs`.
- ReuseSourceaccepted100..399 guardedrawGET, pinnedheaderdefaults/encoding policy andincrementalgzip/xgzip/zlib/rawdeflate/zstd decoderhelpers; originalcompressedHeader remainsdistinct fromdecodedtext/filebytes. Cleancompressed404 requiresdecoder success, corruptcompressed404 stayserror with actualraw nativeproof andnoMissing.
  Tests: `objectstorage/v1/objects/python_cloud_get_test.go:TestPythonCloudObjectGetSharedCodecsAndCompressedMissingFaults`.
- Eager Connection text/output handling shares the cached Swift facade and retained actual receipt, preserving cancellation and invalid-call failures before discovery.
  Tests: `connection_cycle7_records_test.go:TestConnectionObjectGetTextSharesCachedSwiftAndRetainsReceipt`, `connection_cycle7_records_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- This meaningful closed cloud GetObjectText API is a partial named implementation, notgo_mapping completion. Operation remainsunresolved and doesnotincreasefullsupported; wholeSDK-R1/C1/S1 resource/session/cache/Connection foundations also remainunsupported. GeneralProxy.get_object mutableObject/Container/dirtyattrs and existing nativeGetObject/DownloadObject review judgments stayunchanged.
- GoObjectGetResult separates lowercaseHeaders, optionalBody *string, actualReceipt, Missing andBytesWritten. SourceNone onclean404 becomes Missingwith actual404receipt, nofakeheaders/text; errorresults may retainreceivedheader/status/partialproof evenwhenSourcewouldraisebeforetuple.
- Go supports limited explicitheadercharset/BOM/validUTF8 policies with golang.org/x/textv0.41.0 andsimplePythonaliasnormalization; these do notreplaceRequests2.34.2 charset_normalizer3.5.2 apparent_encoding orfullCPython textcodec lookup/errorsemantics. Unresolved detector scopeincludesallunmarkedbytes, notmerelynonUTF8; knownNUL/malformedBOM/unicode_escape/cp1006/palmos negatives retainUnsupported+actualbytes.
- Sourceget_object outfileNone/falseynonNone truthiness is represented by typednilcontrols, explicitemptyOutfile/OmitContent, concretepathorborrowedio.Writer. Pythonarbitrary filelikebool/callbacks/Resources are nottransferredtoGo. QueryString/Streamignored formalcontrols staylibraryowned, noapplicationinterface builderrequired.
- NonemptyGooutput chunk is typedpositive1..16MiB validated beforeHTTP/fileopen, whileSource mayopenpaththenvalidate dynamicchunk type; memory/discardignoreChunkSize. Go detects io.ErrShortWrite whereSource ignoreswrite returncount, optionalFlush supportsstandardunbuffered Go writers, andownedpath/HTTPbody cleanup occurs onall errors instead ofSourcefinally absence.
- Accepted outputorder is rawGET/status→lowerheadercopy→pathopen→decoderinit→reads/writes→Flush/Close. SourceObject rawheadercopy is retained withoutdemanding eagerdescriptorcasts. CallerPathError concreteerrors.As type remainsavailable; no reusedcachedstream-error snapshot canerase it.
- Names followSDK safeUTF8 literalSwiftvalidation andwholeobject PathEscape, source/provider/accountendpoint/version/header facts are capturedbeforeoptions andremainsticky throughnativecallbacks/Read/Write/Flush/Close. Late writer/reader resource.ErrNotFound is aphasefailure with actualreceivedproof, not cleanHTTP404 suppression.
- Decoder behavior andnative clean401/429/498 policyreuse existingownedSwift streamhelpers, accepted100..399 doesnotexpandnativeStreamObject200/206/304. Error bodyproof maybeactual currentcompressedphysicalbytes orcurrentdecodedwriterchunk asstageappropriate; noentireobjectbuffer iskeptforoutputmodes.
- ActualpinnedSource cloud/Proxy/Object/Requests/urllib3 execution has65observations withmemoryHTTPadapter/eagerRequests-like buffering andonlypathopen recordingstub. Sourcefield/runtimefiles arehash-recorded; oracle/mains/fixtures donotprove liveauth/filesystemACL/release. Originalpins areGophercloudv2.15.0 andopenstacksdkef55d7d1666099f50bf1e1c40b59d7e7b72a51fe; fullsupported count remains0.

Unresolved boundaries:

- Requests Response.text with encoding=None runs charset_normalizer3.5.2 statistical apparent_encoding across the entire received bytes. The Go limitedUTF8/BOM policy does not reproduce that detector for allvalidUTF8, NUL, malformedBOM, legacy/binary/mixedlanguage inputs. ActualSource examples: unmarkedcaf e9→UTF16BE U6361/U66E9, validUTF8 00 41 00 42→AB, malformedff fe 41→alternatecodepage. Known detector-required inputs currently returnErrUnsupported withactualfulldecodedbytesproof; exactautomaticdefaulttext parity remainsopen.
- FullCPython3.14 textcodec/alias lookup anderrors=replace semantics are not mappedbythelimitedmanual+IANA/xtext subset. KnownvalidPythonunicode_escape, raw_unicode_escape, cp1006, palmos andother unsupportedcodecs returnErrUnsupported; unknownIANA names canstillincludevalidPythonaliases notclassifiedbythissubset. Entirecodecnamespace andallcodepage mappings/replacementcases needexplicitlibraryownedcoverage before namedget_object can bepromoted.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import sys
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--container", default="assets")
parser.add_argument("--object", default="logs/part.bin")
parser.add_argument("--query")
parser.add_argument("--stream", action="store_true")
parser.add_argument("--chunk-size", type=int, default=1024)
output = parser.add_mutually_exclusive_group()
output.add_argument("--outfile")
output.add_argument("--discard-content", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    result = conn.get_object(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
        resp_chunk_size=args.chunk_size,
        outfile="" if args.discard_content else args.outfile,
    )
    if result is None:
        print("missing", file=sys.stderr)
    else:
        headers, body = result
        print(json.dumps(headers, indent=2), file=sys.stderr)
        if body is not None:
            sys.stdout.write(body)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func run(ctx context.Context, cloud, container, object, query, outfile string,
    stream, discard bool, chunkSize int) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    options := []objectstorage.ObjectGetOption{
        objectstorage.WithObjectGetQueryString(query),
        objectstorage.WithObjectGetStream(stream),
        objectstorage.WithObjectGetChunkSize(chunkSize),
    }
    if outfile != "" { options = append(options, objectstorage.WithObjectGetOutfile(outfile)) }
    if discard {
        if outfile != "" { return fmt.Errorf("choose outfile or discard-content") }
        options = append(options, objectstorage.WithoutObjectGetContent())
    }
    result, getErr := conn.GetObjectText(ctx, container, object, options...)
    if result != nil {
        summary := struct {
            Headers map[string]string `json:"headers"`
            Missing bool `json:"missing"`
            BytesWritten int64 `json:"bytes_written"`
        }{result.Headers, result.Missing, result.BytesWritten}
        encoded, err := json.MarshalIndent(summary, "", "  ")
        if err != nil { return err }
        fmt.Fprintln(os.Stderr, string(encoded))
    }
    if getErr != nil { return getErr }
    if result != nil && result.Body != nil {
        written, err := io.WriteString(os.Stdout, *result.Body)
        if err != nil { return err }
        if written != len(*result.Body) { return io.ErrShortWrite }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name")
    query := flag.String("query", "", "ignored by pinned cloud get_object")
    stream := flag.Bool("stream", false, "ignored by pinned cloud get_object")
    outfile := flag.String("outfile", "", "SDK-owned binary output path")
    discard := flag.Bool("discard-content", false, "falsey non-None output: headers only")
    chunkSize := flag.Int("chunk-size", 1024, "output-only decoded chunk size; 1..16777216")
    timeout := flag.Duration("timeout", 5*time.Minute, "whole request deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *outfile,
        *stream, *discard, *chunkSize); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [cloud get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L512)
- [Proxy.get_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288)
- [Install](../docs/install.md)
- [Python Cloud Get Test](v1/objects/python_cloud_get_test.go)
- [Cloud Object Get Test](cloud_object_get_test.go)
