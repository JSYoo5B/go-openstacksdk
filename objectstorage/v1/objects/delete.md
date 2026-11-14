# Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showDelete(result *objects.DeleteObjectResult) {
    if result == nil { return }
    fmt.Println("ignored missing:", result.IgnoredMissing)
    if result.StaticLargeObject == nil {
        fmt.Println("SLO flag unknown")
    } else {
        fmt.Println("SLO:", *result.StaticLargeObject)
    }
    for _, phase := range []struct {
        name string
        response *objects.DeleteObjectResponse
    }{{"HEAD", result.Discovery}, {"DELETE", result.Deletion}} {
        if phase.response != nil {
            fmt.Printf("%s status=%d transaction=%q observed-bytes=%d\n",
                phase.name, phase.response.StatusCode,
                phase.response.Header.Get("X-Trans-Id"), len(phase.response.Body))
        }
    }
    if result.Bulk != nil {
        fmt.Printf("bulk status=%d deleted=%d not-found=%d errors=%d\n",
            result.Bulk.ResponseCode, result.Bulk.NumberDeleted,
            result.Bulk.NumberNotFound, len(result.Bulk.Errors))
    }
}

func showDeleteError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed physical status=%d observed-bytes=%d\n",
            response.StatusCode, len(response.Body))
    }
    var bulk *objects.ObjectDeleteBulkError
    if errors.As(err, &bulk) {
        fmt.Printf("embedded bulk failure=%q errors=%d\n",
            bulk.ResponseStatus, len(bulk.Errors))
    }
}

func deleteObjects(ctx context.Context, api *objects.API, versionID string) error {
    knownSLO, allowMissing, newest := false, false, true
    auto := []objects.DeleteObjectOption{
        objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            StaticLargeObject: &knownSLO, IgnoreMissing: &allowMissing, Newest: &newest,
        }),
        objects.WithDeleteObjectHeaders(map[string]string{"X-Trace-Label": "object-delete"}),
        request.Header("X-Request-Label", "auto"),
        request.Custom(func(c *objects.DeleteObjectOpts) error { c.StaticLargeObject = nil; return nil }),
        request.IgnoreMissing(false),
        request.Custom(func(c *objects.DeleteObjectOpts) error { c.IgnoreMissing = nil; return nil }),
        request.Custom(func(c *objects.DeleteObjectOpts) error { c.VersionID = versionID; return nil }),
        objects.WithDeleteObjectNewest(true),
        request.Custom(func(c *objects.DeleteObjectOpts) error { c.Newest = nil; return nil }),
        objects.WithDeleteObjectNewest(false),
    }
    result, err := api.DeleteObject(ctx, "backups", "report.txt", auto...)
    showDelete(result)
    if err != nil {
        showDeleteError(err)
        return err
    }

    result, err = api.DeleteObject(ctx, "backups", "manifest",
        objects.WithDeleteObjectStaticLargeObject(true),
        request.IgnoreMissing(false),
    )
    showDelete(result)
    if err != nil { showDeleteError(err) }
    return err
}

func main() {}
```

```python
import openstack

conn = openstack.connect()
conn.object_store.delete_object("report.txt", container="backups")
conn.object_store.delete_object("strict-report.txt", container="backups", ignore_missing=False)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L535-L561)
- [Object._raw_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L374-L407)
- [Swift object API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L449-L625)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L455-L480)
- [DELETE default codes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L598)
- [SLO controller](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1801-L1829)
- [bulk outcome](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/bulk.py#L331-L467)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [public HTTP·missing·native hook](delete_contracts_test.go)
- [Connection Objectstorage Object Delete Test](../../../connectiontest/connection_objectstorage_object_delete_test.go)
- [Swift Object Delete Test](../../../internal/cmd/sdkgen/swift_object_delete_test.go)
