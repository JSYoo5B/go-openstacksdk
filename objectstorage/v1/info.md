# Info

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func reportResponseError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed status=%d transaction=%q observed-bytes=%d\n",
            response.StatusCode, response.Header.Get("X-Trans-Id"), len(response.Body))
    }
}

func showSegment(result *objectstorage.ObjectSegmentSizeResult) {
    if result == nil { return }
    fmt.Printf("requested=%d max=%d min=%d selected=%d fallback=%t status=%d raw-bytes=%d\n",
        result.RequestedSize, result.MaxFileSize, result.MinSegmentSize,
        result.Size, result.UsedFallback, result.StatusCode, len(result.Body))
}

func capabilities(ctx context.Context, service *objectstorage.Service) error {
    info, err := service.GetInfo(ctx,
        objectstorage.WithGetInfoOpts(objectstorage.GetInfoOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        objectstorage.WithGetInfoHeaders(map[string]string{"X-Trace-Label": "capabilities"}),
        request.Header("X-Request-Label", "info"),
    )
    if err != nil {
        reportResponseError(err)
        return err
    }
    fmt.Printf("info status=%d transaction=%q\n", info.StatusCode, info.Header.Get("X-Trans-Id"))
    if info.Swift != nil {
        if maximum, ok := info.Swift["max_file_size"]; ok {
            fmt.Println("advertised max_file_size:", string(maximum))
        }
    }
    fmt.Println("SLO section present:", info.SLO != nil)
    if plugin, ok := info.Body["my_plugin"]; ok {
        fmt.Println("raw plugin:", string(plugin))
    }

    large := int64(8 * 1024 * 1024 * 1024)
    defaultOptions := []objectstorage.ObjectSegmentSizeOption{
        objectstorage.WithObjectSegmentSizeOpts(objectstorage.ObjectSegmentSizeOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Size: &large,
        }),
        objectstorage.WithObjectSegmentSizeHeaders(map[string]string{"X-Trace-Label": "segment"}),
        request.Header("X-Request-Label", "default"),
        request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error { v := int64(2 * 1024 * 1024 * 1024); c.Size = &v; return nil }),
        request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error { c.Size = nil; return nil }),
    }
    selected, err := service.GetObjectSegmentSize(ctx, defaultOptions...)
    showSegment(selected)
    if err != nil {
        reportResponseError(err)
        return err
    }

    selected, err = service.GetObjectSegmentSize(ctx,
        request.Custom(func(c *objectstorage.ObjectSegmentSizeOpts) error { v := large; c.Size = &v; return nil }),
    )
    showSegment(selected)
    if err != nil { reportResponseError(err) }
    return err
}

func main() {}
```

```python
import openstack

conn = openstack.connect()
caps = conn.object_store.get_info()
print("Swift capability:", caps.swift)
print("SLO capability:", caps.slo)
print("default selection:", conn.object_store.get_object_segment_size(None))
print("large selection:", conn.object_store.get_object_segment_size(8 * 1024**3))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L860-L919)
- [Info Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/info.py#L25-L96)
- [Swift `/info` API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage_info.inc#L1-L46)
- [Info](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/info.py#L55-L108)
- [effective constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/server.py#L359-L378)
- [Constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/constraints.py#L28-L66)
- [Slo](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1893-L1922)
- [Info Core Test](info_core_test.go)
- [Info Options Test](info_options_test.go)
- [public HTTP·native retry·fallback](info_contracts_test.go)
- [Connection Objectstorage Info Test](../../connectiontest/connection_objectstorage_info_test.go)
- [Swift Info Test](../../internal/cmd/sdkgen/swift_info_test.go)
