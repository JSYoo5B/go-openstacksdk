# Upload Image

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
with open("image.qcow2", "rb") as source:
    created = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="upload-example",
        visibility="private",
        tags=["example"],
    )
print(created.id, created.status)
```

```go
package main

import (
    "context"
    "encoding/json"
    "os"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func uploadFile(ctx context.Context, service *image.Service, filename string,
    knownSize *int64, owner *string) (*image.ImageUploadResult, error) {
    source, err := os.Open(filename)
    if err != nil { return nil, err }
    defer source.Close()

    options := []image.ImageUploadOption{
        image.WithImageUploadOpts(image.ImageUploadOpts{
            Headers: map[string]string{"X-Request-Label": "example"},
            Fields: map[string]json.RawMessage{"build": json.RawMessage(`"initial"`)},
        }),
        image.WithImageUploadHeaders(map[string]string{"X-Trace-Label": "upload"}),
        request.Header("X-Request-Label", "direct-upload"),
        image.WithImageUploadFields(map[string]json.RawMessage{
            "build": json.RawMessage(`"release"`),
        }),
        image.WithImageUploadField("build", "2026-10-03"),
        image.WithImageUploadField("disk_format", "qcow2"),
        image.WithImageUploadField("container_format", "bare"),
        image.WithImageUploadField("visibility", image.VisibilityPrivate),
        image.WithImageUploadField("protected", false),
        image.WithImageUploadField("os_hidden", false),
        image.WithImageUploadField("min_disk", 0),
        image.WithImageUploadField("min_ram", 0),
        image.WithImageUploadTags("example"),
    }
    if owner != nil {
        options = append(options, image.WithImageUploadField("owner", *owner))
    }
    if knownSize == nil {
        options = append(options, request.Custom(func(c *image.ImageUploadOpts) error { c.Size = nil; return nil }))
    } else {
        options = append(options, request.Custom(func(c *image.ImageUploadOpts) error { v := *knownSize; c.Size = &v; return nil }))
    }
    result, err := service.UploadImage(ctx,
        image.UploadImageRequest{Name: "upload-example", Data: source}, options...)
    return result, err
}

func main() {}
```

## References

- [pinned Python `Proxy.upload_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L583)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462)
- [pinned API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-data.inc#L14)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#upload-binary-image-data)
- [wire/evidence](upload_image_contracts_test.go)
- [preflight/snapshot](upload_image_contracts_test.go)
- [Upload Image Contracts Test](upload_image_contracts_test.go)
- [core](upload_image_core_test.go)
- [options](upload_image_options_test.go)
- [Connection](../connectiontest/connection_image_upload_test.go)
- [generator](../internal/cmd/sdkgen/glance_image_upload_test.go)
