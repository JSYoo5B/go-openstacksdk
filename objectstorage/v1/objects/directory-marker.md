# Directory Marker

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showMarker(result *objects.CreateObjectResult, err error) {
    if result != nil {
        fmt.Printf("source=%s mode=%s size=%d md5=%q sha256=%q\n",
            result.Source, result.Mode, result.Size, result.MD5, result.SHA256)
        if result.Ordinary != nil {
            for _, attempt := range result.Ordinary.Attempts {
                fmt.Printf("logical=%d physical=%d error=%v\n",
                    attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Ordinary.Acknowledgement != nil {
                fmt.Println("acknowledged HTTP", result.Ordinary.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func markerExample(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showMarker(nil, err)
        return
    }
    options := []objects.DirectoryMarkerOption{
        objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{
            Headers: map[string]string{"X-Client": "directory-marker"},
            Metadata: map[string]string{"Owner": "sdk"},
        }),
        request.Header("X-Trace", "initial"),
        objects.WithDirectoryMarkerHeaders(map[string]string{"x-trace": "marker"}),
        objects.WithDirectoryMarkerMetadata(map[string]string{"Purpose": "web-listing"}),
        objects.WithDirectoryMarkerMetadataValue("Build", "2026-10-04"),
    }
    result, err := service.Objects.CreateDirectoryMarkerObject(ctx, "backups", "reports/", options...)
    showMarker(result, err)
}

func main() {}
```

## References

- [pinned cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L231-L258)
- [cloud create_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L260-L321)
- [canonical proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L526)
- [Object.create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L359-L372)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L917-L950)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L48-L66)
- [Obj](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/obj.py#L708-L717)
