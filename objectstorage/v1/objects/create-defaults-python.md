# Create Defaults Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:objectstorage/v1/objects.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateBytes`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateQuery`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithCreateHeader`

- Create retains native escaped object PUT, core/extension headers and query, default content MD5, explicit ETag with NoETag, native201/202 status and actual PUT header decoding. Final plain-nil Content creates an empty object rather than invoking the upstream nil-reader MD5 panic; typed-nil supplied Content is ErrInvalidOption before HTTP under every ETag mode.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyDefaults`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateTypedNilReadersAndPreflight`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateUpdateDeleteHeadersAndEscaping`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectStatusesNamesAndPreflight`, `objectstorage/v1/objects/native_contracts_test.go:TestNativeObjectCreateNilContentUsesEmptyBody`.
- WithCreateBytes snapshots nil/empty/nonempty bytes at option construction and creates a fresh reader on each application. Request options retain last-applied Content precedence; mutation of caller bytes and reusing an option do not change later uploads.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateBytesSnapshotAndOptionOrder`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyDefaults`.
- Native checksum preparation retains physical read/seek causes and Create/objects wrapping without HTTP or closing a failed preflight reader. Empty bodies retain default ETag through an actual native503 retry to201;204 is still a native rejection.
  Tests: `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateDefaultReaderErrorsRetainCause`, `objectstorage/v1/objects/create_defaults_test.go:TestNativeObjectCreateEmptyNativeRetryAndStatus`.

Python/Go differences:

- This declaration maps the native Gophercloud object Create call only. Python Resource and owned CreateObject bytes/file/SLO/DLO workflows remain separately tracked; fixing nil Content does not declare Python create_object full parity.
- The SDK applies an empty-reader default after final typed options, rejecting typed-nil readers before HTTP. Native URL, headers/query, status decoding and reader/retry behavior remain unchanged; WithCreateBytes is a reusable immutable input option, not a replacement for the owned CreateObject replay/evidence workflow.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }

    // Content omitted: one ordinary PUT, with the MD5 of empty bytes.
    empty, err := swift.Objects.Create(ctx, "backups", "empty.bin", objects.CreateOpts{})
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT ETag=%q transaction=%q\n", empty.ETag, empty.TransID)

    // The option snapshots these bytes. Reusing it starts a new reader.
    data := []byte("hello")
    body := objects.WithCreateBytes(data)
    data[0] = 'X'
    for _, name := range []string{"first.txt", "second.txt"} {
        result, err := swift.Objects.Create(ctx, "backups", name,
            objects.CreateOpts{ContentType: "text/plain"}, body,
            objects.WithCreateHeader("X-Object-Meta-Category", "example"))
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s PUT ETag=%q\n", name, result.ETag)
    }

    // Source-shaped bytes workflow: no native ETag calculation, and raw PUT
    // acknowledgement is retained separately from metadata observations.
    uploaded, err := swift.Objects.CreateObject(ctx, "backups", "source-empty.bin",
        objects.CreateObjectInput{Data: []byte{}})
    if err != nil { log.Fatal(err) }
    if uploaded.Ordinary != nil && uploaded.Ordinary.Acknowledgement != nil {
        fmt.Printf("source PUT status=%d\n", uploaded.Ordinary.Acknowledgement.StatusCode)
    }
}
```

```python
import openstack

conn = openstack.connect()
obj = conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "first.txt", data=b"hello",
                                content_type="text/plain")
```

## References

- [owned create workflow](create.md)
- [Create](create.md)
- [Create Defaults Test](create_defaults_test.go)
- [Native Contracts Test](native_contracts_test.go)
