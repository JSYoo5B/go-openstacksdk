# Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func updateQueuedImageExample(ctx context.Context, svc *image.Service, imageID, ownerID string) (*image.ImageInfo, error) {
    return svc.UpdateImage(ctx, resource.ID(imageID),
        image.WithUpdateImageOpts(image.UpdateImageOpts{
            Headers: map[string]string{"X-Example": "ordered-image-patch"},
        }),
        image.WithUpdateImageHeaders(map[string]string{"X-Trace-Id": "image-update"}),
        request.Header("X-Request-Id", "image-update-example"),
        image.WithUpdateImageChanges(image.ImagePatch{
            Op: "add", Path: "/build", Value: json.RawMessage(`"draft"`),
        }),
        image.WithUpdateImageChange(image.ImagePatch{
            Op: "replace", Path: "/build", Value: json.RawMessage(`"ready"`),
        }),
        image.WithUpdateImageField("name", "release-image"),
        image.WithUpdateImageField("visibility", "private"),
        image.WithUpdateImageField("protected", false),
        image.WithUpdateImageField("os_hidden", false),
        image.WithUpdateImageField("owner", ownerID),
        image.WithUpdateImageField("container_format", "bare"),
        image.WithUpdateImageField("disk_format", "qcow2"),
        image.WithUpdateImageField("min_disk", 0),
        image.WithUpdateImageField("min_ram", 0),
        image.WithUpdateImageTags("release", "ready"),
        image.WithUpdateImageField("build~origin", "ci"),
        image.WithUpdateImageFields(map[string]json.RawMessage{
            "build/id": json.RawMessage(`"2026.10"`),
            "legacy": json.RawMessage(`"temporary"`),
        }),
        image.WithUpdateImageRemoveField("legacy"),
    )
}

func setImagePropertiesExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.SetImageProperties(ctx, resource.ID(imageID),
        image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{
            Headers: map[string]string{"X-Example": "property-upsert"},
        }),
        image.WithSetImagePropertiesHeaders(map[string]string{"X-Trace-Id": "image-properties"}),
        request.Header("X-Request-Id", "image-properties-example"),
        image.WithSetImagePropertiesProperties(map[string]json.RawMessage{
            "team": json.RawMessage(`"platform"`),
            "release": json.RawMessage(`"2026"`),
        }),
        image.WithSetImagePropertiesProperty("release", "2026.10"),
    )
}
```

```python
from openstack.image.v2.image import Image

def update_image_example(conn, image_id):
    cached = Image.existing(id=image_id, name="old-name", team="platform")
    updated = conn.image.update_image(cached, name="release-image", is_protected=False)
    did_update = conn.image.update_image_properties(
        image=updated, meta={"team": "platform"}, min_ram=0,
    )
    return updated, did_update
```

## References

- [ImageInfo](images.md)
- [generated/native Update](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L559-L682)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L617-L655)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L713)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1406-L1442)
- [parser](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1450-L1551)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2011)
- [`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)
- [native Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/service_client.go#L144-L163)
- [Update Core Test](update_core_test.go)
- [Update Options Test](update_options_test.go)
- [Update Contracts Test](update_contracts_test.go)
- [Connection Image Update Test](../connectiontest/connection_image_update_test.go)
- [Glance Image Update Test](../internal/cmd/sdkgen/glance_image_update_test.go)
