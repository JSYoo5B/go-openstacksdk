# Python Raw Get

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_object_store/ObjectStoreCloudMixin/get_object_raw`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObject`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.GetObjectRaw`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectQueryString`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithRawObjectStream`

- Default stream=False and query_string=None map to library-owned omitted-query buffered GET; functional concrete options allow explicit encoded query and stream=True. Literal escaped container/object, empty obj selects container, raw repeated query order stays intact, no HEAD/metadata/content decoding. Complete opaque bytes include NUL/invalid UTF8, physical body closes once, accepted HTTP100..999 alone never raises status errors.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectCloudGetObjectRawReturnsEveryStatus`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawBufferedDefaultOwnsOpaqueBytesAndAllowsContainerEndpoint`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectCloudGetObjectRawRejectsInputsBeforeHTTP`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawOptionBoundaryCapturesSourceBeforeCallbacks`.
- Buffered physical Read/Close/sticky restored source/custom cancellation faults retain actual status/header/partial payload without replays, on 200/206/401/404/429/498/503; 401 pointer ErrUnableToReauthenticate public ErrOriginal and ErrReauth retained; willing retry/reauth/backoff hooks remain zero after physical faults. HTTP status clean versus mixed callback errors are classified separately.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawPhysicalFaultsPreserveActualPayloadAndNative401WithoutHooks`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawCleanNativeRenewalAndMixedCallbackErrorsStayDistinct`.
- Full named stream=True branch returns accepted body before any Read, uses shared owned bodyless unread exchange, opaque bytes flow through caller-owned guarded reader, clean EOF and Close are stable and one physical Close, valid extension HTTP777 remains raw. Original admitted policy/body/header/route cannot be broadened by callbacks.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingIsUnreadAndGuardsOpaqueLifecycle`.
- Caller stream Read/Close faults and restored source/custom cancel preserve only the current partial chunk with immutable actual status/header and all causes, despite caller mutating exported result. SDK operation wrapper retains exact EOF; no reauth/retry/backoff or resend after exposure. Early Close succeeds/idempotent and later Read is ErrClosedPipe; cancel after headers closes without a physical Read.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingFaultsRetainImmutableWireEvidenceAndNoReplay`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingCloseBeforeEOFAndCanceledUnreadBody`.
- Clean native401 reauth and429/498 backoff renew live token while fixing URL/query/Accept/conditions/selected version. Final clean native statuses are rebuilt only from actual native bytes already read/closed. Faulty rejected Close/native pointer401 preserve evidence and stop hooks before streaming body is returned.
  Tests: `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawCleanNativeRenewalAndMixedCallbackErrorsStayDistinct`, `objectstorage/v1/objects/python_raw_get_test.go:TestPythonObjectRawStreamingCleanAndFaultyNative401AndBackoff`.

Python/Go differences:

- Go returns RawObjectResponse(StatusCode,Header,Body), rather than mutable Requests Response, and configures omitted query + buffered default via concrete optional functions. The public stream branch preserves actual wire facts and close ownership rather than exposing mutable HTTP implementation state.
- Go preserves one escaped opaque object name, including slash versus literal percent-escape distinction; Python urllib.parse.quote leaves object slashes unescaped. Empty obj selects the container in both. Go validates literal container/object names and already-encoded QueryString UTF8/control/space/fragment before HTTP; Requests automatic URI requoting and custom Python objects are not reproduced.
- Go provider-owned clean native401 renewal and429/498 backoff occur before the final raw result. Native rejected bodies have already been consumed, so final clean statuses and failed native response evidence are rebuilt from their actual bytes, even in stream=True. Mixed callback/handling/source/context failures remain errors and never become clean raw status results.
- Go streamed terminal failure proof retains current partial chunk and immutable physical status/header without buffering prior successful chunks. Buffered failures may return nonnil raw result and error together. Python transport/cache/session/Resource object ownership remain common SDK-R1/C1/S1 contracts, not a full SDK parity claim.
- Fixture/guide validation does not establish live Swift ACL/backend/middleware/auth/cache or release acceptance; those project gates remain tracked.

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
parser.add_argument("--stream", action="store_true")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    response = conn.get_object_raw(
        args.container, args.object,
        query_string=args.query, stream=args.stream,
    )
    try:
        print(f"status={response.status_code}", file=sys.stderr)
        for chunk in response.iter_content(chunk_size=32768):
            sys.stdout.buffer.write(chunk)
    finally:
        response.close()
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
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func run(ctx context.Context, cloud, container, object, query string, stream bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { return err }
    response, requestErr := swift.Objects.GetObjectRaw(ctx, container, object,
        objects.WithRawObjectQueryString(query),
        objects.WithRawObjectStream(stream),
    )
    if response == nil { return requestErr }
    fmt.Fprintf(os.Stderr, "status=%d\n", response.StatusCode)
    _, readErr := io.Copy(os.Stdout, response.Body)
    closeErr := response.Body.Close()
    return errors.Join(requestErr, readErr, closeErr)
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "assets", "literal container name")
    object := flag.String("object", "logs/part.bin", "literal object name; empty selects container")
    query := flag.String("query", "", "already encoded query string")
    stream := flag.Bool("stream", false, "transfer a guarded stream")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *container, *object, *query, *stream); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## References

- [`get_object_raw(container, obj, query_string=None, stream=False)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L449-L484)
- [Install](../../../docs/install.md)
- [Python Raw Get Test](python_raw_get_test.go)
