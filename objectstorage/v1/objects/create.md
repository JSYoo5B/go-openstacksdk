# Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
    "github.com/JSYoo5B/go-openstacksdk/request"
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
        request.Header("X-Trans-Id-Extra", "sdk-upload"),
        objects.WithCreateObjectHeaders(map[string]string{"Cache-Control": "private"}),
        objects.WithCreateObjectMetadata(map[string]string{"owner": "sdk"}),
        objects.WithCreateObjectMetadataValue("empty", ""),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.MD5 = knownMD5; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.SHA256 = knownSHA256; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.SegmentSize = nil; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.UseSLO = nil; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.GenerateChecksums = nil; return nil }),
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
        request.Header("X-Trace", "stale"),
        objects.WithIsObjectStaleHeaders(map[string]string{"X-Trans-Id-Extra": "sdk-stale"}),
        request.Custom(func(c *objects.IsObjectStaleOpts) error { c.MD5 = knownMD5; return nil }),
        request.Custom(func(c *objects.IsObjectStaleOpts) error { c.SHA256 = knownSHA256; return nil }))
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
        request.Custom(func(c *objects.CreateObjectOpts) error { c.MD5 = ""; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { c.SHA256 = ""; return nil }),
        request.Custom(func(c *objects.CreateObjectOpts) error { v := int64(1<<30); c.SegmentSize = &v; return nil }),
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
- [Connection](../../../connectiontest/connection_objectstorage_create_object_test.go)
- [generator](../../../internal/cmd/sdkgen/swift_object_create_test.go)
